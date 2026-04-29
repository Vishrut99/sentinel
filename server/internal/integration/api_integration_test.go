package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/yourusername/incident-ticketing/internal/cache"
	"github.com/yourusername/incident-ticketing/internal/db"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/handler"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/repository"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type integrationApp struct {
	router     http.Handler
	db         *gorm.DB
	categoryID uuid.UUID
	redisURL   string
	redis      *redis.Client
}

func setupIntegrationApp(t *testing.T) *integrationApp {
	t.Helper()

	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("integration tests disabled; set INTEGRATION_TESTS=1")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL is required for integration tests")
	}

	if os.Getenv("JWT_SECRET") == "" {
		t.Setenv("JWT_SECRET", "integration-test-secret")
	}
	if os.Getenv("BOOTSTRAP_ADMIN_SECRET") == "" {
		t.Setenv("BOOTSTRAP_ADMIN_SECRET", "integration-bootstrap-secret")
	}

	database, err := db.Connect(dbURL)
	if err != nil {
		t.Skipf("cannot connect integration database: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("failed to get sql DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	migrations := []string{
		"internal/db/migrations/001_init.sql",
		"internal/db/migrations/002_migrate.sql",
		"internal/db/migrations/003_migrate.sql",
		"internal/db/migrations/004_ai_problem_management.sql",
		"internal/db/migrations/005_skill_assignment.sql",
	}
	for _, migrationPath := range migrations {
		content, err := os.ReadFile(filepath.Join("..", "..", migrationPath))
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", migrationPath, err)
		}
		if err := database.Exec(string(content)).Error; err != nil {
			t.Fatalf("failed to execute migration %s: %v", migrationPath, err)
		}
	}

	if err := database.Exec("TRUNCATE TABLE comments, audit_logs, tickets, agents, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("failed to cleanup test data: %v", err)
	}

	var category domain.Category
	if err := database.WithContext(context.Background()).Where("name = ?", "General").First(&category).Error; err != nil {
		t.Fatalf("failed to fetch default category: %v", err)
	}

	redisURL := os.Getenv("REDIS_URL")
	redisCache, err := cache.NewRedisCache(redisURL)
	if err != nil {
		redisCache = nil
	}
	if redisCache != nil {
		t.Cleanup(func() { _ = redisCache.Close() })
	}

	var redisClient *redis.Client
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			redisClient = redis.NewClient(opt)
			if pingErr := redisClient.Ping(context.Background()).Err(); pingErr != nil {
				_ = redisClient.Close()
				redisClient = nil
			}
		}
	}
	if redisClient != nil {
		t.Cleanup(func() { _ = redisClient.Close() })
	}

	userRepo := repository.NewUserRepository(database)
	authService := service.NewAuthService(userRepo, os.Getenv("BOOTSTRAP_ADMIN_SECRET"))
	authHandler := handler.NewAuthHandler(authService)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	ticketRepo := repository.NewTicketRepository(database)
	ticketService := service.NewTicketService(ticketRepo, database, redisCache, nil)
	ticketHandler := handler.NewTicketHandler(ticketService)

	commentRepo := repository.NewCommentRepository(database)
	commentService := service.NewCommentService(commentRepo, ticketRepo)
	commentHandler := handler.NewCommentHandler(commentService)

	agentRepo := repository.NewAgentRepository(database)
	agentService := service.NewAgentService(agentRepo, database)
	agentHandler := handler.NewAgentHandler(agentService)

	dashboardRepo := repository.NewDashboardRepository(database)
	dashboardService := service.NewDashboardService(dashboardRepo, redisCache)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)

	auditRepo := repository.NewAuditRepository(database)
	auditService := service.NewAuditService(auditRepo, ticketRepo)
	auditHandler := handler.NewAuditHandler(auditService)

	lookupRepo := repository.NewLookupRepository(database)
	lookupService := service.NewLookupService(lookupRepo)
	lookupHandler := handler.NewLookupHandler(lookupService)

	router := chi.NewRouter()
	router.Use(middleware.CORS)
	router.Route("/api/v1", func(r chi.Router) {
		authHandler.RegisterRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)
			ticketHandler.RegisterRoutes(r)
			commentHandler.RegisterRoutes(r)
			agentHandler.RegisterRoutes(r)
			dashboardHandler.RegisterRoutes(r)
			auditHandler.RegisterRoutes(r)
			lookupHandler.RegisterRoutes(r)

			r.With(middleware.RequirePermission(middleware.PermRegisterAgent)).Post("/agents/register", agentHandler.Register)
			r.With(middleware.RequirePermission(middleware.PermManageAgentCapacity)).Patch("/agents/{id}/capacity", agentHandler.UpdateCapacity)
			r.With(middleware.RequirePermission(middleware.PermPromoteAdmin)).Patch("/users/{id}/promote-admin", userHandler.PromoteToAdmin)
			r.With(middleware.RequirePermission(middleware.PermEditTicket)).Patch("/tickets/{id}", ticketHandler.UpdateTicket)
			r.With(middleware.RequirePermission(middleware.PermAssignTicket)).Patch("/tickets/{id}/assign", ticketHandler.AssignTicket)
			r.With(middleware.RequirePermission(middleware.PermResolveTicket)).Patch("/tickets/{id}/status", ticketHandler.ChangeStatus)
			r.With(middleware.RequirePermission(middleware.PermLinkProblem)).Patch("/tickets/{id}/link-problem", ticketHandler.LinkToProblem)
			r.With(middleware.RequirePermission(middleware.PermViewDashboard)).Get("/dashboard", dashboardHandler.GetDashboard)
		})
	})

	return &integrationApp{
		router:     router,
		db:         database,
		categoryID: category.ID,
		redisURL:   redisURL,
		redis:      redisClient,
	}
}

