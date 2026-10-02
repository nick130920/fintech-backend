CREATE TABLE IF NOT EXISTS incomes (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL,
    amount NUMERIC NOT NULL,
    description TEXT NOT NULL,
    source TEXT NOT NULL,
    date TIMESTAMPTZ NOT NULL,
    notes TEXT,
    currency VARCHAR(3) DEFAULT 'USD',
    is_recurring BOOLEAN DEFAULT FALSE,
    frequency VARCHAR(20),
    next_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    recurring_until TIMESTAMPTZ,
    tax_deducted NUMERIC DEFAULT 0,
    net_amount NUMERIC DEFAULT 0,
    CONSTRAINT fk_incomes_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE NO ACTION ON DELETE NO ACTION
);

CREATE INDEX IF NOT EXISTS idx_incomes_deleted_at ON incomes (deleted_at);
CREATE INDEX IF NOT EXISTS idx_incomes_user_id ON incomes (user_id);
