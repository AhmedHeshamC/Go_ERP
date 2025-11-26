package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/tests/integration/testutil"
)

// CustomerRepositoryTestSuite contains all tests for the customer repository
type CustomerRepositoryTestSuite struct {
	suite.Suite
	db          *testutil.TestDatabase
	repo        repositories.CustomerRepository
	cleanupFunc func()
}

// SetupSuite sets up the test suite
func (suite *CustomerRepositoryTestSuite) SetupSuite() {
	// This would typically set up a test database
	suite.T().Skip("Skipping database tests - requires test database setup")
}

// TearDownSuite tears down the test suite
func (suite *CustomerRepositoryTestSuite) TearDownSuite() {
	if suite.cleanupFunc != nil {
		suite.cleanupFunc()
	}
}

// TestCreateCustomer tests creating a new customer
func (suite *CustomerRepositoryTestSuite) TestCreateCustomer() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Verify the customer was created
	retrieved, err := suite.repo.GetByID(ctx, customer.ID)
	require.NoError(suite.T(), err)
	testutil.AssertCustomersEqual(suite.T(), customer, retrieved)
}

// TestGetByID tests retrieving a customer by ID
func (suite *CustomerRepositoryTestSuite) TestGetByID() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Retrieve the customer
	retrieved, err := suite.repo.GetByID(ctx, customer.ID)
	require.NoError(suite.T(), err)
	testutil.AssertCustomersEqual(suite.T(), customer, retrieved)
}

// TestGetByCustomerCode tests retrieving a customer by customer code
func (suite *CustomerRepositoryTestSuite) TestGetByCustomerCode() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Retrieve the customer by code
	retrieved, err := suite.repo.GetByCustomerCode(ctx, customer.CustomerCode)
	require.NoError(suite.T(), err)
	testutil.AssertCustomersEqual(suite.T(), customer, retrieved)
}

// TestUpdateCustomer tests updating a customer
func (suite *CustomerRepositoryTestSuite) TestUpdateCustomer() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Update the customer
	customer.FirstName = "Jane"
	customer.LastName = "Smith"
	customer.Notes = testutil.StringPtr("Updated customer notes")
	customer.UpdatedAt = testutil.CreateTestContext().Value("now").(time.Time)

	err = suite.repo.Update(ctx, customer)
	require.NoError(suite.T(), err)

	// Verify the update
	retrieved, err := suite.repo.GetByID(ctx, customer.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Jane", retrieved.FirstName)
	assert.Equal(suite.T(), "Smith", retrieved.LastName)
	assert.Equal(suite.T(), "Updated customer notes", *retrieved.Notes)
}

// TestDeleteCustomer tests deleting a customer
func (suite *CustomerRepositoryTestSuite) TestDeleteCustomer() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Delete the customer
	err = suite.repo.Delete(ctx, customer.ID)
	require.NoError(suite.T(), err)

	// Verify the customer was deleted
	_, err = suite.repo.GetByID(ctx, customer.ID)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// TestListCustomers tests listing customers with filters
func (suite *CustomerRepositoryTestSuite) TestListCustomers() {
	ctx := testutil.CreateTestContext()

	// Create test customers
	customers := make([]*entities.Customer, 5)
	for i := 0; i < 5; i++ {
		customers[i] = testutil.CreateTestCustomer(suite.T())
		if i%2 == 0 {
			customers[i].Type = "INDIVIDUAL"
		} else {
			customers[i].Type = "BUSINESS"
		}
		customers[i].Email = fmt.Sprintf("customer%d@example.com", i)
		err := suite.repo.Create(ctx, customers[i])
		require.NoError(suite.T(), err)
	}

	// Test listing all customers
	filter := repositories.CustomerFilter{
		Limit: 10,
	}
	allCustomers, err := suite.repo.List(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(allCustomers), 5)

	// Test filtering by type
	filter.Type = "INDIVIDUAL"
	individualCustomers, err := suite.repo.List(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(individualCustomers), 2)
}

// TestCountCustomers tests counting customers with filters
func (suite *CustomerRepositoryTestSuite) TestCountCustomers() {
	ctx := testutil.CreateTestContext()

	// Create test customers
	for i := 0; i < 3; i++ {
		customer := testutil.CreateTestCustomer(suite.T())
		customer.Type = "INDIVIDUAL"
		customer.Email = fmt.Sprintf("count%d@example.com", i)
		err := suite.repo.Create(ctx, customer)
		require.NoError(suite.T(), err)
	}

	// Test counting all customers
	filter := repositories.CustomerFilter{}
	count, err := suite.repo.Count(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), count, 3)

	// Test counting by type
	filter.Type = "INDIVIDUAL"
	individualCount, err := suite.repo.Count(ctx, filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), individualCount, 3)
}

