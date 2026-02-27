package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ricountzero/SubscriptionLedger/internal/model"
	"github.com/ricountzero/SubscriptionLedger/internal/repository"
	"go.uber.org/zap"
)

const dateLayout = "01-2006"

type SubscriptionService struct {
	repo   *repository.SubscriptionRepository
	logger *zap.Logger
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, logger *zap.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, logger: logger}
}

func parseMonthYear(s string) (time.Time, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format %q, expected MM-YYYY", s)
	}
	return t, nil
}

func formatMonthYear(t time.Time) string {
	return t.Format(dateLayout)
}

func toResponse(s *model.Subscription) *model.SubscriptionResponse {
	resp := &model.SubscriptionResponse{
		ID: s.ID, ServiceName: s.ServiceName, Price: s.Price,
		UserID: s.UserID, StartDate: formatMonthYear(s.StartDate),
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.EndDate != nil {
		str := formatMonthYear(*s.EndDate)
		resp.EndDate = &str
	}
	return resp
}

func (s *SubscriptionService) Create(ctx context.Context, req *model.CreateSubscriptionRequest) (*model.SubscriptionResponse, error) {
	startDate, err := parseMonthYear(req.StartDate)
	if err != nil {
		return nil, err
	}
	sub := &model.Subscription{
		ID: uuid.New(), ServiceName: req.ServiceName,
		Price: req.Price, UserID: req.UserID, StartDate: startDate,
	}
	if req.EndDate != nil {
		endDate, err := parseMonthYear(*req.EndDate)
		if err != nil {
			return nil, err
		}
		if !endDate.After(startDate) {
			return nil, fmt.Errorf("end_date must be after start_date")
		}
		sub.EndDate = &endDate
	}
	s.logger.Info("creating subscription", zap.String("service", req.ServiceName))
	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		s.logger.Error("failed to create subscription", zap.Error(err))
		return nil, err
	}
	return toResponse(created), nil
}

func (s *SubscriptionService) GetByID(ctx context.Context, id uuid.UUID) (*model.SubscriptionResponse, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	return toResponse(sub), nil
}

func (s *SubscriptionService) List(ctx context.Context, userID *uuid.UUID, serviceName *string) ([]*model.SubscriptionResponse, error) {
	subs, err := s.repo.List(ctx, userID, serviceName)
	if err != nil {
		return nil, err
	}
	result := make([]*model.SubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		result = append(result, toResponse(sub))
	}
	return result, nil
}

func (s *SubscriptionService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateSubscriptionRequest) (*model.SubscriptionResponse, error) {
	fields := map[string]interface{}{}
	if req.ServiceName != nil {
		fields["service_name"] = *req.ServiceName
	}
	if req.Price != nil {
		if *req.Price < 1 {
			return nil, fmt.Errorf("price must be positive")
		}
		fields["price"] = *req.Price
	}
	if req.StartDate != nil {
		t, err := parseMonthYear(*req.StartDate)
		if err != nil {
			return nil, err
		}
		fields["start_date"] = t
	}
	if req.EndDate != nil {
		t, err := parseMonthYear(*req.EndDate)
		if err != nil {
			return nil, err
		}
		fields["end_date"] = t
	}
	sub, err := s.repo.Update(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	return toResponse(sub), nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.Delete(ctx, id)
}

func (s *SubscriptionService) TotalCost(ctx context.Context, req *model.TotalCostRequest) (*model.TotalCostResponse, error) {
	from, err := parseMonthYear(req.PeriodFrom)
	if err != nil {
		return nil, fmt.Errorf("period_from: %w", err)
	}
	to, err := parseMonthYear(req.PeriodTo)
	if err != nil {
		return nil, fmt.Errorf("period_to: %w", err)
	}
	if to.Before(from) {
		return nil, fmt.Errorf("period_to must be >= period_from")
	}
	total, err := s.repo.TotalCost(ctx, req.UserID, req.ServiceName, from, to)
	if err != nil {
		return nil, err
	}
	return &model.TotalCostResponse{TotalCost: total}, nil
}
