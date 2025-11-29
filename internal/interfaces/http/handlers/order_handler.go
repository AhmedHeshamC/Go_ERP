package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"erpgo/internal/application/services/order"
	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/interfaces/http/dto"
)

// OrderHandler handles order HTTP requests
type OrderHandler struct {
	orderService order.Service
	logger       zerolog.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService order.Service, logger zerolog.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// Order CRUD Operations

// CreateOrder creates a new order
// @Summary Create order
// @Description Create a new order with items and automatic order number generation
// @Tags orders
// @Accept json
// @Produce json
// @Param order body dto.OrderRequest true "Order data"
// @Success 201 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid order creation request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &order.CreateOrderRequest{
		CustomerID:        req.CustomerID.String(),
		Type:              entities.OrderType(req.Type),
		Priority:          entities.OrderPriority(req.Priority),
		ShippingMethod:    entities.ShippingMethod(req.ShippingMethod),
		ShippingAddressID: req.ShippingAddressID.String(),
		BillingAddressID:  req.BillingAddressID.String(),
		Currency:          req.Currency,
		RequiredDate:      req.RequiredDate,
		Notes:             req.Notes,
		CustomerNotes:     req.CustomerNotes,
		DiscountCode:      req.DiscountCode,
		PaymentMethod:     req.PaymentMethod,
	}

	// Convert items
	serviceReq.Items = make([]order.CreateOrderItemRequest, len(req.Items))
	for i, item := range req.Items {
		serviceReq.Items[i] = order.CreateOrderItemRequest{
			ProductID: item.ProductID.String(),
			Quantity:  int(item.Quantity),
			UnitPrice: item.UnitPrice,
			Notes:     item.Notes,
		}
	}

	createdOrder, err := h.orderService.CreateOrder(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(createdOrder)
	c.JSON(http.StatusCreated, response)
}

// GetOrder retrieves an order by ID
// @Summary Get order
// @Description Get an order by its ID
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	order, err := h.orderService.GetOrder(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to get order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(order)
	c.JSON(http.StatusOK, response)
}

// GetOrderByNumber retrieves an order by order number
// @Summary Get order by number
// @Description Get an order by its order number
// @Tags orders
// @Accept json
// @Produce json
// @Param number path string true "Order number"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/number/{number} [get]
func (h *OrderHandler) GetOrderByNumber(c *gin.Context) {
	orderNumber := c.Param("number")
	if orderNumber == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order number is required",
		})
		return
	}

	order, err := h.orderService.GetOrderByNumber(c, orderNumber)
	if err != nil {
		h.logger.Error().Err(err).Str("order_number", orderNumber).Msg("Failed to get order by number")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(order)
	c.JSON(http.StatusOK, response)
}

// UpdateOrder updates an order
// @Summary Update order
// @Description Update an existing order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param order body dto.UpdateOrderRequest true "Order data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id} [put]
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid order update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Convert to service request
	serviceReq := &order.UpdateOrderRequest{
		Priority:          (*entities.OrderPriority)(req.Priority),
		ShippingMethod:    (*entities.ShippingMethod)(req.ShippingMethod),
		ShippingAddressID: uuidPtrToPtrString(req.ShippingAddressID),
		BillingAddressID:  uuidPtrToPtrString(req.BillingAddressID),
		RequiredDate:      req.RequiredDate,
		Notes:             req.Notes,
		CustomerNotes:     req.CustomerNotes,
		InternalNotes:     req.InternalNotes,
	}

	updatedOrder, err := h.orderService.UpdateOrder(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to update order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(updatedOrder)
	c.JSON(http.StatusOK, response)
}

