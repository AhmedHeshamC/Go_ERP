package testutil

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	orderEntities "erpgo/internal/domain/orders/entities"
	productEntities "erpgo/internal/domain/products/entities"
)

// PerformanceTestContainer provides test setup for performance tests
type PerformanceTestContainer struct {
	Orders   []*orderEntities.Order
	Products []*productEntities.Product
	Cleanup  func()
}

// NewPerformanceTestContainer creates a new performance test container
func NewPerformanceTestContainer(t *testing.T) *PerformanceTestContainer {
	cleanup := func() {
		// Cleanup any resources if needed
	}

	return &PerformanceTestContainer{
		Orders:   make([]*orderEntities.Order, 0),
		Products: make([]*productEntities.Product, 0),
		Cleanup:  cleanup,
	}
}

// CreateTestOrderData creates sample order data for performance testing
func (ptc *PerformanceTestContainer) CreateTestOrderData(t *testing.T, numOrders int) {
	// Create test products first
	for i := 0; i < 10; i++ {
		// Create test product
		product := &productEntities.Product{
			ID:    uuid.New(),
			SKU:   fmt.Sprintf("TEST-PROD-%d", i),
			Name:  fmt.Sprintf("Test Product %d", i),
			Price: decimal.NewFromFloat(float64(i+1) * 10.0),
		}
		ptc.Products = append(ptc.Products, product)
	}

	// Create test orders
	for i := 0; i < numOrders; i++ {
		order := &orderEntities.Order{
			ID:     uuid.New(),
			Status: orderEntities.OrderStatusPending,
			TotalAmount: decimal.NewFromFloat(float64(i+1) * 100.0),
		}
		ptc.Orders = append(ptc.Orders, order)
	}
}

// CleanupTestData cleans up test data
func (ptc *PerformanceTestContainer) CleanupTestData(t *testing.T) {
	// Clean up in reverse order of dependencies
	ptc.Orders = make([]*orderEntities.Order, 0)
	ptc.Products = make([]*productEntities.Product, 0)
}