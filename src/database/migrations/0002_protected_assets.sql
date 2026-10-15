-- 0002_protected_assets.sql — tenant-owned protected domains (assets).
--
-- The authoritative, idempotent DDL lives in `internal/db` (db.Migrate runs it
-- on every boot). This file is the static-SQL equivalent for ops who manage
-- schema with a migration runner / psql -f. Keep both in sync.
--
-- A row is a protected domain owned by exactly one tenant. A domain becomes
-- ACTIVE at the edge (tenant resolution) only once status='verified', which
-- requires the tenant to prove DNS ownership of the domain (TXT challenge
-- _garudashield.<domain> = "gs-verify=<token>"). Domains are globally unique so
-- two tenants cannot claim the same name.

CREATE TABLE IF NOT EXISTS protected_assets (
  id           BIGSERIAL PRIMARY KEY,
  tenant_id    TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  domain       TEXT NOT NULL,
  origin_host  TEXT NOT NULL DEFAULT '',
  origin_port  INTEGER NOT NULL DEFAULT 0,
  status       TEXT NOT NULL DEFAULT 'pending',
  verify_token TEXT NOT NULL DEFAULT '',
  created_at   BIGINT NOT NULL DEFAULT 0,
  verified_at  BIGINT NOT NULL DEFAULT 0,
  UNIQUE (domain)
);
CREATE INDEX IF NOT EXISTS idx_protected_assets_tenant ON protected_assets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_protected_assets_domain ON protected_assets(domain, status);