func (app *integrationApp) doJSONRequest(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()

	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		payload = encoded
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	app.router.ServeHTTP(recorder, req)

	var response map[string]any
	if len(recorder.Body.Bytes()) > 0 {
		_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	}

	return recorder.Code, response
}

func (app *integrationApp) registerAndLogin(t *testing.T, email, password, fullName string) (uuid.UUID, string) {
	t.Helper()

	status, registerResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":     email,
		"password":  password,
		"full_name": fullName,
	})
	if status != http.StatusCreated {
		t.Fatalf("register failed, status=%d body=%v", status, registerResponse)
	}

	userIDValue, ok := registerResponse["id"].(string)
	if !ok {
		t.Fatalf("register response missing id: %v", registerResponse)
	}
	userID, err := uuid.Parse(userIDValue)
	if err != nil {
		t.Fatalf("invalid user id in register response: %v", err)
	}

	status, loginResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login failed, status=%d body=%v", status, loginResponse)
	}

	tokenValue, ok := loginResponse["token"].(string)
	if !ok || tokenValue == "" {
		t.Fatalf("login response missing token: %v", loginResponse)
	}

	return userID, tokenValue
}

func TestIntegrationRegisterDuplicateEmailReturnsSpecificConflict(t *testing.T) {
	app := setupIntegrationApp(t)

	status, firstResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":     "duplicate@example.com",
		"password":  "Password123",
		"full_name": "Duplicate User",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected first register to succeed, status=%d body=%v", status, firstResponse)
	}

	status, secondResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":     "duplicate@example.com",
		"password":  "Password123",
		"full_name": "Duplicate User",
	})
	if status != http.StatusConflict {
		t.Fatalf("expected duplicate register to fail with conflict, status=%d body=%v", status, secondResponse)
	}
	if secondResponse["message"] != "email already registered" {
		t.Fatalf("expected duplicate register message to mention email conflict, body=%v", secondResponse)
	}
}

func (app *integrationApp) loginOnly(t *testing.T, email, password string) string {
	t.Helper()

	status, loginResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login failed, status=%d body=%v", status, loginResponse)
	}

	tokenValue, ok := loginResponse["token"].(string)
	if !ok || tokenValue == "" {
		t.Fatalf("login response missing token: %v", loginResponse)
	}

	return tokenValue
}

func (app *integrationApp) bootstrapAdmin(t *testing.T, email, password, fullName string) (uuid.UUID, string) {
	t.Helper()

	status, response := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/bootstrap-admin", "", map[string]any{
		"email":            email,
		"password":         password,
		"full_name":        fullName,
		"bootstrap_secret": os.Getenv("BOOTSTRAP_ADMIN_SECRET"),
	})
	if status != http.StatusCreated {
		t.Fatalf("bootstrap admin failed, status=%d body=%v", status, response)
	}

	userValue, ok := response["user"].(map[string]any)
	if !ok {
		t.Fatalf("bootstrap admin response missing user: %v", response)
	}

	userIDValue, ok := userValue["id"].(string)
	if !ok {
		t.Fatalf("bootstrap admin response missing user id: %v", response)
	}
	userID, err := uuid.Parse(userIDValue)
	if err != nil {
		t.Fatalf("invalid bootstrap admin user id: %v", err)
	}

	tokenValue, ok := response["token"].(string)
	if !ok || tokenValue == "" {
		t.Fatalf("bootstrap admin response missing token: %v", response)
	}

	return userID, tokenValue
}

