package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
)

func setupHandlerTestEnv(t *testing.T) func() {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cleanupDB := setupTestDB(t)

	// Ensure extra tables used by submit, check, notification, otp handlers exist
	if err := database.Db.AutoMigrate(
		&database.DynamicFlag{},
		&database.DynamicFlagClaim{},
		&database.DynamicScoreDirty{},
		&database.Notification{},
		&database.OTP{},
	); err != nil {
		cleanupDB()
		t.Fatalf("auto migrate extra tables: %v", err)
	}
	if err := database.MigrateSubmissionGuards(); err != nil {
		cleanupDB()
		t.Fatalf("migrate submission guards: %v", err)
	}

	prevCfg := config.Cfg
	prevSkip := config.SkipAuthorization
	prevCache := cache.Cache
	prevCacheMux := cache.CacheMutex

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	cache.CacheMutex = &sync.Mutex{}
	cache.Cache = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	config.Cfg = &config.BeastConfig{
		JWTSecret: "secret-test-jwt-key",
		CompetitionInfo: config.CompetitionInfo{
			Name:         "test-ctf",
			StartingTime: "00:00:00 UTC: +05:30, 1 January 2020, Wednesday",
			EndingTime:   "23:59:59 UTC: +05:30, 1 January 2035, Monday",
			TimeZone:     "Asia/Calcutta: UTC +05:30",
			DynamicScore: true,
		},
	}
	config.SkipAuthorization = false
	invalidateLeaderboardCaches()

	return func() {
		processDirtyDynamicScores()
		dynamicScoreWorkerMu.Lock()
		config.Cfg = prevCfg
		config.SkipAuthorization = prevSkip
		if cache.Cache != nil && cache.Cache != prevCache {
			_ = cache.Cache.Close()
		}
		cache.Cache = prevCache
		cache.CacheMutex = prevCacheMux
		invalidateLeaderboardCaches()
		cleanupDB()
		dynamicScoreWorkerMu.Unlock()
	}
}

func TestSubmitFlagHandlerAndDynamicScoreOnStaticFlagChallenge(t *testing.T) {
	cleanup := setupHandlerTestEnv(t)
	defer cleanup()

	author := database.User{
		Name:      "author1",
		Email:     "author1@example.test",
		AuthModel: auth.CreateModel("author1", "pw", core.USER_ROLES["author"]),
	}
	if err := database.Db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	contestant := database.User{
		Name:      "player1",
		Email:     "player1@example.test",
		AuthModel: auth.CreateModel("player1", "pw", core.USER_ROLES["contestant"]),
	}
	if err := database.Db.Create(&contestant).Error; err != nil {
		t.Fatalf("create contestant: %v", err)
	}
	token, err := auth.GenerateJWT(contestant.AuthModel)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	chall := database.Challenge{
		Name:            "static-dyn-score",
		AuthorID:        author.ID,
		DynamicFlag:     false,
		Flag:            "flag{ok}",
		Type:            "static",
		Format:          "static",
		Difficulty:      "easy",
		MaxAttemptLimit: -1,
		Status:          core.DEPLOY_STATUS["deployed"],
		ContainerId:     "cid-static-dyn",
		ImageId:         "img-static-dyn",
		Points:          500,
		MaxPoints:       500,
		MinPoints:       100,
	}
	if err := database.Db.Create(&chall).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	router := gin.New()
	router.POST("/api/submit/challenge", submitFlagHandler)

	form := url.Values{}
	form.Set("chall_id", fmt.Sprintf("%d", chall.ID))
	form.Set("flag", "flag{ok}")

	req := httptest.NewRequest(http.MethodPost, "/api/submit/challenge", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp FlagSubmitResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, got %+v", resp)
	}

	// Verify recomputeDynamicScore works when DynamicFlag is false
	if err := recomputeDynamicScore(database.DynamicScoreDirty{ChallengeID: chall.ID}); err != nil {
		t.Fatalf("recomputeDynamicScore: %v", err)
	}

	var updatedChall database.Challenge
	if err := database.Db.First(&updatedChall, chall.ID).Error; err != nil {
		t.Fatalf("reload challenge: %v", err)
	}
	if updatedChall.Points != 500 {
		t.Fatalf("expected points 500 with 1 solve, got %d", updatedChall.Points)
	}

	// Simulate second solver and verify points drop cleanly
	contestant2 := database.User{
		Name:      "player2",
		Email:     "player2@example.test",
		AuthModel: auth.CreateModel("player2", "pw", core.USER_ROLES["contestant"]),
	}
	if err := database.Db.Create(&contestant2).Error; err != nil {
		t.Fatalf("create contestant2: %v", err)
	}
	token2, _ := auth.GenerateJWT(contestant2.AuthModel)
	req2 := httptest.NewRequest(http.MethodPost, "/api/submit/challenge", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected second submit 200 OK, got %d: %s", w2.Code, w2.Body.String())
	}

	if err := recomputeDynamicScore(database.DynamicScoreDirty{ChallengeID: chall.ID}); err != nil {
		t.Fatalf("recomputeDynamicScore after 2 solves: %v", err)
	}
	if err := database.Db.First(&updatedChall, chall.ID).Error; err != nil {
		t.Fatalf("reload challenge: %v", err)
	}
	if updatedChall.Points >= 500 || updatedChall.Points < 100 {
		t.Fatalf("expected points between 100 and 499 after 2 solves, got %d", updatedChall.Points)
	}
}

