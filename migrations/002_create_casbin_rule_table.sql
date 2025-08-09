-- migrations/002_create_casbin_rule_table.sql

CREATE TABLE IF NOT EXISTS casbin_rule (
    ptype TEXT,
    v0 TEXT,
    v1 TEXT,
    v2 TEXT,
    v3 TEXT,
    v4 TEXT,
    v5 TEXT
);
