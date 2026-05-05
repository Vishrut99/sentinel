package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/yourusername/incident-ticketing/internal/cache"
	"github.com/yourusername/incident-ticketing/internal/config"
	"github.com/yourusername/incident-ticketing/internal/db"
	"github.com/yourusername/incident-ticketing/internal/handler"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/repository"
	"github.com/yourusername/incident-ticketing/internal/service"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load .env file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	log.Printf("database connected")

	if err := db.RunMigrations(database, "internal/db/migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Printf("migrations applied")

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("failed to create sql db handle: %v", err)
	}
	defer sqlDB.Close()

	redisCache, err := cache.NewRedisCache(cfg.REDIS_URL)
	if err != nil {
		log.Printf("warning: redis cache disabled, continuing without cache: %v", err)
	}
	if redisCache != nil {
		defer func() {
			if err := redisCache.Close(); err != nil {
				log.Printf("warning: failed to close redis cache: %v", err)
			}
		}()
	}

	userRepo := repository.NewUserRepository(database)
	authService := service.NewAuthService(userRepo, cfg.BootstrapAdminSecret)
	authHandler := handler.NewAuthHandler(authService)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)
	aiTriageService := service.NewAITriageService(cfg.AI_TRIAGE_URL, cfg.AI_TRIAGE_API_KEY)
	ticketRepo := repository.NewTicketRepository(database)
	ticketService := service.NewTicketService(ticketRepo, database, redisCache, aiTriageService)
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

	slaService := service.NewSLAService(ticketRepo)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recover)
	router.Use(middleware.CORS)
	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		authHandler.RegisterRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)
			ticketHandler.RegisterRoutes(r)
			commentHandler.RegisterRoutes(r)
			agentHandler.RegisterRoutes(r)
			dashboardHandler.RegisterRoutes(r)
			auditHandler.RegisterRoutes(r)
			lookupHandler.RegisterRoutes(r)

			// Permission Restricted Endpoints
			r.With(middleware.RequirePermission(middleware.PermViewAgents)).Get("/agents", agentHandler.List)
			r.With(middleware.RequirePermission(middleware.PermPromoteAdmin)).Get("/users", userHandler.List)
			r.With(middleware.RequirePermission(middleware.PermRegisterAgent)).Post("/agents/register", agentHandler.Register)
			r.With(middleware.RequirePermission(middleware.PermManageAgentCapacity)).Patch("/agents/{id}/capacity", agentHandler.UpdateCapacity)
			r.With(middleware.RequirePermission(middleware.PermPromoteAdmin)).Patch("/users/{id}/promote-admin", userHandler.PromoteToAdmin)
			r.With(middleware.RequirePermission(middleware.PermEditTicket)).Patch("/tickets/{id}", ticketHandler.UpdateTicket)
			r.With(middleware.RequirePermission(middleware.PermAssignTicket)).Patch("/tickets/{id}/assign", ticketHandler.AssignTicket)
			r.With(middleware.RequirePermission(middleware.PermLinkProblem)).Patch("/tickets/{id}/link-problem", ticketHandler.LinkToProblem)
			r.With(middleware.RequirePermission(middleware.PermResolveTicket)).Patch("/tickets/{id}/status", ticketHandler.ChangeStatus)
			r.With(middleware.RequirePermission(middleware.PermViewDashboard)).Get("/dashboard", dashboardHandler.GetDashboard)
		})
	})

	address := ":" + strings.TrimPrefix(cfg.Port, ":")

	// Start SLA watcher
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := slaService.CheckAndMarkBreaches(context.Background()); err != nil {
				log.Printf("SLA checker failed: %v", err)
			}
		}
	}()

	log.Printf("server started on %s", address)
	if err := http.ListenAndServe(address, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
