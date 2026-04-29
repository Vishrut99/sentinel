package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type AgentHandler struct {
	agentService service.AgentService
	validator    *validator.Validate
}

func NewAgentHandler(agentService service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		validator:    validator.New(),
	}
}

func (h *AgentHandler) RegisterRoutes(r chi.Router) {}

func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterAgentReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	res, err := h.agentService.Register(r.Context(), req, actorID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, res)
}

func (h *AgentHandler) UpdateCapacity(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || agentID == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid agent id")
		return
	}

	var req domain.UpdateAgentCapacityReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.agentService.UpdateCapacity(r.Context(), agentID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	agents, err := h.agentService.List(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": agents})
}

func (h *AgentHandler) writeServiceError(w http.ResponseWriter, err error) {
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
