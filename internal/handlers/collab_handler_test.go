package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupCollabHandlerTestDB creates an in-memory SQLite DB with all collaboration
// entities migrated. It swaps db.DB and restores it on cleanup.
func setupCollabHandlerTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.CollabSprint{},
		&entities.CollabRequirement{},
		&entities.CollabTask{},
		&entities.CollabDefect{},
		&entities.CollabTestCase{},
		&entities.CollabTestRun{},
	))
	db.DB = testDB
}

// claimsMiddleware injects app.Claims into the gin context so handlers can
// call api.GetClaims(c).
func claimsMiddleware(userID, username, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("claims", &app.Claims{
			UserID:   userID,
			Username: username,
			Role:     role,
		})
		c.Next()
	}
}

// collabAPIResponse mirrors api.Response for JSON decoding in tests.
type collabAPIResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// ---------- Requirement handler tests ----------

func TestCreateRequirement_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/requirements", CreateRequirement)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/requirements", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error, "error field should be populated for malformed JSON")
	assert.Empty(t, resp.Data, "data field should be empty on error")
}

func TestCreateRequirement_MissingRequiredFields(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/requirements", CreateRequirement)

	// Missing title, status, priority (all required)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/requirements", strings.NewReader(`{"description":"only optional"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateRequirement_ValidRequest(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/requirements", CreateRequirement)

	body := `{"title":"Req 1","status":"triage","priority":"p1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/requirements", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error, "error should be empty on success")
	assert.NotEmpty(t, resp.Data, "data should be populated on success")

	// Verify data shape has expected fields
	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "p1", data["project_id"])
	assert.Equal(t, "Req 1", data["title"])
	assert.Equal(t, "triage", data["status"])
	assert.Equal(t, "p1", data["priority"])
}

func TestTransitionRequirement_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/requirements/:requirementID/transition", TransitionRequirement)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/requirements/r1/transition", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestTransitionRequirement_InvalidTransition(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a requirement
	require.NoError(t, db.DB.Create(&entities.CollabRequirement{
		Base:           entities.Base{ID: "r1"},
		ProjectID:      "p1",
		Title:          "Req 1",
		Status:         "triage",
		Priority:       "p1",
		PlanningStatus: "backlog",
		CreatedBy:      "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/requirements/:requirementID/transition", TransitionRequirement)

	// Try invalid transition: triage -> done (not allowed)
	body := `{"status":"done"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/requirements/r1/transition", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error, "should report invalid transition")
}

// ---------- Sprint handler tests ----------

func TestCreateSprint_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/sprints", CreateSprint)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sprints", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateSprint_MissingRequiredFields(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/sprints", CreateSprint)

	// Missing name, status, start_date, end_date
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sprints", strings.NewReader(`{"goal":"some goal"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateSprint_ValidRequest(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/sprints", CreateSprint)

	body := `{"name":"Sprint 1","status":"planned","start_date":"2025-01-01","end_date":"2025-01-14"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sprints", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.Data)
}

func TestCreateSprint_ResponseUsesDateOnlyFormat(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/sprints", CreateSprint)

	body := `{"name":"Sprint 1","status":"planned","start_date":"2026-03-20","end_date":"2026-03-30"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/sprints", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "2026-03-20", data["start_date"])
	assert.Equal(t, "2026-03-30", data["end_date"])
}

// ---------- Task handler tests ----------

func TestCreateTask_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/tasks", CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/tasks", strings.NewReader(`!!!`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateTask_MissingRequiredFields(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/tasks", CreateTask)

	// Missing title, status, priority
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/tasks", strings.NewReader(`{"description":"desc"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateTask_ValidRequest(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/tasks", CreateTask)

	body := `{"title":"Task 1","status":"todo","priority":"p2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.Data)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "Task 1", data["title"])
	assert.Equal(t, "todo", data["status"])
}

func TestCreateTask_ResponseUsesDateOnlyFormat(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/tasks", CreateTask)

	body := `{"title":"Task 1","status":"todo","priority":"p2","due_date":"2026-03-20"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, "2026-03-20", data["due_date"])
}

func TestTransitionTask_InvalidTransition(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a task
	require.NoError(t, db.DB.Create(&entities.CollabTask{
		Base:      entities.Base{ID: "t1"},
		ProjectID: "p1",
		Title:     "Task 1",
		Status:    "todo",
		Priority:  "p1",
		CreatedBy: "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/tasks/:taskID/transition", TransitionTask)

	// Try invalid transition: todo -> done (must go through in_progress/review first)
	body := `{"status":"done"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/tasks/t1/transition", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error, "should report invalid transition")
}

// ---------- Defect handler tests ----------

func TestCreateDefect_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/defects", CreateDefect)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/defects", strings.NewReader(`{{{`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateDefect_MissingRequiredFields(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/defects", CreateDefect)

	// Missing title, description, severity, status
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/defects", strings.NewReader(`{"assignee_id":"u2"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateDefect_ValidRequest(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a requirement so the upstream link is valid
	require.NoError(t, db.DB.Create(&entities.CollabRequirement{
		Base:      entities.Base{ID: "r-upstream"},
		ProjectID: "p1",
		Title:     "Parent Req",
		Status:    "triage",
		Priority:  "medium",
		CreatedBy: "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/defects", CreateDefect)

	body := `{"title":"Bug 1","description":"Crash on save","severity":"high","status":"new","requirement_id":"r-upstream"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/defects", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.Data)
}

func TestTransitionDefect_InvalidTransition(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a defect
	require.NoError(t, db.DB.Create(&entities.CollabDefect{
		Base:        entities.Base{ID: "d1"},
		ProjectID:   "p1",
		Title:       "Bug 1",
		Description: "desc",
		Severity:    "high",
		Status:      "new",
		CreatedBy:   "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/defects/:defectID/transition", TransitionDefect)

	// Try invalid transition: new -> closed (should go through processing/pending_verify first)
	body := `{"status":"closed"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/defects/d1/transition", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error, "should report invalid transition")
}

// ---------- TestCase handler tests ----------

func TestCreateTestCase_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/test-cases", CreateTestCase)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/test-cases", strings.NewReader(`<not json>`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateTestCase_MissingRequiredFields(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/test-cases", CreateTestCase)

	// Missing title, steps, expected_result
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/test-cases", strings.NewReader(`{"precondition":"some"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateTestCase_ValidRequest(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/test-cases", CreateTestCase)

	body := `{"title":"TC 1","steps":"1. Open app\n2. Click save","expected_result":"Data saved"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/test-cases", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.Data)
}

// ---------- TestRun handler tests ----------

func TestCreateTestRun_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a test case for the route parameter
	require.NoError(t, db.DB.Create(&entities.CollabTestCase{
		Base:           entities.Base{ID: "tc1"},
		ProjectID:      "p1",
		Title:          "TC 1",
		Steps:          "step",
		ExpectedResult: "result",
		CreatedBy:      "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/test-cases/:testCaseID/runs", CreateTestRun)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/test-cases/tc1/runs", strings.NewReader(`broken`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateTestRun_MissingRequiredFields(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.CollabTestCase{
		Base:           entities.Base{ID: "tc1"},
		ProjectID:      "p1",
		Title:          "TC 1",
		Steps:          "step",
		ExpectedResult: "result",
		CreatedBy:      "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/test-cases/:testCaseID/runs", CreateTestRun)

	// Missing status (required)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/test-cases/tc1/runs", strings.NewReader(`{"comment":"no status"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestCreateTestRun_ValidRequest(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.CollabTestCase{
		Base:           entities.Base{ID: "tc1"},
		ProjectID:      "p1",
		Title:          "TC 1",
		Steps:          "step",
		ExpectedResult: "result",
		CreatedBy:      "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/test-cases/:testCaseID/runs", CreateTestRun)

	body := `{"status":"passed","comment":"All good"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/test-cases/tc1/runs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.Data)
}

// ---------- List / empty project tests ----------

func TestListRequirements_EmptyProject(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/projects/:projectID/requirements", ListRequirements)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/requirements?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
	assert.NotEmpty(t, resp.Data)
}

func TestListSprints_EmptyProject(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/projects/:projectID/sprints", ListSprints)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/sprints?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Error)
}

func TestListTasks_EmptyProject(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/projects/:projectID/tasks", ListTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/tasks?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListDefects_EmptyProject(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/projects/:projectID/defects", ListDefects)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/defects?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListTestCases_EmptyProject(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/projects/:projectID/test-cases", ListTestCases)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/p1/test-cases?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- Update with malformed JSON tests ----------

func TestUpdateRequirement_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/requirements/:requirementID", UpdateRequirement)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/requirements/r1", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestUpdateSprint_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/sprints/:sprintID", UpdateSprint)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/sprints/s1", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestUpdateTask_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/tasks/:taskID", UpdateTask)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/tasks/t1", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestUpdateDefect_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/defects/:defectID", UpdateDefect)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/defects/d1", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

func TestUpdateTestCase_MalformedJSON(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.PUT("/api/v1/projects/:projectID/test-cases/:testCaseID", UpdateTestCase)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/p1/test-cases/tc1", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

// ---------- Empty body tests ----------

func TestCreateRequirement_EmptyBody(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/requirements", CreateRequirement)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/requirements", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error)
}

// ---------- Depth limit through handler ----------

func TestCreateRequirementChild_DepthExceeded(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a parent requirement at depth 1 (max)
	require.NoError(t, db.DB.Create(&entities.CollabRequirement{
		Base:           entities.Base{ID: "parent1"},
		ProjectID:      "p1",
		Title:          "Parent",
		Status:         "triage",
		Priority:       "p1",
		PlanningStatus: "backlog",
		Depth:          1,
		CreatedBy:      "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/requirements/:requirementID/children", CreateRequirementChild)

	body := `{"title":"Child","status":"triage","priority":"p2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/requirements/parent1/children", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error, "should reject child at exceeded depth")
	assert.Contains(t, resp.Error, "depth", "error message should mention depth")
}

func TestCreateTaskChild_DepthExceeded(t *testing.T) {
	setupCollabHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	// Seed a parent task at depth 1 (max)
	require.NoError(t, db.DB.Create(&entities.CollabTask{
		Base:      entities.Base{ID: "parent1"},
		ProjectID: "p1",
		Title:     "Parent Task",
		Status:    "todo",
		Priority:  "p1",
		Depth:     1,
		CreatedBy: "u1",
	}).Error)

	r := gin.New()
	r.Use(claimsMiddleware("u1", "alice", app.UserRoleUser))
	r.POST("/api/v1/projects/:projectID/tasks/:taskID/children", CreateTaskChild)

	body := `{"title":"Child Task","status":"todo","priority":"p2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p1/tasks/parent1/children", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp collabAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Error, "should reject child at exceeded depth")
	assert.Contains(t, resp.Error, "depth", "error message should mention depth")
}
