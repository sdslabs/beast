package database

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupSubmissionTestDB(t *testing.T) func() {
	t.Helper()

	dsn := os.Getenv("BEAST_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("set BEAST_TEST_PG_DSN to run PostgreSQL submission race tests")
	}

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open admin postgres connection: %v", err)
	}

	schema := fmt.Sprintf("beast_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	testDB, err := gorm.Open(postgres.Open(withSearchPath(dsn, schema)), &gorm.Config{})
	if err != nil {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema)).Error
		t.Fatalf("open test postgres connection: %v", err)
	}

	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(32)
	sqlDB.SetMaxIdleConns(32)

	previousDB := Db
	previousMux := DBMux
	Db = testDB
	DBMux = &sync.Mutex{}

	if err := Db.AutoMigrate(&Challenge{}, &User{}, &UserChallenges{}, &DynamicFlag{}, &DynamicFlagClaim{}, &DynamicScoreDirty{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := MigrateSubmissionGuards(); err != nil {
		t.Fatalf("migrate submission guards: %v", err)
	}

	return func() {
		Db = previousDB
		DBMux = previousMux
		_ = sqlDB.Close()
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema)).Error
		if adminSQL, err := adminDB.DB(); err == nil {
			_ = adminSQL.Close()
		}
	}
}

func withSearchPath(dsn, schema string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			query := parsed.Query()
			query.Set("search_path", schema)
			parsed.RawQuery = query.Encode()
			return parsed.String()
		}
	}

	return dsn + " search_path=" + schema
}

func createSubmissionTestUser(t *testing.T, username string) User {
	t.Helper()

	user := User{
		Name:  username,
		Email: username + "@example.test",
		AuthModel: auth.AuthModel{
			Username: username,
			Role:     core.USER_ROLES["contestant"],
		},
	}
	if err := Db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return user
}

func createSubmissionTestChallenge(t *testing.T, name string, maxAttempts int, dynamic bool) Challenge {
	t.Helper()

	challenge := Challenge{
		Name:            name,
		Type:            "web",
		Difficulty:      "easy",
		Flag:            "flag{correct}",
		DynamicFlag:     dynamic,
		Points:          500,
		MaxPoints:       500,
		MinPoints:       100,
		MaxAttemptLimit: maxAttempts,
		Status:          core.DEPLOY_STATUS["deployed"],
	}
	if err := Db.Create(&challenge).Error; err != nil {
		t.Fatalf("create challenge %s: %v", name, err)
	}
	return challenge
}

func TestConcurrentCorrectSubmissionsAwardOnce(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	user := createSubmissionTestUser(t, "raceuser")
	challenge := createSubmissionTestChallenge(t, "race-correct", -1, false)

	var awards int32
	errCh := make(chan error, 64)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			now := time.Now()
			attempt, err := ReserveSubmissionAttempt(user.ID, challenge.ID, challenge.MaxAttemptLimit, challenge.Flag, now)
			if err != nil {
				errCh <- err
				return
			}
			if attempt.Status != SubmissionAttemptAccepted {
				return
			}

			result, err := FinalizeSubmissionSolve(user.ID, challenge.ID, challenge.Flag, false, now, false)
			if err != nil {
				errCh <- err
				return
			}
			if !result.Awarded {
				return
			}
			atomic.AddInt32(&awards, 1)
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent submission failed: %v", err)
	}
	if got := atomic.LoadInt32(&awards); got != 1 {
		t.Fatalf("expected exactly one score award, got %d", got)
	}

	var submissions []UserChallenges
	if err := Db.Where("user_id = ? AND challenge_id = ?", user.ID, challenge.ID).Find(&submissions).Error; err != nil {
		t.Fatalf("query submissions: %v", err)
	}
	if len(submissions) != 1 {
		t.Fatalf("expected one user_challenges row, got %d", len(submissions))
	}
	if !submissions[0].Solved {
		t.Fatalf("expected submission row to be solved")
	}

	var refreshed User
	if err := Db.First(&refreshed, user.ID).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	if refreshed.Score != challenge.Points {
		t.Fatalf("expected user score %d, got %d", challenge.Points, refreshed.Score)
	}
}

