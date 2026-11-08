UPDATE banks SET currency = 'USD';

ALTER TABLE users ADD COLUMN last_currency TEXT NOT NULL DEFAULT 'USD';