func TestIntegrationBootstrapAdminAndPromoteAdminFlow(t *testing.T) {
	app := setupIntegrationApp(t)

	_, adminToken := app.bootstrapAdmin(t, "bootstrap-admin@example.com", "Password123", "Bootstrap Admin")

	status, secondBootstrap := app.doJSONRequest(t, http.MethodPost, "/api/v1/auth/bootstrap-admin", "", map[string]any{
		"email":            "second-admin@example.com",
		"password":         "Password123",
		"full_name":        "Second Admin",
		"bootstrap_secret": os.Getenv("BOOTSTRAP_ADMIN_SECRET"),
	})
	if status != http.StatusConflict {
		t.Fatalf("expected second bootstrap to fail with conflict, status=%d body=%v", status, secondBootstrap)
	}

	candidateID, _ := app.registerAndLogin(t, "admin-candidate@example.com", "Password123", "Admin Candidate")

	status, promoteResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/users/%s/promote-admin", candidateID.String()), adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("promote admin failed, status=%d body=%v", status, promoteResponse)
	}

	promotedUser, ok := promoteResponse["user"].(map[string]any)
	if !ok || promotedUser["role"] != "admin" {
		t.Fatalf("expected promoted user payload, got %v", promoteResponse)
	}

	candidateToken := app.loginOnly(t, "admin-candidate@example.com", "Password123")
	status, dashboardResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/dashboard", candidateToken, nil)
	if status != http.StatusOK {
		t.Fatalf("expected promoted admin to access dashboard after re-login, status=%d body=%v", status, dashboardResponse)
	}
}

func TestIntegrationAuthTicketAssignResolveFlow(t *testing.T) {
	app := setupIntegrationApp(t)

	reporterID, reporterToken := app.registerAndLogin(t, "reporter@example.com", "Password123", "Reporter User")
	_, _ = reporterID, reporterToken

	status, createTicketResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/tickets/", reporterToken, map[string]any{
		"title":       "VPN timeout issue",
		"description": "User cannot connect to VPN",
		"priority":    "high",
		"category_id": app.categoryID.String(),
	})
	if status != http.StatusCreated {
		t.Fatalf("create ticket failed, status=%d body=%v", status, createTicketResponse)
	}

	ticketIDText, ok := createTicketResponse["id"].(string)
	if !ok {
		t.Fatalf("ticket create response missing id: %v", createTicketResponse)
	}
	ticketID, err := uuid.Parse(ticketIDText)
	if err != nil {
		t.Fatalf("invalid ticket id: %v", err)
	}

	_, adminToken := app.bootstrapAdmin(t, "admin@example.com", "Password123", "Admin User")

	agentUserID, _ := app.registerAndLogin(t, "agent-user@example.com", "Password123", "Agent User")

	status, registerAgentResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/agents/register", adminToken, map[string]any{
		"user_id":    agentUserID.String(),
		"department": "IT",
	})
	if status != http.StatusCreated {
		t.Fatalf("register agent failed, status=%d body=%v", status, registerAgentResponse)
	}

	agentIDText, ok := registerAgentResponse["id"].(string)
	if !ok {
		t.Fatalf("register agent response missing id: %v", registerAgentResponse)
	}

	status, assignResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/tickets/%s/assign", ticketID.String()), adminToken, map[string]any{
		"agent_id": agentIDText,
	})
	if status != http.StatusOK {
		t.Fatalf("assign failed, status=%d body=%v", status, assignResponse)
	}

	status, resolveResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/tickets/%s/status", ticketID.String()), adminToken, map[string]any{
		"status": "resolved",
	})
	if status != http.StatusOK {
		t.Fatalf("resolve failed, status=%d body=%v", status, resolveResponse)
	}

	var ticket domain.Ticket
	if err := app.db.WithContext(context.Background()).Preload("Status").Where("id = ?", ticketID).First(&ticket).Error; err != nil {
		t.Fatalf("failed to fetch resolved ticket: %v", err)
	}
	if ticket.Status.Name != "resolved" {
		t.Fatalf("expected resolved status, got %s", ticket.Status.Name)
	}
}

