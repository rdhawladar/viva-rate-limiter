package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rdhawladar/viva-rate-limiter/internal/models"
	"github.com/rdhawladar/viva-rate-limiter/internal/repositories"
)

// BillingService handles billing-related operations
type BillingService struct {
	billingRepo  *repositories.BillingRecordRepository
	apiKeyRepo   *repositories.APIKeyRepository
	usageLogRepo *repositories.UsageLogRepository
}

// NewBillingService creates a new billing service
func NewBillingService(
	billingRepo *repositories.BillingRecordRepository,
	apiKeyRepo *repositories.APIKeyRepository,
	usageLogRepo *repositories.UsageLogRepository,
) *BillingService {
	return &BillingService{
		billingRepo:  billingRepo,
		apiKeyRepo:   apiKeyRepo,
		usageLogRepo: usageLogRepo,
	}
}

// GenerateBillingRecords generates billing records for a given period
func (s *BillingService) GenerateBillingRecords(ctx context.Context, periodStart, periodEnd time.Time) ([]*models.BillingRecord, error) {
	// Get all API keys
	apiKeys, err := s.apiKeyRepo.List(100000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	var records []*models.BillingRecord

	for _, apiKey := range apiKeys {
		// Calculate usage for the period
		usage, err := s.calculateUsageForPeriod(ctx, apiKey.ID, periodStart, periodEnd)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate usage for API key %s: %w", apiKey.ID, err)
		}

		// Calculate cost based on tier
		cost := s.calculateCost(apiKey.Tier, usage)

		// Create billing record
		record := &models.BillingRecord{
			ID:          uuid.New(),
			APIKeyID:    apiKey.ID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			TotalUsage:  usage,
			TotalCost:   cost,
			Currency:    "USD",
			Status:      models.PaymentStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Calculate overage if applicable
		if usage > int64(apiKey.RateLimit*24*30) { // Monthly limit
			record.OverageUsage = usage - int64(apiKey.RateLimit*24*30)
			record.OverageCost = s.calculateOverageCost(apiKey.Tier, record.OverageUsage)
			record.TotalCost += record.OverageCost
		}

		// Save billing record
		if err := s.billingRepo.Create(record); err != nil {
			return nil, fmt.Errorf("failed to create billing record: %w", err)
		}

		records = append(records, record)
	}

	return records, nil
}

// GetBillingRecord retrieves a billing record by ID
func (s *BillingService) GetBillingRecord(ctx context.Context, recordID string) (*models.BillingRecord, error) {
	id, err := uuid.Parse(recordID)
	if err != nil {
		return nil, fmt.Errorf("invalid record ID: %w", err)
	}

	return s.billingRepo.FindByID(id)
}

// GetBillingHistory retrieves billing history for an API key
func (s *BillingService) GetBillingHistory(ctx context.Context, apiKeyID string, limit, offset int) ([]*models.BillingRecord, error) {
	id, err := uuid.Parse(apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("invalid API key ID: %w", err)
	}

	return s.billingRepo.FindByAPIKeyID(id, limit, offset)
}

// GetCurrentBillingPeriod gets the current billing period usage
func (s *BillingService) GetCurrentBillingPeriod(ctx context.Context, apiKeyID string) (*models.BillingRecord, error) {
	id, err := uuid.Parse(apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("invalid API key ID: %w", err)
	}

	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	// Check if record exists
	record, err := s.billingRepo.FindByPeriod(id, periodStart, periodEnd)
	if err == nil && record != nil {
		return record, nil
	}

	// Calculate current usage
	usage, err := s.calculateUsageForPeriod(ctx, id, periodStart, now)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current usage: %w", err)
	}

	// Get API key details
	apiKey, err := s.apiKeyRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Create temporary record (not saved)
	return &models.BillingRecord{
		APIKeyID:    id,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		TotalUsage:  usage,
		TotalCost:   s.calculateCost(apiKey.Tier, usage),
		Currency:    "USD",
		Status:      models.PaymentStatusPending,
	}, nil
}

// UpdateBillingStatus updates the status of a billing record
func (s *BillingService) UpdateBillingStatus(ctx context.Context, recordID string, status models.PaymentStatus) error {
	id, err := uuid.Parse(recordID)
	if err != nil {
		return fmt.Errorf("invalid record ID: %w", err)
	}

	record, err := s.billingRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("failed to find billing record: %w", err)
	}

	record.Status = status
	record.UpdatedAt = time.Now()

	if status == models.PaymentStatusPaid {
		now := time.Now()
		record.PaidAt = &now
	}

	return s.billingRepo.Update(record)
}

// calculateUsageForPeriod calculates total usage for an API key in a period
func (s *BillingService) calculateUsageForPeriod(ctx context.Context, apiKeyID uuid.UUID, start, end time.Time) (int64, error) {
	logs, err := s.usageLogRepo.FindByAPIKeyIDAndTimeRange(apiKeyID, start, end)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, log := range logs {
		total += log.RequestCount
	}

	return total, nil
}

// calculateCost calculates the base cost based on tier and usage
func (s *BillingService) calculateCost(tier models.APIKeyTier, usage int64) float64 {
	switch tier {
	case models.APIKeyTierFree:
		return 0.0
	case models.APIKeyTierBasic:
		return 9.99
	case models.APIKeyTierStandard:
		return 29.99
	case models.APIKeyTierPro:
		return 99.99
	case models.APIKeyTierEnterprise:
		return 299.99
	default:
		return 0.0
	}
}

// calculateOverageCost calculates overage charges
func (s *BillingService) calculateOverageCost(tier models.APIKeyTier, overage int64) float64 {
	// Cost per 1000 requests over limit
	var costPer1000 float64

	switch tier {
	case models.APIKeyTierFree:
		return 0.0 // No overage for free tier
	case models.APIKeyTierBasic:
		costPer1000 = 0.01
	case models.APIKeyTierStandard:
		costPer1000 = 0.008
	case models.APIKeyTierPro:
		costPer1000 = 0.005
	case models.APIKeyTierEnterprise:
		costPer1000 = 0.003
	default:
		costPer1000 = 0.01
	}

	return float64(overage) / 1000.0 * costPer1000
}

// GetBillingSummary gets a summary of billing for all API keys
func (s *BillingService) GetBillingSummary(ctx context.Context, periodStart, periodEnd time.Time) (map[string]interface{}, error) {
	records, err := s.billingRepo.FindByPeriodRange(periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing records: %w", err)
	}

	var totalRevenue float64
	var totalUsage int64
	var paidCount, pendingCount int

	for _, record := range records {
		totalRevenue += record.TotalCost
		totalUsage += record.TotalUsage

		switch record.Status {
		case models.PaymentStatusPaid:
			paidCount++
		case models.PaymentStatusPending:
			pendingCount++
		}
	}

	return map[string]interface{}{
		"period_start":    periodStart,
		"period_end":      periodEnd,
		"total_revenue":   totalRevenue,
		"total_usage":     totalUsage,
		"total_records":   len(records),
		"paid_records":    paidCount,
		"pending_records": pendingCount,
		"average_revenue": totalRevenue / float64(len(records)),
	}, nil
}