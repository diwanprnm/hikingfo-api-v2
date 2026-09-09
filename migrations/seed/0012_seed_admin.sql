-- +goose Up
-- T097: seeded platform admin for the admin panel (US4).
-- Password is "Admin123!" argon2id-hashed (dev seed only — rotate before prod).

INSERT INTO users (id, email, password_hash, email_verified_at, display_name, role, status, created_at, updated_at)
VALUES (gen_random_uuid(), 'admin@hikingfo.local',
        '$argon2id$v=19$m=65536,t=1,p=4$lXcK0XohYdD1k0Oj6Zvj+Q$Ptwhb/LSUOp36qcEFaoyBnEKlUKzlxONk1dilf7q17c',
        now(), 'Admin Hikingfo', 'admin', 'active', now(), now())
ON CONFLICT (email) DO NOTHING;
