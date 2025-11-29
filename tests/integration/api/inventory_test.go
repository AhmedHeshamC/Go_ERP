package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"erpgo/internal/interfaces/http/dto"
	"erpgo/internal/interfaces/http/handlers"
	"erpgo/internal/interfaces/http/routes"
	"erpgo/pkg/errors"
)

// TestWarehouseEndpoints tests all warehouse endpoints
func TestWarehouseEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock warehouse service
	warehouseService := &mockWarehouseService{}
	warehouseHandler := NewWarehouseHandler(warehouseService, testLogger)

	// Setup routes
	v1 := router.Group("/api/v1")
	SetupInventoryRoutes(v1, warehouseHandler, nil, nil, mockAuthMiddleware, mockAdminAuthMiddleware, validationMiddleware, testLogger)

	// Test data
	testWarehouseID := uuid.New()
	testManagerID := uuid.New()

	// Test POST /api/v1/warehouses
	t.Run("Create Warehouse", func(t *testing.T) {
		createReq := dto.CreateWarehouseRequest{
			Name:       "Test Warehouse",
			Code:       "TW001",
			Address:    "123 Test St",
			City:       "Test City",
			State:      "TX",
			Country:    "USA",
			PostalCode: "12345",
			Phone:      "+1234567890",
			Email:      "test@example.com",
			ManagerID:  &testManagerID,
			Type:       "RETAIL",
			Capacity:   intPtr(1000),
		}

		jsonData, err := json.Marshal(createReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/warehouses", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.WarehouseResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, createReq.Name, response.Name)
		assert.Equal(t, createReq.Code, response.Code)
		assert.Equal(t, createReq.Address, response.Address)
		assert.Equal(t, createReq.City, response.City)
		assert.Equal(t, createReq.Type, response.Type)
		assert.Equal(t, *createReq.Capacity, *response.Capacity)
		assert.True(t, response.IsActive)
	})

	// Test GET /api/v1/warehouses
	t.Run("List Warehouses", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/warehouses?page=1&limit=10", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.WarehouseListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response.Warehouses)
		assert.NotNil(t, response.Pagination)
		assert.Equal(t, 1, response.Pagination.Page)
		assert.Equal(t, 10, response.Pagination.Limit)
	})

	// Test GET /api/v1/warehouses/:id
	t.Run("Get Warehouse", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/warehouses/%s", testWarehouseID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.WarehouseResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testWarehouseID, response.ID)
	})

	// Test PUT /api/v1/warehouses/:id
	t.Run("Update Warehouse", func(t *testing.T) {
		updateReq := dto.UpdateWarehouseRequest{
			Name:  strPtr("Updated Warehouse Name"),
			Phone: strPtr("+1987654321"),
		}

		jsonData, err := json.Marshal(updateReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/warehouses/%s", testWarehouseID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.WarehouseResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, *updateReq.Name, response.Name)
		assert.Equal(t, *updateReq.Phone, response.Phone)
	})

	// Test POST /api/v1/warehouses/:id/activate
	t.Run("Activate Warehouse", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/warehouses/%s/activate", testWarehouseID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.WarehouseResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response.IsActive)
	})

	// Test PUT /api/v1/warehouses/:id/manager
	t.Run("Assign Manager", func(t *testing.T) {
		managerReq := map[string]string{
			"manager_id": testManagerID.String(),
		}

		jsonData, err := json.Marshal(managerReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/warehouses/%s/manager", testWarehouseID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.WarehouseResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testManagerID, *response.ManagerID)
	})

	// Test GET /api/v1/warehouses/stats
	t.Run("Get Warehouse Stats", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/warehouses/stats", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.WarehouseStatsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, response.TotalWarehouses, 0)
		assert.GreaterOrEqual(t, response.ActiveWarehouses, 0)
		assert.NotNil(t, response.WarehousesByType)
	})

	// Test DELETE /api/v1/warehouses/:id
	t.Run("Delete Warehouse", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/warehouses/%s", testWarehouseID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestInventoryEndpoints tests all inventory endpoints
func TestInventoryEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock inventory service
	inventoryService := &mockInventoryService{}
	inventoryHandler := NewInventoryHandler(inventoryService, testLogger)

	// Setup routes
	v1 := router.Group("/api/v1")
	SetupInventoryRoutes(v1, nil, inventoryHandler, nil, mockAuthMiddleware, mockAdminAuthMiddleware, validationMiddleware, testLogger)

	// Test data
	testProductID := uuid.New()
	testWarehouseID := uuid.New()
	testTransactionID := uuid.New()

	// Test POST /api/v1/inventory/adjust
	t.Run("Adjust Inventory", func(t *testing.T) {
		adjustReq := dto.AdjustInventoryRequest{
			ProductID:     testProductID,
			WarehouseID:   testWarehouseID,
			Adjustment:    10,
			Reason:        "Initial stock",
			ReferenceType: "adjustment",
		}

		jsonData, err := json.Marshal(adjustReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/adjust", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.InventoryTransactionResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, testWarehouseID, response.WarehouseID)
		assert.Equal(t, "ADJUSTMENT", response.TransactionType)
		assert.Equal(t, 10, response.Quantity)
		assert.Equal(t, "Initial stock", response.Reason)
	})

	// Test POST /api/v1/inventory/reserve
	t.Run("Reserve Inventory", func(t *testing.T) {
		reserveReq := dto.ReserveInventoryRequest{
			ProductID:     testProductID,
			WarehouseID:   testWarehouseID,
			Quantity:      5,
			Reason:        "Customer order",
			ReferenceType: "order",
			Priority:      1,
		}

		jsonData, err := json.Marshal(reserveReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/reserve", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.InventoryResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, testWarehouseID, response.WarehouseID)
		assert.Equal(t, 5, response.ReservedQuantity)
		assert.Equal(t, 5, response.AvailableQuantity)
	})

	// Test POST /api/v1/inventory/release
	t.Run("Release Inventory", func(t *testing.T) {
		releaseReq := dto.ReleaseInventoryRequest{
			ReservationID: uuid.New(),
			Quantity:      3,
			Reason:        "Order fulfilled",
		}

		jsonData, err := json.Marshal(releaseReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/release", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, testWarehouseID, response.WarehouseID)
		assert.Equal(t, 2, response.ReservedQuantity)
		assert.Equal(t, 8, response.AvailableQuantity)
	})

	// Test POST /api/v1/inventory/transfer
	t.Run("Transfer Inventory", func(t *testing.T) {
		transferReq := dto.TransferInventoryRequest{
			ProductID:       testProductID,
			FromWarehouseID: testWarehouseID,
			ToWarehouseID:   uuid.New(),
			Quantity:        3,
			Reason:          "Stock relocation",
			ReferenceType:   "transfer",
		}

		jsonData, err := json.Marshal(transferReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/transfer", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.InventoryTransactionResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, transferReq.FromWarehouseID, response.WarehouseID)
		assert.Equal(t, "OUT", response.TransactionType)
		assert.Equal(t, 3, response.Quantity)
	})

	// Test GET /api/v1/inventory
	t.Run("List Inventory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory?page=1&limit=20&in_stock=true", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response.Inventory)
		assert.NotNil(t, response.Pagination)
		assert.Equal(t, 1, response.Pagination.Page)
		assert.Equal(t, 20, response.Pagination.Limit)
	})

	// Test GET /api/v1/inventory/product/:product_id/warehouse/:warehouse_id
	t.Run("Get Specific Inventory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/inventory/product/%s/warehouse/%s", testProductID, testWarehouseID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, testWarehouseID, response.WarehouseID)
		assert.Equal(t, 10, response.Quantity)
	})

	// Test GET /api/v1/inventory/product/:product_id/warehouse/:warehouse_id/check-availability
	t.Run("Check Inventory Availability", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/inventory/product/%s/warehouse/%s/check-availability?quantity=5", testProductID, testWarehouseID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.AvailabilityResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, 5, response.RequestedQty)
		assert.True(t, response.Available)
		assert.True(t, response.CanFulfill)
	})

	// Test GET /api/v1/inventory/stats
	t.Run("Get Inventory Stats", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory/stats", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryStatsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, response.TotalProducts, 0)
		assert.GreaterOrEqual(t, response.TotalStockQuantity, 0)
		assert.GreaterOrEqual(t, response.LowStockItems, 0)
	})

	// Test GET /api/v1/inventory/low-stock
	t.Run("Get Low Stock Items", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory/low-stock", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response.Inventory)
	})

	// Test POST /api/v1/inventory/bulk-adjust
	t.Run("Bulk Inventory Adjustment", func(t *testing.T) {
		bulkReq := dto.BulkInventoryAdjustmentRequest{
			Adjustments: []dto.AdjustInventoryRequest{
				{
					ProductID:     testProductID,
					WarehouseID:   testWarehouseID,
					Adjustment:    5,
					Reason:        "Bulk adjustment 1",
					ReferenceType: "adjustment",
				},
				{
					ProductID:     uuid.New(),
					WarehouseID:   testWarehouseID,
					Adjustment:    -2,
					Reason:        "Bulk adjustment 2",
					ReferenceType: "adjustment",
				},
			},
			DryRun: false,
		}

		jsonData, err := json.Marshal(bulkReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/bulk-adjust", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.BulkInventoryOperationResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 2, response.TotalCount)
		assert.GreaterOrEqual(t, response.SuccessCount, 0)
		assert.GreaterOrEqual(t, response.FailedCount, 0)
	})
}