func TestCheckFlagHandlerMissingChallengeAndMissingInstance(t *testing.T) {
	cleanup := setupHandlerTestEnv(t)
	defer cleanup()

	config.SkipAuthorization = true

	defaultUser := database.User{
		Name:      core.DEFAULT_USER_NAME,
		Email:     "default-noauth@example.test",
		AuthModel: auth.CreateModel(core.DEFAULT_USER_NAME, "pw", core.USER_ROLES["contestant"]),
	}
	if err := database.Db.Create(&defaultUser).Error; err != nil {
		t.Fatalf("create default user: %v", err)
	}

	chall := database.Challenge{
		Name:            "check-instanced",
		AuthorID:        defaultUser.ID,
		DynamicFlag:     false,
		Flag:            "check.sh",
		Type:            "service",
		Format:          "service",
		MaxAttemptLimit: 3,
		Status:          core.DEPLOY_STATUS["deployed"],
		Instanced:       true,
		ContainerId:     "cid-check-inst",
		ImageId:         "img-check-inst",
		Points:          200,
	}
	if err := database.Db.Create(&chall).Error; err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	router := gin.New()
	router.POST("/api/check/challenge", checkFlagHandler)

	// Missing challenge should return 404 (not panic)
	formMissing := url.Values{}
	formMissing.Set("chall_id", "99999")
	formMissing.Set("instance_id", "does-not-exist")
	req1 := httptest.NewRequest(http.MethodPost, "/api/check/challenge", strings.NewReader(formMissing.Encode()))
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing challenge, got %d: %s", w1.Code, w1.Body.String())
	}

	// Missing instance should return 404 without consuming an attempt (provided Redis is available; if Redis isn't reachable skip instance check)
	formInst := url.Values{}
	formInst.Set("chall_id", fmt.Sprintf("%d", chall.ID))
	formInst.Set("instance_id", "missing-instance-id")
	req2 := httptest.NewRequest(http.MethodPost, "/api/check/challenge", strings.NewReader(formInst.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing instance, got %d: %s", w2.Code, w2.Body.String())
	}

	var triesCount int64
	database.Db.Model(&database.UserChallenges{}).Where("user_id = ? AND challenge_id = ?", defaultUser.ID, chall.ID).Count(&triesCount)
	if triesCount != 0 {
		t.Fatalf("expected 0 consumed attempts on missing instance, got %d", triesCount)
	}
}

