ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT 'en';
ALTER TABLE users ADD COLUMN referred_by INTEGER;

ALTER TABLE banks ADD COLUMN currency TEXT NOT NULL DEFAULT 'USD';

CREATE TABLE operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    bank_id INTEGER REFERENCES banks (id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    balance_after_cents INTEGER,
    meta TEXT,
    created_at TEXT NOT NULL,
    CHECK (type IN ('add', 'spend', 'set', 'delete', 'transfer'))
);
