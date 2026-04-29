package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/repository"
)

type UserService interface {
	ListUsers(ctx context.Context, filter domain.UserListFilter) ([]domain.UserSummary, error)
	PromoteToAdmin(ctx context.Context, userID uuid.UUID) (*domain.UserSummary, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) ListUsers(ctx context.Context, filter domain.UserListFilter) ([]domain.UserSummary, error) {
	users, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("UserService.ListUsers: %w", err)
	}

	summaries := make([]domain.UserSummary, 0, len(users))
	for i := range users {
		user := users[i]
		summaries = append(summaries, domain.UserSummary{
			ID:       user.ID,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     user.Role,
		})
	}

	return summaries, nil
}

func (s *userService) PromoteToAdmin(ctx context.Context, userID uuid.UUID) (*domain.UserSummary, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserService.PromoteToAdmin: %w", err)
	}

	switch user.Role {
	case "admin":
		return nil, fmt.Errorf("UserService.PromoteToAdmin: %w", domain.ErrConflict.WithErr(fmt.Errorf("user is already an admin")))
	case "agent":
		return nil, fmt.Errorf("UserService.PromoteToAdmin: %w", domain.ErrConflict.WithErr(fmt.Errorf("agent users cannot be promoted through this endpoint")))
	}

	if err := s.userRepo.UpdateRole(ctx, userID, "admin"); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("UserService.PromoteToAdmin: %w", err)
		}
		return nil, fmt.Errorf("UserService.PromoteToAdmin: %w", err)
	}

	user.Role = "admin"
	return &domain.UserSummary{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     user.Role,
	}, nil
}