// DeleteOrder deletes an order
// @Summary Delete order
// @Description Delete an order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 204
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id} [delete]
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	err := h.orderService.DeleteOrder(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to delete order")
		handleOrderError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListOrders retrieves a paginated list of orders
// @Summary List orders
// @Description Get a paginated list of orders with optional filtering
// @Tags orders
// @Accept json
// @Produce json
// @Param customer_id query string false "Customer ID"
// @Param status query string false "Order status"
// @Param type query string false "Order type"
// @Param priority query string false "Order priority"
// @Param payment_status query string false "Payment status"
// @Param fulfillment_status query string false "Fulfillment status"
// @Param currency query string false "Currency"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param search query string false "Search term"
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.ListOrdersResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	var req dto.ListOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid order list request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert to service request
	serviceReq := &order.ListOrdersRequest{
		CustomerID: uuidPtrToPtrString(req.CustomerID),
		Search:     ptrStringToString(req.Search),
		Page:       req.Page,
		Limit:      req.Limit,
		SortBy:     ptrStringToString(req.SortBy),
		SortOrder:  ptrStringToString(req.SortOrder),
	}

	result, err := h.orderService.ListOrders(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list orders")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retrieve orders",
		})
		return
	}

	// Convert to response DTO
	orders := make([]*dto.OrderResponse, len(result.Orders))
	for i, o := range result.Orders {
		orders[i] = h.orderToResponse(o)
	}

	response := &dto.ListOrdersResponse{
		Orders: orders,
		Pagination: &dto.Pagination{
			Page:       result.Pagination.Page,
			Limit:      result.Pagination.Limit,
			Total:      result.Pagination.Total,
			TotalPages: result.Pagination.TotalPages,
			HasNext:    result.Pagination.HasNext,
			HasPrev:    result.Pagination.HasPrev,
		},
	}

	c.JSON(http.StatusOK, response)
}

// SearchOrders searches orders by query
// @Summary Search orders
// @Description Search orders by query string
// @Tags orders
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param customer_id query string false "Customer ID"
// @Param status query string false "Order status"
// @Param type query string false "Order type"
// @Param created_after query string false "Created after (ISO 8601)"
// @Param created_before query string false "Created before (ISO 8601)"
// @Param search_fields query []string false "Search fields"
// @Param sort_by query string false "Sort field" default("created_at")
// @Param sort_order query string false "Sort order" Enums(asc,desc) default("desc")
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.SearchOrdersResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/search [get]
func (h *OrderHandler) SearchOrders(c *gin.Context) {
	var req dto.SearchOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid order search request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid query parameters",
			Details: err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Convert to service request
	serviceReq := &order.SearchOrdersRequest{
		Query: req.Query,
		Limit: req.Limit,
	}

	result, err := h.orderService.SearchOrders(c, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to search orders")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to search orders",
		})
		return
	}

	// Convert to response DTO
	orders := make([]*dto.OrderResponse, len(result.Orders))
	for i, o := range result.Orders {
		orders[i] = h.orderToResponse(o)
	}

	response := &dto.SearchOrdersResponse{
		Orders:     orders,
		TotalCount: result.Total,
	}

	c.JSON(http.StatusOK, response)
}

// Order Status Management

// UpdateOrderStatus updates order status
// @Summary Update order status
// @Description Update the status of an order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param status body dto.UpdateOrderStatusRequest true "Status update data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid status update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &order.UpdateOrderStatusRequest{
		Status:    entities.OrderStatus(req.Status),
		Reason:    ptrStringToString(req.Notes),
		Notify:    req.NotifyCustomer,
		UpdatedBy: "", // TODO: Get from authenticated user context
	}

	updatedOrder, err := h.orderService.UpdateOrderStatus(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to update order status")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(updatedOrder)
	c.JSON(http.StatusOK, response)
}

// CancelOrder cancels an order
// @Summary Cancel order
// @Description Cancel an order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param cancel body dto.CancelOrderRequest true "Cancellation data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id}/cancel [post]
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid cancel order request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &order.CancelOrderRequest{
		Reason:      req.Reason,
		Refund:      req.RefundPayment,
		Notify:      req.NotifyCustomer,
		CancelledBy: "", // TODO: Get from authenticated user context
	}

	canceledOrder, err := h.orderService.CancelOrder(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to cancel order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(canceledOrder)
	c.JSON(http.StatusOK, response)
}

