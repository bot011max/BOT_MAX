-- Таблица для связи пользователей с Telegram
CREATE TABLE telegram_users (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    telegram_id BIGINT UNIQUE NOT NULL,
    chat_id BIGINT NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10),
    is_active BOOLEAN DEFAULT true,
    auth_token VARCHAR(50),
    token_expires TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_telegram_users_user_id ON telegram_users(user_id);
CREATE INDEX idx_telegram_users_telegram_id ON telegram_users(telegram_id);
CREATE INDEX idx_telegram_users_auth_token ON telegram_users(auth_token) WHERE auth_token IS NOT NULL;

-- Таблица для сессий диалогов
CREATE TABLE telegram_sessions (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL REFERENCES telegram_users(telegram_id) ON DELETE CASCADE,
    state VARCHAR(50) DEFAULT 'none',
    temp_data TEXT,
    last_message_id INTEGER,
    last_command VARCHAR(255),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_telegram_sessions_telegram_id ON telegram_sessions(telegram_id);
CREATE INDEX idx_telegram_sessions_state ON telegram_sessions(state);

-- Таблица для напоминаний
CREATE TABLE telegram_reminders (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    telegram_id BIGINT REFERENCES telegram_users(telegram_id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    scheduled_for TIMESTAMP NOT NULL,
    sent_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_telegram_reminders_user_id ON telegram_reminders(user_id);
CREATE INDEX idx_telegram_reminders_scheduled ON telegram_reminders(scheduled_for) WHERE status = 'pending';
