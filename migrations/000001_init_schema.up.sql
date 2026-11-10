CREATE TABLE IF NOT EXISTS regions (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ONLINE',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS resource_types (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    is_divisible BOOLEAN NOT NULL DEFAULT true,
    min_amount BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS region_capacities (
    id BIGSERIAL PRIMARY KEY,
    region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE RESTRICT,
    resource_type_id VARCHAR(50) NOT NULL REFERENCES resource_types(id) ON DELETE RESTRICT,
    total_capacity BIGINT NOT NULL,
    allocated_capacity BIGINT NOT NULL DEFAULT 0,
    reserved_capacity BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_region_resource UNIQUE (region_id, resource_type_id),
    CONSTRAINT chk_non_negative_capacities CHECK (
        allocated_capacity >= 0 AND 
        reserved_capacity >= 0 AND 
        total_capacity >= 0
    ),
    CONSTRAINT chk_capacity_limit CHECK (
        allocated_capacity + reserved_capacity <= total_capacity
    )
);

CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150) NOT NULL UNIQUE,
    tier VARCHAR(50) NOT NULL DEFAULT 'STANDARD',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customer_limits (
    id BIGSERIAL PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    resource_type_id VARCHAR(50) NOT NULL REFERENCES resource_types(id) ON DELETE RESTRICT,
    max_total_limit BIGINT NOT NULL,
    min_per_request BIGINT NOT NULL DEFAULT 1,
    max_per_request BIGINT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_customer_resource UNIQUE (customer_id, resource_type_id)
);

CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE RESTRICT,
    idempotency_key VARCHAR(128) NOT NULL,
    status VARCHAR(20) NOT NULL,
    ttl_seconds INT NOT NULL DEFAULT 600,
    expires_at TIMESTAMPTZ NOT NULL,
    committed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_customer_idempotency UNIQUE (customer_id, idempotency_key),
    CONSTRAINT chk_reservation_status CHECK (
        status IN ('RESERVED', 'COMMITTED', 'CANCELLED', 'EXPIRED')
    )
);

CREATE TABLE IF NOT EXISTS reservation_items (
    id BIGSERIAL PRIMARY KEY,
    reservation_id UUID NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    resource_type_id VARCHAR(50) NOT NULL REFERENCES resource_types(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL CHECK (amount > 0),
    CONSTRAINT uq_reservation_resource UNIQUE (reservation_id, resource_type_id)
);

CREATE TABLE IF NOT EXISTS capacity_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    reservation_id UUID REFERENCES reservations(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    region_id VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reservations_status_expires 
    ON reservations(status, expires_at) 
    WHERE status = 'RESERVED';

CREATE INDEX IF NOT EXISTS idx_reservations_customer_history 
    ON reservations(customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reservations_customer_active
    ON reservations(customer_id)
    WHERE status IN ('RESERVED', 'COMMITTED');

CREATE INDEX IF NOT EXISTS idx_region_capacities_region 
    ON region_capacities(region_id);

CREATE INDEX IF NOT EXISTS idx_capacity_audit_logs_res 
    ON capacity_audit_logs(reservation_id);