// ProcessOrder processes an order
// @Summary Process order
// @Description Process an order for fulfillment
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param process body dto.ProcessOrderRequest true "Processing data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id}/process [post]
func (h *OrderHandler) ProcessOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.ProcessOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid process order request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	processedOrder, err := h.orderService.ProcessOrder(c, id)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to process order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(processedOrder)
	c.JSON(http.StatusOK, response)
}

// ShipOrder ships an order
// @Summary Ship order
// @Description Mark an order as shipped with tracking information
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param ship body dto.ShipOrderRequest true "Shipping data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id}/ship [post]
func (h *OrderHandler) ShipOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.ShipOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid ship order request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &order.ShipOrderRequest{
		TrackingNumber: req.TrackingNumber,
		Carrier:        req.Carrier,
		ShippingDate:   req.ShippingDate,
		Notify:         req.NotifyCustomer,
		ShippedBy:      "", // TODO: Get from authenticated user context
	}

	shippedOrder, err := h.orderService.ShipOrder(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to ship order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(shippedOrder)
	c.JSON(http.StatusOK, response)
}

// DeliverOrder marks an order as delivered
// @Summary Deliver order
// @Description Mark an order as delivered
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param deliver body dto.DeliverOrderRequest true "Delivery data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id}/deliver [post]
func (h *OrderHandler) DeliverOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.DeliverOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid deliver order request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &order.DeliverOrderRequest{
		DeliveryDate: req.DeliveryDate,
		Proof:        req.PhotoProofURL,
		Notes:        req.Notes,
		Notify:       req.NotifyCustomer,
		DeliveredBy:  "", // TODO: Get from authenticated user context
	}

	deliveredOrder, err := h.orderService.DeliverOrder(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to deliver order")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(deliveredOrder)
	c.JSON(http.StatusOK, response)
}

// Payment Processing

// ProcessPayment processes a payment for an order
// @Summary Process payment
// @Description Process a payment for an order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param payment body dto.ProcessPaymentRequest true "Payment data"
// @Success 200 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/orders/{id}/payment [post]
func (h *OrderHandler) ProcessPayment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Order ID is required",
		})
		return
	}

	var req dto.ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid payment request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	serviceReq := &order.ProcessPaymentRequest{
		PaymentMethod: req.PaymentMethod,
		Amount:        req.Amount,
		TransactionID: ptrStringToString(req.TransactionID),
		Notes:         req.Notes,
		PaymentBy:     "", // TODO: Get from authenticated user context
	}

	order, err := h.orderService.ProcessPayment(c, id, serviceReq)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", id).Msg("Failed to process payment")
		handleOrderError(c, err)
		return
	}

	response := h.orderToResponse(order)
	c.JSON(http.StatusOK, response)
}

// Helper Methods