// TestSearchCustomers tests searching customers
func (suite *CustomerRepositoryTestSuite) TestSearchCustomers() {
	ctx := testutil.CreateTestContext()

	// Create test customers with specific data
	customer1 := testutil.CreateTestCustomer(suite.T())
	customer1.FirstName = "Searchable"
	customer1.CompanyName = testutil.StringPtr("Search Company")
	err := suite.repo.Create(ctx, customer1)
	require.NoError(suite.T(), err)

	customer2 := testutil.CreateTestCustomer(suite.T())
	customer2.LastName = "Findable"
	customer2.Email = "findable@example.com"
	err = suite.repo.Create(ctx, customer2)
	require.NoError(suite.T(), err)

	// Search for customers
	filter := repositories.CustomerFilter{
		Limit: 10,
	}

	// Search by first name
	customers, err := suite.repo.Search(ctx, "Searchable", filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(customers), 1)

	// Search by company name
	customers, err = suite.repo.Search(ctx, "Search Company", filter)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(customers), 1)
}

// TestExistsByEmail tests checking if a customer exists by email
func (suite *CustomerRepositoryTestSuite) TestExistsByEmail() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Test existing customer
	exists, err := suite.repo.ExistsByEmail(ctx, customer.Email)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test non-existing customer
	exists, err = suite.repo.ExistsByEmail(ctx, "nonexistent@example.com")
	require.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestExistsByCustomerCode tests checking if a customer exists by customer code
func (suite *CustomerRepositoryTestSuite) TestExistsByCustomerCode() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Test existing customer
	exists, err := suite.repo.ExistsByCustomerCode(ctx, customer.CustomerCode)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test non-existing customer
	exists, err = suite.repo.ExistsByCustomerCode(ctx, "NON-EXISTENT-CODE")
	require.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestGetActiveCustomers tests retrieving active customers
func (suite *CustomerRepositoryTestSuite) TestGetActiveCustomers() {
	ctx := testutil.CreateTestContext()

	// Create test customers
	for i := 0; i < 3; i++ {
		customer := testutil.CreateTestCustomer(suite.T())
		customer.IsActive = true
		customer.Email = fmt.Sprintf("active%d@example.com", i)
		err := suite.repo.Create(ctx, customer)
		require.NoError(suite.T(), err)
	}

	// Retrieve active customers
	activeCustomers, err := suite.repo.GetActiveCustomers(ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(activeCustomers), 3)

	// Verify all returned customers are active
	for _, customer := range activeCustomers {
		assert.True(suite.T(), customer.IsActive)
	}
}

// TestGetCustomersByType tests retrieving customers by type
func (suite *CustomerRepositoryTestSuite) TestGetCustomersByType() {
	ctx := testutil.CreateTestContext()

	// Create test business customers
	for i := 0; i < 3; i++ {
		customer := testutil.CreateTestCustomer(suite.T())
		customer.Type = "BUSINESS"
		customer.Email = fmt.Sprintf("business%d@example.com", i)
		err := suite.repo.Create(ctx, customer)
		require.NoError(suite.T(), err)
	}

	// Retrieve business customers
	businessCustomers, err := suite.repo.GetCustomersByType(ctx, "BUSINESS")
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(businessCustomers), 3)

	// Verify all returned customers are business customers
	for _, customer := range businessCustomers {
		assert.Equal(suite.T(), "BUSINESS", customer.Type)
	}
}

// TestUpdateCreditUsed tests updating customer credit used amount
func (suite *CustomerRepositoryTestSuite) TestUpdateCreditUsed() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Update credit used amount
	newCreditUsed := decimal.NewFromFloat(250.00)
	err = suite.repo.UpdateCreditUsed(ctx, customer.ID, newCreditUsed)
	require.NoError(suite.T(), err)

	// Verify the update
	retrieved, err := suite.repo.GetByID(ctx, customer.ID)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), newCreditUsed.Equal(retrieved.CreditUsed))
}

// TestGetCustomersWithCreditLimit tests retrieving customers with credit limits
func (suite *CustomerRepositoryTestSuite) TestGetCustomersWithCreditLimit() {
	ctx := testutil.CreateTestContext()

	// Create test customers with credit limits
	for i := 0; i < 3; i++ {
		customer := testutil.CreateTestCustomer(suite.T())
		customer.CreditLimit = decimal.NewFromFloat(float64(i+1) * 1000.00)
		customer.Email = fmt.Sprintf("credit%d@example.com", i)
		err := suite.repo.Create(ctx, customer)
		require.NoError(suite.T(), err)
	}

	// Retrieve customers with credit limits
	customers, err := suite.repo.GetCustomersWithCreditLimit(ctx)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(customers), 3)

	// Verify all returned customers have credit limits
	for _, customer := range customers {
		assert.Greater(suite.T(), customer.CreditLimit, decimal.Zero)
	}
}

