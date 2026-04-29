package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog maps to the audit_logs table (read-only from Go code).
type AuditLog struct {
	ID        uuid.UUID   `gorm:"column:id;type:uuid;primaryKey"`
	TicketID  uuid.UUID   `gorm:"column:ticket_id;type:uuid;not null"`
	ActorID   uuid.UUID   `gorm:"column:actor_id;type:uuid;not null"`
	Action    string      `gorm:"column:action;not null"`
	OldValue  JSONPayload `gorm:"column:old_value;type:jsonb"`
	NewValue  JSONPayload `gorm:"column:new_value;type:jsonb"`
	CreatedAt time.Time   `gorm:"column:created_at;not null"`

	Actor User `gorm:"foreignKey:ActorID"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// AuditLogRes is the API response object for ticket audit endpoints.
type AuditLogRes struct {
	ID        uuid.UUID      `json:"id"`
	Action    string         `json:"action"`
	Actor     UserSummary    `json:"actor"`
	OldValue  map[string]any `json:"old_value,omitempty"`
	NewValue  map[string]any `json:"new_value,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// DashboardStatsRes is the response shape for GET /dashboard.
type DashboardStatsRes struct {
	TotalTickets      int64            `json:"total_tickets"`
	OpenTickets       int64            `json:"open_tickets"`
	InProgressTickets int64            `json:"in_progress_tickets"`
	ResolvedToday     int64            `json:"resolved_today"`
	SLABreached       int64            `json:"sla_breached"`
	AvgResolveHours   float64          `json:"avg_resolve_hours"`
	ByPriority        map[string]int64 `json:"by_priority"`
	ByCategory        map[string]int64 `json:"by_category"`
}
