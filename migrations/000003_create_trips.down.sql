-- Reverte el modulo de viajes (turismo).

-- Restaurar restricciones NOT NULL en expenses (solo si no quedan filas con NULL)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM expenses WHERE budget_id IS NULL) THEN
        ALTER TABLE expenses ALTER COLUMN budget_id SET NOT NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM expenses WHERE allocation_id IS NULL) THEN
        ALTER TABLE expenses ALTER COLUMN allocation_id SET NOT NULL;
    END IF;
END$$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_expenses_paid_by') THEN
        ALTER TABLE expenses DROP CONSTRAINT fk_expenses_paid_by;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_expenses_trip') THEN
        ALTER TABLE expenses DROP CONSTRAINT fk_expenses_trip;
    END IF;
END$$;

ALTER TABLE expenses DROP COLUMN IF EXISTS paid_by_member_id;
ALTER TABLE expenses DROP COLUMN IF EXISTS trip_id;

DROP INDEX IF EXISTS idx_categories_is_trip_category;
ALTER TABLE categories DROP COLUMN IF EXISTS is_trip_category;

DROP TABLE IF EXISTS trip_itinerary_items;
DROP TABLE IF EXISTS settlements;
DROP TABLE IF EXISTS expense_splits;
DROP TABLE IF EXISTS trip_budget_allocations;
DROP TABLE IF EXISTS trip_invitations;
DROP TABLE IF EXISTS trip_members;
DROP TABLE IF EXISTS trips;
