package leaderboard

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core/database/models"
	dbrepo "github.com/sdslabs/beastv4/core/database/repository"
	redisrepo "github.com/sdslabs/beastv4/core/redis/repository"
	"gorm.io/gorm"
)

const (
	solveFetchBatchSize    = 1000
	userHintFetchBatchSize = 1000
)

func AddScoreGraphPoint(ctx context.Context, rdb *redis.Client, entries []redisrepo.LeaderboardEntry) error {
	if rdb == nil {
		return redisrepo.ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	return redisrepo.AppendScoreGraphPoints(ctx, rdb, time.Now().UTC(), entries)
}

func RebuildScoreGraphFromSolves(ctx context.Context, db *gorm.DB, rdb *redis.Client, opts CalculateOptions) (map[uuid.UUID][]redisrepo.ScoreGraphPoint, error) {
	if db == nil {
		return nil, dbrepo.ErrNoDatabase
	}
	if rdb == nil {
		return nil, redisrepo.ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	scoringPolicy := "dynamic"
	var startsAt *time.Time
	cfg, err := dbrepo.GetCompetitionConfig(ctx, db)
	if err != nil {
		if !errors.Is(err, dbrepo.ErrNotFound) {
			return nil, err
		}
	} else if cfg != nil {
		if cfg.ScoringPolicy != "" {
			scoringPolicy = cfg.ScoringPolicy
		}
		startsAt = cfg.StartsAt
	}

	teams, err := loadActiveTeams(ctx, db)
	if err != nil {
		return nil, err
	}

	challenges, err := dbrepo.ListChallenges(ctx, db, opts.IncludeHidden)
	if err != nil {
		return nil, err
	}

	solves, err := loadAllSolves(ctx, db)
	if err != nil {
		return nil, err
	}

	hints, err := loadAllUserHints(ctx, db)
	if err != nil {
		return nil, err
	}

	seriesByTeam := buildScoreGraphSeries(scoringPolicy, startsAt, teams, challenges, solves, hints)
	if err := redisrepo.ReplaceScoreGraph(ctx, rdb, seriesByTeam); err != nil {
		return nil, err
	}
	return seriesByTeam, nil
}

type scoreEventKind int

const (
	scoreEventSolve scoreEventKind = iota
	scoreEventHint
)

type scoreEvent struct {
	timestamp   time.Time
	kind        scoreEventKind
	teamID      uuid.UUID
	challengeID uuid.UUID
	costPoints  int
}

func buildScoreGraphSeries(
	scoringPolicy string,
	startsAt *time.Time,
	teams []models.Team,
	challenges []models.Challenge,
	solves []models.Solve,
	hints []models.UserHint,
) map[uuid.UUID][]redisrepo.ScoreGraphPoint {
	activeTeams := make(map[uuid.UUID]struct{}, len(teams))
	for _, team := range teams {
		if !team.IsBanned {
			activeTeams[team.ID] = struct{}{}
		}
	}

	challengesByID := make(map[uuid.UUID]models.Challenge, len(challenges))
	challengePoints := make(map[uuid.UUID]int, len(challenges))
	for _, ch := range challenges {
		challengesByID[ch.ID] = ch
		challengePoints[ch.ID] = ch.MaxPoints
	}

	events := make([]scoreEvent, 0, len(solves)+len(hints))
	for _, solve := range solves {
		if _, ok := activeTeams[solve.TeamID]; !ok {
			continue
		}
		if _, ok := challengesByID[solve.ChallengeID]; !ok {
			continue
		}
		events = append(events, scoreEvent{
			timestamp:   solve.SolvedAt.UTC(),
			kind:        scoreEventSolve,
			teamID:      solve.TeamID,
			challengeID: solve.ChallengeID,
		})
	}
	for _, hint := range hints {
		if _, ok := activeTeams[hint.TeamID]; !ok {
			continue
		}
		if hint.CostPoints == 0 {
			continue
		}
		events = append(events, scoreEvent{
			timestamp:  hint.UnlockedAt.UTC(),
			kind:       scoreEventHint,
			teamID:     hint.TeamID,
			costPoints: hint.CostPoints,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		a, b := events[i], events[j]
		if !a.timestamp.Equal(b.timestamp) {
			return a.timestamp.Before(b.timestamp)
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		if a.teamID != b.teamID {
			return a.teamID.String() < b.teamID.String()
		}
		return a.challengeID.String() < b.challengeID.String()
	})

	seriesByTeam := make(map[uuid.UUID][]redisrepo.ScoreGraphPoint, len(activeTeams))
	if startsAt != nil {
		startTS := startsAt.UTC()
		if len(events) == 0 || !startTS.After(events[0].timestamp) {
			for teamID := range activeTeams {
				seriesByTeam[teamID] = []redisrepo.ScoreGraphPoint{{
					Timestamp: startTS,
					Score:     0,
				}}
			}
		}
	}

	solveCounts := make(map[uuid.UUID]int, len(challengesByID))
	solvedTeamsByChallenge := make(map[uuid.UUID][]uuid.UUID, len(challengesByID))
	teamScores := make(map[uuid.UUID]int, len(activeTeams))

	appendPoint := func(teamID uuid.UUID, ts time.Time, score int) {
		pts := seriesByTeam[teamID]
		if n := len(pts); n > 0 && pts[n-1].Timestamp.Equal(ts) {
			pts[n-1].Score = score
			return
		}
		seriesByTeam[teamID] = append(pts, redisrepo.ScoreGraphPoint{
			Timestamp: ts,
			Score:     score,
		})
	}

	for _, ev := range events {
		switch ev.kind {
		case scoreEventSolve:
			ch := challengesByID[ev.challengeID]
			oldPts := challengePoints[ev.challengeID]
			solveCounts[ev.challengeID]++

			newPts := ch.MaxPoints
			if scoringPolicy != "standard" {
				newPts = CalculateChallengePoints(ch.MaxPoints, ch.MinPoints, ch.DecaySolves, solveCounts[ev.challengeID])
			}
			challengePoints[ev.challengeID] = newPts

			if delta := newPts - oldPts; delta != 0 {
				for _, prevTeamID := range solvedTeamsByChallenge[ev.challengeID] {
					teamScores[prevTeamID] += delta
					appendPoint(prevTeamID, ev.timestamp, teamScores[prevTeamID])
				}
			}

			solvedTeamsByChallenge[ev.challengeID] = append(solvedTeamsByChallenge[ev.challengeID], ev.teamID)
			teamScores[ev.teamID] += newPts
			appendPoint(ev.teamID, ev.timestamp, teamScores[ev.teamID])

		case scoreEventHint:
			teamScores[ev.teamID] -= ev.costPoints
			appendPoint(ev.teamID, ev.timestamp, teamScores[ev.teamID])
		}
	}

	return seriesByTeam
}

func loadAllSolves(ctx context.Context, db *gorm.DB) ([]models.Solve, error) {
	var allSolves []models.Solve
	for offset := 0; ; offset += solveFetchBatchSize {
		batch, err := dbrepo.ListSolves(ctx, db, offset, solveFetchBatchSize)
		if err != nil {
			return nil, err
		}
		allSolves = append(allSolves, batch...)
		if len(batch) < solveFetchBatchSize {
			break
		}
	}
	return allSolves, nil
}

func loadAllUserHints(ctx context.Context, db *gorm.DB) ([]models.UserHint, error) {
	var allHints []models.UserHint
	for offset := 0; ; offset += userHintFetchBatchSize {
		batch, err := dbrepo.ListUserHints(ctx, db, offset, userHintFetchBatchSize)
		if err != nil {
			return nil, err
		}
		allHints = append(allHints, batch...)
		if len(batch) < userHintFetchBatchSize {
			break
		}
	}
	return allHints, nil
}
