package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/viva/rate-limiter/internal/models"
)

// BillingRecordRepository defines the interface for billing record data access
type BillingRecordRepository interface {
	Create(ctx context.Context, record *models.BillingRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.BillingRecord, error)
	Update(ctx context.Context, record *models.BillingRecord) error
	List(ctx context.Context, filter *BillingFilter, pagination *PaginationParams) (*PaginatedResult, error)
	GetByAPIKey(ctx context.Context, apiKeyID uuid.UUID, startDate, endDate time.Time) ([]*models.BillingRecord, error)
	GetUnpaidRecords(ctx context.Context, apiKeyID uuid.UUID) ([]*models.BillingRecord, error)
	GetCurrentPeriodRecord(ctx context.Context, apiKeyID uuid.UUID) (*models.BillingRecord, error)
	UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status models.PaymentStatus, paidAt *time.Time) error
	GetRevenueSummary(ctx context.Context, startDate, endDate time.Time) (*RevenueSummary, error)
	GetOverdueRecords(ctx context.Context, daysOverdue int) ([]*models.BillingRecord, error)
	GetTopRevenue(ctx context.Context, startDate, endDate time.Time, limit int) ([]*RevenueByAPIKey, error)
	CalculateTotalOwed(ctx context.Context, apiKeyID uuid.UUID) (float64, error)
}

// BillingFilter contains filter parameters for billing record queries
type BillingFilter struct {
	APIKeyID      *uuid.UUID             `json:"api_key_id"`
	PaymentStatus *models.PaymentStatus  `json:"payment_status"`
	StartDate     *time.Time             `json:"start_date"`
	EndDate       *time.Time             `json:"end_date"`
	MinAmount     *float64               `json:"min_amount"`
	MaxAmount     *float64               `json:"max_amount"`
	HasOverage    *bool                  `json:"has_overage"`
}

// RevenueSummary contains aggregated revenue statistics
type RevenueSummary struct {
	TotalRevenue      float64            `json:"total_revenue"`
	PaidRevenue       float64            `json:"paid_revenue"`
	UnpaidRevenue     float64            `json:"unpaid_revenue"`
	OverdueRevenue    float64            `json:"overdue_revenue"`
	TotalOverages     float64            `json:"total_overages"`
	RecordCount       int64              `json:"record_count"`
	ByPaymentStatus   map[string]float64 `json:"by_payment_status"`
	AverageRecordValue float64           `json:"average_record_value"`
}

// RevenueByAPIKey contains revenue information grouped by API key
type RevenueByAPIKey struct {
	APIKeyID      uuid.UUID `json:"api_key_id"`
	TotalRevenue  float64   `json:"total_revenue"`
	PaidRevenue   float64   `json:"paid_revenue"`
	UnpaidRevenue float64   `json:"unpaid_revenue"`
	RecordCount   int64     `json:"record_count"`
}

// billingRecordRepository implements BillingRecordRepository interface
type billingRecordRepository struct {
	*baseRepository
}

// NewBillingRecordRepository creates a new billing record repository
func NewBillingRecordRepository(db *gorm.DB) BillingRecordRepository {
	return &billingRecordRepository{
		baseRepository: NewBaseRepository(db),
	}
}

// Create creates a new billing record
func (r *billingRecordRepository) Create(ctx context.Context, record *models.BillingRecord) error {
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("failed to create billing record: %w", err)
	}
	return nil
}

// GetByID retrieves a billing record by ID
func (r *billingRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BillingRecord, error) {
	var record models.BillingRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("billing record not found")
		}
		return nil, fmt.Errorf("failed to get billing record: %w", err)
	}
	return &record, nil
}