// orderToResponse converts an order entity to a response DTO
func (h *OrderHandler) orderToResponse(o *entities.Order) *dto.OrderResponse {
	// Convert order items
	items := make([]dto.OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		// Validate quantities are within int32 range
		quantity := item.Quantity
		if quantity > 0x7FFFFFFF || quantity < -0x80000000 {
			quantity = 0 // Fallback to 0 if out of range
		}
		shippedQty := item.QuantityShipped
		if shippedQty > 0x7FFFFFFF || shippedQty < -0x80000000 {
			shippedQty = 0
		}
		returnedQty := item.QuantityReturned
		if returnedQty > 0x7FFFFFFF || returnedQty < -0x80000000 {
			returnedQty = 0
		}

		items[i] = dto.OrderItemResponse{
			ID:               item.ID,
			ProductID:        item.ProductID,
			ProductSKU:       item.ProductSKU,
			ProductName:      item.ProductName,
			Quantity:         int32(quantity), // #nosec G115 - Validated above
			UnitPrice:        item.UnitPrice,
			TotalPrice:       item.TotalPrice,
			TaxAmount:        item.TaxAmount,
			DiscountAmount:   item.DiscountAmount,
			Weight:           decimal.NewFromFloat(item.Weight),
			Status:           item.Status,
			ShippedQuantity:  int32(shippedQty),  // #nosec G115 - Validated above
			ReturnedQuantity: int32(returnedQty), // #nosec G115 - Validated above
			Notes:            item.Notes,
		}
	}

	// Get customer name from the relationship
	customerName := ""
	if o.Customer != nil {
		customerName = o.Customer.FirstName + " " + o.Customer.LastName
	}

	return &dto.OrderResponse{
		ID:                 o.ID,
		OrderNumber:        o.OrderNumber,
		CustomerID:         o.CustomerID,
		CustomerName:       customerName,
		Type:               string(o.Type),
		Status:             string(o.Status),
		Priority:           string(o.Priority),
		PaymentStatus:      string(o.PaymentStatus),
		FulfillmentStatus:  string(o.Status), // Using Status as FulfillmentStatus
		Currency:           o.Currency,
		Subtotal:           o.Subtotal,
		TaxAmount:          o.TaxAmount,
		ShippingAmount:     o.ShippingAmount,
		DiscountAmount:     o.DiscountAmount,
		TotalAmount:        o.TotalAmount,
		PaidAmount:         o.PaidAmount,
		RefundedAmount:     o.RefundedAmount,
		Weight:             decimal.NewFromFloat(0), // TODO: Calculate from items
		ShippingMethod:     string(o.ShippingMethod),
		TrackingNumber:     ptrStringToString(o.TrackingNumber),
		ShippingAddress:    h.addressToResponse(o.ShippingAddress),
		BillingAddress:     h.addressToResponse(o.BillingAddress),
		Items:              items,
		Notes:              o.Notes,
		CustomerNotes:      o.CustomerNotes,
		InternalNotes:      o.InternalNotes,
		RequiredDate:       o.RequiredDate,
		ShippedDate:        o.ShippingDate,
		DeliveredDate:      o.DeliveryDate,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
		ApprovedAt:         o.ApprovedAt,
		ApprovedBy:         uuidPtrToPtrString(o.ApprovedBy),
		CancelledAt:        o.CancelledDate,
		CancelledBy:        nil, // No field in entity
		CancellationReason: nil, // No field in entity
	}
}

// addressToResponse converts an address entity to a response DTO
func (h *OrderHandler) addressToResponse(addr *entities.OrderAddress) *dto.AddressResponse {
	if addr == nil {
		return nil
	}

	return &dto.AddressResponse{
		ID:         addr.ID,
		Type:       addr.Type,
		FirstName:  addr.FirstName,
		LastName:   addr.LastName,
		Company:    addr.Company,
		Address1:   addr.AddressLine1,
		Address2:   addr.AddressLine2,
		City:       addr.City,
		State:      addr.State,
		PostalCode: addr.PostalCode,
		Country:    addr.Country,
		Phone:      addr.Phone,
		Email:      addr.Email,
		IsDefault:  addr.IsDefault,
	}
}

// handleOrderError handles order service errors
func handleOrderError(c *gin.Context, err error) {
	switch {
	case err.Error() == "order not found":
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Order not found",
			Details: err.Error(),
		})
	case err.Error() == "order already exists":
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error:   "Order already exists",
			Details: err.Error(),
		})
	case err.Error() == "invalid order status":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid order status",
			Details: err.Error(),
		})
	case err.Error() == "cannot cancel processed order":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Cannot cancel processed order",
			Details: err.Error(),
		})
	case err.Error() == "insufficient inventory":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Insufficient inventory",
			Details: err.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Details: err.Error(),
		})
	}
}

// uuidPtrToString converts a UUID pointer to string
func uuidPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// uuidPtrToPtrString converts a UUID pointer to string pointer
func uuidPtrToPtrString(ptr *uuid.UUID) *string {
	if ptr == nil {
		return nil
	}
	str := ptr.String()
	return &str
}

// ptrStringToString converts a string pointer to string
func ptrStringToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
