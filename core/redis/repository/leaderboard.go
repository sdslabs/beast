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
	keyLeaderboardScores          = "leaderboard:scores"
	keyLeaderboardTeams           = "leaderboard:teams"
	keyLeaderboardChallengePoints = "leaderboard:challenge_points"
	keyLeaderboardUpdatedAt       = "leaderboard:updated_at"
)

type LeaderboardEntry struct {
	Rank        int        `json:"rank"`
	TeamID      uuid.UUID  `json:"team_id"`
	TeamName    string     `json:"team_name"`
	Affiliation *string    `json:"affiliation,omitempty"`
	CountryCode *string    `json:"country_code,omitempty"`
	Score       int        `json:"score"`
	SolveCount  int        `json:"solve_count"`
	LastSolveAt *time.Time `json:"last_solve_at,omitempty"`
}

func SetChallengePoints(ctx context.Context, rdb *redis.Client, points map[uuid.UUID]int) error {
	if rdb == nil {
		return ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	keys, err := coreredis.FormatKeys(keyLeaderboardChallengePoints, keyLeaderboardChallengePoints+":tmp")
	if err != nil {
		return err
	}
	targetKey, tmpKey := keys[0], keys[1]

	if len(points) == 0 {
		return rdb.Del(ctx, targetKey, tmpKey).Err()
	}

	fields := make([]any, 0, len(points)*2)
	for id, pts := range points {
		fields = append(fields, id.String(), pts)
	}

	pipe := rdb.TxPipeline()
	pipe.Del(ctx, tmpKey)
	pipe.HSet(ctx, tmpKey, fields...)
	pipe.Rename(ctx, tmpKey, targetKey)
	_, err = pipe.Exec(ctx)
	return err
}

func GetChallengePoints(ctx context.Context, rdb *redis.Client, challengeID uuid.UUID) (int, bool, error) {
	if rdb == nil {
		return 0, false, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key, err := coreredis.FormatKey(keyLeaderboardChallengePoints)
	if err != nil {
		return 0, false, err
	}

	val, err := rdb.HGet(ctx, key, challengeID.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, false, nil
		}
		return 0, false, err
	}

	pts, err := strconv.Atoi(val)
	if err != nil {
		return 0, false, fmt.Errorf("invalid cached challenge points for %s: %w", challengeID, err)
	}
	return pts, true, nil
}

func GetAllChallengePoints(ctx context.Context, rdb *redis.Client) (map[uuid.UUID]int, error) {
	if rdb == nil {
		return nil, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key, err := coreredis.FormatKey(keyLeaderboardChallengePoints)
	if err != nil {
		return nil, err
	}

	raw, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	res := make(map[uuid.UUID]int, len(raw))
	for idStr, ptsStr := range raw {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid challenge uuid %q in redis cache: %w", idStr, err)
		}
		pts, err := strconv.Atoi(ptsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid challenge points %q for %s: %w", ptsStr, idStr, err)
		}
		res[id] = pts
	}
	return res, nil
}

func SaveLeaderboard(ctx context.Context, rdb *redis.Client, entries []LeaderboardEntry) error {
	if rdb == nil {
		return ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	keys, err := coreredis.FormatKeys(
		keyLeaderboardScores,
		keyLeaderboardTeams,
		keyLeaderboardScores+":tmp",
		keyLeaderboardTeams+":tmp",
		keyLeaderboardUpdatedAt,
	)
	if err != nil {
		return err
	}
	scoresKey, teamsKey := keys[0], keys[1]
	tmpScoresKey, tmpTeamsKey := keys[2], keys[3]
	updatedAtKey := keys[4]

	nowStr := time.Now().UTC().Format(time.RFC3339Nano)

	if len(entries) == 0 {
		pipe := rdb.TxPipeline()
		pipe.Del(ctx, scoresKey, teamsKey, tmpScoresKey, tmpTeamsKey)
		pipe.Set(ctx, updatedAtKey, nowStr, 0)
		_, err = pipe.Exec(ctx)
		return err
	}

	scores := make([]redis.Z, 0, len(entries))
	teamFields := make([]any, 0, len(entries)*2)

	for _, entry := range entries {
		idStr := entry.TeamID.String()
		payload, marshalErr := json.Marshal(entry)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal leaderboard entry for team %s: %w", idStr, marshalErr)
		}
		scores = append(scores, redis.Z{
			Score:  float64(entry.Score),
			Member: idStr,
		})
		teamFields = append(teamFields, idStr, string(payload))
	}

	pipe := rdb.TxPipeline()
	pipe.Del(ctx, tmpScoresKey, tmpTeamsKey)
	pipe.ZAdd(ctx, tmpScoresKey, scores...)
	pipe.HSet(ctx, tmpTeamsKey, teamFields...)
	pipe.Rename(ctx, tmpScoresKey, scoresKey)
	pipe.Rename(ctx, tmpTeamsKey, teamsKey)
	pipe.Set(ctx, updatedAtKey, nowStr, 0)

	_, err = pipe.Exec(ctx)
	return err
}

func GetLeaderboard(ctx context.Context, rdb *redis.Client, offset, limit int) ([]LeaderboardEntry, error) {
	if rdb == nil {
		return nil, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if offset < 0 {
		offset = 0
	}

	keys, err := coreredis.FormatKeys(keyLeaderboardScores, keyLeaderboardTeams)
	if err != nil {
		return nil, err
	}
	scoresKey, teamsKey := keys[0], keys[1]

	stop := int64(-1)
	if limit > 0 {
		stop = int64(offset + limit - 1)
	}

	ids, err := rdb.ZRevRange(ctx, scoresKey, int64(offset), stop).Result()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []LeaderboardEntry{}, nil
	}

	rawEntries, err := rdb.HMGet(ctx, teamsKey, ids...).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, 0, len(rawEntries))
	for i, raw := range rawEntries {
		if raw == nil {
			continue
		}
		str, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected redis value type for team %s: %T", ids[i], raw)
		}
		var entry LeaderboardEntry
		if err := json.Unmarshal([]byte(str), &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal leaderboard entry for team %s: %w", ids[i], err)
		}
		entry.Rank = offset + i + 1
		entries = append(entries, entry)
	}
	return entries, nil
}