// Update updates a billing record
func (r *billingRecordRepository) Update(ctx context.Context, record *models.BillingRecord) error {
	result := r.db.WithContext(ctx).Model(record).Updates(record)
	if result.Error != nil {
		return fmt.Errorf("failed to update billing record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("billing record not found")
	}
	return nil
}

// List retrieves billing records with filtering and pagination
func (r *billingRecordRepository) List(ctx context.Context, filter *BillingFilter, pagination *PaginationParams) (*PaginatedResult, error) {
	query := r.db.WithContext(ctx).Model(&models.BillingRecord{})

	// Apply filters
	if filter != nil {
		if filter.APIKeyID != nil {
			query = query.Where("api_key_id = ?", *filter.APIKeyID)
		}
		if filter.PaymentStatus != nil {
			query = query.Where("payment_status = ?", *filter.PaymentStatus)
		}
		if filter.StartDate != nil {
			query = query.Where("period_start >= ?", *filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("period_end <= ?", *filter.EndDate)
		}
		if filter.MinAmount != nil {
			query = query.Where("total_amount >= ?", *filter.MinAmount)
		}
		if filter.MaxAmount != nil {
			query = query.Where("total_amount <= ?", *filter.MaxAmount)
		}
		if filter.HasOverage != nil {
			if *filter.HasOverage {
				query = query.Where("overage_charges > 0")
			} else {
				query = query.Where("overage_charges = 0")
			}
		}
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count billing records: %w", err)
	}

	// Apply pagination
	var records []models.BillingRecord
	if err := query.
		Order(pagination.GetOrderBy()).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list billing records: %w", err)
	}

	return NewPaginatedResult(records, total, pagination), nil
}

// GetByAPIKey retrieves billing records for a specific API key within a date range
func (r *billingRecordRepository) GetByAPIKey(ctx context.Context, apiKeyID uuid.UUID, startDate, endDate time.Time) ([]*models.BillingRecord, error) {
	var records []*models.BillingRecord
	query := r.db.WithContext(ctx).Where("api_key_id = ?", apiKeyID)

	if !startDate.IsZero() {
		query = query.Where("period_start >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("period_end <= ?", endDate)
	}

	if err := query.Order("period_start DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get billing records by api key: %w", err)
	}
	return records, nil
}

// GetUnpaidRecords retrieves unpaid billing records for an API key
func (r *billingRecordRepository) GetUnpaidRecords(ctx context.Context, apiKeyID uuid.UUID) ([]*models.BillingRecord, error) {
	var records []*models.BillingRecord
	query := r.db.WithContext(ctx).
		Where("payment_status IN ?", []models.PaymentStatus{
			models.PaymentStatusPending,
			models.PaymentStatusOverdue,
			models.PaymentStatusFailed,
		})

	if apiKeyID != uuid.Nil {
		query = query.Where("api_key_id = ?", apiKeyID)
	}

	if err := query.Order("period_start ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get unpaid records: %w", err)
	}
	return records, nil
}

// GetCurrentPeriodRecord retrieves the billing record for the current period
func (r *billingRecordRepository) GetCurrentPeriodRecord(ctx context.Context, apiKeyID uuid.UUID) (*models.BillingRecord, error) {
	var record models.BillingRecord
	now := time.Now()

	if err := r.db.WithContext(ctx).
		Where("api_key_id = ? AND period_start <= ? AND period_end >= ?", 
			apiKeyID, now, now).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No current period record
		}
		return nil, fmt.Errorf("failed to get current period record: %w", err)
	}
	return &record, nil
}

// UpdatePaymentStatus updates the payment status of a billing record
func (r *billingRecordRepository) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status models.PaymentStatus, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"payment_status": status,
	}
	if paidAt != nil {
		updates["paid_at"] = *paidAt
	}

	result := r.db.WithContext(ctx).
		Model(&models.BillingRecord{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update payment status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("billing record not found")
	}
	return nil
}

