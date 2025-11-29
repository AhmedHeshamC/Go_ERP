package repositories

import (
	"context"
	"fmt"
	"time"

	"erpgo/internal/domain/orders/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Additional analytics types not defined in the domain package
type RegionSalesStats struct {
	Country       string          `json:"country"`
	State         string          `json:"state"`
	City          string          `json:"city"`
	OrderCount    int64           `json:"order_count"`
	TotalRevenue  decimal.Decimal `json:"total_revenue"`
	AvgOrderValue decimal.Decimal `json:"avg_order_value"`
}

type CustomerRetentionStats struct {
	Period                   string          `json:"period"`
	TotalCustomers           int64           `json:"total_customers"`
	NewCustomers             int64           `json:"new_customers"`
	ReturningCustomers       int64           `json:"returning_customers"`
	RetentionRate            decimal.Decimal `json:"retention_rate"`
	TotalOrders              int64           `json:"total_orders"`
	AverageOrdersPerCustomer decimal.Decimal `json:"average_orders_per_customer"`
	CustomerLifetimeValue    decimal.Decimal `json:"customer_lifetime_value"`
}

type InventoryTurnoverStats struct {
	ProductID    uuid.UUID       `json:"product_id"`
	ProductSKU   string          `json:"product_sku"`
	ProductName  string          `json:"product_name"`
	QuantitySold int64           `json:"quantity_sold"`
	AverageStock decimal.Decimal `json:"average_stock"`
	TurnoverRate decimal.Decimal `json:"turnover_rate"`
	DaysInStock  int64           `json:"days_in_stock"`
}

type PaymentAnalysisStats struct {
	TotalOrders            int64                           `json:"total_orders"`
	PaymentStatusBreakdown map[string]int64                `json:"payment_status_breakdown"`
	PaymentMethodBreakdown map[string]int64                `json:"payment_method_breakdown"`
	AveragePaymentTime     decimal.Decimal                 `json:"average_payment_time"`
	OverdueAmount          decimal.Decimal                 `json:"overdue_amount"`
	PaymentTrends          []*repositories.RevenueByPeriod `json:"payment_trends"`
}

type SeasonalStats struct {
	Season        string          `json:"season"`
	Year          int             `json:"year"`
	OrderCount    int64           `json:"order_count"`
	TotalRevenue  decimal.Decimal `json:"total_revenue"`
	AvgOrderValue decimal.Decimal `json:"avg_order_value"`
	GrowthRate    decimal.Decimal `json:"growth_rate"`
}

// PostgresOrderAnalyticsRepository implements advanced analytics queries for orders
type PostgresOrderAnalyticsRepository struct {
	db *database.Database
}

// NewPostgresOrderAnalyticsRepository creates a new PostgreSQL order analytics repository
func NewPostgresOrderAnalyticsRepository(db *database.Database) *PostgresOrderAnalyticsRepository {
	return &PostgresOrderAnalyticsRepository{
		db: db,
	}
}

// GetRevenueByPeriod retrieves revenue data grouped by time period
func (r *PostgresOrderAnalyticsRepository) GetRevenueByPeriod(ctx context.Context, startDate, endDate time.Time, groupBy string) ([]*repositories.RevenueByPeriod, error) {
	var dateFormat string
	switch groupBy {
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "YYYY-\"WW\""
	case "month":
		dateFormat = "YYYY-MM"
	case "quarter":
		dateFormat = "YYYY-\"Q\"Q"
	case "year":
		dateFormat = "YYYY"
	default:
		dateFormat = "YYYY-MM"
	}

	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(order_date, '%s') as period,
			COALESCE(SUM(total_amount), 0) as revenue,
			COUNT(*) as order_count,
			COALESCE(AVG(total_amount), 0) as average_order_value
		FROM orders
		WHERE order_date >= $1 AND order_date <= $2
		AND status NOT IN ('CANCELLED', 'REFUNDED')
		GROUP BY TO_CHAR(order_date, '%s')
		ORDER BY period
	`, dateFormat, dateFormat)

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by period: %w", err)
	}
	defer rows.Close()

	var results []*repositories.RevenueByPeriod
	for rows.Next() {
		result := &repositories.RevenueByPeriod{}
		err := rows.Scan(
			&result.Period,
			&result.Revenue,
			&result.OrderCount,
			&result.AverageOrderValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue by period row: %w", err)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue by period rows: %w", err)
	}

	return results, nil
}

// GetTopCustomers retrieves top customers by revenue
func (r *PostgresOrderAnalyticsRepository) GetTopCustomers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*repositories.CustomerOrderStats, error) {
	query := `
		SELECT
			c.id,
			COALESCE(c.first_name, '') || ' ' || COALESCE(c.last_name, '') as customer_name,
			COALESCE(c.email, '') as customer_email,
			COUNT(o.id) as order_count,
			COALESCE(SUM(o.total_amount), 0) as total_revenue,
			COALESCE(AVG(o.total_amount), 0) as average_order_value,
			MAX(o.order_date) as last_order_date
		FROM customers c
		INNER JOIN orders o ON c.id = o.customer_id
		WHERE o.order_date >= $1 AND o.order_date <= $2
		AND o.status NOT IN ('CANCELLED', 'REFUNDED')
		GROUP BY c.id, c.first_name, c.last_name, c.email
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top customers: %w", err)
	}
	defer rows.Close()

	var customers []*repositories.CustomerOrderStats
	for rows.Next() {
		customer := &repositories.CustomerOrderStats{}
		err := rows.Scan(
			&customer.CustomerID,
			&customer.CustomerName,
			&customer.CustomerEmail,
			&customer.OrderCount,
			&customer.TotalRevenue,
			&customer.AverageOrderValue,
			&customer.LastOrderDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top customer row: %w", err)
		}
		customers = append(customers, customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating top customer rows: %w", err)
	}

	return customers, nil
}

// GetSalesByProduct retrieves top-selling products
func (r *PostgresOrderAnalyticsRepository) GetSalesByProduct(ctx context.Context, startDate, endDate time.Time, limit int) ([]*repositories.ProductSalesStats, error) {
	query := `
		SELECT
			p.id,
			p.sku,
			p.name,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.total_price), 0) as total_revenue,
			COUNT(DISTINCT oi.order_id) as order_count
		FROM products p
		INNER JOIN order_items oi ON p.id = oi.product_id
		INNER JOIN orders o ON oi.order_id = o.id
		WHERE o.order_date >= $1 AND o.order_date <= $2
		AND o.status NOT IN ('CANCELLED', 'REFUNDED')
		GROUP BY p.id, p.sku, p.name
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales by product: %w", err)
	}
	defer rows.Close()

	var products []*repositories.ProductSalesStats
	for rows.Next() {
		product := &repositories.ProductSalesStats{}
		err := rows.Scan(
			&product.ProductID,
			&product.ProductSKU,
			&product.ProductName,
			&product.QuantitySold,
			&product.TotalRevenue,
			&product.OrderCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sales by product row: %w", err)
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sales by product rows: %w", err)
	}

	return products, nil
}

// GetOrderTrends retrieves order trends over time
func (r *PostgresOrderAnalyticsRepository) GetOrderTrends(ctx context.Context, startDate, endDate time.Time, groupBy string) ([]*repositories.RevenueByPeriod, error) {
	var dateFormat string
	switch groupBy {
	case "day":
		dateFormat = "YYYY-MM-DD"
	case "week":
		dateFormat = "YYYY-\"WW\""
	case "month":
		dateFormat = "YYYY-MM"
	case "year":
		dateFormat = "YYYY"
	default:
		dateFormat = "YYYY-MM"
	}

	query := fmt.Sprintf(`
		WITH order_periods AS (
			SELECT
				TO_CHAR(order_date, '%s') as period,
				COUNT(*) as total_orders,
				COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending_orders,
				COUNT(CASE WHEN status = 'CONFIRMED' THEN 1 END) as confirmed_orders,
				COUNT(CASE WHEN status = 'PROCESSING' THEN 1 END) as processing_orders,
				COUNT(CASE WHEN status = 'SHIPPED' THEN 1 END) as shipped_orders,
				COUNT(CASE WHEN status = 'DELIVERED' THEN 1 END) as delivered_orders,
				COUNT(CASE WHEN status = 'CANCELLED' THEN 1 END) as cancelled_orders,
				COALESCE(SUM(total_amount), 0) as total_revenue,
				COALESCE(AVG(total_amount), 0) as average_order_value
			FROM orders
			WHERE order_date >= $1 AND order_date <= $2
			GROUP BY TO_CHAR(order_date, '%s')
		)
		SELECT
			period,
			total_revenue,
			total_orders as order_count,
			average_order_value
		FROM order_periods
		ORDER BY period
	`, dateFormat, dateFormat)

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get order trends: %w", err)
	}
	defer rows.Close()

	var trends []*repositories.RevenueByPeriod
	for rows.Next() {
		trend := &repositories.RevenueByPeriod{}
		err := rows.Scan(
			&trend.Period,
			&trend.Revenue,
			&trend.OrderCount,
			&trend.AverageOrderValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order trend row: %w", err)
		}
		trends = append(trends, trend)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order trend rows: %w", err)
	}

	return trends, nil
}

// GetSalesByRegion retrieves sales data by geographic region
func (r *PostgresOrderAnalyticsRepository) GetSalesByRegion(ctx context.Context, startDate, endDate time.Time, limit int) ([]*RegionSalesStats, error) {

	query := `
		SELECT
			COALESCE(shipping.country, '') as country,
			COALESCE(shipping.state, '') as state,
			COALESCE(shipping.city, '') as city,
			COUNT(DISTINCT o.id) as order_count,
			COALESCE(SUM(o.total_amount), 0) as total_revenue,
			COALESCE(AVG(o.total_amount), 0) as avg_order_value
		FROM orders o
		LEFT JOIN order_addresses shipping ON o.shipping_address_id = shipping.id
		WHERE o.order_date >= $1 AND o.order_date <= $2
		AND o.status NOT IN ('CANCELLED', 'REFUNDED')
		GROUP BY shipping.country, shipping.state, shipping.city
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales by region: %w", err)
	}
	defer rows.Close()

	var regions []*RegionSalesStats
	for rows.Next() {
		region := &RegionSalesStats{}
		err := rows.Scan(
			&region.Country,
			&region.State,
			&region.City,
			&region.OrderCount,
			&region.TotalRevenue,
			&region.AvgOrderValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sales by region row: %w", err)
		}
		regions = append(regions, region)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sales by region rows: %w", err)
	}

	return regions, nil
}