func TestChallengesMetadataFiltersAndNotificationDeleteAndOTPNoAuth(t *testing.T) {
	cleanup := setupHandlerTestEnv(t)
	defer cleanup()

	config.SkipAuthorization = true

	author := database.User{
		Name:      "Author One",
		Email:     "authorone@example.test",
		AuthModel: auth.CreateModel("authorone", "pw", core.USER_ROLES["author"]),
	}
	if err := database.Db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	defaultUser := database.User{
		Name:      core.DEFAULT_USER_NAME,
		Email:     "default-filter@example.test",
		AuthModel: auth.CreateModel(core.DEFAULT_USER_NAME, "pw", core.USER_ROLES["contestant"]),
	}
	if err := database.Db.Create(&defaultUser).Error; err != nil {
		t.Fatalf("create default user: %v", err)
	}

	c1 := database.Challenge{
		Name:        "filter-match",
		AuthorID:    author.ID,
		Type:        "web",
		Status:      core.DEPLOY_STATUS["deployed"],
		ContainerId: "cid-f1",
		ImageId:     "img-f1",
		Points:      350,
	}
	c2 := database.Challenge{
		Name:        "filter-other",
		AuthorID:    defaultUser.ID,
		Type:        "web",
		Status:      core.DEPLOY_STATUS["deployed"],
		ContainerId: "cid-f2",
		ImageId:     "img-f2",
		Points:      100,
	}
	if err := database.Db.Create(&c1).Error; err != nil {
		t.Fatalf("create c1: %v", err)
	}
	if err := database.Db.Create(&c2).Error; err != nil {
		t.Fatalf("create c2: %v", err)
	}

	n := database.Notification{Title: "hello", Description: "world"}
	if err := database.Db.Create(&n).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}

	router := gin.New()
	router.GET("/api/info/challenges", challengesMetadataHandler)
	router.DELETE("/api/notification/delete", removeNotification)
	router.POST("/auth/send-otp", sendOTPHandler)

	// 1. challengesMetadataHandler filter by author and by score
	reqAuthor := httptest.NewRequest(http.MethodGet, "/api/info/challenges?filter=author&value=authorone", nil)
	wAuthor := httptest.NewRecorder()
	router.ServeHTTP(wAuthor, reqAuthor)
	if wAuthor.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from author filter, got %d: %s", wAuthor.Code, wAuthor.Body.String())
	}
	var authorList []ChallengeMetadata
	if err := json.Unmarshal(wAuthor.Body.Bytes(), &authorList); err != nil {
		t.Fatalf("unmarshal author metadata: %v", err)
	}
	if len(authorList) != 1 || authorList[0].Name != "filter-match" {
		t.Fatalf("expected [filter-match] for author filter, got %+v", authorList)
	}

	reqScore := httptest.NewRequest(http.MethodGet, "/api/info/challenges?filter=score&value=350", nil)
	wScore := httptest.NewRecorder()
	router.ServeHTTP(wScore, reqScore)
	if wScore.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from score filter, got %d: %s", wScore.Code, wScore.Body.String())
	}
	var scoreList []ChallengeMetadata
	if err := json.Unmarshal(wScore.Body.Bytes(), &scoreList); err != nil {
		t.Fatalf("unmarshal score metadata: %v", err)
	}
	if len(scoreList) != 1 || scoreList[0].Name != "filter-match" {
		t.Fatalf("expected [filter-match] for score filter, got %+v", scoreList)
	}

	// 2. removeNotification 404 for nonexistent ID, 200 for JSON body ID
	reqDelMissing := httptest.NewRequest(http.MethodDelete, "/api/notification/delete?id=999999", nil)
	wDelMissing := httptest.NewRecorder()
	router.ServeHTTP(wDelMissing, reqDelMissing)
	if wDelMissing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 deleting missing notification, got %d: %s", wDelMissing.Code, wDelMissing.Body.String())
	}

	jsonBody, _ := json.Marshal(map[string]uint{"id": n.ID})
	reqDelOK := httptest.NewRequest(http.MethodDelete, "/api/notification/delete", bytes.NewReader(jsonBody))
	reqDelOK.Header.Set("Content-Type", "application/json")
	wDelOK := httptest.NewRecorder()
	router.ServeHTTP(wDelOK, reqDelOK)
	if wDelOK.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting notification, got %d: %s", wDelOK.Code, wDelOK.Body.String())
	}

	// 3. sendOTPHandler returns 200 JSON in no-auth (-n) mode without SMTP server
	otpForm := url.Values{}
	otpForm.Set("email", "newuser@example.test")
	reqOTP := httptest.NewRequest(http.MethodPost, "/auth/send-otp", strings.NewReader(otpForm.Encode()))
	reqOTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wOTP := httptest.NewRecorder()
	router.ServeHTTP(wOTP, reqOTP)
	if wOTP.Code != http.StatusOK {
		t.Fatalf("expected 200 from sendOTPHandler in SkipAuthorization mode, got %d: %s", wOTP.Code, wOTP.Body.String())
	}

	// Ensure unused time import isn't flagged if not needed; check time.Now()
	_ = time.Now()
}

