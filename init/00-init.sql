CREATE TABLE users (
    uuid UUID PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    password_hash CHAR(60) NOT NULL,
    email VARCHAR(255) UNIQUE
);

CREATE TABLE ip (
    id BIGSERIAL PRIMARY KEY,
    version SMALLINT NOT NULL,
    ip_range CIDR NOT NULL UNIQUE,
    is_whitelist BOOLEAN NOT NULL,
    create_at TIMESTAMPTZ DEFAULT NOW(),
    update_at TIMESTAMPTZ DEFAULT NOW(),
    note TEXT
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_ip_update_at BEFORE UPDATE ON ip
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO ip (version, ip_range, is_whitelist, note)
VALUES
    (4, '127.0.0.1/32', true, 'localhost'),
    (4, '10.0.0.0/8', true, 'allow private ip'),
    (4, '192.168.0.0/16', true, 'allow private ip')