func TestIntegrationTicketEndpointsReturnDTOShape(t *testing.T) {
	app := setupIntegrationApp(t)

	_, reporterToken := app.registerAndLogin(t, "dto-reporter@example.com", "Password123", "DTO Reporter")

	status, createTicketResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/tickets/", reporterToken, map[string]any{
		"title":       "Email delivery delay",
		"description": "Messages are delayed by fifteen minutes",
		"priority":    "medium",
		"category_id": app.categoryID.String(),
	})
	if status != http.StatusCreated {
		t.Fatalf("create ticket failed, status=%d body=%v", status, createTicketResponse)
	}

	if createTicketResponse["ticket_number"] == "" {
		t.Fatalf("expected ticket_number in create response, got %v", createTicketResponse)
	}
	if createTicketResponse["category_id"] != app.categoryID.String() {
		t.Fatalf("expected category_id=%s, got %v", app.categoryID.String(), createTicketResponse["category_id"])
	}
	createdBy, ok := createTicketResponse["created_by"].(map[string]any)
	if !ok || createdBy["role"] != "user" {
		t.Fatalf("expected created_by user summary, got %v", createTicketResponse)
	}

	ticketID, ok := createTicketResponse["id"].(string)
	if !ok || ticketID == "" {
		t.Fatalf("expected id in create response, got %v", createTicketResponse)
	}

	status, getTicketResponse := app.doJSONRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%s", ticketID), reporterToken, nil)
	if status != http.StatusOK {
		t.Fatalf("get ticket failed, status=%d body=%v", status, getTicketResponse)
	}
	if getTicketResponse["id"] != ticketID {
		t.Fatalf("expected get ticket to return DTO id=%s, got %v", ticketID, getTicketResponse)
	}

	status, listResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/tickets/?page=1&per_page=20", reporterToken, nil)
	if status != http.StatusOK {
		t.Fatalf("list tickets failed, status=%d body=%v", status, listResponse)
	}
	if listResponse["page"] != float64(1) || listResponse["per_page"] != float64(20) {
		t.Fatalf("expected pagination fields in DTO list response, got %v", listResponse)
	}
	data, ok := listResponse["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatalf("expected DTO list data, got %v", listResponse)
	}
	firstTicket, ok := data[0].(map[string]any)
	if !ok || firstTicket["ticket_number"] == "" {
		t.Fatalf("expected ticket DTO object in list response, got %v", listResponse)
	}
}

