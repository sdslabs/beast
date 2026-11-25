package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	coreredis "github.com/sdslabs/beastv4/core/redis"
)

const (
	keyScoreGraphTeams      = "leaderboard:graph:teams"
	keyScoreGraphTeamPrefix = "leaderboard:graph:team:"
)

type ScoreGraphPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
}

func scoreGraphTeamKey(teamID uuid.UUID) (string, error) {
	return coreredis.FormatKey(keyScoreGraphTeamPrefix + teamID.String())
}

func AppendScoreGraphPoints(ctx context.Context, rdb *redis.Client, timestamp time.Time, entries []LeaderboardEntry) error {
	if rdb == nil {
		return ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if len(entries) == 0 {
		return nil
	}

	teamsKey, err := coreredis.FormatKey(keyScoreGraphTeams)
	if err != nil {
		return err
	}

	ts := timestamp.UTC()
	score := float64(ts.UnixNano())

	pipe := rdb.TxPipeline()
	teamMembers := make([]any, 0, len(entries))

	for _, entry := range entries {
		teamKey, keyErr := scoreGraphTeamKey(entry.TeamID)
		if keyErr != nil {
			return keyErr
		}
		payload, marshalErr := json.Marshal(ScoreGraphPoint{
			Timestamp: ts,
			Score:     entry.Score,
		})
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal score graph point for team %s: %w", entry.TeamID, marshalErr)
		}

		teamMembers = append(teamMembers, entry.TeamID.String())
		pipe.ZAdd(ctx, teamKey, redis.Z{
			Score:  score,
			Member: string(payload),
		})
	}

	pipe.SAdd(ctx, teamsKey, teamMembers...)
	_, err = pipe.Exec(ctx)
	return err
}

