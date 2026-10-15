-- 0001_commercial.sql — deployment artifact for the commercial control plane.
--
-- The authoritative, idempotent DDL lives in `internal/db` (db.Migrate runs it
-- on every boot against DATABASE_URL/GS_DATABASE_URL, including `IF NOT EXISTS`
-- guards). This file is the generated equivalent for ops who prefer managing
-- schema as static SQL (e.g. psql -f or a migration runner). Keep the two in
-- sync when the schema changes.

CREATE TABLE IF NOT EXISTS tenants (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  plan          TEXT NOT NULL DEFAULT 'free',
  domain        TEXT UNIQUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  license_key   TEXT,
  license_exp   BIGINT NOT NULL DEFAULT 0,
  status        TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS users (
  id              BIGSERIAL PRIMARY KEY,
  tenant_id       TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email           TEXT NOT NULL,
  password_hash   TEXT NOT NULL,
  role            TEXT NOT NULL DEFAULT 'owner',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  session_version BIGINT NOT NULL DEFAULT 0,
  UNIQUE (tenant_id, email)
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);

CREATE TABLE IF NOT EXISTS subscriptions (
  tenant_id          TEXT PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  status             TEXT NOT NULL DEFAULT 'active',
  provider           TEXT,
  plan               TEXT NOT NULL,
  current_period_end BIGINT NOT NULL DEFAULT 0,
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
  id            BIGSERIAL PRIMARY KEY,
  tenant_id     TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  prefix        TEXT NOT NULL,
  key_hash      TEXT NOT NULL,
  scopes        TEXT NOT NULL DEFAULT 'read',
  created_at    BIGINT NOT NULL DEFAULT 0,
  last_used_at  BIGINT NOT NULL DEFAULT 0,
  expires_at    BIGINT NOT NULL DEFAULT 0,
  revoked_at    BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);

CREATE TABLE IF NOT EXISTS audit_log (
  id         BIGSERIAL PRIMARY KEY,
  tenant_id  TEXT,
  actor      TEXT,
  action     TEXT NOT NULL,
  detail     TEXT,
  ip         TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_log(tenant_id);

-- Per-plan daily request budgets, shared across engine instances via PG.
CREATE TABLE IF NOT EXISTS usage_quota (
  tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  period    TEXT NOT NULL,
  requests  BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, period)
);

-- Second factor (TOTP) and single-use recovery codes + password reset tokens.
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS mfa_recovery (
  code_hash TEXT PRIMARY KEY,
  user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  used_at   BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_mfa_recovery_user ON mfa_recovery(user_id);

CREATE TABLE IF NOT EXISTS password_resets (
  token_hash TEXT PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at BIGINT NOT NULL DEFAULT 0,
  expires_at BIGINT NOT NULL,
  used_at    BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_password_resets_user ON password_resets(user_id);