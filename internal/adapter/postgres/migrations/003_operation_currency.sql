ALTER TABLE operations DROP CONSTRAINT operations_type_check;

ALTER TABLE operations ADD CONSTRAINT operations_check
    CHECK (type IN ('add', 'spend', 'set', 'delete', 'transfer', 'rename', 'currency'));
