package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type TicketHandler struct {
	ticketService service.TicketService
	validator     *validator.Validate
}

func NewTicketHandler(ticketService service.TicketService) *TicketHandler {
	return &TicketHandler{
		ticketService: ticketService,
		validator:     validator.New(),
	}
}

func (t *TicketHandler) RegisterRoutes(r chi.Router) {
	r.Route("/tickets", func(r chi.Router) {
		r.Post("/", t.CreateTicket)
		r.Get("/", t.ListTickets)
		r.Get("/{id}", t.GetTicket)
	})
}

func (t *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTicketReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if err := t.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	res, err := t.ticketService.CreateTicket(r.Context(), req, userID, role)
	if err != nil {
		t.writeServiceError(w, err)
		return
	}
	WriteJSON(w, 201, toTicketResponse(res))
}

func (t *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
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

	// parse sla_breached
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

	// parse page and per_page
	page := 0
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		parsedPage, err := strconv.Atoi(pageParam)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid page value")
			return
		}
		page = parsedPage
	}
	perPage := 0
	if perPageParam := r.URL.Query().Get("per_page"); perPageParam != "" {
		parsedPerPage, err := strconv.Atoi(perPageParam)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid per_page value")
			return
		}
		perPage = parsedPerPage
	}
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 10
	}

	filter := domain.TicketListFilter{
		Status:          r.URL.Query().Get("status"),
		Priority:        r.URL.Query().Get("priority"),
		CategoryID:      categoryID,
		Query:           r.URL.Query().Get("q"),
		IsProblem:       isProblem,
		SLABreached:     slaBreached,
		SortBy:          r.URL.Query().Get("sort_by"),
		SortDirection:   r.URL.Query().Get("sort_dir"),
		Page:            page,
		PerPage:         perPage,
		AssignedAgentID: assignedAgentID,
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	tickets, total, err := t.ticketService.ListTicket(r.Context(), filter, userID, role)
	if err != nil {
		t.writeServiceError(w, err)
		return
	}
	// then write
	WriteJSON(w, 200, toTicketListResponse(tickets, total, page, perPage))

}

func (t *TicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateTicketReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket id")
		return
	}

	if err := t.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	updatedTicket, err := t.ticketService.UpdateTicket(r.Context(), id, req, actorID, role)
	if err != nil {
		t.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, toTicketResponse(updatedTicket))
}

func (t *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket id")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	ticket, err := t.ticketService.GetTicket(r.Context(), id, userID, role)
	if err != nil {
		t.writeServiceError(w, err)
		return
	}
	WriteJSON(w, 200, toTicketResponse(ticket))

}

func (t *TicketHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateStatusReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket id")
		return
	}

	if err := t.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	err = t.ticketService.ChangeStatus(r.Context(), id, req.Status, actorID, role)
	if err != nil {
		t.writeServiceError(w, err)
		return
	}
	WriteJSON(w, 200, map[string]any{"message": "status updated"})
}

func (t *TicketHandler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	var req domain.AssignReq

	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket id")
		return
	}
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	err = t.ticketService.AssignTicket(r.Context(), id, req.AgentID, actorID)
	if err != nil {
		t.writeServiceError(w, err)
		return
	}
	WriteJSON(w, 200, map[string]any{"message": "ticket Assigned"})
}

func (t *TicketHandler) LinkToProblem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket id")
		return
	}

	var req domain.LinkProblemReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}
	if err := t.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	if err := t.ticketService.LinkToProblem(r.Context(), id, req, actorID, role); err != nil {
		t.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"message": "ticket linked to problem"})
}

func (t *TicketHandler) writeServiceError(w http.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		status := http.StatusInternalServerError
		switch appErr.Code {
		case "BAD_REQUEST":
			status = http.StatusBadRequest
		case "UNAUTHORIZED":
			status = http.StatusUnauthorized
		case "FORBIDDEN":
			status = http.StatusForbidden
		case "NOT_FOUND":
			status = http.StatusNotFound
		case "CONFLICT":
			status = http.StatusConflict
		}
		WriteError(w, status, appErr.Code, appErr.Message)
		return
	}

	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
