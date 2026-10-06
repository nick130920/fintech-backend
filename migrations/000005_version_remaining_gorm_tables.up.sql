CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    balance NUMERIC(15,2) DEFAULT 0,
    initial_balance NUMERIC(15,2) DEFAULT 0,
    credit_limit NUMERIC(15,2) DEFAULT 0,
    bank_name TEXT,
    account_number TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    currency VARCHAR(3) DEFAULT 'USD',
    color TEXT DEFAULT '#007bff',
    icon TEXT,
    low_balance_alert BOOLEAN DEFAULT FALSE,
    low_balance_limit NUMERIC(15,2) DEFAULT 0
);

CREATE INDEX idx_accounts_deleted_at ON accounts (deleted_at);
CREATE INDEX idx_accounts_user_id ON accounts (user_id);

CREATE TABLE bank_accounts (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    bank_name TEXT NOT NULL,
    bank_code TEXT,
    branch_code TEXT,
    branch_name TEXT,
    account_number TEXT,
    account_number_mask TEXT NOT NULL,
    account_alias TEXT NOT NULL,
    type TEXT NOT NULL,
    color TEXT DEFAULT '#007bff',
    icon TEXT DEFAULT 'credit_card',
    is_active BOOLEAN DEFAULT TRUE,
    is_notification_enabled BOOLEAN DEFAULT TRUE,
    currency VARCHAR(3) DEFAULT 'USD',
    last_balance NUMERIC(15,2),
    last_balance_update TIMESTAMPTZ,
    notification_phone TEXT,
    notification_email TEXT,
    min_amount_to_notify NUMERIC(15,2) DEFAULT 0,
    notes TEXT,
    external_id TEXT,
    imported_from TEXT
);

CREATE INDEX idx_bank_accounts_deleted_at ON bank_accounts (deleted_at);
CREATE INDEX idx_bank_accounts_user_id ON bank_accounts (user_id);

CREATE TABLE bank_notification_patterns (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    bank_account_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    channel TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    message_pattern TEXT,
    example_message TEXT,
    keywords_trigger TEXT,
    keywords_exclude TEXT,
    amount_regex TEXT,
    date_regex TEXT,
    description_regex TEXT,
    merchant_regex TEXT,
    requires_validation BOOLEAN DEFAULT TRUE,
    confidence_threshold NUMERIC(3,2) DEFAULT 0.8,
    auto_approve BOOLEAN DEFAULT FALSE,
    match_count BIGINT DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    success_rate NUMERIC(5,2) DEFAULT 0,
    last_matched_at TIMESTAMPTZ,
    priority BIGINT DEFAULT 100,
    is_default BOOLEAN DEFAULT FALSE,
    tags TEXT,
    metadata TEXT
);

CREATE INDEX idx_bank_notification_patterns_deleted_at ON bank_notification_patterns (deleted_at);
CREATE INDEX idx_bank_notification_patterns_user_id ON bank_notification_patterns (user_id);
CREATE INDEX idx_bank_notification_patterns_bank_account_id ON bank_notification_patterns (bank_account_id);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    bank_account_id BIGINT,
    to_account_id BIGINT,
    type TEXT NOT NULL,
    status TEXT DEFAULT 'completed',
    amount NUMERIC(15,2) NOT NULL,
    description TEXT NOT NULL,
    category_id BIGINT,
    category_name TEXT,
    tags TEXT,
    transaction_date TIMESTAMPTZ NOT NULL,
    location TEXT,
    reference TEXT,
    notes TEXT,
    recurring BOOLEAN DEFAULT FALSE,
    recurring_id BIGINT,
    currency TEXT DEFAULT 'USD',
    exchange_rate NUMERIC(10,6) DEFAULT 1,
    source TEXT DEFAULT 'manual',
    validation_status TEXT DEFAULT 'auto',
    raw_notification TEXT,
    ai_confidence NUMERIC(3,2) DEFAULT 0,
    pattern_id BIGINT,
    imported_from TEXT,
    external_id TEXT
);

CREATE INDEX idx_transactions_deleted_at ON transactions (deleted_at);
CREATE INDEX idx_transactions_user_id ON transactions (user_id);
CREATE INDEX idx_transactions_account_id ON transactions (account_id);
CREATE INDEX idx_transactions_bank_account_id ON transactions (bank_account_id);
CREATE INDEX idx_transactions_to_account_id ON transactions (to_account_id);
CREATE INDEX idx_transactions_category_id ON transactions (category_id);
CREATE INDEX idx_transactions_transaction_date ON transactions (transaction_date);
CREATE INDEX idx_transactions_recurring_id ON transactions (recurring_id);
CREATE INDEX idx_transactions_pattern_id ON transactions (pattern_id);

CREATE TABLE budget_suggestion_slug_stats (
    id BIGSERIAL PRIMARY KEY,
    stat_date DATE NOT NULL,
    category_slug VARCHAR(32) NOT NULL,
    hit_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_budget_slug_stat ON budget_suggestion_slug_stats (stat_date, category_slug);

CREATE TABLE budget_suggestion_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL,
    messages_json TEXT NOT NULL,
    result_json TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_budget_suggestion_jobs_user_id ON budget_suggestion_jobs (user_id);
CREATE INDEX idx_budget_suggestion_jobs_status ON budget_suggestion_jobs (status);

CREATE TABLE pending_notifications (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    raw_message TEXT NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'sms',
    phone VARCHAR(30),
    received_at TIMESTAMPTZ,
    attempts BIGINT DEFAULT 0,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending'
);

CREATE INDEX idx_pending_notifications_deleted_at ON pending_notifications (deleted_at);
CREATE INDEX idx_pending_notifications_user_id ON pending_notifications (user_id);
CREATE INDEX idx_pending_notifications_received_at ON pending_notifications (received_at);
CREATE INDEX idx_pending_notifications_status ON pending_notifications (status);

CREATE TABLE user_email_connections (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    provider VARCHAR(32) NOT NULL,
    email_address VARCHAR(255) NOT NULL,
    refresh_token_enc TEXT,
    access_token_enc TEXT,
    access_expires_at TIMESTAMPTZ,
    last_history_id VARCHAR(64),
    last_synced_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_user_email_connections_deleted_at ON user_email_connections (deleted_at);
CREATE UNIQUE INDEX idx_user_email_provider ON user_email_connections (user_id, provider);

CREATE TABLE processed_email_messages (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    provider VARCHAR(32) NOT NULL,
    provider_message_id VARCHAR(128) NOT NULL
);

CREATE UNIQUE INDEX idx_proc_email_dedupe ON processed_email_messages (user_id, provider, provider_message_id);
