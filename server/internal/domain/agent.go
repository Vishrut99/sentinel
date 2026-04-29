package domain

import (
	"time"

	"github.com/google/uuid"
)

// Agent maps to the agents table.
type Agent struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID      `gorm:"column:user_id;type:uuid;not null;uniqueIndex"`
	Department  *string        `gorm:"column:department"`
	IsAvailable bool           `gorm:"column:is_available;not null;default:true"`
	MaxTickets  int            `gorm:"column:max_tickets;not null;default:10"`
	Skills      SkillScoreList `gorm:"column:skills;type:jsonb;not null;default:'[]'::jsonb"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null"`

	User          User `gorm:"foreignKey:UserID"`
	ActiveTickets int  `gorm:"-:all"`
}

func (Agent) TableName() string { return "agents" }

// RegisterAgentReq is the payload for POST /agents.
type RegisterAgentReq struct {
	UserID     uuid.UUID      `json:"user_id" validate:"required"`
	Department string         `json:"department" validate:"required,min=2,max=100"`
	MaxTickets *int           `json:"max_tickets,omitempty" validate:"omitempty,min=1,max=100"`
	Skills     SkillScoreList `json:"skills,omitempty"`
}

type UpdateAgentCapacityReq struct {
	IsAvailable *bool           `json:"is_available,omitempty"`
	MaxTickets  *int            `json:"max_tickets,omitempty" validate:"omitempty,min=1,max=100"`
	Skills      *SkillScoreList `json:"skills,omitempty"`
}

// AgentSummary is the public agent payload used in ticket/agent responses.
type AgentSummary struct {
	ID                 uuid.UUID      `json:"id"`
	FullName           string         `json:"full_name"`
	Department         string         `json:"department,omitempty"`
	IsAvailable        bool           `json:"is_available"`
	MaxTickets         int            `json:"max_tickets"`
	Skills             SkillScoreList `json:"skills,omitempty"`
	ActiveTickets      int            `json:"active_tickets"`
	UtilizationPercent int            `json:"utilization_percent"`
}
