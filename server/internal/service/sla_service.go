package service

import (
	"context"
	"log"

	"github.com/yourusername/incident-ticketing/internal/repository"
)

type SLAService interface {
	CheckAndMarkBreaches(ctx context.Context) error
}

type slaService struct {
	ticketRepo repository.TicketRepository
}

func NewSLAService(ticketRepo repository.TicketRepository) SLAService {
	return &slaService{
		ticketRepo: ticketRepo,
	}
}

func (s *slaService) CheckAndMarkBreaches(ctx context.Context) error {
	err := s.ticketRepo.MarkSLABreached(ctx)
	if err != nil {
		log.Printf("Error marking SLA breaches: %v", err)
		return err
	}
	return nil
}
