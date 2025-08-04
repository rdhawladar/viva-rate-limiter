package repositories

import (
	"context"

	"gorm.io/gorm"
)

// BaseRepository provides common database operations
type BaseRepository interface {
	// GetDB returns the database instance
	GetDB() *gorm.DB
	
	// WithTx executes operations within a transaction
	WithTx(tx *gorm.DB) BaseRepository
}

// baseRepository implements common repository functionality
type baseRepository struct {
	db *gorm.DB
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *gorm.DB) *baseRepository {
	return &baseRepository{
		db: db,
	}
}

// GetDB returns the database instance
func (r *baseRepository) GetDB() *gorm.DB {
	return r.db
}

// WithTx returns a repository instance with transaction
func (r *baseRepository) WithTx(tx *gorm.DB) *baseRepository {
	return &baseRepository{
		db: tx,
	}
}

// PaginationParams contains pagination parameters
type PaginationParams struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	OrderBy  string `json:"order_by"`
	Order    string `json:"order"` // asc or desc
}

// DefaultPagination returns default pagination parameters
func DefaultPagination() *PaginationParams {
	return &PaginationParams{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		Order:    "desc",
	}
}

// GetOffset calculates the offset for pagination
func (p *PaginationParams) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the limit for pagination
func (p *PaginationParams) GetLimit() int {
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.PageSize
}

// GetOrderBy returns the order by clause
func (p *PaginationParams) GetOrderBy() string {
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
	return p.OrderBy + " " + p.Order
}

// PaginatedResult contains paginated query results
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// NewPaginatedResult creates a new paginated result
func NewPaginatedResult(data interface{}, total int64, params *PaginationParams) *PaginatedResult {
	totalPages := int(total) / params.GetLimit()
	if int(total)%params.GetLimit() > 0 {
		totalPages++
	}

	return &PaginatedResult{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.GetLimit(),
		TotalPages: totalPages,
	}
}

// FilterParams contains common filter parameters
type FilterParams struct {
	Search    string                 `json:"search"`
	Status    string                 `json:"status"`
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Transaction executes a function within a database transaction
func Transaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}