func TestAlreadySolvedSubmissionCannotBeReAwarded(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	user := createSubmissionTestUser(t, "solveduser")
	challenge := createSubmissionTestChallenge(t, "already-solved", -1, false)

	now := time.Now()
	attempt, err := ReserveSubmissionAttempt(user.ID, challenge.ID, challenge.MaxAttemptLimit, "first", now)
	if err != nil {
		t.Fatalf("reserve first attempt: %v", err)
	}
	if attempt.Status != SubmissionAttemptAccepted {
		t.Fatalf("expected first attempt accepted, got %v", attempt.Status)
	}

	first, err := FinalizeSubmissionSolve(user.ID, challenge.ID, "first", false, now, false)
	if err != nil {
		t.Fatalf("finalize first solve: %v", err)
	}
	if !first.Awarded {
		t.Fatalf("expected first solve to award")
	}

	second, err := FinalizeSubmissionSolve(user.ID, challenge.ID, "second", false, time.Now(), false)
	if err != nil {
		t.Fatalf("finalize duplicate solve: %v", err)
	}
	if second.Awarded {
		t.Fatalf("duplicate solve awarded score")
	}

	attempt, err = ReserveSubmissionAttempt(user.ID, challenge.ID, challenge.MaxAttemptLimit, "third", time.Now())
	if err != nil {
		t.Fatalf("reserve after solved: %v", err)
	}
	if attempt.Status != SubmissionAttemptAlreadySolved {
		t.Fatalf("expected already solved status, got %v", attempt.Status)
	}

	var refreshed User
	if err := Db.First(&refreshed, user.ID).Error; err != nil {
		t.Fatalf("query user: %v", err)
	}
	if refreshed.Score != challenge.Points {
		t.Fatalf("expected user score %d, got %d", challenge.Points, refreshed.Score)
	}
}

func TestDynamicScoreFinalizationUsesSignedDelta(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	userA := createSubmissionTestUser(t, "dynamicusera")
	userB := createSubmissionTestUser(t, "dynamicuserb")
	challenge := createSubmissionTestChallenge(t, "dynamic-finalize", -1, true)

	if _, err := ReserveSubmissionAttempt(userA.ID, challenge.ID, challenge.MaxAttemptLimit, "first", time.Now()); err != nil {
		t.Fatalf("reserve first dynamic attempt: %v", err)
	}
	first, err := FinalizeSubmissionSolve(userA.ID, challenge.ID, "first", false, time.Now(), true)
	if err != nil {
		t.Fatalf("finalize first dynamic solve: %v", err)
	}
	if !first.Awarded || first.Points != challenge.MaxPoints {
		t.Fatalf("expected first dynamic solve to award max points, got awarded=%v points=%d", first.Awarded, first.Points)
	}

	if err := Db.Model(&User{}).Where("id = ?", userA.ID).Update("score", uint(1)).Error; err != nil {
		t.Fatalf("force low score: %v", err)
	}

	if _, err := ReserveSubmissionAttempt(userB.ID, challenge.ID, challenge.MaxAttemptLimit, "second", time.Now()); err != nil {
		t.Fatalf("reserve second dynamic attempt: %v", err)
	}
	second, err := FinalizeSubmissionSolve(userB.ID, challenge.ID, "second", false, time.Now(), true)
	if err != nil {
		t.Fatalf("finalize second dynamic solve: %v", err)
	}
	expectedPoints := DynamicChallengeScore(challenge.MaxPoints, challenge.MinPoints, 2)
	if !second.Awarded || second.Points != expectedPoints {
		t.Fatalf("expected second dynamic solve to award %d points, got awarded=%v points=%d", expectedPoints, second.Awarded, second.Points)
	}
	if second.DynamicDelta >= 0 {
		t.Fatalf("expected negative dynamic delta, got %d", second.DynamicDelta)
	}

	var refreshedA, refreshedB User
	if err := Db.First(&refreshedA, userA.ID).Error; err != nil {
		t.Fatalf("query first user: %v", err)
	}
	if err := Db.First(&refreshedB, userB.ID).Error; err != nil {
		t.Fatalf("query second user: %v", err)
	}
	if refreshedA.Score != 0 {
		t.Fatalf("expected negative delta to clamp first user score to 0, got %d", refreshedA.Score)
	}
	if refreshedB.Score != expectedPoints {
		t.Fatalf("expected second user score %d, got %d", expectedPoints, refreshedB.Score)
	}

	var refreshedChallenge Challenge
	if err := Db.First(&refreshedChallenge, challenge.ID).Error; err != nil {
		t.Fatalf("query challenge: %v", err)
	}
	if refreshedChallenge.Points != expectedPoints {
		t.Fatalf("expected challenge points %d, got %d", expectedPoints, refreshedChallenge.Points)
	}
}