func ReplaceScoreGraph(ctx context.Context, rdb *redis.Client, seriesByTeam map[uuid.UUID][]ScoreGraphPoint) error {
	if rdb == nil {
		return ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	keys, err := coreredis.FormatKeys(keyScoreGraphTeams, keyScoreGraphTeams+":tmp")
	if err != nil {
		return err
	}
	teamsKey, tmpTeamsKey := keys[0], keys[1]

	existingIDs, err := rdb.SMembers(ctx, teamsKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	pipe := rdb.TxPipeline()
	pipe.Del(ctx, tmpTeamsKey)

	newIDs := make(map[string]struct{}, len(seriesByTeam))
	teamMembers := make([]any, 0, len(seriesByTeam))

	for teamID, points := range seriesByTeam {
		idStr := teamID.String()
		teamKey, keyErr := scoreGraphTeamKey(teamID)
		if keyErr != nil {
			return keyErr
		}
		tmpTeamKey, tmpKeyErr := coreredis.FormatKey(keyScoreGraphTeamPrefix + idStr + ":tmp")
		if tmpKeyErr != nil {
			return tmpKeyErr
		}

		if len(points) == 0 {
			pipe.Del(ctx, teamKey, tmpTeamKey)
			continue
		}

		newIDs[idStr] = struct{}{}
		teamMembers = append(teamMembers, idStr)

		zs := make([]redis.Z, 0, len(points))
		for _, pt := range points {
			ts := pt.Timestamp.UTC()
			payload, marshalErr := json.Marshal(ScoreGraphPoint{
				Timestamp: ts,
				Score:     pt.Score,
			})
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal score graph point for team %s: %w", idStr, marshalErr)
			}
			zs = append(zs, redis.Z{
				Score:  float64(ts.UnixNano()),
				Member: string(payload),
			})
		}

		pipe.Del(ctx, tmpTeamKey)
		pipe.ZAdd(ctx, tmpTeamKey, zs...)
		pipe.Rename(ctx, tmpTeamKey, teamKey)
	}

	for _, oldIDStr := range existingIDs {
		if _, kept := newIDs[oldIDStr]; kept {
			continue
		}
		staleKey, keyErr := coreredis.FormatKey(keyScoreGraphTeamPrefix + oldIDStr)
		if keyErr != nil {
			return keyErr
		}
		pipe.Del(ctx, staleKey)
	}

	if len(teamMembers) > 0 {
		pipe.SAdd(ctx, tmpTeamsKey, teamMembers...)
		pipe.Rename(ctx, tmpTeamsKey, teamsKey)
	} else {
		pipe.Del(ctx, teamsKey)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func GetTeamScoreGraph(ctx context.Context, rdb *redis.Client, teamID uuid.UUID, before *time.Time) ([]ScoreGraphPoint, error) {
	if rdb == nil {
		return nil, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	teamKey, err := scoreGraphTeamKey(teamID)
	if err != nil {
		return nil, err
	}

	maxScore := "+inf"
	if before != nil {
		maxScore = strconv.FormatInt(before.UTC().UnixNano(), 10)
	}

	rawPoints, err := rdb.ZRangeByScore(ctx, teamKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: maxScore,
	}).Result()
	if err != nil {
		return nil, err
	}

	points := make([]ScoreGraphPoint, 0, len(rawPoints))
	for _, raw := range rawPoints {
		var pt ScoreGraphPoint
		if err := json.Unmarshal([]byte(raw), &pt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal score graph point for team %s: %w", teamID, err)
		}
		points = append(points, pt)
	}
	return points, nil
}

func GetScoreGraph(ctx context.Context, rdb *redis.Client, teamIDs []uuid.UUID, before *time.Time) (map[uuid.UUID][]ScoreGraphPoint, error) {
	if rdb == nil {
		return nil, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if len(teamIDs) == 0 {
		teamsKey, err := coreredis.FormatKey(keyScoreGraphTeams)
		if err != nil {
			return nil, err
		}
		rawIDs, err := rdb.SMembers(ctx, teamsKey).Result()
		if err != nil {
			return nil, err
		}
		teamIDs = make([]uuid.UUID, 0, len(rawIDs))
		for _, idStr := range rawIDs {
			id, parseErr := uuid.Parse(idStr)
			if parseErr != nil {
				return nil, fmt.Errorf("invalid team uuid %q in score graph: %w", idStr, parseErr)
			}
			teamIDs = append(teamIDs, id)
		}
	}

	res := make(map[uuid.UUID][]ScoreGraphPoint, len(teamIDs))
	if len(teamIDs) == 0 {
		return res, nil
	}

	maxScore := "+inf"
	if before != nil {
		maxScore = strconv.FormatInt(before.UTC().UnixNano(), 10)
	}

	pipe := rdb.Pipeline()
	cmds := make(map[uuid.UUID]*redis.StringSliceCmd, len(teamIDs))
	for _, id := range teamIDs {
		teamKey, err := scoreGraphTeamKey(id)
		if err != nil {
			return nil, err
		}
		cmds[id] = pipe.ZRangeByScore(ctx, teamKey, &redis.ZRangeBy{
			Min: "-inf",
			Max: maxScore,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	for id, cmd := range cmds {
		rawPoints, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		points := make([]ScoreGraphPoint, 0, len(rawPoints))
		for _, raw := range rawPoints {
			var pt ScoreGraphPoint
			if err := json.Unmarshal([]byte(raw), &pt); err != nil {
				return nil, fmt.Errorf("failed to unmarshal score graph point for team %s: %w", id, err)
			}
			points = append(points, pt)
		}
		res[id] = points
	}

	return res, nil
}

func ClearScoreGraph(ctx context.Context, rdb *redis.Client) error {
	if rdb == nil {
		return ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	teamsKey, err := coreredis.FormatKey(keyScoreGraphTeams)
	if err != nil {
		return err
	}

	idStrs, err := rdb.SMembers(ctx, teamsKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	keysToDelete := make([]string, 0, len(idStrs)+1)
	keysToDelete = append(keysToDelete, teamsKey)
	for _, idStr := range idStrs {
		teamKey, keyErr := coreredis.FormatKey(keyScoreGraphTeamPrefix + idStr)
		if keyErr != nil {
			return keyErr
		}
		keysToDelete = append(keysToDelete, teamKey)
	}

	return rdb.Del(ctx, keysToDelete...).Err()
}
