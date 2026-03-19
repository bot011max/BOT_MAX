-- Расширенное шифрование и защита данных

-- Включение расширения для шифрования
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Функция для шифрования данных
CREATE OR REPLACE FUNCTION encrypt_data(data TEXT, key TEXT)
RETURNS BYTEA AS $$
BEGIN
    RETURN pgp_sym_encrypt(data, key);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Функция для дешифрования данных
CREATE OR REPLACE FUNCTION decrypt_data(encrypted BYTEA, key TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN pgp_sym_decrypt(encrypted, key);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Функция для хеширования с солью
CREATE OR REPLACE FUNCTION hash_with_salt(data TEXT, salt TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(digest(data || salt, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Таблица для ключей шифрования (хранятся в HSM в production)
CREATE TABLE IF NOT EXISTS encryption_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_name TEXT UNIQUE NOT NULL,
    key_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_encryption_keys_name ON encryption_keys(key_name);
CREATE INDEX idx_encryption_keys_expires ON encryption_keys(expires_at);

-- Таблица для аномалий безопасности
CREATE TABLE IF NOT EXISTS security_anomalies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    anomaly_type TEXT NOT NULL,
    ip_address INET,
    user_id UUID REFERENCES users(id),
    details JSONB,
    score FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_security_anomalies_created ON security_anomalies(created_at);
CREATE INDEX idx_security_anomalies_ip ON security_anomalies(ip_address);
