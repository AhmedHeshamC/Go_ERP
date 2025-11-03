package entities

import (
	"time"

	"github.com/google/uuid"
)

// Customer represents a customer in the system
type Customer struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	CustomerCode    string     `json:"customer_code" db:"customer_code"`
	Name            string     `json:"name" db:"name"`
	Email           string     `json:"email,omitempty" db:"email"`
	Phone           string     `json:"phone,omitempty" db:"phone"`
	BillingAddress  string     `json:"billing_address" db:"billing_address"`
	ShippingAddress *string    `json:"shipping_address,omitempty" db:"shipping_address"`
	CreditLimit     float64    `json:"credit_limit" db:"credit_limit"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// Order represents an order in the system
type Order struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	OrderNumber   string     `json:"order_number" db:"order_number"`
	CustomerID    uuid.UUID  `json:"customer_id" db:"customer_id"`
	Status        string     `json:"status" db:"status"` // PENDING, CONFIRMED, SHIPPED, DELIVERED, CANCELLED
	Subtotal      float64    `json:"subtotal" db:"subtotal"`
	TaxAmount     float64    `json:"tax_amount" db:"tax_amount"`
	ShippingAmount float64   `json:"shipping_amount" db:"shipping_amount"`
	TotalAmount   float64    `json:"total_amount" db:"total_amount"`
	Currency      string     `json:"currency" db:"currency"`
	OrderDate     time.Time  `json:"order_date" db:"order_date"`
	ShippingDate  *time.Time `json:"shipping_date,omitempty" db:"shipping_date"`
	DeliveryDate  *time.Time `json:"delivery_date,omitempty" db:"delivery_date"`
	Notes         *string    `json:"notes,omitempty" db:"notes"`
	CreatedBy     uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	OrderID      uuid.UUID  `json:"order_id" db:"order_id"`
	ProductID    uuid.UUID  `json:"product_id" db:"product_id"`
	Quantity     int        `json:"quantity" db:"quantity"`
	UnitPrice    float64    `json:"unit_price" db:"unit_price"`
	DiscountAmount float64   `json:"discount_amount" db:"discount_amount"`
	TotalPrice   float64    `json:"total_price" db:"total_price"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// Invoice represents an invoice in the system
type Invoice struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	InvoiceNumber string     `json:"invoice_number" db:"invoice_number"`
	OrderID       *uuid.UUID `json:"order_id,omitempty" db:"order_id"`
	CustomerID    uuid.UUID  `json:"customer_id" db:"customer_id"`
	InvoiceDate   time.Time  `json:"invoice_date" db:"invoice_date"`
	DueDate       time.Time  `json:"due_date" db:"due_date"`
	Subtotal      float64    `json:"subtotal" db:"subtotal"`
	TaxAmount     float64    `json:"tax_amount" db:"tax_amount"`
	TotalAmount   float64    `json:"total_amount" db:"total_amount"`
	Status        string     `json:"status" db:"status"` // DRAFT, SENT, PAID, OVERDUE, CANCELLED
	PaidAmount    float64    `json:"paid_amount" db:"paid_amount"`
	PaidAt        *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	CreatedBy     uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}