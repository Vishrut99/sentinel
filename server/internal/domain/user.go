package domain

import (
	"time"

	"github.com/google/uuid"
)

// User maps to the users table.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"column:email;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	FullName     string    `gorm:"column:full_name;not null"`
	Role         string    `gorm:"column:role;not null;default:user"`
	IsActive     bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (User) TableName() string { return "users" }

// RegisterReq is the payload for POST /auth/register.
type RegisterReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
}

// LoginReq is the payload for POST /auth/login.
type LoginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type BootstrapAdminReq struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	FullName        string `json:"full_name" validate:"required,min=2,max=100"`
	BootstrapSecret string `json:"bootstrap_secret" validate:"required"`
}

type UserListFilter struct {
	Role   string `json:"role,omitempty"`
	Search string `json:"search,omitempty"`
}

// UserSummary is the public user payload returned in API responses.
type UserSummary struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
}

// LoginRes is the response for POST /auth/login.
type LoginRes struct {
	Token string      `json:"token"`
	User  UserSummary `json:"user"`
}