func TestIntegrationTicketAutoAssignmentAndSkillEvaluationFlow(t *testing.T) {
	app := setupIntegrationApp(t)

	_, adminToken := app.bootstrapAdmin(t, "auto-admin@example.com", "Password123", "Auto Admin")
	_, reporterToken := app.registerAndLogin(t, "auto-reporter@example.com", "Password123", "Auto Reporter")
	agentUserID, _ := app.registerAndLogin(t, "vpn-agent@example.com", "Password123", "VPN Agent")

	status, registerAgentResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/agents/register", adminToken, map[string]any{
		"user_id":     agentUserID.String(),
		"department":  "Network",
		"max_tickets": 4,
		"skills": []map[string]any{
			{"name": "VPN", "score": 60},
			{"name": "Network", "score": 75},
		},
	})
	if status != http.StatusCreated {
		t.Fatalf("register agent failed, status=%d body=%v", status, registerAgentResponse)
	}

	agentIDText, ok := registerAgentResponse["id"].(string)
	if !ok || agentIDText == "" {
		t.Fatalf("register agent response missing id: %v", registerAgentResponse)
	}
	agentID, err := uuid.Parse(agentIDText)
	if err != nil {
		t.Fatalf("invalid agent id: %v", err)
	}

	status, createTicketResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/tickets/", reporterToken, map[string]any{
		"title":       "VPN login blocked after MFA challenge",
		"description": "Remote access fails for the whole team after the MFA step completes",
		"priority":    "high",
		"category_id": app.categoryID.String(),
	})
	if status != http.StatusCreated {
		t.Fatalf("create ticket failed, status=%d body=%v", status, createTicketResponse)
	}

	assignedAgent, ok := createTicketResponse["assigned_agent"].(map[string]any)
	if !ok || assignedAgent["id"] != agentID.String() {
		t.Fatalf("expected ticket to be auto-assigned to VPN agent, got %v", createTicketResponse["assigned_agent"])
	}
	if createTicketResponse["assignment_justification"] == "" {
		t.Fatalf("expected assignment justification in response, got %v", createTicketResponse)
	}
	requiredSkills, ok := createTicketResponse["required_skills"].([]any)
	if !ok || len(requiredSkills) == 0 {
		t.Fatalf("expected required_skills in create response, got %v", createTicketResponse["required_skills"])
	}

	ticketIDText, ok := createTicketResponse["id"].(string)
	if !ok || ticketIDText == "" {
		t.Fatalf("expected ticket id in create response, got %v", createTicketResponse)
	}

	status, resolveResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/tickets/%s/status", ticketIDText), adminToken, map[string]any{
		"status": "resolved",
	})
	if status != http.StatusOK {
		t.Fatalf("resolve failed, status=%d body=%v", status, resolveResponse)
	}

	var storedAgent domain.Agent
	if err := app.db.WithContext(context.Background()).Where("id = ?", agentID).First(&storedAgent).Error; err != nil {
		t.Fatalf("failed to fetch updated agent: %v", err)
	}

	vpnScore := 0
	for _, skill := range storedAgent.Skills {
		if skill.Name == "VPN" {
			vpnScore = skill.Score
			break
		}
	}
	if vpnScore <= 60 {
		t.Fatalf("expected VPN skill score to increase after resolution, got %d", vpnScore)
	}

	var storedTicket domain.Ticket
	if err := app.db.WithContext(context.Background()).Where("id = ?", ticketIDText).First(&storedTicket).Error; err != nil {
		t.Fatalf("failed to fetch updated ticket: %v", err)
	}
	if storedTicket.SkillsEvaluatedAt == nil {
		t.Fatalf("expected ticket skills_evaluated_at to be set")
	}
}

func TestIntegrationTicketUpdateRequiresReasonResetsSLAAndSupportsSearch(t *testing.T) {
	app := setupIntegrationApp(t)

	_, reporterToken := app.registerAndLogin(t, "edit-reporter@example.com", "Password123", "Edit Reporter")

	status, createTicketResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/tickets/", reporterToken, map[string]any{
		"title":       "VPN outage in branch office",
		"description": "Users cannot connect from the branch office",
		"priority":    "high",
		"category_id": app.categoryID.String(),
	})
	if status != http.StatusCreated {
		t.Fatalf("create ticket failed, status=%d body=%v", status, createTicketResponse)
	}

	ticketIDText, ok := createTicketResponse["id"].(string)
	if !ok || ticketIDText == "" {
		t.Fatalf("expected ticket id, got %v", createTicketResponse)
	}
	ticketID, err := uuid.Parse(ticketIDText)
	if err != nil {
		t.Fatalf("invalid ticket id: %v", err)
	}

	_, adminToken := app.bootstrapAdmin(t, "edit-admin@example.com", "Password123", "Edit Admin")

	newCategory := domain.Category{
		ID:        uuid.New(),
		Name:      "Infrastructure",
		CreatedAt: time.Now(),
	}
	if err := app.db.WithContext(context.Background()).Create(&newCategory).Error; err != nil {
		t.Fatalf("failed to seed extra category: %v", err)
	}

	status, missingReasonResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/tickets/%s", ticketID.String()), adminToken, map[string]any{
		"priority": "critical",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected missing change reason to fail, status=%d body=%v", status, missingReasonResponse)
	}

	status, updateResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/tickets/%s", ticketID.String()), adminToken, map[string]any{
		"title":         "VPN outage in branch office escalated",
		"priority":      "critical",
		"category_id":   newCategory.ID.String(),
		"change_reason": "Customer impact is much larger than initially reported",
	})
	if status != http.StatusOK {
		t.Fatalf("ticket update failed, status=%d body=%v", status, updateResponse)
	}
	if updateResponse["priority"] != "critical" {
		t.Fatalf("expected updated priority in response, got %v", updateResponse)
	}
	if updateResponse["category"] != newCategory.Name {
		t.Fatalf("expected updated category in response, got %v", updateResponse)
	}

	var storedTicket domain.Ticket
	if err := app.db.WithContext(context.Background()).
		Preload("Priority").
		Preload("Category").
		Preload("Status").
		Where("id = ?", ticketID).
		First(&storedTicket).Error; err != nil {
		t.Fatalf("failed to reload ticket: %v", err)
	}
	if storedTicket.Priority.Name != "critical" {
		t.Fatalf("expected priority to be critical, got %s", storedTicket.Priority.Name)
	}
	if storedTicket.Category.Name != newCategory.Name {
		t.Fatalf("expected category to be %s, got %s", newCategory.Name, storedTicket.Category.Name)
	}
	if storedTicket.DueAt == nil {
		t.Fatalf("expected due_at to be recalculated")
	}
	dueIn := time.Until(*storedTicket.DueAt)
	if dueIn < 3*time.Hour || dueIn > 5*time.Hour {
		t.Fatalf("expected due_at to reset near 4 hours, got %s", dueIn)
	}
	if storedTicket.SLABreached {
		t.Fatalf("expected sla_breached to be reset")
	}

	var comments []domain.Comment
	if err := app.db.WithContext(context.Background()).
		Where("ticket_id = ? AND is_internal = ?", ticketID, true).
		Order("created_at desc").
		Find(&comments).Error; err != nil {
		t.Fatalf("failed to fetch comments: %v", err)
	}
	if len(comments) == 0 {
		t.Fatalf("expected an internal comment for the classification change")
	}
	if !strings.Contains(comments[0].Body, "Customer impact is much larger than initially reported") {
		t.Fatalf("expected reason in internal comment, got %q", comments[0].Body)
	}

	status, searchResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/tickets/?q=branch%20office%20escalated&sort_by=title&sort_dir=asc&page=1&per_page=10", adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("search request failed, status=%d body=%v", status, searchResponse)
	}
	results, ok := searchResponse["data"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected search results, got %v", searchResponse)
	}
	firstResult, ok := results[0].(map[string]any)
	if !ok || firstResult["id"] != ticketID.String() {
		t.Fatalf("expected updated ticket in search results, got %v", searchResponse)
	}
}

