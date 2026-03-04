-- seeds/dev/001_seed_users.sql
-- Passwords are bcrypt hashes of "Password123!" (cost 10)

BEGIN;

INSERT INTO users (id, email, password, name, phone, role)
VALUES
    (
        '550e8400-e29b-41d4-a716-446655440001',
        'admin@project.com',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        'Admin',
        '+6281234567890',
        'admin'
    ),
    (
        '550e8400-e29b-41d4-a716-446655440002',
        'user@project.com',
        '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
        'User',
        '+6289876543210',
        'user'
    )
    ON CONFLICT (id) DO NOTHING;

COMMIT;