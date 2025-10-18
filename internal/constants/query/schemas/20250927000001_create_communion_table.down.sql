-- Drop communion table and related objects
DROP TRIGGER IF EXISTS trigger_communion_updated_at ON communion;
DROP FUNCTION IF EXISTS update_communion_updated_at();
DROP TABLE IF EXISTS communion;