func TestIntegrationDashboardCacheHitMissInvalidate(t *testing.T) {
	app := setupIntegrationApp(t)
	if app.redis == nil {
		t.Skip("redis is unavailable; skipping dashboard cache integration test")
	}

	_, adminToken := app.bootstrapAdmin(t, "cache-admin@example.com", "Password123", "Cache Admin")

	dummyPayload := `{"total_tickets":999,"open_tickets":100,"in_progress_tickets":200,"resolved_today":300,"sla_breached":1,"avg_resolve_hours":0,"by_priority":{},"by_category":{}}`
	if err := app.redis.Set(context.Background(), cache.DashboardStatsCacheKey, dummyPayload, 2*time.Minute).Err(); err != nil {
		t.Fatalf("failed to seed dashboard cache: %v", err)
	}

	status, dashboardResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/dashboard", adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard request failed, status=%d body=%v", status, dashboardResponse)
	}
	totalTickets, ok := dashboardResponse["total_tickets"].(float64)
	if !ok || int(totalTickets) != 999 {
		t.Fatalf("expected cached dashboard response, got %v", dashboardResponse)
	}

	_, reporterToken := app.registerAndLogin(t, "cache-reporter@example.com", "Password123", "Cache Reporter")
	status, createTicketResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/tickets/", reporterToken, map[string]any{
		"title":       "Cached dashboard invalidation check",
		"description": "invalidate dashboard cache",
		"priority":    "medium",
		"category_id": app.categoryID.String(),
	})
	if status != http.StatusCreated {
		t.Fatalf("create ticket failed, status=%d body=%v", status, createTicketResponse)
	}

	_, err := app.redis.Get(context.Background(), cache.DashboardStatsCacheKey).Result()
	if err != redis.Nil {
		t.Fatalf("expected cache key to be deleted, get err=%v", err)
	}

	status, dashboardResponse = app.doJSONRequest(t, http.MethodGet, "/api/v1/dashboard", adminToken, nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard miss request failed, status=%d body=%v", status, dashboardResponse)
	}

	_, err = app.redis.Get(context.Background(), cache.DashboardStatsCacheKey).Result()
	if err != nil {
		t.Fatalf("expected cache key to be repopulated, get err=%v", err)
	}
}

