-- +goose Up
-- System admin account: admin@marketplace.local / admin123
INSERT INTO users (email, password_hash, full_name, role)
VALUES ('admin@marketplace.local', '$2a$10$NCZo//GZesuCZdRDpb3kE.Uya2mnWx0f5m.eBxGzDCNN5YNof.V.W', 'System Admin', 'admin')
ON CONFLICT DO NOTHING;

-- System analyst account: analyst@marketplace.local / analyst123
INSERT INTO users (email, password_hash, full_name, role)
VALUES ('analyst@marketplace.local', '$2a$10$fRwTo9oql3Jwf.TDPpxpH.RCmcOkjFaOr.eSBJafHdkoCJUpyxH26', 'System Analyst', 'analyst')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM users WHERE email IN ('admin@marketplace.local', 'analyst@marketplace.local');