func TestConcurrentWrongSubmissionsRespectMaxAttempts(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	user := createSubmissionTestUser(t, "wronguser")
	challenge := createSubmissionTestChallenge(t, "race-wrong", 3, false)

	var accepted int32
	errCh := make(chan error, 64)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			attempt, err := ReserveSubmissionAttempt(user.ID, challenge.ID, challenge.MaxAttemptLimit, fmt.Sprintf("wrong-%d", i), time.Now())
			if err != nil {
				errCh <- err
				return
			}
			if attempt.Status == SubmissionAttemptAccepted {
				atomic.AddInt32(&accepted, 1)
			}
		}(i)
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent wrong submission failed: %v", err)
	}
	if got := atomic.LoadInt32(&accepted); got != int32(challenge.MaxAttemptLimit) {
		t.Fatalf("expected %d accepted attempts, got %d", challenge.MaxAttemptLimit, got)
	}

	var submission UserChallenges
	if err := Db.Where("user_id = ? AND challenge_id = ?", user.ID, challenge.ID).First(&submission).Error; err != nil {
		t.Fatalf("query submission: %v", err)
	}
	if submission.Tries != uint(challenge.MaxAttemptLimit) {
		t.Fatalf("expected tries %d, got %d", challenge.MaxAttemptLimit, submission.Tries)
	}
	if submission.Solved {
		t.Fatalf("wrong submissions must not mark challenge solved")
	}
}

func TestDynamicFlagClaimFirstClaimWins(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	userA := createSubmissionTestUser(t, "claimusera")
	userB := createSubmissionTestUser(t, "claimuserb")
	challenge := createSubmissionTestChallenge(t, "dynamic-claim", -1, true)

	start := make(chan struct{})
	results := make(chan DynamicFlagClaimResult, 2)
	errCh := make(chan error, 2)
	for _, user := range []User{userA, userB} {
		go func(user User) {
			<-start
			result, err := ClaimDynamicFlag(challenge.ID, user.ID, "flag{dynamic}", time.Now())
			if err != nil {
				errCh <- err
				return
			}
			results <- result
		}(user)
	}

	close(start)

	var got []DynamicFlagClaimResult
	for len(got) < 2 {
		select {
		case err := <-errCh:
			t.Fatalf("claim dynamic flag: %v", err)
		case result := <-results:
			got = append(got, result)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for dynamic flag claims")
		}
	}

	created := 0
	claimedByOther := 0
	for _, result := range got {
		switch result.Status {
		case DynamicFlagClaimCreated:
			created++
		case DynamicFlagClaimedByOtherUser:
			claimedByOther++
		default:
			t.Fatalf("unexpected dynamic claim status %v", result.Status)
		}
	}
	if created != 1 || claimedByOther != 1 {
		t.Fatalf("expected one created and one duplicate claim, got created=%d duplicate=%d", created, claimedByOther)
	}

	var claims []DynamicFlagClaim
	if err := Db.Where("challenge_id = ? AND flag = ?", challenge.ID, "flag{dynamic}").Find(&claims).Error; err != nil {
		t.Fatalf("query claims: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected one dynamic flag claim row, got %d", len(claims))
	}
}

func TestDynamicScoreDirtyCoalescesConcurrentMarks(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	challenge := createSubmissionTestChallenge(t, "dynamic-score-dirty", -1, true)
	user := createSubmissionTestUser(t, "dirtyuser")

	errCh := make(chan error, 64)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := MarkDynamicScoreDirty(challenge.ID, user.ID, time.Now()); err != nil {
				errCh <- err
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("mark dynamic score dirty: %v", err)
	}

	dirty, err := QueryDirtyDynamicScores(10)
	if err != nil {
		t.Fatalf("query dirty dynamic scores: %v", err)
	}
	if len(dirty) != 1 {
		t.Fatalf("expected one coalesced dirty-score marker, got %d", len(dirty))
	}
	if dirty[0].ChallengeID != challenge.ID {
		t.Fatalf("expected dirty marker for challenge %d, got %d", challenge.ID, dirty[0].ChallengeID)
	}
}
