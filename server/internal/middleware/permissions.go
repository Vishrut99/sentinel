package middleware

import (
	"encoding/json"
	"net/http"
)

const (
	PermEditTicket           = "edit:ticket"
	PermAssignTicket         = "assign:ticket"
	PermResolveTicket        = "resolve:ticket"
	PermLinkProblem          = "link:problem"
	PermRegisterAgent        = "register:agent"
	PermViewDashboard        = "view:dashboard"
	PermViewLookups          = "view:lookups"
	PermManageAgentCapacity  = "manage:agent-capacity"
	PermCommentAnyTicket     = "comment:any-ticket"
	PermViewAnyTicketComment = "view:any-ticket-comment"
	PermCommentInternal      = "comment:internal"
	PermViewInternalComment  = "view:internal-comment"
	PermPromoteAdmin         = "promote:admin"
	PermViewAgents           = "view:agents"
	PermCreateProblemTicket  = "create:problem-ticket"
	PermViewAllTickets       = "view:all-tickets"
	PermViewAssignedTickets  = "view:assigned-tickets"
)

// RolePermissions maps a role to a list of allowed permissions.
var RolePermissions = map[string][]string{
	"admin": {
		PermEditTicket, PermAssignTicket, PermResolveTicket, PermLinkProblem,
		PermRegisterAgent, PermViewDashboard, PermViewLookups, PermManageAgentCapacity,
		PermCommentAnyTicket, PermViewAnyTicketComment, PermCommentInternal, PermViewInternalComment,
		PermPromoteAdmin, PermViewAgents, PermCreateProblemTicket, PermViewAllTickets,
	},
	"agent": {
		PermEditTicket, PermResolveTicket, PermLinkProblem, PermViewDashboard, PermViewLookups,
		PermCommentAnyTicket, PermViewAnyTicketComment, PermCommentInternal, PermViewInternalComment,
		PermViewAgents, PermCreateProblemTicket, PermViewAssignedTickets,
	},
	"user": {PermViewLookups},
}

// HasPermission checks whether a role includes the requested permission.
func HasPermission(role, perm string) bool {
	perms, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, candidate := range perms {
		if candidate == perm {
			return true
		}
	}

	return false
}

// RequirePermission is a middleware that enforces RBAC authorization.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "UNAUTHORIZED", "message": "missing authorization role"})
				return
			}

			if !HasPermission(role, perm) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "FORBIDDEN", "message": "You are not authorized to perform this operation."})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