// GetCustomerRetentionAnalysis analyzes customer retention rates
func (r *PostgresOrderAnalyticsRepository) GetCustomerRetentionAnalysis(ctx context.Context, startDate, endDate time.Time) (*CustomerRetentionStats, error) {

	query := `
		WITH customer_periods AS (
			SELECT
				c.id,
				CASE
					WHEN MIN(o.order_date) >= $1 THEN 'new'
					ELSE 'returning'
				END as customer_type,
				COUNT(o.id) as order_count,
				COALESCE(SUM(o.total_amount), 0) as total_spent
			FROM customers c
			LEFT JOIN orders o ON c.id = o.customer_id
			AND o.order_date >= $1 AND o.order_date <= $2
			AND o.status NOT IN ('CANCELLED', 'REFUNDED')
			WHERE c.created_at <= $2
			GROUP BY c.id
		)
		SELECT
			COUNT(*) as total_customers,
			COUNT(CASE WHEN customer_type = 'new' THEN 1 END) as new_customers,
			COUNT(CASE WHEN customer_type = 'returning' THEN 1 END) as returning_customers,
			COALESCE(AVG(order_count), 0) as avg_orders_per_customer,
			COALESCE(AVG(total_spent), 0) as avg_customer_value
		FROM customer_periods
	`

	stats := &CustomerRetentionStats{}
	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&stats.TotalCustomers,
		&stats.NewCustomers,
		&stats.ReturningCustomers,
		&stats.AverageOrdersPerCustomer,
		&stats.CustomerLifetimeValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer retention analysis: %w", err)
	}

	// Calculate retention rate
	if stats.TotalCustomers > 0 {
		stats.RetentionRate = decimal.NewFromInt(stats.ReturningCustomers).Div(decimal.NewFromInt(stats.TotalCustomers))
	}

	return stats, nil
}