func GetTeamLeaderboardEntry(ctx context.Context, rdb *redis.Client, teamID uuid.UUID) (*LeaderboardEntry, bool, error) {
	if rdb == nil {
		return nil, false, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	keys, err := coreredis.FormatKeys(keyLeaderboardTeams, keyLeaderboardScores)
	if err != nil {
		return nil, false, err
	}
	teamsKey, scoresKey := keys[0], keys[1]

	idStr := teamID.String()
	raw, err := rdb.HGet(ctx, teamsKey, idStr).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var entry LeaderboardEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal leaderboard entry for team %s: %w", teamID, err)
	}

	rank, err := rdb.ZRevRank(ctx, scoresKey, idStr).Result()
	if err == nil {
		entry.Rank = int(rank) + 1
	} else if !errors.Is(err, redis.Nil) {
		return nil, false, err
	}

	return &entry, true, nil
}

func GetTeamScore(ctx context.Context, rdb *redis.Client, teamID uuid.UUID) (int, bool, error) {
	if rdb == nil {
		return 0, false, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key, err := coreredis.FormatKey(keyLeaderboardScores)
	if err != nil {
		return 0, false, err
	}

	score, err := rdb.ZScore(ctx, key, teamID.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return int(score), true, nil
}

func GetLeaderboardUpdatedAt(ctx context.Context, rdb *redis.Client) (*time.Time, bool, error) {
	if rdb == nil {
		return nil, false, ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key, err := coreredis.FormatKey(keyLeaderboardUpdatedAt)
	if err != nil {
		return nil, false, err
	}

	raw, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}

	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, false, fmt.Errorf("invalid leaderboard updated_at timestamp %q: %w", raw, err)
	}
	return &ts, true, nil
}

func ClearLeaderboard(ctx context.Context, rdb *redis.Client) error {
	if rdb == nil {
		return ErrNoRedis
	}
	if ctx == nil {
		ctx = context.Background()
	}

	keys, err := coreredis.FormatKeys(
		keyLeaderboardScores,
		keyLeaderboardTeams,
		keyLeaderboardChallengePoints,
		keyLeaderboardUpdatedAt,
	)
	if err != nil {
		return err
	}
	return rdb.Del(ctx, keys...).Err()
}