// TestInventoryTransactionEndpoints tests all inventory transaction endpoints
func TestInventoryTransactionEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock transaction service
	transactionService := &mockTransactionService{}
	transactionHandler := NewInventoryTransactionHandler(transactionService, testLogger)

	// Setup routes
	v1 := router.Group("/api/v1")
	SetupInventoryRoutes(v1, nil, nil, transactionHandler, mockAuthMiddleware, mockAdminAuthMiddleware, validationMiddleware, testLogger)

	// Test data
	testTransactionID := uuid.New()
	testProductID := uuid.New()
	testWarehouseID := uuid.New()
	testUserID := uuid.New()

	// Test GET /api/v1/inventory/transactions
	t.Run("List Transactions", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory/transactions?page=1&limit=20", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryTransactionListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response.Transactions)
		assert.NotNil(t, response.Pagination)
		assert.Equal(t, 1, response.Pagination.Page)
		assert.Equal(t, 20, response.Pagination.Limit)
	})

	// Test GET /api/v1/inventory/transactions/:id
	t.Run("Get Transaction", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/inventory/transactions/%s", testTransactionID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryTransactionResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransactionID, response.ID)
		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, testWarehouseID, response.WarehouseID)
	})

	// Test POST /api/v1/inventory/transactions/:id/approve
	t.Run("Approve Transaction", func(t *testing.T) {
		approveReq := dto.ApproveTransactionRequest{
			ApprovedBy: testUserID,
			Notes:      "Approved for fulfillment",
		}

		jsonData, err := json.Marshal(approveReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/inventory/transactions/%s/approve", testTransactionID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryTransactionResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransactionID, response.ID)
		assert.Equal(t, testUserID, *response.ApprovedBy)
		assert.NotNil(t, response.ApprovedAt)
		assert.Equal(t, "Approved for fulfillment", response.ApprovalNotes)
	})

	// Test POST /api/v1/inventory/transactions/:id/reject
	t.Run("Reject Transaction", func(t *testing.T) {
		rejectReq := map[string]string{
			"reason": "Insufficient stock available",
		}

		jsonData, err := json.Marshal(rejectReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/inventory/transactions/%s/reject", testTransactionID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryTransactionResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransactionID, response.ID)
		assert.Equal(t, "REJECTED", response.TransactionType) // This would be status in real implementation
	})

	// Test GET /api/v1/inventory/transactions/pending
	t.Run("Get Pending Approvals", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory/transactions/pending", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.InventoryTransactionListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response.Transactions)
	})

	// Test GET /api/v1/inventory/transactions/stats
	t.Run("Get Transaction Stats", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory/transactions/stats", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.TransactionStatsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, response.TotalTransactions, 0)
		assert.GreaterOrEqual(t, response.PendingApprovals, 0)
		assert.NotNil(t, response.TransactionsByType)
	})

	// Test POST /api/v1/inventory/alerts/low-stock
	t.Run("Create Low Stock Alert", func(t *testing.T) {
		alertReq := dto.LowStockAlertRequest{
			ProductID:   &testProductID,
			WarehouseID: &testWarehouseID,
			Threshold:   intPtr(10),
			IsActive:    boolPtr(true),
		}

		jsonData, err := json.Marshal(alertReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/alerts/low-stock", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.LowStockAlertResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testProductID, response.ProductID)
		assert.Equal(t, testWarehouseID, response.WarehouseID)
		assert.Equal(t, 10, response.Threshold)
		assert.True(t, response.IsActive)
	})

	// Test GET /api/v1/inventory/alerts/low-stock
	t.Run("List Low Stock Alerts", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/inventory/alerts/low-stock?page=1&limit=20", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.LowStockAlertListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response.Alerts)
		assert.NotNil(t, response.Pagination)
	})

	// Test PUT /api/v1/inventory/alerts/low-stock/:id
	t.Run("Update Low Stock Alert", func(t *testing.T) {
		alertID := uuid.New()
		updateReq := dto.LowStockAlertRequest{
			Threshold: intPtr(15),
			IsActive:  boolPtr(false),
		}

		jsonData, err := json.Marshal(updateReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/inventory/alerts/low-stock/%s", alertID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.LowStockAlertResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, alertID, response.ID)
		assert.Equal(t, 15, response.Threshold)
		assert.False(t, response.IsActive)
	})

	// Test DELETE /api/v1/inventory/alerts/low-stock/:id
	t.Run("Delete Low Stock Alert", func(t *testing.T) {
		alertID := uuid.New()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/inventory/alerts/low-stock/%s", alertID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// Error handling tests
func TestInventoryErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock service that returns errors
	errorService := &mockErrorService{}
	warehouseHandler := NewWarehouseHandler(errorService, testLogger)
	inventoryHandler := NewInventoryHandler(errorService, testLogger)
	transactionHandler := NewInventoryTransactionHandler(errorService, testLogger)

	// Setup routes
	v1 := router.Group("/api/v1")
	SetupInventoryRoutes(v1, warehouseHandler, inventoryHandler, transactionHandler, mockAuthMiddleware, mockAdminAuthMiddleware, validationMiddleware, testLogger)

	// Test validation errors
	t.Run("Validation Error", func(t *testing.T) {
		// Invalid warehouse creation request
		invalidReq := map[string]interface{}{
			"name": "",  // Empty name should fail validation
			"code": "A", // Too short code should fail validation
		}

		jsonData, err := json.Marshal(invalidReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/warehouses", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse dto.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.NotEmpty(t, errorResponse.Error)
	})

	// Test not found error
	t.Run("Not Found Error", func(t *testing.T) {
		nonExistentID := uuid.New()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/warehouses/%s", nonExistentID), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var errorResponse dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Warehouse not found", errorResponse.Error)
	})

	// Test conflict error
	t.Run("Conflict Error", func(t *testing.T) {
		// Try to create duplicate warehouse
		duplicateReq := dto.CreateWarehouseRequest{
			Name:       "Duplicate Warehouse",
			Code:       "DUP001", // This should cause a conflict
			Address:    "123 Test St",
			City:       "Test City",
			State:      "TX",
			Country:    "USA",
			PostalCode: "12345",
		}

		jsonData, err := json.Marshal(duplicateReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/warehouses", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResponse dto.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Warehouse already exists", errorResponse.Error)
	})

	// Test insufficient stock error
	t.Run("Insufficient Stock Error", func(t *testing.T) {
		// Try to reserve more stock than available
		reserveReq := dto.ReserveInventoryRequest{
			ProductID:     uuid.New(),
			WarehouseID:   uuid.New(),
			Quantity:      1000, // More than available
			Reason:        "Large order",
			ReferenceType: "order",
		}

		jsonData, err := json.Marshal(reserveReq)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/inventory/reserve", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var errorResponse dto.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
		require.NoError(t, err)

		assert.Equal(t, "Insufficient stock", errorResponse.Error)
	})
}

// Mock implementations (these would be replaced with actual service implementations in real tests)

type mockWarehouseService struct{}

func (m *mockWarehouseService) CreateWarehouse(ctx *gin.Context, req *dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:                    uuid.New(),
		Name:                  req.Name,
		Code:                  req.Code,
		Address:               req.Address,
		City:                  req.City,
		State:                 req.State,
		Country:               req.Country,
		PostalCode:            req.PostalCode,
		Phone:                 req.Phone,
		Email:                 req.Email,
		ManagerID:             req.ManagerID,
		Type:                  req.Type,
		Capacity:              req.Capacity,
		SquareFootage:         req.SquareFootage,
		DockCount:             req.DockCount,
		TemperatureControlled: req.TemperatureControlled,
		SecurityLevel:         req.SecurityLevel,
		Description:           req.Description,
		IsActive:              true,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}, nil
}

func (m *mockWarehouseService) GetWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        id,
		Name:      "Test Warehouse",
		Code:      "TW001",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) GetWarehouseByCode(ctx *gin.Context, code string) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        uuid.New(),
		Code:      code,
		Name:      "Test Warehouse",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) UpdateWarehouse(ctx *gin.Context, id uuid.UUID, req *dto.UpdateWarehouseRequest) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        id,
		Name:      *req.Name,
		Phone:     *req.Phone,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) DeleteWarehouse(ctx *gin.Context, id uuid.UUID) error {
	return nil
}