// GetInventoryTurnoverAnalysis analyzes inventory turnover rates
func (r *PostgresOrderAnalyticsRepository) GetInventoryTurnoverAnalysis(ctx context.Context, startDate, endDate time.Time) ([]*InventoryTurnoverStats, error) {

	query := `
		WITH product_sales AS (
			SELECT
				p.id as product_id,
				p.sku as product_sku,
				p.name as product_name,
				COALESCE(SUM(oi.quantity), 0) as quantity_sold
			FROM products p
			LEFT JOIN order_items oi ON p.id = oi.product_id
			LEFT JOIN orders o ON oi.order_id = o.id
			AND o.order_date >= $1 AND o.order_date <= $2
			AND o.status NOT IN ('CANCELLED', 'REFUNDED')
			GROUP BY p.id, p.sku, p.name
		),
		average_inventory AS (
			SELECT
				product_id,
				COALESCE(AVG(quantity_available), 0) as avg_stock
			FROM inventory
			WHERE recorded_at >= $1 AND recorded_at <= $2
			GROUP BY product_id
		)
		SELECT
			ps.product_id,
			ps.product_sku,
			ps.product_name,
			ps.quantity_sold,
			COALESCE(ai.avg_stock, 0) as average_stock,
			CASE
				WHEN COALESCE(ai.avg_stock, 0) > 0
				THEN CAST(ps.quantity_sold as decimal) / ai.avg_stock
				ELSE 0
			END as turnover_rate
		FROM product_sales ps
		LEFT JOIN average_inventory ai ON ps.product_id = ai.product_id
		WHERE ps.quantity_sold > 0
		ORDER BY turnover_rate DESC
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory turnover analysis: %w", err)
	}
	defer rows.Close()

	var stats []*InventoryTurnoverStats
	for rows.Next() {
		stat := &InventoryTurnoverStats{}
		err := rows.Scan(
			&stat.ProductID,
			&stat.ProductSKU,
			&stat.ProductName,
			&stat.QuantitySold,
			&stat.AverageStock,
			&stat.TurnoverRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inventory turnover row: %w", err)
		}

		// Calculate days in stock (simplified calculation)
		if stat.TurnoverRate.GreaterThan(decimal.Zero) {
			days := endDate.Sub(startDate).Hours() / 24.0
			stat.DaysInStock = int64(days / stat.TurnoverRate.InexactFloat64())
		}

		stats = append(stats, stat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inventory turnover rows: %w", err)
	}

	return stats, nil
}

// GetPaymentAnalysis analyzes payment patterns and trends
func (r *PostgresOrderAnalyticsRepository) GetPaymentAnalysis(ctx context.Context, startDate, endDate time.Time) (*PaymentAnalysisStats, error) {

	query := `
		SELECT
			COUNT(*) as total_orders,
			COUNT(CASE WHEN payment_status = 'PAID' THEN 1 END) as paid_orders,
			COUNT(CASE WHEN payment_status = 'PARTIALLY_PAID' THEN 1 END) as partially_paid_orders,
			COUNT(CASE WHEN payment_status = 'PENDING' THEN 1 END) as pending_orders,
			COUNT(CASE WHEN payment_status = 'OVERDUE' THEN 1 END) as overdue_orders,
			COUNT(CASE WHEN payment_status = 'FAILED' THEN 1 END) as failed_orders,
			COALESCE(SUM(CASE WHEN payment_status = 'OVERDUE' THEN total_amount - paid_amount ELSE 0 END), 0) as overdue_amount
		FROM orders
		WHERE order_date >= $1 AND order_date <= $2
	`

	stats := &PaymentAnalysisStats{
		PaymentStatusBreakdown: make(map[string]int64),
		PaymentMethodBreakdown: make(map[string]int64),
	}

	var paidOrders, partiallyPaidOrders, pendingOrders, overdueOrders, failedOrders int64

	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&stats.TotalOrders,
		&paidOrders,
		&partiallyPaidOrders,
		&pendingOrders,
		&overdueOrders,
		&failedOrders,
		&stats.OverdueAmount,
	)

	// Fill the breakdown map
	stats.PaymentStatusBreakdown["PAID"] = paidOrders
	stats.PaymentStatusBreakdown["PARTIALLY_PAID"] = partiallyPaidOrders
	stats.PaymentStatusBreakdown["PENDING"] = pendingOrders
	stats.PaymentStatusBreakdown["OVERDUE"] = overdueOrders
	stats.PaymentStatusBreakdown["FAILED"] = failedOrders
	if err != nil {
		return nil, fmt.Errorf("failed to get payment analysis: %w", err)
	}

	// Get payment trends
	trends, err := r.GetRevenueByPeriod(ctx, startDate, endDate, "month")
	if err != nil {
		return nil, fmt.Errorf("failed to get payment trends: %w", err)
	}
	stats.PaymentTrends = trends

	return stats, nil
}

// GetSeasonalAnalysis analyzes seasonal patterns in orders
func (r *PostgresOrderAnalyticsRepository) GetSeasonalAnalysis(ctx context.Context, years int) ([]*SeasonalStats, error) {

	query := `
		WITH seasonal_data AS (
			SELECT
				EXTRACT(YEAR FROM order_date) as year,
				CASE
					WHEN EXTRACT(MONTH FROM order_date) IN (12, 1, 2) THEN 'Winter'
					WHEN EXTRACT(MONTH FROM order_date) IN (3, 4, 5) THEN 'Spring'
					WHEN EXTRACT(MONTH FROM order_date) IN (6, 7, 8) THEN 'Summer'
					WHEN EXTRACT(MONTH FROM order_date) IN (9, 10, 11) THEN 'Fall'
				END as season,
				COUNT(*) as order_count,
				COALESCE(SUM(total_amount), 0) as total_revenue,
				COALESCE(AVG(total_amount), 0) as avg_order_value
			FROM orders
			WHERE order_date >= CURRENT_DATE - INTERVAL '%d years'
			AND status NOT IN ('CANCELLED', 'REFUNDED')
			GROUP BY EXTRACT(YEAR FROM order_date), season
		)
		SELECT season, year, order_count, total_revenue, avg_order_value
		FROM seasonal_data
		ORDER BY year DESC, season
	`

	rows, err := r.db.Query(ctx, query, years)
	if err != nil {
		return nil, fmt.Errorf("failed to get seasonal analysis: %w", err)
	}
	defer rows.Close()

	var seasonalStats []*SeasonalStats
	for rows.Next() {
		stat := &SeasonalStats{}
		err := rows.Scan(
			&stat.Season,
			&stat.Year,
			&stat.OrderCount,
			&stat.TotalRevenue,
			&stat.AvgOrderValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan seasonal analysis row: %w", err)
		}
		seasonalStats = append(seasonalStats, stat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating seasonal analysis rows: %w", err)
	}

	return seasonalStats, nil
}
