CREATE TABLE operations_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    bank_id INTEGER REFERENCES banks (id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    balance_after_cents INTEGER,
    meta TEXT,
    created_at TEXT NOT NULL,
    CHECK (type IN ('add', 'spend', 'set', 'delete', 'transfer', 'rename'))
);

INSERT INTO operations_new (
    id, user_id, bank_id, type, amount_cents, balance_after_cents, meta, created_at
)
SELECT
    id, user_id, bank_id, type, amount_cents, balance_after_cents, meta, created_at
FROM operations;

DROP TABLE operations;

ALTER TABLE operations_new RENAME TO operations;
