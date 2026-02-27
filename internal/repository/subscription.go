package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ricountzero/SubscriptionLedger/internal/model"
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, s *model.Subscription) (*model.Subscription, error) {
	query := `INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`
	row := r.db.QueryRow(ctx, query, s.ID, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate)
	return scanSubscription(row)
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)
	sub, err := scanSubscription(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return sub, err
}

func (r *SubscriptionRepository) List(ctx context.Context, userID *uuid.UUID, serviceName *string) ([]*model.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	argN := 1
	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argN)
		args = append(args, *userID)
		argN++
	}
	if serviceName != nil {
		query += fmt.Sprintf(" AND service_name ILIKE $%d", argN)
		args = append(args, "%"+*serviceName+"%")
		argN++
	}
	query += " ORDER BY created_at DESC"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []*model.Subscription
	for rows.Next() {
		sub := &model.Subscription{}
		if err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID,
			&sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SubscriptionRepository) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (*model.Subscription, error) {
	if len(fields) == 0 {
		return r.GetByID(ctx, id)
	}
	query := "UPDATE subscriptions SET updated_at = NOW()"
	args := []interface{}{}
	argN := 1
	for k, v := range fields {
		query += fmt.Sprintf(", %s = $%d", k, argN)
		args = append(args, v)
		argN++
	}
	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at", argN)
	args = append(args, id)
	row := r.db.QueryRow(ctx, query, args...)
	sub, err := scanSubscription(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return sub, err
}

func (r *SubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.db.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (r *SubscriptionRepository) TotalCost(ctx context.Context, userID *uuid.UUID, serviceName *string, from, to time.Time) (int, error) {
	query := `SELECT COALESCE(SUM(price), 0) FROM subscriptions
		WHERE start_date <= $1 AND (end_date IS NULL OR end_date >= $2)`
	args := []interface{}{to, from}
	argN := 3
	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argN)
		args = append(args, *userID)
		argN++
	}
	if serviceName != nil {
		query += fmt.Sprintf(" AND service_name ILIKE $%d", argN)
		args = append(args, "%"+*serviceName+"%")
		argN++
	}
	var total int
	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	return total, err
}

func scanSubscription(row pgx.Row) (*model.Subscription, error) {
	sub := &model.Subscription{}
	err := row.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID,
		&sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return sub, nil
}