func TestIntegrationAgentRegisterAndCapacityUpdate(t *testing.T) {
	app := setupIntegrationApp(t)

	_, adminToken := app.bootstrapAdmin(t, "capacity-admin@example.com", "Password123", "Capacity Admin")

	agentUserID, _ := app.registerAndLogin(t, "capacity-agent@example.com", "Password123", "Capacity Agent")

	status, registerAgentResponse := app.doJSONRequest(t, http.MethodPost, "/api/v1/agents/register", adminToken, map[string]any{
		"user_id":     agentUserID.String(),
		"department":  "IT",
		"max_tickets": 2,
	})
	if status != http.StatusCreated {
		t.Fatalf("register agent failed, status=%d body=%v", status, registerAgentResponse)
	}

	agentIDText, ok := registerAgentResponse["id"].(string)
	if !ok {
		t.Fatalf("register agent response missing id: %v", registerAgentResponse)
	}
	maxTickets, ok := registerAgentResponse["max_tickets"].(float64)
	if !ok || int(maxTickets) != 2 {
		t.Fatalf("expected max_tickets to be returned as 2, got %v", registerAgentResponse)
	}

	agentID, err := uuid.Parse(agentIDText)
	if err != nil {
		t.Fatalf("invalid agent id: %v", err)
	}

	var storedAgent domain.Agent
	if err := app.db.WithContext(context.Background()).Where("id = ?", agentID).First(&storedAgent).Error; err != nil {
		t.Fatalf("failed to fetch stored agent: %v", err)
	}
	if storedAgent.MaxTickets != 2 {
		t.Fatalf("expected stored max_tickets=2, got %d", storedAgent.MaxTickets)
	}

	status, updateResponse := app.doJSONRequest(t, http.MethodPatch, fmt.Sprintf("/api/v1/agents/%s/capacity", agentID.String()), adminToken, map[string]any{
		"is_available": false,
		"max_tickets":  4,
	})
	if status != http.StatusOK {
		t.Fatalf("capacity update failed, status=%d body=%v", status, updateResponse)
	}

	if err := app.db.WithContext(context.Background()).Where("id = ?", agentID).First(&storedAgent).Error; err != nil {
		t.Fatalf("failed to reload agent: %v", err)
	}
	if storedAgent.IsAvailable {
		t.Fatalf("expected agent to be unavailable")
	}
	if storedAgent.MaxTickets != 4 {
		t.Fatalf("expected updated max_tickets=4, got %d", storedAgent.MaxTickets)
	}
}

func TestIntegrationLookupEndpointsReturnSeededMasterData(t *testing.T) {
	app := setupIntegrationApp(t)

	_, userToken := app.registerAndLogin(t, "lookup-user@example.com", "Password123", "Lookup User")

	status, categoriesResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/lookups/categories", userToken, nil)
	if status != http.StatusOK {
		t.Fatalf("categories lookup failed, status=%d body=%v", status, categoriesResponse)
	}
	categories, ok := categoriesResponse["data"].([]any)
	if !ok || len(categories) == 0 {
		t.Fatalf("expected categories data, got %v", categoriesResponse)
	}

	firstCategory, ok := categories[0].(map[string]any)
	if !ok || firstCategory["name"] == "" {
		t.Fatalf("expected category object, got %v", categoriesResponse)
	}

	status, prioritiesResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/lookups/priorities", userToken, nil)
	if status != http.StatusOK {
		t.Fatalf("priorities lookup failed, status=%d body=%v", status, prioritiesResponse)
	}
	priorities, ok := prioritiesResponse["data"].([]any)
	if !ok || len(priorities) < 4 {
		t.Fatalf("expected seeded priorities, got %v", prioritiesResponse)
	}

	status, statusesResponse := app.doJSONRequest(t, http.MethodGet, "/api/v1/lookups/statuses", userToken, nil)
	if status != http.StatusOK {
		t.Fatalf("statuses lookup failed, status=%d body=%v", status, statusesResponse)
	}
	statuses, ok := statusesResponse["data"].([]any)
	if !ok || len(statuses) < 5 {
		t.Fatalf("expected seeded statuses, got %v", statusesResponse)
	}
	firstStatus, ok := statuses[0].(map[string]any)
	if !ok || firstStatus["name"] != "open" {
		t.Fatalf("expected first seeded status to be open, got %v", statusesResponse)
	}
}
