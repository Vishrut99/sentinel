package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type LookupHandler struct {
	lookupService service.LookupService
}

func NewLookupHandler(lookupService service.LookupService) *LookupHandler {
	return &LookupHandler{lookupService: lookupService}
}

func (h *LookupHandler) RegisterRoutes(r chi.Router) {
	r.Route("/lookups", func(r chi.Router) {
		r.Use(middleware.RequirePermission(middleware.PermViewLookups))
		r.Get("/categories", h.ListCategories)
		r.Get("/priorities", h.ListPriorities)
		r.Get("/statuses", h.ListStatuses)
	})
}

func (h *LookupHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.lookupService.ListCategories(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch categories")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": categories})
}

func (h *LookupHandler) ListPriorities(w http.ResponseWriter, r *http.Request) {
	priorities, err := h.lookupService.ListPriorities(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch priorities")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": priorities})
}

func (h *LookupHandler) ListStatuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.lookupService.ListStatuses(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch statuses")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": statuses})
}
