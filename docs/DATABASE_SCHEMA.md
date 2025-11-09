# ERPGo Database Schema Documentation

## Overview

ERPGo uses PostgreSQL as its primary database, designed with a focus on data integrity, performance, and scalability. The schema follows third normal form (3NF) principles while maintaining query efficiency through strategic indexing.

## Table of Contents

1. [Database Architecture](#database-architecture)
2. [Core Tables](#core-tables)
3. [Supporting Tables](#supporting-tables)
4. [Relationships and Foreign Keys](#relationships-and-foreign-keys)
5. [Indexes and Performance](#indexes-and-performance)
6. [Constraints and Validation](#constraints-and-validation)
7. [Triggers and Automation](#triggers-and-automation)
8. [Data Types and Formats](#data-types-and-formats)
9. [Migration Strategy](#migration-strategy)
10. [Backup and Recovery](#backup-and-recovery)
11. [Security Considerations](#security-considerations)
12. [Performance Tuning](#performance-tuning)

## Database Architecture

### Database Configuration

- **Database Engine**: PostgreSQL 14+
- **Default Charset**: UTF8
- **Collation**: en_US.UTF-8
- **Timezone**: UTC (with timezone support)
- **Extensions**: uuid-ossp, pg_trgm, btree_gin

### Design Principles

1. **UUID Primary Keys**: All tables use UUID primary keys for global uniqueness
2. **Audit Fields**: All tables include created_at, updated_at timestamps
3. **Soft Deletes**: Critical tables use is_active flags instead of hard deletes
4. **Referential Integrity**: Foreign key constraints ensure data consistency
5. **Index Strategy**: Strategic indexing for optimal query performance

## Core Tables

### 1. Users Table

**Purpose**: User authentication and profile management

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    is_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

**Key Features**:
- Unique email and username constraints
- Password hashing with bcrypt
- Account status tracking (active, verified)
- Last login timestamp for security monitoring

**Indexes**:
- `idx_users_email` - Unique lookup by email
- `idx_users_username` - Unique lookup by username
- `idx_users_active` - Filter active users
- `idx_users_created_at` - Sort by creation date

### 2. Roles Table

**Purpose**: Role-based access control (RBAC)

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    permissions JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

**Key Features**:
- Flexible JSONB permissions structure
- Hierarchical role support through permissions
- Active/inactive status for role management

### 3. User Roles Junction Table

**Purpose**: Many-to-many relationship between users and roles

```sql
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by UUID REFERENCES users(id),
    UNIQUE(user_id, role_id)
);
```

### 4. Product Categories Table

**Purpose**: Hierarchical product categorization

```sql
CREATE TABLE product_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES product_categories(id) ON DELETE RESTRICT,
    path VARCHAR(1000),  -- Materialized path for hierarchy
    level INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

**Key Features**:
- Self-referencing for hierarchical structure
- Materialized path for efficient hierarchy queries
- Level-based organization for performance

### 5. Products Table

**Purpose**: Product catalog management

```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(300) NOT NULL,
    description TEXT,
    short_description VARCHAR(500),
    category_id UUID NOT NULL REFERENCES product_categories(id) ON DELETE RESTRICT,
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    cost DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (cost >= 0),
    weight DECIMAL(10,3) NOT NULL DEFAULT 0,
    dimensions VARCHAR(100),
    length DECIMAL(8,2) NOT NULL DEFAULT 0,
    width DECIMAL(8,2) NOT NULL DEFAULT 0,
    height DECIMAL(8,2) NOT NULL DEFAULT 0,
    volume DECIMAL(10,3) NOT NULL DEFAULT 0,
    barcode VARCHAR(50),
    track_inventory BOOLEAN NOT NULL DEFAULT true,
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    min_stock_level INTEGER NOT NULL DEFAULT 0,
    max_stock_level INTEGER NOT NULL DEFAULT 0,
    allow_backorder BOOLEAN NOT NULL DEFAULT false,
    requires_shipping BOOLEAN NOT NULL DEFAULT true,
    taxable BOOLEAN NOT NULL DEFAULT true,
    tax_rate DECIMAL(5,2) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    is_digital BOOLEAN NOT NULL DEFAULT false,
    download_url VARCHAR(1000),
    max_downloads INTEGER NOT NULL DEFAULT 0,
    expiry_days INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Key Features**:
- Comprehensive product attributes
- Inventory tracking capabilities
- Digital product support
- Tax and shipping configurations
- Barcode integration

**Business Constraints**:
- Cost cannot exceed price
- Digital products cannot require shipping
- Inventory levels must be logical (min ≤ max)
- Tax settings must be consistent

### 6. Product Variants Table

**Purpose**: Product variations (size, color, etc.)

```sql
CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,
    name VARCHAR(300) NOT NULL,
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    cost DECIMAL(12,2) NOT NULL DEFAULT 0,
    weight DECIMAL(10,3) NOT NULL DEFAULT 0,
    barcode VARCHAR(50),
    track_inventory BOOLEAN NOT NULL DEFAULT true,
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    min_stock_level INTEGER NOT NULL DEFAULT 0,
    max_stock_level INTEGER NOT NULL DEFAULT 0,
    allow_backorder BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    attributes JSONB DEFAULT '{}',  -- {color: "red", size: "L"}
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(product_id, sku)
);
```

**Key Features**:
- Flexible attribute storage using JSONB
- Independent inventory tracking per variant
- Unique SKU within product scope

### 7. Customers Table

**Purpose**: Customer information and relationship management

```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    customer_code VARCHAR(50) UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    company VARCHAR(200),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    type VARCHAR(20) NOT NULL DEFAULT 'INDIVIDUAL' CHECK (type IN ('INDIVIDUAL', 'BUSINESS')),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED')),
    credit_limit DECIMAL(12,2) DEFAULT 0,
    balance DECIMAL(12,2) DEFAULT 0,
    tax_exempt BOOLEAN DEFAULT false,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Key Features**:
- Link to user accounts for customer portal access
- Customer classification (individual/business)
- Credit management capabilities
- Tax exemption support

### 8. Orders Table

**Purpose**: Order processing and fulfillment management

```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) NOT NULL UNIQUE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PENDING', 'CONFIRMED', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'CANCELLED', 'REFUNDED', 'RETURNED', 'ON_HOLD', 'PARTIALLY_SHIPPED')),
    previous_status VARCHAR(20) CHECK (previous_status IN ('DRAFT', 'PENDING', 'CONFIRMED', 'PROCESSING', 'SHIPPED', 'DELIVERED', 'CANCELLED', 'REFUNDED', 'RETURNED', 'ON_HOLD', 'PARTIALLY_SHIPPED')),
    priority VARCHAR(10) NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW', 'NORMAL', 'HIGH', 'URGENT', 'CRITICAL')),
    type VARCHAR(15) NOT NULL DEFAULT 'SALES' CHECK (type IN ('SALES', 'PURCHASE', 'RETURN', 'EXCHANGE', 'TRANSFER', 'ADJUSTMENT')),
    payment_status VARCHAR(15) NOT NULL DEFAULT 'PENDING' CHECK (payment_status IN ('PENDING', 'PAID', 'PARTIALLY_PAID', 'OVERDUE', 'REFUNDED', 'FAILED')),
    shipping_method VARCHAR(20) NOT NULL DEFAULT 'STANDARD' CHECK (shipping_method IN ('STANDARD', 'EXPRESS', 'OVERNIGHT', 'INTERNATIONAL', 'PICKUP', 'DIGITAL')),

    -- Financial fields
    subtotal DECIMAL(12,2) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    shipping_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    paid_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    refunded_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    -- Date fields
    order_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    required_date TIMESTAMP WITH TIME ZONE,
    shipping_date TIMESTAMP WITH TIME ZONE,
    delivery_date TIMESTAMP WITH TIME ZONE,
    cancelled_date TIMESTAMP WITH TIME ZONE,

    -- Address references
    shipping_address_id UUID NOT NULL REFERENCES order_addresses(id) ON DELETE RESTRICT,
    billing_address_id UUID NOT NULL REFERENCES order_addresses(id) ON DELETE RESTRICT,

    -- Metadata
    notes TEXT,
    internal_notes TEXT,
    customer_notes TEXT,
    tracking_number VARCHAR(100),
    carrier VARCHAR(50),

    -- System fields
    created_by UUID NOT NULL,
    approved_by UUID,
    shipped_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE,
    shipped_at TIMESTAMP WITH TIME ZONE
);
```

**Key Features**:
- Comprehensive order lifecycle management
- Financial tracking (payments, refunds)
- Multiple order types and priorities
- Audit trail with created_by, approved_by, shipped_by

**Business Constraints**:
- Total calculation must be consistent
- Paid amount cannot exceed total amount
- Refunded amount cannot exceed paid amount
- Date relationships must be logical

### 9. Order Items Table

**Purpose**: Line items within orders

```sql
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    variant_id UUID REFERENCES product_variants(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(12,2) NOT NULL CHECK (unit_price > 0),
    total_price DECIMAL(12,2) NOT GENERATED ALWAYS AS (quantity * unit_price) STORED,
    tax_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    final_price DECIMAL(12,2) NOT NULL GENERATED ALWAYS AS (total_price + tax_amount - discount_amount) STORED,
    weight DECIMAL(10,3) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'CONFIRMED', 'SHIPPED', 'DELIVERED', 'CANCELLED', 'RETURNED')),
    shipped_quantity INTEGER NOT NULL DEFAULT 0,
    returned_quantity INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Key Features**:
- Support for product variants
- Computed columns for totals
- Separate tracking for shipped/returned quantities
- Individual item status tracking

## Supporting Tables

### 10. Warehouses Table

**Purpose**: Warehouse and location management

```sql
CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(100),
    postal_code VARCHAR(20),
    phone VARCHAR(20),
    email VARCHAR(255),
    manager_id UUID REFERENCES users(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

### 11. Product Inventory Table

**Purpose**: Multi-warehouse inventory tracking

```sql
CREATE TABLE product_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    quantity_available INTEGER NOT NULL DEFAULT 0,
    quantity_reserved INTEGER NOT NULL DEFAULT 0,
    reorder_level INTEGER NOT NULL DEFAULT 0,
    max_stock INTEGER,
    last_updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_by UUID,
    UNIQUE(product_id, warehouse_id)
);
```

### 12. Inventory Transactions Table

**Purpose**: Audit trail for inventory movements

```sql
CREATE TABLE inventory_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('IN', 'OUT', 'ADJUST', 'TRANSFER')),
    quantity INTEGER NOT NULL,
    reference_id UUID,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID
);
```

### 13. Order Addresses Table

**Purpose**: Shipping and billing addresses for orders

```sql
CREATE TABLE order_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('shipping', 'billing')),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    company VARCHAR(200),
    address1 VARCHAR(255) NOT NULL,
    address2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

## Relationships and Foreign Keys

### Entity Relationship Diagram

```
Users (1) ----< (M) UserRoles (M) >---- (1) Roles
  |
  +---- (1) ----< (M) Orders
                         |
                         +---- (1) ----< (M) OrderItems ----> (1) Products
                         |                                    |
                         +---- (1) ----< (1) OrderAddresses      +---- (1) ----< (M) ProductVariants
                         |
                         +---- (1) ----< (1) Customers ----> (1) Users

ProductCategories (1) ----< (M) Products ----> (1) ProductInventory ----> (1) Warehouses
      |                                                    |
      +---- (1) ----< (M) ProductCategories (self)       +---- (1) ----< (M) InventoryTransactions
```

### Foreign Key Constraints

1. **User Management**:
   - `user_roles.user_id` → `users.id` (CASCADE)
   - `user_roles.role_id` → `roles.id` (CASCADE)
   - `customers.user_id` → `users.id` (SET NULL)

2. **Product Management**:
   - `products.category_id` → `product_categories.id` (RESTRICT)
   - `product_categories.parent_id` → `product_categories.id` (RESTRICT)
   - `product_variants.product_id` → `products.id` (CASCADE)

3. **Order Management**:
   - `orders.customer_id` → `customers.id` (RESTRICT)
   - `order_items.order_id` → `orders.id` (CASCADE)
   - `order_items.product_id` → `products.id` (RESTRICT)
   - `order_items.variant_id` → `product_variants.id` (RESTRICT)
   - `orders.shipping_address_id` → `order_addresses.id` (RESTRICT)
   - `orders.billing_address_id` → `order_addresses.id` (RESTRICT)

4. **Inventory Management**:
   - `product_inventory.product_id` → `products.id` (CASCADE)
   - `product_inventory.warehouse_id` → `warehouses.id` (CASCADE)
   - `inventory_transactions.product_id` → `products.id` (CASCADE)
   - `inventory_transactions.warehouse_id` → `warehouses.id` (CASCADE)

## Indexes and Performance

### Primary Indexes

All tables have primary key indexes on UUID columns using B-tree structure.

### Secondary Indexes

#### Users Table
```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_active ON users(is_active);
CREATE INDEX idx_users_verified ON users(is_verified);
CREATE INDEX idx_users_created_at ON users(created_at);
```

#### Products Table
```sql
-- Search and filtering indexes
CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_name ON products USING gin(to_tsvector('english', name));
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_price ON products(price);
CREATE INDEX idx_products_is_active ON products(is_active);
CREATE INDEX idx_products_is_featured ON products(is_featured);

-- Composite indexes for common queries
CREATE INDEX idx_products_category_active ON products(category_id, is_active) WHERE is_active = true;
CREATE INDEX idx_products_active_featured ON products(is_active, is_featured) WHERE is_active = true AND is_featured = true;

-- Inventory indexes
CREATE INDEX idx_products_stock_quantity ON products(stock_quantity) WHERE track_inventory = true;
```

#### Orders Table
```sql
-- Basic filtering indexes
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_order_date ON orders(order_date);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);

-- Composite indexes for common queries
CREATE INDEX idx_orders_customer_status ON orders(customer_id, status);
CREATE INDEX idx_orders_status_date ON orders(status, order_date);
CREATE INDEX idx_orders_currency_total ON orders(currency, total_amount);

-- Shipping and tracking indexes
CREATE INDEX idx_orders_tracking_number ON orders(tracking_number) WHERE tracking_number IS NOT NULL;
```

#### Order Items Table
```sql
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_variant_id ON order_items(variant_id);
```

### Performance Optimization Strategies

1. **Partial Indexes**: Used for filtering active records
   ```sql
   CREATE INDEX idx_active_products ON products(id) WHERE is_active = true;
   ```

2. **Functional Indexes**: Used for case-insensitive searches
   ```sql
   CREATE INDEX idx_customers_email_lower ON customers(lower(email));
   ```

3. **Covering Indexes**: Include frequently accessed columns
   ```sql
   CREATE INDEX idx_orders_covering ON orders(customer_id, status, total_amount);
   ```

## Constraints and Validation

### Check Constraints

#### Financial Validation
```sql
-- Product pricing constraints
ALTER TABLE products ADD CONSTRAINT check_cost_not_higher_than_price
    CHECK (cost <= price OR cost = 0);

-- Order calculation constraints
ALTER TABLE orders ADD CONSTRAINT check_total_calculation
    CHECK (total_amount = subtotal + tax_amount + shipping_amount - discount_amount);
```

#### Business Logic Constraints
```sql
-- Order status progression
ALTER TABLE orders ADD CONSTRAINT check_status_progression
    CHECK (
        (status = 'DRAFT' AND previous_status IS NULL) OR
        (previous_status IS NOT NULL)
    );

-- Inventory quantity constraints
ALTER TABLE product_inventory ADD CONSTRAINT check_inventory_quantities
    CHECK (
        quantity_available >= 0 AND
        quantity_reserved >= 0 AND
        quantity_available >= quantity_reserved
    );
```

### Unique Constraints

```sql
-- Email uniqueness across users and customers
CREATE UNIQUE INDEX idx_unique_emails ON users(email) WHERE is_active = true;
CREATE UNIQUE INDEX idx_unique_customer_emails ON customers(email) WHERE status = 'ACTIVE';

-- SKU uniqueness
CREATE UNIQUE INDEX idx_unique_skus ON products(sku) WHERE is_active = true;
CREATE UNIQUE INDEX idx_unique_variant_skus ON product_variants(sku, product_id) WHERE is_active = true;
```

## Triggers and Automation

### Update Timestamp Triggers

```sql
-- Generic update timestamp function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to all relevant tables
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### Inventory Management Triggers

```sql
-- Update product stock level when inventory changes
CREATE OR REPLACE FUNCTION update_product_stock()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE products
    SET stock_quantity = (
        SELECT COALESCE(SUM(quantity_available), 0)
        FROM product_inventory
        WHERE product_id = NEW.product_id
    )
    WHERE id = NEW.product_id;

    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_update_product_stock
    AFTER INSERT OR UPDATE ON product_inventory
    FOR EACH ROW EXECUTE FUNCTION update_product_stock();
```

### Order Status Triggers

```sql
-- Track previous order status
CREATE OR REPLACE FUNCTION track_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
    NEW.previous_status = OLD.status;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_order_status_change
    BEFORE UPDATE OF status ON orders
    FOR EACH ROW EXECUTE FUNCTION track_order_status_change();
```

## Data Types and Formats

### UUID Generation

- **Primary Keys**: Use `uuid_generate_v4()` for random UUIDs
- **External References**: Use `gen_random_uuid()` for security

### Numeric Types

```sql
-- Financial values (12,2 supports up to 99,999,999,999.99)
DECIMAL(12,2)

-- Quantities (supports large inventory counts)
INTEGER

-- Measurements (supports precision)
DECIMAL(10,3)  -- Weight with 3 decimal places
DECIMAL(8,2)   -- Dimensions with 2 decimal places
```

### Timestamps

All timestamps use `TIMESTAMP WITH TIME ZONE` for global consistency:
```sql
created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
```

### JSON Fields

Flexible data storage using JSONB:
```sql
-- Role permissions
permissions JSONB DEFAULT '{}'

-- Product variant attributes
attributes JSONB DEFAULT '{"color": "red", "size": "L"}'
```

## Migration Strategy

### Migration Files

Migration files follow the naming convention:
```
XXX_description.up.sql   -- Forward migration
XXX_description.down.sql -- Rollback migration
```

### Migration Commands

```bash
# Run all pending migrations
go run cmd/migrate/main.go up

# Run specific migration
go run cmd/migrate/main.go up 001

# Rollback migration
go run cmd/migrate/main.go down 001

# Get migration status
go run cmd/migrate/main.go version
```

### Migration Best Practices

1. **Idempotent Operations**: Use `IF NOT EXISTS` clauses
2. **Rollback Support**: Always provide down migrations
3. **Performance**: Add indexes after data loading for large tables
4. **Validation**: Include data integrity checks
5. **Testing**: Test migrations on staging before production

## Backup and Recovery

### Backup Strategy

#### Full Backups
```bash
# Daily full backup
pg_dump -h localhost -U erpgo -d erp -Fc > backup_$(date +%Y%m%d).dump

# Compressed backup
pg_dump -h localhost -U erpgo -d erp | gzip > backup_$(date +%Y%m%d).sql.gz
```

#### Incremental Backups
```bash
# WAL archive for point-in-time recovery
archive_command = 'cp %p /backup/wal/%f'
```

### Recovery Procedures

#### Full Recovery
```bash
# Restore from backup
pg_restore -h localhost -U erpgo -d erp_new backup_20240108.dump

# Verify data integrity
psql -h localhost -U erpgo -d erp_new -c "SELECT COUNT(*) FROM users;"
```

#### Point-in-Time Recovery
```bash
# Restore to specific time
pg_basebackup -h localhost -D /backup/base -U erpgo -v -P
# Then restore WAL files to target time
```

## Security Considerations

### Database Security

#### User Permissions
```sql
-- Application user with limited privileges
CREATE USER erpgo_app WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE erp TO erpgo_app;
GRANT USAGE ON SCHEMA public TO erpgo_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO erpgo_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO erpgo_app;
```

#### Row Level Security (RLS)
```sql
-- Enable RLS on sensitive tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy for users to see their own data
CREATE POLICY user_data_policy ON users
    FOR ALL
    TO erpgo_app
    USING (id = current_setting('app.current_user_id')::uuid);
```

### Data Encryption

#### Column-Level Encryption
```sql
-- Extension for encryption
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Encrypt sensitive data
INSERT INTO users (email, password_hash)
VALUES (
    pgp_sym_encrypt('user@example.com', 'encryption_key'),
    crypt('password', gen_salt('bf'))
);
```

#### Transparent Data Encryption (TDE)
Configure at database level for encryption at rest.

## Performance Tuning

### Query Optimization

#### Slow Query Analysis
```sql
-- Enable slow query logging
ALTER SYSTEM SET log_min_duration_statement = 1000;
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();

-- Analyze slow queries
SELECT query, mean_time, calls, rows
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
```

#### Index Usage Analysis
```sql
-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Unused indexes
SELECT schemaname, tablename, indexname
FROM pg_stat_user_indexes
WHERE idx_scan = 0;
```

### Configuration Optimization

#### Memory Settings
```sql
-- shared_buffers: 25% of RAM
shared_buffers = 256MB

-- effective_cache_size: 75% of RAM
effective_cache_size = 768MB

-- work_mem: Per query memory
work_mem = 4MB

-- maintenance_work_mem: For maintenance operations
maintenance_work_mem = 64MB
```

#### Connection Settings
```sql
-- max_connections: Based on application needs
max_connections = 200

-- connection pooling
max_pool_size = 20
min_pool_size = 5
```

### Monitoring Queries

#### Database Statistics
```sql
-- Table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Active connections
SELECT
    datname,
    usename,
    application_name,
    state,
    query_start,
    query
FROM pg_stat_activity
WHERE state = 'active';
```

---

**Note**: This documentation represents the current database schema. Always refer to the latest migration files for the most up-to-date schema definition.