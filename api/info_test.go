package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()

	dsn := os.Getenv("BEAST_TEST_PG_DSN")
	if dsn == "" {
		dsn = "postgres://beast_database_postgress_user:doorword@localhost:5432/beast?sslmode=disable"
	}

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping test, cannot connect to test postgres: %v", err)
	}

	schema := fmt.Sprintf("beast_info_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	var schemaDSN string
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			q := parsed.Query()
			q.Set("search_path", schema)
			parsed.RawQuery = q.Encode()
			schemaDSN = parsed.String()
		}
	}
	if schemaDSN == "" {
		schemaDSN = dsn + " search_path=" + schema
	}

	testDB, err := gorm.Open(postgres.Open(schemaDSN), &gorm.Config{})
	if err != nil {
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema)).Error
		t.Fatalf("open test db: %v", err)
	}

	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}

	previousDB := database.Db
	previousMux := database.DBMux

	database.Db = testDB
	database.DBMux = &sync.Mutex{}

	if err := database.Db.AutoMigrate(
		&database.Challenge{},
		&database.User{},
		&database.Port{},
		&database.Tag{},
		&database.Hint{},
		&database.UserChallenges{},
	); err != nil {
		t.Fatalf("auto migrate test db: %v", err)
	}

	// Setup Redis cache
	if cache.Cache == nil {
		cache.CacheMutex = &sync.Mutex{}
		cache.Cache = redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
	}

	auth.Init(100, 32, 3600, "beast-test", "secret-test-jwt-key", []string{"manager"}, []string{"admin"}, []string{"contestant"})

	return func() {
		database.Db = previousDB
		database.DBMux = previousMux
		_ = sqlDB.Close()
		_ = adminDB.Exec(fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schema)).Error
		if adminSQL, err := adminDB.DB(); err == nil {
			_ = adminSQL.Close()
		}
	}
}

func TestChallengeInfoHandlerReturnsPortsAndInstancedFlag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)

	// 1. Create a contestant user
	user := database.User{
		Name:      "challuser",
		Email:     "challuser@example.test",
		AuthModel: auth.CreateModel("challuser", "password123", core.USER_ROLES["contestant"]),
	}
	if err := database.Db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	token, err := auth.GenerateJWT(user.AuthModel)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}

	// 2. Create a non-instanced challenge with allocated ports
	nonInstancedChall := database.Challenge{
		Name:           "test-non-instanced",
		Type:           "bare",
		Difficulty:     "easy",
		Status:         core.DEPLOY_STATUS["deployed"],
		ServerDeployed: "localhost",
		ContainerId:    "cid-non-instanced",
		ImageId:        "img-non-instanced",
		Instanced:      false,
		Points:         100,
	}
	if err := database.Db.Create(&nonInstancedChall).Error; err != nil {
		t.Fatalf("create non-instanced challenge: %v", err)
	}
	portEntry := database.Port{
		ChallengeID: nonInstancedChall.ID,
		PortNo:      10005,
	}
	if err := database.Db.Create(&portEntry).Error; err != nil {
		t.Fatalf("create port entry: %v", err)
	}

	// 3. Create an instanced challenge
	instancedChall := database.Challenge{
		Name:               "test-instanced",
		Type:               "service",
		Difficulty:         "medium",
		Status:             core.DEPLOY_STATUS["deployed"],
		ServerDeployed:     "localhost",
		ContainerId:        "cid-instanced",
		ImageId:            "img-instanced",
		Instanced:          true,
		InstanceExpiration: 600,
		Points:             200,
	}
	if err := database.Db.Create(&instancedChall).Error; err != nil {
		t.Fatalf("create instanced challenge: %v", err)
	}

	router := gin.New()
	router.GET("/api/info/challenge/:name", challengeInfoHandler)

	// 4. Test non-instanced challenge info query -> should return [10005]
	reqNonInstanced := httptest.NewRequest("GET", "/api/info/challenge/test-non-instanced", nil)
	reqNonInstanced.Header.Set("Authorization", "Bearer "+token)
	wNonInstanced := httptest.NewRecorder()
	router.ServeHTTP(wNonInstanced, reqNonInstanced)

	if wNonInstanced.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wNonInstanced.Code, wNonInstanced.Body.String())
	}

	var respNonInstanced Challenge
	if err := json.Unmarshal(wNonInstanced.Body.Bytes(), &respNonInstanced); err != nil {
		t.Fatalf("unmarshal non-instanced response: %v", err)
	}

	if respNonInstanced.Instanced {
		t.Errorf("expected instanced to be false, got true")
	}
	if len(respNonInstanced.Ports) != 1 || respNonInstanced.Ports[0] != 10005 {
		t.Errorf("expected ports [10005], got %v", respNonInstanced.Ports)
	}

	// 5. Test instanced challenge info query WITHOUT active instance -> should return []
	reqInstanced := httptest.NewRequest("GET", "/api/info/challenge/test-instanced", nil)
	reqInstanced.Header.Set("Authorization", "Bearer "+token)
	wInstanced := httptest.NewRecorder()
	router.ServeHTTP(wInstanced, reqInstanced)

	if wInstanced.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", wInstanced.Code, wInstanced.Body.String())
	}

	var respInstanced Challenge
	if err := json.Unmarshal(wInstanced.Body.Bytes(), &respInstanced); err != nil {
		t.Fatalf("unmarshal instanced response: %v", err)
	}

	if !respInstanced.Instanced {
		t.Errorf("expected instanced to be true, got false")
	}
	if respInstanced.InstanceExpiration != 600 {
		t.Errorf("expected instanceExpiration 600, got %d", respInstanced.InstanceExpiration)
	}
	if len(respInstanced.Ports) != 0 {
		t.Errorf("expected empty ports array for unspawned instanced challenge, got %v", respInstanced.Ports)
	}

	// 6. Test instanced challenge info query WITH active instance -> should return active instance port [31234]
	if cache.Cache != nil && cache.Cache.Ping(context.Background()).Err() == nil {
		activeInstance := &cache.Instance{
			InstanceID:    "inst-active-test",
			ChallengeName: "test-instanced",
			UserID:        fmt.Sprintf("%d", user.ID),
			Port:          []uint32{31234},
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		}
		if err := cache.SaveInstance(activeInstance, 10*time.Minute); err == nil {
			defer func() {
				_ = cache.DeleteInstance("inst-active-test")
			}()

			reqActive := httptest.NewRequest("GET", "/api/info/challenge/test-instanced", nil)
			reqActive.Header.Set("Authorization", "Bearer "+token)
			wActive := httptest.NewRecorder()
			router.ServeHTTP(wActive, reqActive)

			if wActive.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d: %s", wActive.Code, wActive.Body.String())
			}

			var respActive Challenge
			if err := json.Unmarshal(wActive.Body.Bytes(), &respActive); err != nil {
				t.Fatalf("unmarshal active instanced response: %v", err)
			}

			if !respActive.Instanced {
				t.Errorf("expected instanced to be true, got false")
			}
			if len(respActive.Ports) != 1 || respActive.Ports[0] != 31234 {
				t.Errorf("expected active instance ports [31234], got %v", respActive.Ports)
			}
		}
	}
}