func (m *mockWarehouseService) ListWarehouses(ctx *gin.Context, req *dto.ListWarehousesRequest) (*dto.WarehouseListResponse, error) {
	return &dto.WarehouseListResponse{
		Warehouses: []*dto.WarehouseResponse{},
		Pagination: &dto.PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *mockWarehouseService) ActivateWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        id,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) DeactivateWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        id,
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) AssignManager(ctx *gin.Context, id uuid.UUID, managerID uuid.UUID) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        id,
		ManagerID: &managerID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) RemoveManager(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error) {
	return &dto.WarehouseResponse{
		ID:        id,
		ManagerID: nil,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockWarehouseService) GetWarehouseStats(ctx *gin.Context) (*dto.WarehouseStatsResponse, error) {
	return &dto.WarehouseStatsResponse{
		TotalWarehouses:    5,
		ActiveWarehouses:   4,
		InactiveWarehouses: 1,
		WarehousesByType:   map[string]int{"RETAIL": 2, "WHOLESALE": 2, "DISTRIBUTION": 1},
	}, nil
}

type mockInventoryService struct{}

func (m *mockInventoryService) AdjustInventory(ctx *gin.Context, req *dto.AdjustInventoryRequest) (*dto.InventoryTransactionResponse, error) {
	return &dto.InventoryTransactionResponse{
		ID:               uuid.New(),
		ProductID:        req.ProductID,
		WarehouseID:      req.WarehouseID,
		TransactionType:  "ADJUSTMENT",
		Quantity:         req.Adjustment,
		PreviousQuantity: 0,
		NewQuantity:      req.Adjustment,
		Reason:           req.Reason,
		ReferenceType:    req.ReferenceType,
		CreatedBy:        uuid.New(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil
}

func (m *mockInventoryService) ReserveInventory(ctx *gin.Context, req *dto.ReserveInventoryRequest) (*dto.InventoryResponse, error) {
	return &dto.InventoryResponse{
		ID:                uuid.New(),
		ProductID:         req.ProductID,
		ProductSKU:        "TEST-001",
		ProductName:       "Test Product",
		WarehouseID:       req.WarehouseID,
		WarehouseCode:     "WH001",
		WarehouseName:     "Test Warehouse",
		Quantity:          10,
		ReservedQuantity:  req.Quantity,
		AvailableQuantity: 10 - req.Quantity,
		MinStockLevel:     5,
		IsLowStock:        false,
		IsOutOfStock:      false,
		LastUpdated:       time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (m *mockInventoryService) ReleaseInventory(ctx *gin.Context, req *dto.ReleaseInventoryRequest) (*dto.InventoryResponse, error) {
	return &dto.InventoryResponse{
		ID:                uuid.New(),
		Quantity:          10,
		ReservedQuantity:  2,
		AvailableQuantity: 8,
		MinStockLevel:     5,
		IsLowStock:        false,
		IsOutOfStock:      false,
		LastUpdated:       time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (m *mockInventoryService) TransferInventory(ctx *gin.Context, req *dto.TransferInventoryRequest) (*dto.InventoryTransactionResponse, error) {
	return &dto.InventoryTransactionResponse{
		ID:              uuid.New(),
		ProductID:       req.ProductID,
		WarehouseID:     req.FromWarehouseID,
		TransactionType: "OUT",
		Quantity:        req.Quantity,
		Reason:          req.Reason,
		ReferenceType:   req.ReferenceType,
		CreatedBy:       uuid.New(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func (m *mockInventoryService) ListInventory(ctx *gin.Context, req *dto.ListInventoryRequest) (*dto.InventoryListResponse, error) {
	return &dto.InventoryListResponse{
		Inventory: []*dto.InventoryResponse{},
		Pagination: &dto.PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *mockInventoryService) GetInventoryByProductAndWarehouse(ctx *gin.Context, productID, warehouseID uuid.UUID) (*dto.InventoryResponse, error) {
	return &dto.InventoryResponse{
		ID:                uuid.New(),
		ProductID:         productID,
		WarehouseID:       warehouseID,
		Quantity:          10,
		ReservedQuantity:  0,
		AvailableQuantity: 10,
		MinStockLevel:     5,
		IsLowStock:        false,
		IsOutOfStock:      false,
		LastUpdated:       time.Now(),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil
}

func (m *mockInventoryService) GetInventoryStats(ctx *gin.Context) (*dto.InventoryStatsResponse, error) {
	return &dto.InventoryStatsResponse{
		TotalProducts:        100,
		TotalWarehouses:      5,
		TotalStockQuantity:   1000,
		LowStockItems:        10,
		OutOfStockItems:      2,
		TotalReservations:    50,
		TopWarehousesByStock: []dto.WarehouseStockInfo{},
		TopProductsByValue:   []dto.ProductValueInfo{},
	}, nil
}

func (m *mockInventoryService) GetLowStockItems(ctx *gin.Context, warehouseID *uuid.UUID) (*dto.InventoryListResponse, error) {
	return &dto.InventoryListResponse{
		Inventory: []*dto.InventoryResponse{},
		Pagination: &dto.PaginationInfo{
			Page:       1,
			Limit:      20,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *mockInventoryService) BulkInventoryAdjustment(ctx *gin.Context, req *dto.BulkInventoryAdjustmentRequest) (*dto.BulkInventoryOperationResponse, error) {
	return &dto.BulkInventoryOperationResponse{
		SuccessCount: len(req.Adjustments),
		FailedCount:  0,
		TotalCount:   len(req.Adjustments),
		Results:      make([]dto.BulkInventoryOperationResult, len(req.Adjustments)),
		Summary: &dto.BulkOperationSummary{
			TotalAdjustments:    len(req.Adjustments),
			PositiveAdjustments: 1,
			NegativeAdjustments: 1,
			AffectedProducts:    len(req.Adjustments),
			AffectedWarehouses:  1,
		},
	}, nil
}

func (m *mockInventoryService) CheckInventoryAvailability(ctx *gin.Context, productID, warehouseID uuid.UUID, quantity int) (*dto.AvailabilityResponse, error) {
	return &dto.AvailabilityResponse{
		ProductID:        productID,
		RequestedQty:     quantity,
		Available:        true,
		CanFulfill:       true,
		BackorderAllowed: false,
	}, nil
}

func (m *mockInventoryService) GetInventoryHistory(ctx *gin.Context, productID, warehouseID uuid.UUID, limit int) ([]*dto.InventoryTransactionResponse, error) {
	return []*dto.InventoryTransactionResponse{}, nil
}

type mockTransactionService struct{}

func (m *mockTransactionService) GetTransaction(ctx *gin.Context, id uuid.UUID) (*dto.InventoryTransactionResponse, error) {
	return &dto.InventoryTransactionResponse{
		ID:              id,
		ProductID:       uuid.New(),
		WarehouseID:     uuid.New(),
		TransactionType: "ADJUSTMENT",
		Quantity:        10,
		Reason:          "Test transaction",
		CreatedBy:       uuid.New(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func (m *mockTransactionService) ListTransactions(ctx *gin.Context, req *dto.ListInventoryTransactionsRequest) (*dto.InventoryTransactionListResponse, error) {
	return &dto.InventoryTransactionListResponse{
		Transactions: []*dto.InventoryTransactionResponse{},
		Pagination: &dto.PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *mockTransactionService) ApproveTransaction(ctx *gin.Context, id uuid.UUID, req *dto.ApproveTransactionRequest) (*dto.InventoryTransactionResponse, error) {
	return &dto.InventoryTransactionResponse{
		ID:            id,
		ApprovedBy:    &req.ApprovedBy,
		ApprovedAt:    &time.Time{},
		ApprovalNotes: req.Notes,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (m *mockTransactionService) RejectTransaction(ctx *gin.Context, id uuid.UUID, reason string) (*dto.InventoryTransactionResponse, error) {
	return &dto.InventoryTransactionResponse{
		ID:        id,
		Reason:    reason,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockTransactionService) GetTransactionStats(ctx *gin.Context, warehouseID, productID *uuid.UUID) (*dto.TransactionStatsResponse, error) {
	return &dto.TransactionStatsResponse{
		TotalTransactions:    100,
		PendingApprovals:     5,
		ApprovedTransactions: 90,
		RejectedTransactions: 5,
		TransactionsByType:   map[string]int{"ADJUSTMENT": 30, "RESERVATION": 40, "TRANSFER": 30},
		TotalQuantityIn:      500,
		TotalQuantityOut:     400,
		NetQuantityChange:    100,
		MostActiveProducts:   []dto.ProductTransactionStats{},
		MostActiveWarehouses: []dto.WarehouseTransactionStats{},
	}, nil
}

func (m *mockTransactionService) GetPendingApprovals(ctx *gin.Context, warehouseID *uuid.UUID) (*dto.InventoryTransactionListResponse, error) {
	return &dto.InventoryTransactionListResponse{
		Transactions: []*dto.InventoryTransactionResponse{},
		Pagination: &dto.PaginationInfo{
			Page:       1,
			Limit:      20,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *mockTransactionService) CreateLowStockAlert(ctx *gin.Context, req *dto.LowStockAlertRequest) (*dto.LowStockAlertResponse, error) {
	return &dto.LowStockAlertResponse{
		ID:            uuid.New(),
		ProductID:     *req.ProductID,
		WarehouseID:   *req.WarehouseID,
		CurrentStock:  5,
		MinStockLevel: 10,
		Threshold:     *req.Threshold,
		IsActive:      *req.IsActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (m *mockTransactionService) ListLowStockAlerts(ctx *gin.Context, req *dto.ListLowStockAlertsRequest) (*dto.LowStockAlertListResponse, error) {
	return &dto.LowStockAlertListResponse{
		Alerts: []*dto.LowStockAlertResponse{},
		Pagination: &dto.PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}, nil
}

func (m *mockTransactionService) UpdateLowStockAlert(ctx *gin.Context, id uuid.UUID, req *dto.LowStockAlertRequest) (*dto.LowStockAlertResponse, error) {
	return &dto.LowStockAlertResponse{
		ID:        id,
		Threshold: *req.Threshold,
		IsActive:  *req.IsActive,
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockTransactionService) DeleteLowStockAlert(ctx *gin.Context, id uuid.UUID) error {
	return nil
}

func (m *mockTransactionService) GetLowStockAlertsByWarehouse(ctx *gin.Context, warehouseID uuid.UUID) ([]*dto.LowStockAlertResponse, error) {
	return []*dto.LowStockAlertResponse{}, nil
}

type mockErrorService struct{}

func (m *mockErrorService) CreateWarehouse(ctx *gin.Context, req *dto.CreateWarehouseRequest) (*dto.WarehouseResponse, error) {
	if req.Name == "" || len(req.Code) < 2 {
		return nil, errors.NewValidationError("invalid warehouse data")
	}
	if req.Code == "DUP001" {
		return nil, errors.NewConflictError("warehouse already exists")
	}
	return &dto.WarehouseResponse{}, nil
}

func (m *mockErrorService) GetWarehouse(ctx *gin.Context, id uuid.UUID) (*dto.WarehouseResponse, error) {
	return nil, errors.NewNotFoundError("warehouse not found")
}

// ... (other error service methods would be implemented similarly)

// Helper functions and middleware
func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Set("user_role", "user")
		c.Next()
	}
}

func mockAdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Set("user_role", "admin")
		c.Next()
	}
}

func validationMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

var testLogger = zerolog.New(io.Discard)

// Utility functions
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