func TestServeAssetsSecurityAndRoots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpGlobal := t.TempDir()
	prevGlobal := core.BEAST_GLOBAL_DIR
	core.BEAST_GLOBAL_DIR = tmpGlobal
	defer func() { core.BEAST_GLOBAL_DIR = prevGlobal }()

	stagingDir := filepath.Join(tmpGlobal, core.BEAST_STAGING_DIR, "test-chall")
	staticDir := filepath.Join(stagingDir, core.BEAST_STATIC_FOLDER)
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatalf("mkdir static: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "beast.toml"), []byte("secret"), 0644); err != nil {
		t.Fatalf("write beast.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, ".env"), []byte("KEY=secret"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "docker-compose.yml"), []byte("services:{}"), 0644); err != nil {
		t.Fatalf("write docker-compose.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "secret.txt"), []byte("should not be served"), 0644); err != nil {
		t.Fatalf("write secret.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "readme.txt"), []byte("public ok"), 0644); err != nil {
		t.Fatalf("write readme.txt: %v", err)
	}

	router := gin.New()
	router.GET("/api/info/download", serveAssets)

	for _, asset := range []string{"beast.toml", ".env", "docker-compose.yml"} {
		req := httptest.NewRequest(http.MethodGet, "/api/info/download?challenge=test-chall&asset="+url.QueryEscape(asset), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for sensitive asset %q, got %d: %s", asset, w.Code, w.Body.String())
		}
	}

	reqRoot := httptest.NewRequest(http.MethodGet, "/api/info/download?challenge=test-chall&asset=secret.txt", nil)
	wRoot := httptest.NewRecorder()
	router.ServeHTTP(wRoot, reqRoot)
	if wRoot.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for file only in staging root, got %d: %s", wRoot.Code, wRoot.Body.String())
	}

	reqOK := httptest.NewRequest(http.MethodGet, "/api/info/download?challenge=test-chall&asset=readme.txt", nil)
	wOK := httptest.NewRecorder()
	router.ServeHTTP(wOK, reqOK)
	if wOK.Code != http.StatusOK || wOK.Body.String() != "public ok" {
		t.Fatalf("expected 200 with 'public ok', got %d: %s", wOK.Code, wOK.Body.String())
	}
}

func TestUpdateCompetitionInfoPartialUpdate(t *testing.T) {
	cleanup := setupHandlerTestEnv(t)
	defer cleanup()

	tmpGlobal := t.TempDir()
	prevGlobal := core.BEAST_GLOBAL_DIR
	core.BEAST_GLOBAL_DIR = tmpGlobal
	defer func() { core.BEAST_GLOBAL_DIR = prevGlobal }()

	initialToml := `
[competition_info]
name = "Original CTF"
about = "Original About"
prizes = "First Prize"
starting_time = "00:00:00 UTC: +05:30, 1 January 2020, Wednesday"
ending_time = "23:59:59 UTC: +05:30, 1 January 2035, Monday"
timezone = "Asia/Calcutta: UTC +05:30"
dynamic_score = true
`
	if err := os.WriteFile(filepath.Join(tmpGlobal, core.BEAST_CONFIG_FILE_NAME), []byte(initialToml), 0644); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}
	config.Cfg.CompetitionInfo = config.CompetitionInfo{
		Name:         "Original CTF",
		About:        "Original About",
		Prizes:       "First Prize",
		StartingTime: "00:00:00 UTC: +05:30, 1 January 2020, Wednesday",
		EndingTime:   "23:59:59 UTC: +05:30, 1 January 2035, Monday",
		TimeZone:     "Asia/Calcutta: UTC +05:30",
		DynamicScore: true,
	}

	router := gin.New()
	router.POST("/api/config/competition-info", updateCompetitionInfoHandler)

	form := url.Values{}
	form.Set("about", "Updated About")
	req := httptest.NewRequest(http.MethodPost, "/api/config/competition-info", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	got := config.Cfg.CompetitionInfo
	if got.About != "Updated About" || got.Name != "Original CTF" || got.Prizes != "First Prize" ||
		got.StartingTime != "00:00:00 UTC: +05:30, 1 January 2020, Wednesday" ||
		got.EndingTime != "23:59:59 UTC: +05:30, 1 January 2035, Monday" ||
		got.TimeZone != "Asia/Calcutta: UTC +05:30" || !got.DynamicScore {
		t.Fatalf("partial update corrupted fields: %+v", got)
	}

	form2 := url.Values{}
	form2.Set("dynamic_score", "false")
	req2 := httptest.NewRequest(http.MethodPost, "/api/config/competition-info", strings.NewReader(form2.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for dynamic_score update, got %d: %s", w2.Code, w2.Body.String())
	}
	if config.Cfg.CompetitionInfo.DynamicScore {
		t.Fatalf("expected DynamicScore=false after update")
	}
	if config.Cfg.CompetitionInfo.Name != "Original CTF" || config.Cfg.CompetitionInfo.About != "Updated About" {
		t.Fatalf("expected other fields preserved after dynamic_score update, got %+v", config.Cfg.CompetitionInfo)
	}
}

func TestChallengeVisibilityByRole(t *testing.T) {
	cleanup := setupHandlerTestEnv(t)
	defer cleanup()

	author := database.User{
		Name:      "authorvis",
		Email:     "authorvis@example.test",
		AuthModel: auth.CreateModel("authorvis", "pw", core.USER_ROLES["author"]),
	}
	if err := database.Db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	contestant := database.User{
		Name:      "playervis",
		Email:     "playervis@example.test",
		AuthModel: auth.CreateModel("playervis", "pw", core.USER_ROLES["contestant"]),
	}
	if err := database.Db.Create(&contestant).Error; err != nil {
		t.Fatalf("create contestant: %v", err)
	}

	authorToken, _ := auth.GenerateJWT(author.AuthModel)
	playerToken, _ := auth.GenerateJWT(contestant.AuthModel)

	deployed := database.Challenge{
		Name:        "vis-deployed",
		AuthorID:    author.ID,
		Type:        "web",
		Status:      core.DEPLOY_STATUS["deployed"],
		ContainerId: "cid-vis-1",
		ImageId:     "img-vis-1",
		Points:      100,
	}
	staging := database.Challenge{
		Name:        "vis-staging",
		AuthorID:    author.ID,
		Type:        "web",
		Status:      core.DEPLOY_STATUS["staging"],
		ContainerId: "cid-vis-2",
		ImageId:     "img-vis-2",
		Points:      200,
	}
	if err := database.Db.Create(&deployed).Error; err != nil {
		t.Fatalf("create deployed: %v", err)
	}
	if err := database.Db.Create(&staging).Error; err != nil {
		t.Fatalf("create staging: %v", err)
	}

	router := gin.New()
	router.GET("/api/info/challenges", challengesMetadataHandler)
	router.GET("/api/info/challenge/:name", challengeInfoHandler)

	// Contestant sees only deployed challenge in list and gets 404 on staging challenge info
	reqListPlayer := httptest.NewRequest(http.MethodGet, "/api/info/challenges", nil)
	reqListPlayer.Header.Set("Authorization", "Bearer "+playerToken)
	wListPlayer := httptest.NewRecorder()
	router.ServeHTTP(wListPlayer, reqListPlayer)
	if wListPlayer.Code != http.StatusOK {
		t.Fatalf("expected 200 for player challenges list, got %d: %s", wListPlayer.Code, wListPlayer.Body.String())
	}
	var playerList []ChallengeMetadata
	if err := json.Unmarshal(wListPlayer.Body.Bytes(), &playerList); err != nil {
		t.Fatalf("unmarshal player list: %v", err)
	}
	if len(playerList) != 1 || playerList[0].Name != "vis-deployed" {
		t.Fatalf("expected only vis-deployed for contestant, got %+v", playerList)
	}

	reqInfoPlayer := httptest.NewRequest(http.MethodGet, "/api/info/challenge/vis-staging", nil)
	reqInfoPlayer.Header.Set("Authorization", "Bearer "+playerToken)
	wInfoPlayer := httptest.NewRecorder()
	router.ServeHTTP(wInfoPlayer, reqInfoPlayer)
	if wInfoPlayer.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for contestant querying staging challenge, got %d: %s", wInfoPlayer.Code, wInfoPlayer.Body.String())
	}

	// Author sees both challenges and gets 200 on staging challenge info
	reqListAuthor := httptest.NewRequest(http.MethodGet, "/api/info/challenges", nil)
	reqListAuthor.Header.Set("Authorization", "Bearer "+authorToken)
	wListAuthor := httptest.NewRecorder()
	router.ServeHTTP(wListAuthor, reqListAuthor)
	if wListAuthor.Code != http.StatusOK {
		t.Fatalf("expected 200 for author challenges list, got %d: %s", wListAuthor.Code, wListAuthor.Body.String())
	}
	var authorList []ChallengeMetadata
	if err := json.Unmarshal(wListAuthor.Body.Bytes(), &authorList); err != nil {
		t.Fatalf("unmarshal author list: %v", err)
	}
	if len(authorList) != 2 {
		t.Fatalf("expected 2 challenges for author, got %d", len(authorList))
	}

	reqInfoAuthor := httptest.NewRequest(http.MethodGet, "/api/info/challenge/vis-staging", nil)
	reqInfoAuthor.Header.Set("Authorization", "Bearer "+authorToken)
	wInfoAuthor := httptest.NewRecorder()
	router.ServeHTTP(wInfoAuthor, reqInfoAuthor)
	if wInfoAuthor.Code != http.StatusOK {
		t.Fatalf("expected 200 for author querying staging challenge, got %d: %s", wInfoAuthor.Code, wInfoAuthor.Body.String())
	}
}

func TestUserEmailRedaction(t *testing.T) {
	cleanup := setupHandlerTestEnv(t)
	defer cleanup()

	alice := database.User{
		Name:      "Alice",
		Email:     "alice@example.test",
		AuthModel: auth.CreateModel("alice", "pw", core.USER_ROLES["contestant"]),
	}
	bob := database.User{
		Name:      "Bob",
		Email:     "bob@example.test",
		AuthModel: auth.CreateModel("bob", "pw", core.USER_ROLES["contestant"]),
	}
	admin := database.User{
		Name:      "Admin",
		Email:     "admin@example.test",
		AuthModel: auth.CreateModel("adminuser", "pw", core.USER_ROLES["admin"]),
	}
	for _, u := range []*database.User{&alice, &bob, &admin} {
		if err := database.Db.Create(u).Error; err != nil {
			t.Fatalf("create user %s: %v", u.Username, err)
		}
	}

	aliceToken, _ := auth.GenerateJWT(alice.AuthModel)
	adminToken, _ := auth.GenerateJWT(admin.AuthModel)

	router := gin.New()
	router.GET("/api/info/user/:username", userInfoHandler)
	router.GET("/api/info/users", getAllUsersInfoHandler)

	// Contestant alice querying bob -> bob's email is redacted
	reqBob := httptest.NewRequest(http.MethodGet, "/api/info/user/bob", nil)
	reqBob.Header.Set("Authorization", "Bearer "+aliceToken)
	wBob := httptest.NewRecorder()
	router.ServeHTTP(wBob, reqBob)
	if wBob.Code != http.StatusOK {
		t.Fatalf("expected 200 querying bob, got %d: %s", wBob.Code, wBob.Body.String())
	}
	var bobResp UserResp
	if err := json.Unmarshal(wBob.Body.Bytes(), &bobResp); err != nil {
		t.Fatalf("unmarshal bobResp: %v", err)
	}
	if bobResp.Email != "" {
		t.Fatalf("expected bob's email to be redacted for contestant, got %q", bobResp.Email)
	}

	// Contestant alice querying self -> email visible
	reqSelf := httptest.NewRequest(http.MethodGet, "/api/info/user/alice", nil)
	reqSelf.Header.Set("Authorization", "Bearer "+aliceToken)
	wSelf := httptest.NewRecorder()
	router.ServeHTTP(wSelf, reqSelf)
	var selfResp UserResp
	if err := json.Unmarshal(wSelf.Body.Bytes(), &selfResp); err != nil {
		t.Fatalf("unmarshal selfResp: %v", err)
	}
	if selfResp.Email != "alice@example.test" {
		t.Fatalf("expected alice's own email visible, got %q", selfResp.Email)
	}

	// Contestant alice listing all users -> emails redacted
	reqAllPlayer := httptest.NewRequest(http.MethodGet, "/api/info/users", nil)
	reqAllPlayer.Header.Set("Authorization", "Bearer "+aliceToken)
	wAllPlayer := httptest.NewRecorder()
	router.ServeHTTP(wAllPlayer, reqAllPlayer)
	var listPlayer []UsersResp
	if err := json.Unmarshal(wAllPlayer.Body.Bytes(), &listPlayer); err != nil {
		t.Fatalf("unmarshal listPlayer: %v", err)
	}
	for _, u := range listPlayer {
		if u.Email != "" {
			t.Fatalf("expected redacted email in /api/info/users for contestant, got %+v", u)
		}
	}

	// Admin querying bob & all users -> emails visible
	reqBobAdmin := httptest.NewRequest(http.MethodGet, "/api/info/user/bob", nil)
	reqBobAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	wBobAdmin := httptest.NewRecorder()
	router.ServeHTTP(wBobAdmin, reqBobAdmin)
	var bobAdminResp UserResp
	if err := json.Unmarshal(wBobAdmin.Body.Bytes(), &bobAdminResp); err != nil {
		t.Fatalf("unmarshal bobAdminResp: %v", err)
	}
	if bobAdminResp.Email != "bob@example.test" {
		t.Fatalf("expected bob's email visible to admin, got %q", bobAdminResp.Email)
	}

	reqAllAdmin := httptest.NewRequest(http.MethodGet, "/api/info/users", nil)
	reqAllAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	wAllAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAllAdmin, reqAllAdmin)
	var listAdmin []UsersResp
	if err := json.Unmarshal(wAllAdmin.Body.Bytes(), &listAdmin); err != nil {
		t.Fatalf("unmarshal listAdmin: %v", err)
	}
	for _, u := range listAdmin {
		if u.Email == "" {
			t.Fatalf("expected email visible to admin in /api/info/users, got %+v", u)
		}
	}
}
