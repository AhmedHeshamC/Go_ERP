package repositories

import (
	"context"
	"fmt"
	"strings"

	"erpgo/internal/domain/orders/entities"
	"erpgo/internal/domain/orders/repositories"
	"erpgo/pkg/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PostgresCompanyRepository implements CompanyRepository for PostgreSQL
type PostgresCompanyRepository struct {
	db *database.Database
}

// NewPostgresCompanyRepository creates a new PostgreSQL company repository
func NewPostgresCompanyRepository(db *database.Database) *PostgresCompanyRepository {
	return &PostgresCompanyRepository{
		db: db,
	}
}

// Create creates a new company and returns the created company
func (r *PostgresCompanyRepository) Create(ctx context.Context, company *entities.Company) (*entities.Company, error) {
	query := `
		INSERT INTO companies (
			id, company_name, legal_name, tax_id, industry, website, phone,
			email, address, city, state, country, postal_code, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	_, err := r.db.Exec(ctx, query,
		company.ID,
		company.CompanyName,
		company.LegalName,
		company.TaxID,
		company.Industry,
		company.Website,
		company.Phone,
		company.Email,
		company.Address,
		company.City,
		company.State,
		company.Country,
		company.PostalCode,
		company.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return company, nil
}

// GetByID retrieves a company by ID
func (r *PostgresCompanyRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Company, error) {
	query := `
		SELECT
			id, company_name, legal_name, tax_id, industry, website, phone,
			email, address, city, state, country, postal_code, is_active,
			created_at, updated_at
		FROM companies
		WHERE id = $1
	`

	company := &entities.Company{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&company.ID,
		&company.CompanyName,
		&company.LegalName,
		&company.TaxID,
		&company.Industry,
		&company.Website,
		&company.Phone,
		&company.Email,
		&company.Address,
		&company.City,
		&company.State,
		&company.Country,
		&company.PostalCode,
		&company.IsActive,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("company with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get company by id: %w", err)
	}

	return company, nil
}

// GetByTaxID retrieves a company by tax ID
func (r *PostgresCompanyRepository) GetByTaxID(ctx context.Context, taxID string) (*entities.Company, error) {
	query := `
		SELECT
			id, company_name, legal_name, tax_id, industry, website, phone,
			email, address, city, state, country, postal_code, is_active,
			created_at, updated_at
		FROM companies
		WHERE tax_id = $1
	`

	company := &entities.Company{}
	err := r.db.QueryRow(ctx, query, taxID).Scan(
		&company.ID,
		&company.CompanyName,
		&company.LegalName,
		&company.TaxID,
		&company.Industry,
		&company.Website,
		&company.Phone,
		&company.Email,
		&company.Address,
		&company.City,
		&company.State,
		&company.Country,
		&company.PostalCode,
		&company.IsActive,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("company with tax id %s not found", taxID)
		}
		return nil, fmt.Errorf("failed to get company by tax id: %w", err)
	}

	return company, nil
}

// Update updates a company
func (r *PostgresCompanyRepository) Update(ctx context.Context, company *entities.Company) (*entities.Company, error) {
	query := `
		UPDATE companies SET
			company_name = $2, legal_name = $3, tax_id = $4, industry = $5,
			website = $6, phone = $7, email = $8, address = $9, city = $10,
			state = $11, country = $12, postal_code = $13, is_active = $14,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		company.ID,
		company.CompanyName,
		company.LegalName,
		company.TaxID,
		company.Industry,
		company.Website,
		company.Phone,
		company.Email,
		company.Address,
		company.City,
		company.State,
		company.Country,
		company.PostalCode,
		company.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update company: %w", err)
	}

	return company, nil
}

// Delete deletes a company
func (r *PostgresCompanyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM companies WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	return nil
}