// TestGetCustomerStats tests retrieving customer statistics
func (suite *CustomerRepositoryTestSuite) TestGetCustomerStats() {
	ctx := testutil.CreateTestContext()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	// Create test customers
	for i := 0; i < 5; i++ {
		customer := testutil.CreateTestCustomer(suite.T())
		customer.CreatedAt = time.Now().AddDate(0, 0, -i)
		customer.Email = fmt.Sprintf("stats%d@example.com", i)
		err := suite.repo.Create(ctx, customer)
		require.NoError(suite.T(), err)
	}

	// Retrieve customer statistics
	filter := repositories.CustomerStatsFilter{
		StartDate: startDate,
		EndDate:   endDate,
	}
	stats, err := suite.repo.GetCustomerStats(ctx, filter)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), stats.TotalCustomers, int64(0))
	assert.GreaterOrEqual(suite.T(), stats.ActiveCustomers, int64(0))
	assert.GreaterOrEqual(suite.T(), stats.NewCustomers, int64(0))
	assert.NotNil(suite.T(), stats.CustomersByType)
	assert.NotNil(suite.T(), stats.CustomersBySource)
}

// TestGetCustomerOrdersSummary tests retrieving a customer's order summary
func (suite *CustomerRepositoryTestSuite) TestGetCustomerOrdersSummary() {
	ctx := testutil.CreateTestContext()
	customer := testutil.CreateTestCustomer(suite.T())

	// Create the customer first
	err := suite.repo.Create(ctx, customer)
	require.NoError(suite.T(), err)

	// Retrieve customer order summary
	summary, err := suite.repo.GetCustomerOrdersSummary(ctx, customer.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), customer.ID, summary.CustomerID)
	assert.GreaterOrEqual(suite.T(), summary.TotalOrders, int64(0))
	assert.GreaterOrEqual(suite.T(), summary.TotalRevenue, decimal.Zero)
	assert.NotNil(suite.T(), summary.StatusCounts)
}

// TestBulkCreate tests bulk creating customers
func (suite *CustomerRepositoryTestSuite) TestBulkCreate() {
	ctx := testutil.CreateTestContext()

	// Create test customers
	customers := make([]*entities.Customer, 3)
	for i := 0; i < 3; i++ {
		customers[i] = testutil.CreateTestCustomer(suite.T())
		customers[i].Email = fmt.Sprintf("bulk%d@example.com", i)
		customers[i].CustomerCode = fmt.Sprintf("BULK-CODE-%d", i)
	}

	// Bulk create customers
	err := suite.repo.BulkCreate(ctx, customers)
	require.NoError(suite.T(), err)

	// Verify the customers were created
	for _, customer := range customers {
		retrieved, err := suite.repo.GetByID(ctx, customer.ID)
		require.NoError(suite.T(), err)
		testutil.AssertCustomersEqual(suite.T(), customer, retrieved)
	}
}

// TestBulkUpdate tests bulk updating customers
func (suite *CustomerRepositoryTestSuite) TestBulkUpdate() {
	ctx := testutil.CreateTestContext()

	// Create test customers
	customers := make([]*entities.Customer, 3)
	for i := 0; i < 3; i++ {
		customers[i] = testutil.CreateTestCustomer(suite.T())
		customers[i].Email = fmt.Sprintf("bulkupdate%d@example.com", i)
		customers[i].CustomerCode = fmt.Sprintf("BULK-UPDATE-CODE-%d", i)
		err := suite.repo.Create(ctx, customers[i])
		require.NoError(suite.T(), err)
	}

	// Update the customers
	for _, customer := range customers {
		customer.Notes = testutil.StringPtr("Bulk updated notes")
	}

	// Bulk update customers
	err := suite.repo.BulkUpdate(ctx, customers)
	require.NoError(suite.T(), err)

	// Verify the updates
	for _, customer := range customers {
		retrieved, err := suite.repo.GetByID(ctx, customer.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Bulk updated notes", *retrieved.Notes)
	}
}

// Run the test suite
func TestCustomerRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(CustomerRepositoryTestSuite))
}

// BenchmarkCreateCustomer benchmarks customer creation
func BenchmarkCreateCustomer(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}

// BenchmarkGetCustomer benchmarks customer retrieval
func BenchmarkGetCustomer(b *testing.B) {
	// This would require a test database setup
	b.Skip("Skipping benchmark - requires test database setup")
}