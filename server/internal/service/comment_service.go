package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
)

type commentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	ListByTicketID(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]domain.Comment, error)
}

type CommentService interface {
	CreateComment(ctx context.Context, ticketID uuid.UUID, req domain.CreateCommentReq, actorID uuid.UUID, role string) (*domain.Comment, error)
	ListComments(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, role string) ([]domain.Comment, error)
}

type commentService struct {
	commentRepo commentRepository
	ticketRepo  ticketAccessReader
}

func NewCommentService(commentRepo commentRepository, ticketRepo ticketAccessReader) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		ticketRepo:  ticketRepo,
	}
}

func (s *commentService) CreateComment(ctx context.Context, ticketID uuid.UUID, req domain.CreateCommentReq, actorID uuid.UUID, role string) (*domain.Comment, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("CommentService.CreateComment: %w", err)
	}
	if err := ensureTicketAccess(ticket, actorID, role); err != nil {
		return nil, fmt.Errorf("CommentService.CreateComment: %w", err)
	}
	if req.IsInternal && !middleware.HasPermission(role, middleware.PermCommentInternal) {
		return nil, fmt.Errorf("CommentService.CreateComment: %w", domain.ErrForbidden)
	}

	comment := &domain.Comment{
		ID:         uuid.New(),
		TicketID:   ticketID,
		AuthorID:   actorID,
		Body:       req.Body,
		IsInternal: req.IsInternal,
		CreatedAt:  time.Now(),
	}
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("CommentService.CreateComment: %w", err)
	}

	return comment, nil
}

func (s *commentService) ListComments(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, role string) ([]domain.Comment, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("CommentService.ListComments: %w", err)
	}
	if err := ensureTicketAccess(ticket, actorID, role); err != nil {
		return nil, fmt.Errorf("CommentService.ListComments: %w", err)
	}

	includeInternal := middleware.HasPermission(role, middleware.PermViewInternalComment)
	comments, err := s.commentRepo.ListByTicketID(ctx, ticketID, includeInternal)
	if err != nil {
		return nil, fmt.Errorf("CommentService.ListComments: %w", err)
	}

	return comments, nil
}
