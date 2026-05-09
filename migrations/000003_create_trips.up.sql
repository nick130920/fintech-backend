-- Modulo de viajes (turismo): tablas y columnas asociadas para gestionar
-- presupuestos estimados y gastos compartidos con grupos.

-- Trips
CREATE TABLE IF NOT EXISTS trips (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    owner_user_id BIGINT NOT NULL,
    name VARCHAR(120) NOT NULL,
    destination VARCHAR(200) NOT NULL,
    country_code VARCHAR(3),
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    primary_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    base_currency_at_creation VARCHAR(3) DEFAULT 'USD',
    fx_rate_at_creation NUMERIC(18,8) DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'planning',
    cover_image_url TEXT,
    estimated_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    spent_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    notes TEXT,
    CONSTRAINT fk_trips_owner FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_trips_owner_user_id ON trips(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips(status);
CREATE INDEX IF NOT EXISTS idx_trips_start_date ON trips(start_date);
CREATE INDEX IF NOT EXISTS idx_trips_end_date ON trips(end_date);
CREATE INDEX IF NOT EXISTS idx_trips_deleted_at ON trips(deleted_at);

-- Trip members (usuarios reales o invitados fantasma)
CREATE TABLE IF NOT EXISTS trip_members (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    trip_id BIGINT NOT NULL,
    user_id BIGINT NULL,
    display_name VARCHAR(120) NOT NULL,
    email VARCHAR(150),
    avatar_url TEXT,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    is_ghost BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_trip_members_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    CONSTRAINT fk_trip_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_trip_members_trip_id ON trip_members(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_members_user_id ON trip_members(user_id);
CREATE INDEX IF NOT EXISTS idx_trip_members_deleted_at ON trip_members(deleted_at);
CREATE INDEX IF NOT EXISTS idx_trip_members_is_ghost ON trip_members(is_ghost);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_trip_members_trip_user
    ON trip_members(trip_id, user_id) WHERE user_id IS NOT NULL AND deleted_at IS NULL;

-- Trip invitations
CREATE TABLE IF NOT EXISTS trip_invitations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    trip_id BIGINT NOT NULL,
    token VARCHAR(512) NOT NULL,
    email VARCHAR(150),
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    created_by_user_id BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    accepted_by_user_id BIGINT NULL,
    CONSTRAINT fk_trip_invitations_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    CONSTRAINT fk_trip_invitations_created_by FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_trip_invitations_accepted_by FOREIGN KEY (accepted_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_trip_invitations_token ON trip_invitations(token);
CREATE INDEX IF NOT EXISTS idx_trip_invitations_trip_id ON trip_invitations(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_invitations_email ON trip_invitations(email);
CREATE INDEX IF NOT EXISTS idx_trip_invitations_expires_at ON trip_invitations(expires_at);

-- Trip budget allocations (estimaciones por categoría)
CREATE TABLE IF NOT EXISTS trip_budget_allocations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    trip_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    estimated_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    spent_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    notes TEXT,
    CONSTRAINT fk_trip_alloc_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    CONSTRAINT fk_trip_alloc_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_trip_alloc_trip_id ON trip_budget_allocations(trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_alloc_category_id ON trip_budget_allocations(category_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_trip_alloc_trip_category
    ON trip_budget_allocations(trip_id, category_id) WHERE deleted_at IS NULL;

-- Expense splits (división del gasto entre miembros)
CREATE TABLE IF NOT EXISTS expense_splits (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    expense_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    share_type VARCHAR(20) NOT NULL DEFAULT 'equal',
    share_value NUMERIC(15,4) NOT NULL DEFAULT 0,
    share_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    paid_at TIMESTAMPTZ NULL,
    CONSTRAINT fk_expense_splits_expense FOREIGN KEY (expense_id) REFERENCES expenses(id) ON DELETE CASCADE,
    CONSTRAINT fk_expense_splits_member FOREIGN KEY (member_id) REFERENCES trip_members(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_expense_splits_expense_id ON expense_splits(expense_id);
CREATE INDEX IF NOT EXISTS idx_expense_splits_member_id ON expense_splits(member_id);
CREATE INDEX IF NOT EXISTS idx_expense_splits_is_paid ON expense_splits(is_paid);

-- Settlements (pagos entre miembros)
CREATE TABLE IF NOT EXISTS settlements (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    trip_id BIGINT NOT NULL,
    from_member_id BIGINT NOT NULL,
    to_member_id BIGINT NOT NULL,
    recorded_by_user BIGINT NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    fx_rate NUMERIC(18,8) NOT NULL DEFAULT 1,
    paid_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    CONSTRAINT fk_settlements_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    CONSTRAINT fk_settlements_from FOREIGN KEY (from_member_id) REFERENCES trip_members(id) ON DELETE CASCADE,
    CONSTRAINT fk_settlements_to FOREIGN KEY (to_member_id) REFERENCES trip_members(id) ON DELETE CASCADE,
    CONSTRAINT fk_settlements_user FOREIGN KEY (recorded_by_user) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_settlements_trip_id ON settlements(trip_id);
CREATE INDEX IF NOT EXISTS idx_settlements_from ON settlements(from_member_id);
CREATE INDEX IF NOT EXISTS idx_settlements_to ON settlements(to_member_id);

-- Trip itinerary items
CREATE TABLE IF NOT EXISTS trip_itinerary_items (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    trip_id BIGINT NOT NULL,
    day TIMESTAMPTZ NOT NULL,
    "time" VARCHAR(5),
    type VARCHAR(20) NOT NULL DEFAULT 'activity',
    title VARCHAR(200) NOT NULL,
    description TEXT,
    location VARCHAR(200),
    estimated_cost NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    expense_id BIGINT NULL,
    CONSTRAINT fk_itinerary_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
    CONSTRAINT fk_itinerary_expense FOREIGN KEY (expense_id) REFERENCES expenses(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_itinerary_trip_id ON trip_itinerary_items(trip_id);
CREATE INDEX IF NOT EXISTS idx_itinerary_day ON trip_itinerary_items(day);
CREATE INDEX IF NOT EXISTS idx_itinerary_expense_id ON trip_itinerary_items(expense_id);

-- Ampliar expenses con la vinculación opcional a viajes
ALTER TABLE expenses
    ADD COLUMN IF NOT EXISTS trip_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS paid_by_member_id BIGINT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_expenses_trip'
    ) THEN
        ALTER TABLE expenses
            ADD CONSTRAINT fk_expenses_trip FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_expenses_paid_by'
    ) THEN
        ALTER TABLE expenses
            ADD CONSTRAINT fk_expenses_paid_by FOREIGN KEY (paid_by_member_id) REFERENCES trip_members(id) ON DELETE SET NULL;
    END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_expenses_trip_id ON expenses(trip_id);
CREATE INDEX IF NOT EXISTS idx_expenses_paid_by_member_id ON expenses(paid_by_member_id);

-- Flag para identificar categorías de viaje
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS is_trip_category BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_categories_is_trip_category ON categories(is_trip_category);

-- Permitir gastos puramente del viaje sin presupuesto mensual asociado
ALTER TABLE expenses ALTER COLUMN budget_id DROP NOT NULL;
ALTER TABLE expenses ALTER COLUMN allocation_id DROP NOT NULL;
