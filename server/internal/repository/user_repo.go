package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourusername/incident-ticketing/internal/domain"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, error)
	ExistsByRole(ctx context.Context, role string) (bool, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("UserRepository.Create: %w", err)
	}
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("UserRepository.GetByEmail: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("UserRepository.GetByEmail: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("UserRepository.GetByID: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("UserRepository.GetByID: %w", err)
	}
	return &user, nil
}

func (r *userRepository) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, error) {
	query := r.db.WithContext(ctx).Model(&domain.User{})
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("full_name ILIKE ? OR email ILIKE ?", search, search)
	}

	var users []domain.User
	if err := query.Order("full_name ASC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("UserRepository.List: %w", err)
	}

	return users, nil
}

func (r *userRepository) ExistsByRole(ctx context.Context, role string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", role).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("UserRepository.ExistsByRole: %w", err)
	}
	return count > 0, nil
}

func (r *userRepository) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Update("role", role)
	if result.Error != nil {
		return fmt.Errorf("UserRepository.UpdateRole: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