// GetRevenueSummary retrieves aggregated revenue statistics
func (r *billingRecordRepository) GetRevenueSummary(ctx context.Context, startDate, endDate time.Time) (*RevenueSummary, error) {
	summary := &RevenueSummary{
		ByPaymentStatus: make(map[string]float64),
	}

	// Base query
	baseQuery := r.db.WithContext(ctx).Model(&models.BillingRecord{})
	if !startDate.IsZero() {
		baseQuery = baseQuery.Where("period_start >= ?", startDate)
	}
	if !endDate.IsZero() {
		baseQuery = baseQuery.Where("period_end <= ?", endDate)
	}

	// Total revenue and count
	var totals struct {
		TotalRevenue float64
		RecordCount  int64
	}
	if err := baseQuery.
		Select("COALESCE(SUM(total_amount), 0) as total_revenue, COUNT(*) as record_count").
		Scan(&totals).Error; err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}
	summary.TotalRevenue = totals.TotalRevenue
	summary.RecordCount = totals.RecordCount

	if summary.RecordCount > 0 {
		summary.AverageRecordValue = summary.TotalRevenue / float64(summary.RecordCount)
	}

	// Revenue by payment status
	statuses := []models.PaymentStatus{
		models.PaymentStatusPending,
		models.PaymentStatusPaid,
		models.PaymentStatusOverdue,
		models.PaymentStatusFailed,
		models.PaymentStatusRefunded,
	}

	for _, status := range statuses {
		var amount float64
		if err := baseQuery.
			Where("payment_status = ?", status).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&amount).Error; err != nil {
			return nil, fmt.Errorf("failed to get revenue for status %s: %w", status, err)
		}
		summary.ByPaymentStatus[string(status)] = amount

		// Set specific status totals
		switch status {
		case models.PaymentStatusPaid:
			summary.PaidRevenue = amount
		case models.PaymentStatusPending:
			summary.UnpaidRevenue += amount
		case models.PaymentStatusOverdue:
			summary.OverdueRevenue = amount
			summary.UnpaidRevenue += amount
		case models.PaymentStatusFailed:
			summary.UnpaidRevenue += amount
		}
	}

	// Total overages
	if err := baseQuery.
		Select("COALESCE(SUM(overage_charges), 0)").
		Scan(&summary.TotalOverages).Error; err != nil {
		return nil, fmt.Errorf("failed to get total overages: %w", err)
	}

	return summary, nil
}

// GetOverdueRecords retrieves billing records that are overdue
func (r *billingRecordRepository) GetOverdueRecords(ctx context.Context, daysOverdue int) ([]*models.BillingRecord, error) {
	var records []*models.BillingRecord
	cutoffDate := time.Now().AddDate(0, 0, -daysOverdue)

	if err := r.db.WithContext(ctx).
		Where("payment_status = ? AND period_end < ?", 
			models.PaymentStatusPending, cutoffDate).
		Order("period_end ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get overdue records: %w", err)
	}

	// Update status to overdue if needed
	if len(records) > 0 {
		var ids []uuid.UUID
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		r.db.WithContext(ctx).
			Model(&models.BillingRecord{}).
			Where("id IN ?", ids).
			Update("payment_status", models.PaymentStatusOverdue)
	}

	return records, nil
}

// GetTopRevenue retrieves API keys with highest revenue
func (r *billingRecordRepository) GetTopRevenue(ctx context.Context, startDate, endDate time.Time, limit int) ([]*RevenueByAPIKey, error) {
	var results []*RevenueByAPIKey

	query := r.db.WithContext(ctx).
		Model(&models.BillingRecord{}).
		Select(`
			api_key_id,
			COUNT(*) as record_count,
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COALESCE(SUM(CASE WHEN payment_status = ? THEN total_amount ELSE 0 END), 0) as paid_revenue,
			COALESCE(SUM(CASE WHEN payment_status IN (?, ?, ?) THEN total_amount ELSE 0 END), 0) as unpaid_revenue
		`, models.PaymentStatusPaid, models.PaymentStatusPending, models.PaymentStatusOverdue, models.PaymentStatusFailed).
		Group("api_key_id").
		Order("total_revenue DESC").
		Limit(limit)

	if !startDate.IsZero() {
		query = query.Where("period_start >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("period_end <= ?", endDate)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get top revenue: %w", err)
	}

	return results, nil
}

// CalculateTotalOwed calculates the total amount owed by an API key
func (r *billingRecordRepository) CalculateTotalOwed(ctx context.Context, apiKeyID uuid.UUID) (float64, error) {
	var total float64
	
	if err := r.db.WithContext(ctx).
		Model(&models.BillingRecord{}).
		Where("api_key_id = ? AND payment_status IN ?", 
			apiKeyID, 
			[]models.PaymentStatus{
				models.PaymentStatusPending,
				models.PaymentStatusOverdue,
				models.PaymentStatusFailed,
			}).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("failed to calculate total owed: %w", err)
	}

	return total, nil
}