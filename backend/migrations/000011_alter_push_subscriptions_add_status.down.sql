DROP INDEX IF EXISTS idx_push_subscriptions_is_active;
ALTER TABLE push_subscriptions
    DROP COLUMN IF EXISTS last_failed_at,
    DROP COLUMN IF EXISTS is_active;
