package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/cache"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/repository"
)

const (
	dashboardStatsTTL = 2 * time.Minute
)

type DashboardService interface {
	GetDashboard(ctx context.Context, role string, userID uuid.UUID, filter domain.TicketListFilter) (*domain.DashboardStatsRes, error)
}

type dashboardService struct {
	dashboardRepo repository.DashboardRepository
	redisCache    *cache.RedisCache
}

func NewDashboardService(dashboardRepo repository.DashboardRepository, redisCache *cache.RedisCache) DashboardService {
	return &dashboardService{
		dashboardRepo: dashboardRepo,
		redisCache:    redisCache,
	}
}

func (s *dashboardService) GetDashboard(ctx context.Context, role string, userID uuid.UUID, filter domain.TicketListFilter) (*domain.DashboardStatsRes, error) {
	if !middleware.HasPermission(role, middleware.PermViewDashboard) {
		return nil, domain.ErrForbidden
	}

	if !middleware.HasPermission(role, middleware.PermViewAllTickets) {
		filter.AssignedUserID = userID
	}

	useCache := shouldUseDashboardCache(role, filter)
	cacheKey := cache.DashboardStatsCacheKey
	if useCache {
		cacheKey = buildDashboardCacheKey(role, userID, filter)
	}
	if s.redisCache != nil && useCache {
		cachedValue, hit, err := s.redisCache.Get(ctx, cacheKey)
		if err != nil {
			log.Printf("DashboardService.GetDashboard cache get failed: %v", err)
		} else if hit {
			var cachedStats domain.DashboardStatsRes
			if err := json.Unmarshal(cachedValue, &cachedStats); err == nil {
				return &cachedStats, nil
			}
			log.Printf("DashboardService.GetDashboard cache unmarshal failed: %v", err)
		}
	}

	stats, err := s.dashboardRepo.GetDashboardStats(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("DashboardService.GetDashboard: %w", err)
	}

	if s.redisCache != nil && useCache {
		cacheValue, err := json.Marshal(stats)
		if err != nil {
			log.Printf("DashboardService.GetDashboard cache marshal failed: %v", err)
		} else if err := s.redisCache.Set(ctx, cacheKey, cacheValue, dashboardStatsTTL); err != nil {
			log.Printf("DashboardService.GetDashboard cache set failed: %v", err)
		}
	}

	return stats, nil
}

func shouldUseDashboardCache(role string, filter domain.TicketListFilter) bool {
	return middleware.HasPermission(role, middleware.PermViewAllTickets) &&
		filter.Status == "" &&
		filter.Priority == "" &&
		filter.CategoryID == uuid.Nil &&
		filter.Query == "" &&
		filter.IsProblem == nil &&
		filter.SLABreached == nil &&
		filter.CreatedBy == uuid.Nil &&
		filter.AssignedUserID == uuid.Nil &&
		filter.AssignedAgentID == uuid.Nil
}

func buildDashboardCacheKey(role string, userID uuid.UUID, filter domain.TicketListFilter) string {
	payload, err := json.Marshal(struct {
		Role   string                  `json:"role"`
		UserID uuid.UUID               `json:"user_id"`
		Filter domain.TicketListFilter `json:"filter"`
	}{
		Role:   role,
		UserID: userID,
		Filter: filter,
	})
	if err != nil {
		return cache.DashboardStatsCacheKey
	}

	hash := sha1.Sum(payload)
	return cache.DashboardStatsCacheKey + ":" + hex.EncodeToString(hash[:])
}
