-- Disposable/test rollback only. This is not a production rollback procedure.
-- Drop the v1 prerequisite tables in FK-safe reverse dependency order.

DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS budget_allocations;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS revoked_tokens;
DROP TABLE IF EXISTS users;
