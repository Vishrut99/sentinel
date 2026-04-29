package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type DashboardHandler struct {
	dashboardService service.DashboardService
}

func NewDashboardHandler(dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

func (h *DashboardHandler) RegisterRoutes(r chi.Router) {
	// Dashboard requires explicit elevated permissions mapping centrally in main
}

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	var categoryID uuid.UUID
	if categoryIDParam := r.URL.Query().Get("category_id"); categoryIDParam != "" {
		parsedCategoryID, err := uuid.Parse(categoryIDParam)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid category id")
			return
		}
		categoryID = parsedCategoryID
	}

	var assignedAgentID uuid.UUID
	if assignedAgentParam := r.URL.Query().Get("assigned_agent_id"); assignedAgentParam != "" {
		parsedAssignedAgentID, err := uuid.Parse(assignedAgentParam)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid assigned agent id")
			return
		}
		assignedAgentID = parsedAssignedAgentID
	}

	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	var slaBreached *bool
	if val := r.URL.Query().Get("sla_breached"); val != "" {
		b, err := strconv.ParseBool(val)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid sla_breached value")
			return
		}
		slaBreached = &b
	}

	var isProblem *bool
	if val := r.URL.Query().Get("is_problem"); val != "" {
		b, err := strconv.ParseBool(val)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid is_problem value")
			return
		}
		isProblem = &b
	}

	filter := domain.TicketListFilter{
		Status:          r.URL.Query().Get("status"),
		Priority:        r.URL.Query().Get("priority"),
		CategoryID:      categoryID,
		Query:           r.URL.Query().Get("q"),
		IsProblem:       isProblem,
		SLABreached:     slaBreached,
		AssignedAgentID: assignedAgentID,
	}

	stats, err := h.dashboardService.GetDashboard(r.Context(), role, userID, filter)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, stats)
}
