package service

import (
	"context"
	"fmt"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/repository"
)

type LookupService interface {
	ListCategories(ctx context.Context) ([]domain.CategoryLookupRes, error)
	ListPriorities(ctx context.Context) ([]domain.PriorityLookupRes, error)
	ListStatuses(ctx context.Context) ([]domain.StatusLookupRes, error)
}

type lookupService struct {
	lookupRepo repository.LookupRepository
}

func NewLookupService(lookupRepo repository.LookupRepository) LookupService {
	return &lookupService{lookupRepo: lookupRepo}
}

func (s *lookupService) ListCategories(ctx context.Context) ([]domain.CategoryLookupRes, error) {
	categories, err := s.lookupRepo.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("LookupService.ListCategories: %w", err)
	}

	response := make([]domain.CategoryLookupRes, 0, len(categories))
	for _, category := range categories {
		description := ""
		if category.Description != nil {
			description = *category.Description
		}
		response = append(response, domain.CategoryLookupRes{
			ID:          category.ID,
			Name:        category.Name,
			Description: description,
		})
	}

	return response, nil
}

func (s *lookupService) ListPriorities(ctx context.Context) ([]domain.PriorityLookupRes, error) {
	priorities, err := s.lookupRepo.ListPriorities(ctx)
	if err != nil {
		return nil, fmt.Errorf("LookupService.ListPriorities: %w", err)
	}

	response := make([]domain.PriorityLookupRes, 0, len(priorities))
	for _, priority := range priorities {
		response = append(response, domain.PriorityLookupRes{
			ID:             priority.ID,
			Name:           priority.Name,
			SLAResponseHrs: priority.SLAResponseHrs,
			SLAResolveHrs:  priority.SLAResolveHrs,
		})
	}

	return response, nil
}

func (s *lookupService) ListStatuses(ctx context.Context) ([]domain.StatusLookupRes, error) {
	statuses, err := s.lookupRepo.ListStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("LookupService.ListStatuses: %w", err)
	}

	response := make([]domain.StatusLookupRes, 0, len(statuses))
	for _, status := range statuses {
		code := ""
		if status.Code != nil {
			code = *status.Code
		}
		response = append(response, domain.StatusLookupRes{
			ID:   status.ID,
			Name: status.Name,
			Code: code,
		})
	}

	return response, nil
}
