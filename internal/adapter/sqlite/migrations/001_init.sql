CREATE TABLE users (
    telegram_id INTEGER PRIMARY KEY,
    username TEXT,
    last_activity_at TEXT NOT NULL,
    plan TEXT NOT NULL DEFAULT 'trial',
    trial_ends_at TEXT NOT NULL,
    discount_percent INTEGER,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (discount_percent IS NULL OR (discount_percent >= 0 AND discount_percent <= 100))
);

CREATE TABLE banks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    balance_cents INTEGER NOT NULL DEFAULT 0,
    include_in_total INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (user_id, name_normalized),
    CHECK (include_in_total IN (0, 1))
);
