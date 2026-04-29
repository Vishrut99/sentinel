package domain

import (
	"time"

	"github.com/google/uuid"
)

// TicketStatus maps to ticket_statuses lookup table.
type TicketStatus struct {
	ID   int16   `gorm:"column:id;primaryKey;autoIncrement"`
	Name string  `gorm:"column:name;not null"`
	Code *string `gorm:"column:code"`
}

func (TicketStatus) TableName() string { return "ticket_statuses" }

// Priority maps to priorities lookup table.
type Priority struct {
	ID             int16  `gorm:"column:id;primaryKey;autoIncrement"`
	Name           string `gorm:"column:name;not null"`
	SLAResponseHrs int    `gorm:"column:sla_response_hrs;not null"`
	SLAResolveHrs  int    `gorm:"column:sla_resolve_hrs;not null"`
}

func (Priority) TableName() string { return "priorities" }

// Category maps to categories lookup table.
type Category struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Name        string    `gorm:"column:name;not null"`
	Description *string   `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

func (Category) TableName() string { return "categories" }

// Ticket maps to the tickets table.
type Ticket struct {
	ID                      uuid.UUID     `gorm:"column:id;type:uuid;primaryKey"`
	TicketNumber            string        `gorm:"column:ticket_number;uniqueIndex;not null"`
	Title                   string        `gorm:"column:title;not null"`
	Description             *string       `gorm:"column:description"`
	AIInsights              JSONPayload   `gorm:"column:ai_insights;type:jsonb"`
	RequiredSkills          SkillNameList `gorm:"column:required_skills;type:jsonb;not null;default:'[]'::jsonb"`
	StatusID                int16         `gorm:"column:status_id;not null"`
	PriorityID              int16         `gorm:"column:priority_id;not null"`
	CategoryID              uuid.UUID     `gorm:"column:category_id;type:uuid;not null"`
	CreatedBy               uuid.UUID     `gorm:"column:created_by;type:uuid;not null"`
	AssignedTo              *uuid.UUID    `gorm:"column:assigned_to;type:uuid"`
	AssignmentJustification *string       `gorm:"column:assignment_justification"`
	ParentID                *uuid.UUID    `gorm:"column:parent_id;type:uuid"`
	IsProblem               bool          `gorm:"column:is_problem;not null;default:false"`
	SLABreached             bool          `gorm:"column:sla_breached;not null;default:false"`
	DueAt                   *time.Time    `gorm:"column:due_at"`
	ResolvedAt              *time.Time    `gorm:"column:resolved_at"`
	SkillsEvaluatedAt       *time.Time    `gorm:"column:skills_evaluated_at"`
	CreatedAt               time.Time     `gorm:"column:created_at;not null"`
	UpdatedAt               time.Time     `gorm:"column:updated_at;not null"`

	Status   TicketStatus `gorm:"foreignKey:StatusID"`
	Priority Priority     `gorm:"foreignKey:PriorityID"`
	Category Category     `gorm:"foreignKey:CategoryID"`
	Creator  User         `gorm:"foreignKey:CreatedBy"`
	Assignee *Agent       `gorm:"foreignKey:AssignedTo"`
}

func (Ticket) TableName() string { return "tickets" }

// CreateTicketReq is the payload for POST /tickets.
type CreateTicketReq struct {
	Title          string     `json:"title" validate:"required,min=5,max=255"`
	Description    string     `json:"description,omitempty"`
	Priority       string     `json:"priority" validate:"required,oneof=low medium high critical"`
	CategoryID     uuid.UUID  `json:"category_id" validate:"required"`
	IsProblem      bool       `json:"is_problem,omitempty"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	RequiredSkills []string   `json:"required_skills,omitempty"`
}

type LinkProblemReq struct {
	ParentID uuid.UUID `json:"parent_id" validate:"required"`
}

// UpdateTicketReq is the payload for PATCH /tickets/{id}.
type UpdateTicketReq struct {
	Title        *string    `json:"title,omitempty" validate:"omitempty,min=5,max=255"`
	Description  *string    `json:"description,omitempty" validate:"omitempty,max=10000"`
	Priority     *string    `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty"`
	ChangeReason *string    `json:"change_reason,omitempty" validate:"omitempty,min=3,max=1000"`
}

// UpdateStatusReq is the payload for PATCH /tickets/{id}/status.
type UpdateStatusReq struct {
	Status string `json:"status" validate:"required,oneof=open in_progress resolved closed cancelled"`
}

// AssignReq is the payload for PATCH /tickets/{id}/assign.
type AssignReq struct {
	AgentID uuid.UUID `json:"agent_id" validate:"required"`
}

// TicketListFilter represents query filters for listing tickets.
type TicketListFilter struct {
	Status          string    `json:"status,omitempty"`
	Priority        string    `json:"priority,omitempty"`
	CategoryID      uuid.UUID `json:"category_id,omitempty"`
	Query           string    `json:"q,omitempty"`
	IsProblem       *bool     `json:"is_problem,omitempty"`
	SLABreached     *bool     `json:"sla_breached,omitempty"`
	SortBy          string    `json:"sort_by,omitempty"`
	SortDirection   string    `json:"sort_dir,omitempty"`
	Page            int       `json:"page,omitempty"`
	PerPage         int       `json:"per_page,omitempty"`
	CreatedBy       uuid.UUID `json:"created_by,omitempty"`
	AssignedUserID  uuid.UUID `json:"assigned_user_id,omitempty"`
	AssignedAgentID uuid.UUID `json:"assigned_agent_id,omitempty"`
}

// TicketRes is the API response object for ticket endpoints.
type TicketRes struct {
	ID                      uuid.UUID     `json:"id"`
	TicketNumber            string        `json:"ticket_number"`
	Title                   string        `json:"title"`
	Description             string        `json:"description,omitempty"`
	AIInsights              JSONPayload   `json:"ai_insights,omitempty"`
	RequiredSkills          []string      `json:"required_skills,omitempty"`
	Status                  string        `json:"status"`
	Priority                string        `json:"priority"`
	CategoryID              uuid.UUID     `json:"category_id"`
	Category                string        `json:"category"`
	CreatedBy               UserSummary   `json:"created_by"`
	AssignedAgent           *AgentSummary `json:"assigned_agent,omitempty"`
	AssignmentJustification *string       `json:"assignment_justification,omitempty"`
	IsProblem               bool          `json:"is_problem"`
	ParentID                *uuid.UUID    `json:"parent_id,omitempty"`
	SLABreached             bool          `json:"sla_breached"`
	DueAt                   *time.Time    `json:"due_at,omitempty"`
	ResolvedAt              *time.Time    `json:"resolved_at,omitempty"`
	CreatedAt               time.Time     `json:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at"`
}

// TicketListRes wraps paginated ticket list responses.
type TicketListRes struct {
	Data    []TicketRes `json:"data"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}
