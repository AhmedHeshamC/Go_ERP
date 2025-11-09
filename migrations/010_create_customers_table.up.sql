-- Create companies table first (for business customers)
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(200) NOT NULL,
    legal_name VARCHAR(200) NOT NULL,
    tax_id VARCHAR(50) NOT NULL,
    industry VARCHAR(100),
    website VARCHAR(500),
    phone VARCHAR(50),
    email VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for companies table
CREATE INDEX idx_companies_company_name ON companies(company_name);
CREATE INDEX idx_companies_tax_id ON companies(tax_id);
CREATE INDEX idx_companies_industry ON companies(industry);
CREATE INDEX idx_companies_is_active ON companies(is_active);
CREATE INDEX idx_companies_created_at ON companies(created_at);

-- Create trigger for companies table
CREATE OR REPLACE FUNCTION update_companies_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW
    EXECUTE FUNCTION update_companies_updated_at();

-- Add comments for companies table
COMMENT ON TABLE companies IS 'Companies for business customers';
COMMENT ON COLUMN companies.id IS 'Unique identifier for the company';
COMMENT ON COLUMN companies.company_name IS 'Display name of the company';
COMMENT ON COLUMN companies.legal_name IS 'Legal name of the company';
COMMENT ON COLUMN companies.tax_id IS 'Tax identification number';
COMMENT ON COLUMN companies.industry IS 'Industry classification';
COMMENT ON COLUMN companies.website IS 'Company website URL';
COMMENT ON COLUMN companies.phone IS 'Company phone number';
COMMENT ON COLUMN companies.email IS 'Company email address';
COMMENT ON COLUMN companies.address IS 'Street address';
COMMENT ON COLUMN companies.city IS 'City name';
COMMENT ON COLUMN companies.state IS 'State or province';
COMMENT ON COLUMN companies.country IS 'Country name';
COMMENT ON COLUMN companies.postal_code IS 'Postal or ZIP code';
COMMENT ON COLUMN companies.is_active IS 'Whether the company is active';
COMMENT ON COLUMN companies.created_at IS 'Timestamp when the company was created';
COMMENT ON COLUMN companies.updated_at IS 'Timestamp when the company was last updated';

-- Create customers table
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_code VARCHAR(50) NOT NULL UNIQUE,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('INDIVIDUAL', 'BUSINESS', 'GOVERNMENT', 'NON_PROFIT')),

    -- Basic information
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    website VARCHAR(500),

    -- Business information
    company_name VARCHAR(200),
    tax_id VARCHAR(50),
    industry VARCHAR(100),

    -- Financial information
    credit_limit DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (credit_limit >= 0),
    credit_used DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (credit_used >= 0),
    terms VARCHAR(20) NOT NULL CHECK (terms ~ '^NET\d+$'),

    -- Status and settings
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_vat_exempt BOOLEAN NOT NULL DEFAULT false,
    preferred_currency VARCHAR(3) NOT NULL DEFAULT 'USD' CHECK (preferred_currency ~ '^[A-Z]{3}$'),

    -- Metadata
    notes TEXT,
    source VARCHAR(20) NOT NULL CHECK (source IN ('WEB', 'PHONE', 'EMAIL', 'REFERRAL', 'WALK_IN', 'SOCIAL', 'ADVERTISEMENT', 'OTHER')),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for customers table
CREATE INDEX idx_customers_customer_code ON customers(customer_code);
CREATE INDEX idx_customers_company_id ON customers(company_id);
CREATE INDEX idx_customers_type ON customers(type);
CREATE INDEX idx_customers_first_name ON customers(first_name);
CREATE INDEX idx_customers_last_name ON customers(last_name);
CREATE INDEX idx_customers_email ON customers(email) WHERE email IS NOT NULL;
CREATE INDEX idx_customers_phone ON customers(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_customers_is_active ON customers(is_active);
CREATE INDEX idx_customers_preferred_currency ON customers(preferred_currency);
CREATE INDEX idx_customers_source ON customers(source);
CREATE INDEX idx_customers_created_at ON customers(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_customers_name_active ON customers(last_name, first_name, is_active) WHERE is_active = true;
CREATE INDEX idx_customers_type_active ON customers(type, is_active) WHERE is_active = true;
CREATE INDEX idx_customers_company_active ON customers(company_id, is_active) WHERE is_active = true AND company_id IS NOT NULL;

-- Create trigger for customers table
CREATE TRIGGER trigger_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION update_companies_updated_at();

-- Add check constraints for business logic
ALTER TABLE customers ADD CONSTRAINT check_credit_limit_used
    CHECK (credit_used <= credit_limit);

ALTER TABLE customers ADD CONSTRAINT check_business_customer_consistency
    CHECK (
        (type = 'BUSINESS' AND company_name IS NOT NULL) OR
        (type != 'BUSINESS')
    );

-- Add comments for customers table
COMMENT ON TABLE customers IS 'Customers for orders and sales';
COMMENT ON COLUMN customers.id IS 'Unique identifier for the customer';
COMMENT ON COLUMN customers.customer_code IS 'Unique customer code';
COMMENT ON COLUMN customers.company_id IS 'Reference to company for business customers';
COMMENT ON COLUMN customers.type IS 'Customer type: INDIVIDUAL, BUSINESS, GOVERNMENT, NON_PROFIT';
COMMENT ON COLUMN customers.first_name IS 'Customer first name';
COMMENT ON COLUMN customers.last_name IS 'Customer last name';
COMMENT ON COLUMN customers.email IS 'Customer email address';
COMMENT ON COLUMN customers.phone IS 'Customer phone number';
COMMENT ON COLUMN customers.website IS 'Customer website URL';
COMMENT ON COLUMN customers.company_name IS 'Company name (for business customers)';
COMMENT ON COLUMN customers.tax_id IS 'Tax identification number';
COMMENT ON COLUMN customers.industry IS 'Industry classification';
COMMENT ON COLUMN customers.credit_limit IS 'Maximum credit limit';
COMMENT ON COLUMN customers.credit_used IS 'Currently used credit';
COMMENT ON COLUMN customers.terms IS 'Payment terms (e.g., NET30, NET60)';
COMMENT ON COLUMN customers.is_active IS 'Whether the customer is active';
COMMENT ON COLUMN customers.is_vat_exempt IS 'Whether the customer is VAT exempt';
COMMENT ON COLUMN customers.preferred_currency IS 'Preferred currency for transactions';
COMMENT ON COLUMN customers.notes IS 'Additional notes about the customer';
COMMENT ON COLUMN customers.source IS 'How the customer was acquired';
COMMENT ON COLUMN customers.created_at IS 'Timestamp when the customer was created';
COMMENT ON COLUMN customers.updated_at IS 'Timestamp when the customer was last updated';