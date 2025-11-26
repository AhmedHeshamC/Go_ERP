package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestWarehouse_ComprehensiveCoverage tests all warehouse methods for complete coverage
func TestWarehouse_ComprehensiveCoverage(t *testing.T) {
	warehouseID := uuid.New()
	managerID := uuid.New()

	t.Run("ToSafeWarehouse", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:          warehouseID,
			Name:        "Main Warehouse",
			Code:        "WH-001",
			Address:     "123 Storage St",
			City:        "New York",
			State:       "NY",
			Country:     "US",
			PostalCode:  "10001",
			Phone:       "+1-555-0100",
			Email:       "warehouse@example.com",
			ManagerID:   &managerID,
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		safe := warehouse.ToSafeWarehouse()
		if safe.ID != warehouse.ID {
			t.Error("Safe warehouse ID mismatch")
		}
		if safe.Name != warehouse.Name {
			t.Error("Safe warehouse Name mismatch")
		}
	})

	t.Run("UpdateDetails", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:          warehouseID,
			Name:        "Main Warehouse",
			Code:        "WH-001",
			Address:     "123 Storage St",
			City:        "New York",
			State:       "NY",
			Country:     "US",
			PostalCode:  "10001",
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		// Valid update
		err := warehouse.UpdateDetails("Updated Warehouse", "456 New St", "Boston", "MA", "US", "02101")
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if warehouse.Name != "Updated Warehouse" {
			t.Error("Name not updated")
		}
		if warehouse.City != "Boston" {
			t.Error("City not updated")
		}

		// Empty name
		err = warehouse.UpdateDetails("", "456 New St", "Boston", "MA", "US", "02101")
		if err == nil {
			t.Error("Expected error for empty name")
		}

		// Name too long
		longName := string(make([]byte, 201))
		err = warehouse.UpdateDetails(longName, "456 New St", "Boston", "MA", "US", "02101")
		if err == nil {
			t.Error("Expected error for name too long")
		}

		// Empty address
		err = warehouse.UpdateDetails("Name", "", "Boston", "MA", "US", "02101")
		if err == nil {
			t.Error("Expected error for empty address")
		}
	})

	t.Run("UpdateContactInfo", func(t *testing.T) {
		warehouse := &Warehouse{
			ID:          warehouseID,
			Name:        "Main Warehouse",
			Code:        "WH-001",
			Address:     "123 Storage St",
			City:        "New York",
			State:       "NY",
			Country:     "US",
			PostalCode:  "10001",
			Phone:       "+1-555-0100",
			Email:       "warehouse@example.com",
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		// Valid update
		err := warehouse.UpdateContactInfo("+1-555-0200", "newwarehouse@example.com")
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if warehouse.Phone != "+1-555-0200" {
			t.Error("Phone not updated")
		}
		if warehouse.Email != "newwarehouse@example.com" {
			t.Error("Email not updated")
		}

		// Invalid phone
		err = warehouse.UpdateContactInfo("invalid-phone", "newwarehouse@example.com")
		if err == nil {
			t.Error("Expected error for invalid phone")
		}

		// Invalid email
		err = warehouse.UpdateContactInfo("+1-555-0200", "invalid-email")
		if err == nil {
			t.Error("Expected error for invalid email")
		}
	})

	t.Run("UpdateType", func(t *testing.T) {
		capacity := 10000
		warehouse := &WarehouseExtended{
			Warehouse: Warehouse{
				ID:          warehouseID,
				Name:        "Main Warehouse",
				Code:        "WH-001",
				Address:     "123 Storage St",
				City:        "New York",
				State:       "NY",
				Country:     "US",
				PostalCode:  "10001",
				IsActive:    true,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
			Type:     WarehouseTypeDistribution,
			Capacity: &capacity,
		}

		// Valid update
		err := warehouse.UpdateType(WarehouseTypeFulfillment)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if warehouse.Type != WarehouseTypeFulfillment {
			t.Error("Type not updated")
		}

		// Invalid type
		err = warehouse.UpdateType(WarehouseType("INVALID_TYPE"))
		if err == nil {
			t.Error("Expected error for invalid type")
		}
	})

	t.Run("UpdateCapacity", func(t *testing.T) {
		capacity := 10000
		warehouse := &WarehouseExtended{
			Warehouse: Warehouse{
				ID:          warehouseID,
				Name:        "Main Warehouse",
				Code:        "WH-001",
				Address:     "123 Storage St",
				City:        "New York",
				State:       "NY",
				Country:     "US",
				PostalCode:  "10001",
				IsActive:    true,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
			Type:     WarehouseTypeDistribution,
			Capacity: &capacity,
		}

		// Valid update
		err := warehouse.UpdateCapacity(15000)
		if err != nil {
			t.Errorf("Expected successful update, got error: %v", err)
		}
		if *warehouse.Capacity != 15000 {
			t.Error("Capacity not updated")
		}

		// Negative capacity
		err = warehouse.UpdateCapacity(-1000)
		if err == nil {
			t.Error("Expected error for negative capacity")
		}

		// Capacity exceeding max
		err = warehouse.UpdateCapacity(1000000000)
		if err == nil {
			t.Error("Expected error for capacity exceeding max")
		}
	})

	t.Run("GetTypeName", func(t *testing.T) {
		tests := []struct {
			warehouseType WarehouseType
			expected      string
		}{
			{WarehouseTypeDistribution, "Distribution"},
			{WarehouseTypeFulfillment, "Fulfillment"},
			{WarehouseTypeRetail, "Retail"},
			{WarehouseTypeWholesale, "Wholesale"},
			{WarehouseTypeReturn, "Return"},
			{WarehouseType("UNKNOWN"), "Unknown"},
		}

		for _, tt := range tests {
			warehouse := &WarehouseExtended{
				Warehouse: Warehouse{
					ID:          warehouseID,
					Name:        "Test Warehouse",
					Code:        "WH-001",
					Address:     "123 Storage St",
					City:        "New York",
					State:       "NY",
					Country:     "US",
					PostalCode:  "10001",
					IsActive:    true,
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				},
				Type: tt.warehouseType,
			}

			name := warehouse.GetTypeName()
			if name != tt.expected {
				t.Errorf("For type %s, expected name '%s', got '%s'", tt.warehouseType, tt.expected, name)
			}
		}
	})

	t.Run("GetUtilizationPercentage", func(t *testing.T) {
		capacity := 10000
		warehouse := &WarehouseExtended{
			Warehouse: Warehouse{
				ID:          warehouseID,
				Name:        "Main Warehouse",
				Code:        "WH-001",
				Address:     "123 Storage St",
				City:        "New York",
				State:       "NY",
				Country:     "US",
				PostalCode:  "10001",
				IsActive:    true,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
			Type:     WarehouseTypeDistribution,
			Capacity: &capacity,
		}

		// Valid calculation
		utilization, err := warehouse.GetUtilizationPercentage(7500)
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if utilization != 75.0 {
			t.Errorf("Expected utilization 75.0%%, got %.2f%%", utilization)
		}

		// Zero capacity
		zeroCapacity := 0
		warehouse.Capacity = &zeroCapacity
		_, err = warehouse.GetUtilizationPercentage(7500)
		if err == nil {
			t.Error("Expected error for zero capacity")
		}

		// Nil capacity
		warehouse.Capacity = nil
		_, err = warehouse.GetUtilizationPercentage(7500)
		if err == nil {
			t.Error("Expected error for nil capacity")
		}

		// Negative current stock
		warehouse.Capacity = &capacity
		_, err = warehouse.GetUtilizationPercentage(-100)
		if err == nil {
			t.Error("Expected error for negative current stock")
		}

		// Current stock exceeding capacity
		utilization, err = warehouse.GetUtilizationPercentage(15000)
		if err != nil {
			t.Errorf("Expected successful calculation, got error: %v", err)
		}
		if utilization != 100.0 {
			t.Errorf("Expected utilization capped at 100.0%%, got %.2f%%", utilization)
		}
	})
}
