package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"erpgo/internal/domain/products/entities"
)

// ProductRepository defines the interface for product data operations
type ProductRepository interface {
	Create(ctx context.Context, product *entities.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Product, error)
	GetBySKU(ctx context.Context, sku string) (*entities.Product, error)
	Update(ctx context.Context, product *entities.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ProductFilter) ([]*entities.Product, error)
	Search(ctx context.Context, query string, limit int) ([]*entities.Product, error)
	Count(ctx context.Context, filter ProductFilter) (int, error)
	GetByCategory(ctx context.Context, categoryID uuid.UUID) ([]*entities.Product, error)
	GetByCategories(ctx context.Context, categoryIDs []uuid.UUID) ([]*entities.Product, error)
	GetFeatured(ctx context.Context, limit int) ([]*entities.Product, error)
	GetActive(ctx context.Context, limit int) ([]*entities.Product, error)
	GetLowStock(ctx context.Context, threshold int) ([]*entities.Product, error)
	ExistsBySKU(ctx context.Context, sku string) (bool, error)
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) error
	AdjustStock(ctx context.Context, productID uuid.UUID, adjustment int) error
	GetPrice(ctx context.Context, productID uuid.UUID) (decimal.Decimal, error)
	UpdatePrice(ctx context.Context, productID uuid.UUID, price decimal.Decimal) error
	BulkUpdateStatus(ctx context.Context, productIDs []uuid.UUID, isActive bool) error
	GetProductStats(ctx context.Context, filter ProductFilter) (*ProductStats, error)
}

// CategoryRepository defines the interface for product category data operations
type CategoryRepository interface {
	Create(ctx context.Context, category *entities.ProductCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ProductCategory, error)
	GetByPath(ctx context.Context, path string) (*entities.ProductCategory, error)
	Update(ctx context.Context, category *entities.ProductCategory) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter CategoryFilter) ([]*entities.ProductCategory, error)
	ListRoot(ctx context.Context) ([]*entities.ProductCategory, error)
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]*entities.ProductCategory, error)
	GetDescendants(ctx context.Context, categoryID uuid.UUID) ([]*entities.ProductCategory, error)
	GetAncestors(ctx context.Context, categoryID uuid.UUID) ([]*entities.ProductCategory, error)
	GetPath(ctx context.Context, categoryID uuid.UUID) ([]*entities.ProductCategory, error)
	Count(ctx context.Context, filter CategoryFilter) (int, error)
	CountProducts(ctx context.Context, categoryID uuid.UUID) (int, error)
	ExistsByPath(ctx context.Context, path string) (bool, error)
	ExistsByName(ctx context.Context, name string, parentID *uuid.UUID) (bool, error)
	RebuildPaths(ctx context.Context) error
	RebuildCategoryPaths(ctx context.Context, categoryID uuid.UUID) error
	BulkUpdateSortOrder(ctx context.Context, updates []CategorySortUpdate) error
}

// ProductVariantRepository defines the interface for product variant data operations
type ProductVariantRepository interface {
	Create(ctx context.Context, variant *entities.ProductVariant) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ProductVariant, error)
	GetBySKU(ctx context.Context, sku string) (*entities.ProductVariant, error)
	GetByProductID(ctx context.Context, productID uuid.UUID) ([]*entities.ProductVariant, error)
	Update(ctx context.Context, variant *entities.ProductVariant) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ProductVariantFilter) ([]*entities.ProductVariant, error)
	Count(ctx context.Context, filter ProductVariantFilter) (int, error)
	ExistsBySKU(ctx context.Context, sku string) (bool, error)
	UpdateStock(ctx context.Context, variantID uuid.UUID, quantity int) error
	AdjustStock(ctx context.Context, variantID uuid.UUID, adjustment int) error
	GetActiveByProductID(ctx context.Context, productID uuid.UUID) ([]*entities.ProductVariant, error)
	BulkCreate(ctx context.Context, variants []*entities.ProductVariant) error
	BulkDelete(ctx context.Context, variantIDs []uuid.UUID) error
}

// VariantAttributeRepository defines the interface for variant attribute data operations
type VariantAttributeRepository interface {
	Create(ctx context.Context, attribute *entities.VariantAttribute) error
	GetByVariantID(ctx context.Context, variantID uuid.UUID) ([]*entities.VariantAttribute, error)
	Update(ctx context.Context, attribute *entities.VariantAttribute) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByVariantID(ctx context.Context, variantID uuid.UUID) error
	BulkCreate(ctx context.Context, attributes []*entities.VariantAttribute) error
}

// VariantImageRepository defines the interface for variant image data operations
type VariantImageRepository interface {
	Create(ctx context.Context, image *entities.VariantImage) error
	GetByVariantID(ctx context.Context, variantID uuid.UUID) ([]*entities.VariantImage, error)
	GetMainImage(ctx context.Context, variantID uuid.UUID) (*entities.VariantImage, error)
	Update(ctx context.Context, image *entities.VariantImage) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByVariantID(ctx context.Context, variantID uuid.UUID) error
	SetMainImage(ctx context.Context, variantID, imageID uuid.UUID) error
	BulkCreate(ctx context.Context, images []*entities.VariantImage) error
}

// ProductFilter defines filtering options for product queries
type ProductFilter struct {
	Search         string
	CategoryID     *uuid.UUID
	CategoryIDs    []uuid.UUID
	SKU            string
	MinPrice       *decimal.Decimal
	MaxPrice       *decimal.Decimal
	IsActive       *bool
	IsFeatured     *bool
	IsDigital      *bool
	TrackInventory *bool
	InStock        *bool
	LowStock       *bool
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	Page           int
	Limit          int
	SortBy         string
	SortOrder      string
}

// CategoryFilter defines filtering options for category queries
type CategoryFilter struct {
	Search   string
	ParentID *uuid.UUID
	IsActive *bool
	Level    *int
	Page     int
	Limit    int
	SortBy   string
	SortOrder string
}

// ProductVariantFilter defines filtering options for product variant queries
type ProductVariantFilter struct {
	ProductID      *uuid.UUID
	Search         string
	SKU            string
	MinPrice       *decimal.Decimal
	MaxPrice       *decimal.Decimal
	IsActive       *bool
	IsDigital      *bool
	TrackInventory *bool
	InStock        *bool
	LowStock       *bool
	Page           int
	Limit          int
	SortBy         string
	SortOrder      string
}

// ProductStats represents product statistics
type ProductStats struct {
	TotalProducts    int     `json:"total_products"`
	ActiveProducts   int     `json:"active_products"`
	InactiveProducts int     `json:"inactive_products"`
	FeaturedProducts int     `json:"featured_products"`
	LowStockProducts int     `json:"low_stock_products"`
	OutOfStockProducts int   `json:"out_of_stock_products"`
	DigitalProducts  int     `json:"digital_products"`
	PhysicalProducts int     `json:"physical_products"`
	AveragePrice     decimal.Decimal `json:"average_price"`
	MinPrice         decimal.Decimal `json:"min_price"`
	MaxPrice         decimal.Decimal `json:"max_price"`
	TotalStockValue  decimal.Decimal `json:"total_stock_value"`
}

// CategorySortUpdate represents a category sort order update
type CategorySortUpdate struct {
	CategoryID uuid.UUID
	SortOrder  int
}

// RepositoryResult represents a paginated repository result
type RepositoryResult[T any] struct {
	Items       []T   `json:"items"`
	TotalCount  int   `json:"total_count"`
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}