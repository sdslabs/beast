package leaderboard

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core/database/models"
	dbrepo "github.com/sdslabs/beastv4/core/database/repository"
	redisrepo "github.com/sdslabs/beastv4/core/redis/repository"
	"gorm.io/gorm"
)

const teamFetchBatchSize = 500

type CalculateOptions struct {
	IncludeHidden bool
}

func CalculateChallengePoints(maxPoints, minPoints, decaySolves, solves int) int {
	if maxPoints < minPoints {
		return minPoints
	}
	if solves <= 0 || maxPoints == minPoints {
		return maxPoints
	}
	if decaySolves <= 0 || solves >= decaySolves {
		return minPoints
	}

	diff := float64(minPoints - maxPoints)
	ratio := float64(solves) / float64(decaySolves)
	value := diff*(ratio*ratio) + float64(maxPoints)
	points := int(math.Ceil(value))

	if points < minPoints {
		return minPoints
	}
	if points > maxPoints {
		return maxPoints
	}
	return points
}

func UpdateLeaderboard(ctx context.Context, db *gorm.DB, rdb *redis.Client) error {
	if db == nil {
		return dbrepo.ErrNoDatabase
	}
	if rdb == nil {
		return redisrepo.ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	challengePoints, entries, err := CalculateLeaderboard(ctx, db, rdb, CalculateOptions{})
	if err != nil {
		return err
	}

	if err := redisrepo.SetChallengePoints(ctx, rdb, challengePoints); err != nil {
		return err
	}
	return redisrepo.SaveLeaderboard(ctx, rdb, entries)
}

func CalculateLeaderboard(ctx context.Context, db *gorm.DB, rdb *redis.Client, opts CalculateOptions) (map[uuid.UUID]int, []redisrepo.LeaderboardEntry, error) {
	if db == nil {
		return nil, nil, dbrepo.ErrNoDatabase
	}
	if ctx == nil {
		ctx = context.Background()
	}

	scoringPolicy := "dynamic"
	cfg, err := dbrepo.GetCompetitionConfig(ctx, db)
	if err != nil {
		if !errors.Is(err, dbrepo.ErrNotFound) {
			return nil, nil, err
		}
	} else if cfg != nil && cfg.ScoringPolicy != "" {
		scoringPolicy = cfg.ScoringPolicy
	}

	challenges, err := dbrepo.ListChallenges(ctx, db, opts.IncludeHidden)
	if err != nil {
		return nil, nil, err
	}

	solveCounts, err := dbrepo.ListSolveCountsByChallenge(ctx, db)
	if err != nil {
		return nil, nil, err
	}

	challengePoints := make(map[uuid.UUID]int, len(challenges))
	for _, ch := range challenges {
		if scoringPolicy == "standard" {
			challengePoints[ch.ID] = ch.MaxPoints
			continue
		}
		challengePoints[ch.ID] = CalculateChallengePoints(ch.MaxPoints, ch.MinPoints, ch.DecaySolves, int(solveCounts[ch.ID]))
	}

	teams, err := loadActiveTeams(ctx, db)
	if err != nil {
		return nil, nil, err
	}

	hintCostsByTeam, err := dbrepo.ListHintCostTotalsPerTeam(ctx, db)
	if err != nil {
		return nil, nil, err
	}

	entries := make([]redisrepo.LeaderboardEntry, 0, len(teams))
	for _, team := range teams {
		teamSolves, err := dbrepo.ListSolvesByTeamID(ctx, db, team.ID)
		if err != nil {
			return nil, nil, err
		}

		score := 0
		solveCount := 0
		var lastSolveAt *time.Time

		for _, solve := range teamSolves {
			pts, visible := challengePoints[solve.ChallengeID]
			if !visible {
				continue
			}
			score += pts
			solveCount++
			t := solve.SolvedAt
			lastSolveAt = &t
		}

		entries = append(entries, redisrepo.LeaderboardEntry{
			TeamID:      team.ID,
			TeamName:    team.Name,
			Affiliation: team.Affiliation,
			CountryCode: team.CountryCode,
			Score:       score - int(hintCostsByTeam[team.ID]),
			SolveCount:  solveCount,
			LastSolveAt: lastSolveAt,
		})
	}

	if rdb != nil {
		if err := AddScoreGraphPoint(ctx, rdb, challengePoints, entries); err != nil {
			return nil, nil, err
		}
	}

	return challengePoints, entries, nil
}

func loadActiveTeams(ctx context.Context, db *gorm.DB) ([]models.Team, error) {
	var teams []models.Team
	for offset := 0; ; offset += teamFetchBatchSize {
		batch, err := dbrepo.ListTeams(ctx, db, offset, teamFetchBatchSize)
		if err != nil {
			return nil, err
		}
		for _, team := range batch {
			if !team.IsBanned {
				teams = append(teams, team)
			}
		}
		if len(batch) < teamFetchBatchSize {
			break
		}
	}
	return teams, nil
}
