-- Versioned baseline for the six prerequisite application tables.
-- v3 owns categories.is_trip_category and the trip-related expenses columns.

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    date_of_birth TIMESTAMPTZ,
    password TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    locale TEXT DEFAULT 'es',
    timezone TEXT DEFAULT 'America/Mexico_City',
    currency TEXT DEFAULT 'USD',
    default_account_id BIGINT,
    last_login_at TIMESTAMPTZ,
    login_count BIGINT DEFAULT 0,
    password_reset_token TEXT,
    password_reset_expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_default_account_id ON users(default_account_id);
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token ON users(password_reset_token);

CREATE TABLE IF NOT EXISTS revoked_tokens (
    id BIGSERIAL PRIMARY KEY,
    token_jti TEXT NOT NULL,
    user_id BIGINT NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_revoked_tokens_token_jti ON revoked_tokens(token_jti);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id ON revoked_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);

CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    color TEXT DEFAULT '#007bff',
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    sort_order BIGINT DEFAULT 0,
    user_id BIGINT
);

CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories(deleted_at);
CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);

CREATE TABLE IF NOT EXISTS budgets (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    year BIGINT NOT NULL,
    month BIGINT NOT NULL,
    total_amount NUMERIC(15,2) NOT NULL,
    spent_amount NUMERIC(15,2) DEFAULT 0,
    remaining_amount NUMERIC(15,2),
    is_active BOOLEAN DEFAULT TRUE,
    auto_create_next BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_budgets_deleted_at ON budgets(deleted_at);
CREATE INDEX IF NOT EXISTS idx_budgets_user_id ON budgets(user_id);
CREATE INDEX IF NOT EXISTS idx_budgets_year ON budgets(year);
CREATE INDEX IF NOT EXISTS idx_budgets_month ON budgets(month);

CREATE TABLE IF NOT EXISTS budget_allocations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    budget_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    allocated_amount NUMERIC(15,2) NOT NULL,
    spent_amount NUMERIC(15,2) DEFAULT 0,
    remaining_amount NUMERIC(15,2),
    daily_limit NUMERIC(15,2),
    current_daily_limit NUMERIC(15,2),
    last_calculated_at TIMESTAMPTZ,
    alert_threshold NUMERIC(3,2) DEFAULT 0.8,
    is_over_budget BOOLEAN DEFAULT FALSE,
    CONSTRAINT fk_budgets_allocations FOREIGN KEY (budget_id) REFERENCES budgets(id),
    CONSTRAINT fk_budget_allocations_category FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_budget_allocations_deleted_at ON budget_allocations(deleted_at);
CREATE INDEX IF NOT EXISTS idx_budget_allocations_budget_id ON budget_allocations(budget_id);
CREATE INDEX IF NOT EXISTS idx_budget_allocations_category_id ON budget_allocations(category_id);

CREATE TABLE IF NOT EXISTS expenses (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    budget_id BIGINT,
    category_id BIGINT NOT NULL,
    allocation_id BIGINT,
    amount NUMERIC(15,2) NOT NULL,
    description TEXT NOT NULL,
    date TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    status TEXT DEFAULT 'confirmed',
    location TEXT,
    merchant TEXT,
    reference TEXT,
    raw_data TEXT,
    confidence NUMERIC,
    tags TEXT,
    notes TEXT,
    receipt_url TEXT,
    exchange_rate NUMERIC(10,6) DEFAULT 1,
    currency VARCHAR(3) DEFAULT 'USD',
    triggered_alert BOOLEAN DEFAULT FALSE,
    alert_sent BOOLEAN DEFAULT FALSE,
    CONSTRAINT fk_expenses_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_budgets_expenses FOREIGN KEY (budget_id) REFERENCES budgets(id),
    CONSTRAINT fk_expenses_category FOREIGN KEY (category_id) REFERENCES categories(id),
    CONSTRAINT fk_budget_allocations_expenses FOREIGN KEY (allocation_id) REFERENCES budget_allocations(id)
);

CREATE INDEX IF NOT EXISTS idx_expenses_deleted_at ON expenses(deleted_at);
CREATE INDEX IF NOT EXISTS idx_expenses_user_id ON expenses(user_id);
CREATE INDEX IF NOT EXISTS idx_expenses_budget_id ON expenses(budget_id);
CREATE INDEX IF NOT EXISTS idx_expenses_category_id ON expenses(category_id);
CREATE INDEX IF NOT EXISTS idx_expenses_allocation_id ON expenses(allocation_id);
CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
