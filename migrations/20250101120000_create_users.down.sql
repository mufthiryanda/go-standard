-- migrations/20250101120000_create_users.down.sql

DROP TRIGGER IF EXISTS set_updated_at ON users;
DROP TABLE IF EXISTS users CASCADE;