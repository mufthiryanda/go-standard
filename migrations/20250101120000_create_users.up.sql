-- migrations/20250101120000_create_users.up.sql

CREATE TABLE IF NOT EXISTS users (
                                     id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255)    NOT NULL,
    password    VARCHAR(255)    NOT NULL,
    name        VARCHAR(100)    NOT NULL,
    phone       VARCHAR(20),
    role        VARCHAR(50)     NOT NULL DEFAULT 'user',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
    );

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email
    ON users (email)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users (deleted_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_updated_at
    ON users (updated_at);

CREATE INDEX IF NOT EXISTS idx_users_role
    ON users (role);

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE users IS 'Application user accounts';
COMMENT ON COLUMN users.password IS 'bcrypt-hashed password, never returned to client';
COMMENT ON COLUMN users.role IS 'Authorization role: user, admin';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp, NULL means active';