// List retrieves a list of companies with filtering
func (r *PostgresCompanyRepository) List(ctx context.Context, filter repositories.CompanyFilter) ([]*entities.Company, error) {
	baseQuery, args, err := r.buildCompanyQuery(filter, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build company query: %w", err)
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []*entities.Company
	for rows.Next() {
		company := &entities.Company{}
		err := rows.Scan(
			&company.ID,
			&company.CompanyName,
			&company.LegalName,
			&company.TaxID,
			&company.Industry,
			&company.Website,
			&company.Phone,
			&company.Email,
			&company.Address,
			&company.City,
			&company.State,
			&company.Country,
			&company.PostalCode,
			&company.IsActive,
			&company.CreatedAt,
			&company.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan company row: %w", err)
		}
		companies = append(companies, company)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company rows: %w", err)
	}

	return companies, nil
}

// Count returns the count of companies matching the filter
func (r *PostgresCompanyRepository) Count(ctx context.Context, filter repositories.CompanyFilter) (int, error) {
	baseQuery, args, err := r.buildCompanyQuery(filter, true)
	if err != nil {
		return 0, fmt.Errorf("failed to build company count query: %w", err)
	}

	var count int
	err = r.db.QueryRow(ctx, baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count companies: %w", err)
	}

	return count, nil
}

// buildCompanyQuery builds the SQL query for companies based on filter
func (r *PostgresCompanyRepository) buildCompanyQuery(filter repositories.CompanyFilter, isCount bool) (string, []interface{}, error) {
	var selectClause string
	if isCount {
		selectClause = "SELECT COUNT(*)"
	} else {
		selectClause = `
			SELECT
				id, company_name, legal_name, tax_id, industry, website, phone,
				email, address, city, state, country, postal_code, is_active,
				created_at, updated_at
		`
	}

	baseQuery := selectClause + " FROM companies WHERE 1=1"

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Search filter
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(company_name ILIKE $%d OR legal_name ILIKE $%d OR tax_id ILIKE $%d OR email ILIKE $%d)",
			argIndex, argIndex+1, argIndex+2, argIndex+3))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		argIndex += 4
	}

	// Industry filter
	if filter.Industry != nil {
		conditions = append(conditions, fmt.Sprintf("industry = $%d", argIndex))
		args = append(args, *filter.Industry)
		argIndex++
	}

	// Active status filter
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	// Date filters
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY for non-count queries
	if !isCount {
		sortBy := "created_at"
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}

		sortOrder := "DESC"
		if filter.SortOrder != "" {
			sortOrder = strings.ToUpper(filter.SortOrder)
		}
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

		// Add LIMIT and OFFSET for pagination
		if filter.Limit > 0 {
			baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, filter.Limit)
			argIndex++

			if filter.Offset > 0 {
				baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
				args = append(args, filter.Offset)
			} else if filter.Page > 1 {
				offset := (filter.Page - 1) * filter.Limit
				baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
				args = append(args, offset)
			}
		}
	}

	return baseQuery, args, nil
}

// Search performs advanced search on companies
func (r *PostgresCompanyRepository) Search(ctx context.Context, query string, filter repositories.CompanyFilter) ([]*entities.Company, error) {
	filter.Search = query
	return r.List(ctx, filter)
}

// GetActiveCompanies retrieves active companies
func (r *PostgresCompanyRepository) GetActiveCompanies(ctx context.Context) ([]*entities.Company, error) {
	filter := repositories.CompanyFilter{
		IsActive: func() *bool { b := true; return &b }(),
		Limit:    1000,
	}
	return r.List(ctx, filter)
}

// GetInactiveCompanies retrieves inactive companies
func (r *PostgresCompanyRepository) GetInactiveCompanies(ctx context.Context) ([]*entities.Company, error) {
	filter := repositories.CompanyFilter{
		IsActive: func() *bool { b := false; return &b }(),
		Limit:    1000,
	}
	return r.List(ctx, filter)
}

// GetCompaniesByIndustry retrieves companies by industry
func (r *PostgresCompanyRepository) GetCompaniesByIndustry(ctx context.Context, industry string) ([]*entities.Company, error) {
	filter := repositories.CompanyFilter{
		Industry: &industry,
		Limit:    1000,
	}
	return r.List(ctx, filter)
}

// ExistsByTaxID checks if a company exists by tax ID
func (r *PostgresCompanyRepository) ExistsByTaxID(ctx context.Context, taxID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM companies WHERE tax_id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, taxID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if company exists by tax id: %w", err)
	}

	return exists, nil
}

// ExistsByName checks if a company exists by name
func (r *PostgresCompanyRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM companies WHERE company_name = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if company exists by name: %w", err)
	}

	return exists, nil
}
