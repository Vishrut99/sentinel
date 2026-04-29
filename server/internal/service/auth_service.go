package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/repository"
	"github.com/yourusername/incident-ticketing/pkg/jwtutil"
)

// AuthService defines business operations for authentication.
type AuthService interface {
	Register(ctx context.Context, req domain.RegisterReq) (*domain.UserSummary, error)
	Login(ctx context.Context, req domain.LoginReq) (*domain.LoginRes, error)
	BootstrapAdmin(ctx context.Context, req domain.BootstrapAdminReq) (*domain.LoginRes, error)
}

type authService struct {
	userRepo             repository.UserRepository
	bootstrapAdminSecret string
}

func NewAuthService(userRepo repository.UserRepository, bootstrapAdminSecret string) AuthService {
	return &authService{
		userRepo:             userRepo,
		bootstrapAdminSecret: strings.TrimSpace(bootstrapAdminSecret),
	}
}

func (s *authService) Register(ctx context.Context, req domain.RegisterReq) (*domain.UserSummary, error) {
	user, err := s.createUser(ctx, req.Email, req.Password, req.FullName, "user")
	if err != nil {
		return nil, fmt.Errorf("AuthService.Register: %w", err)
	}

	return buildUserSummary(user), nil
}

func (s *authService) Login(ctx context.Context, req domain.LoginReq) (*domain.LoginRes, error) {
	return s.loginWithPassword(ctx, req.Email, req.Password)
}

func (s *authService) BootstrapAdmin(ctx context.Context, req domain.BootstrapAdminReq) (*domain.LoginRes, error) {
	if s.bootstrapAdminSecret == "" {
		return nil, fmt.Errorf("AuthService.BootstrapAdmin: %w", &domain.AppError{
			Code:    domain.ErrForbidden.Code,
			Message: "admin bootstrap is disabled",
		})
	}
	if strings.TrimSpace(req.BootstrapSecret) != s.bootstrapAdminSecret {
		return nil, fmt.Errorf("AuthService.BootstrapAdmin: %w", &domain.AppError{
			Code:    domain.ErrForbidden.Code,
			Message: "invalid bootstrap secret",
		})
	}

	adminExists, err := s.userRepo.ExistsByRole(ctx, "admin")
	if err != nil {
		return nil, fmt.Errorf("AuthService.BootstrapAdmin: %w", err)
	}
	if adminExists {
		return nil, fmt.Errorf("AuthService.BootstrapAdmin: %w", &domain.AppError{
			Code:    domain.ErrConflict.Code,
			Message: "admin already exists",
		})
	}

	user, err := s.createUser(ctx, req.Email, req.Password, req.FullName, "admin")
	if err != nil {
		return nil, fmt.Errorf("AuthService.BootstrapAdmin: %w", err)
	}

	return s.buildLoginResponse(user)
}

func (s *authService) loginWithPassword(ctx context.Context, emailInput, password string) (*domain.LoginRes, error) {
	email := strings.TrimSpace(strings.ToLower(emailInput))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("AuthService.Login: %w", domain.ErrUnauthorized.WithErr(fmt.Errorf("invalid email or password")))
		}
		return nil, fmt.Errorf("AuthService.Login: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("AuthService.Login: %w", domain.ErrUnauthorized.WithErr(fmt.Errorf("user is inactive")))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("AuthService.Login: %w", domain.ErrUnauthorized.WithErr(fmt.Errorf("invalid email or password")))
	}

	return s.buildLoginResponse(user)
}

func (s *authService) createUser(ctx context.Context, emailInput, password, fullName, role string) (*domain.User, error) {
	email := strings.TrimSpace(strings.ToLower(emailInput))
	if email == "" {
		return nil, domain.ErrBadRequest.WithErr(fmt.Errorf("email is required"))
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, &domain.AppError{
			Code:    domain.ErrConflict.Code,
			Message: "email already registered",
			Err:     fmt.Errorf("email already registered"),
		}
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		FullName:     strings.TrimSpace(fullName),
		Role:         role,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, &domain.AppError{
				Code:    domain.ErrConflict.Code,
				Message: "email already registered",
				Err:     fmt.Errorf("email already registered"),
			}
		}
		return nil, err
	}

	return user, nil
}

func (s *authService) buildLoginResponse(user *domain.User) (*domain.LoginRes, error) {
	token, err := jwtutil.SignToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &domain.LoginRes{
		Token: token,
		User:  *buildUserSummary(user),
	}, nil
}

func buildUserSummary(user *domain.User) *domain.UserSummary {
	return &domain.UserSummary{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     user.Role,
	}
}
