package domain

import (
	"time"

	"github.com/google/uuid"
)

// Comment maps to the comments table.
type Comment struct {
	ID         uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	TicketID   uuid.UUID `gorm:"column:ticket_id;type:uuid;not null"`
	AuthorID   uuid.UUID `gorm:"column:author_id;type:uuid;not null"`
	Body       string    `gorm:"column:body;not null"`
	IsInternal bool      `gorm:"column:is_internal;not null;default:false"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`

	Ticket Ticket `json:"-" gorm:"foreignKey:TicketID"`
	Author User   `gorm:"foreignKey:AuthorID"`
}

func (Comment) TableName() string { return "comments" }

// CreateCommentReq is the payload for POST /tickets/{id}/comments.
type CreateCommentReq struct {
	Body       string `json:"body" validate:"required,min=1,max=10000"`
	IsInternal bool   `json:"is_internal"`
}

// CommentRes is the API response object for comment endpoints.
type CommentRes struct {
	ID         uuid.UUID   `json:"id"`
	TicketID   uuid.UUID   `json:"ticket_id"`
	Author     UserSummary `json:"author"`
	Body       string      `json:"body"`
	IsInternal bool        `json:"is_internal"`
	CreatedAt  time.Time   `json:"created_at"`
}
