#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "== build engine =="
( cd "$ROOT" && go build -o bin/gs ./cmd/garudashield )

echo "== start engine (background) =="
# Commercial control plane (PostgreSQL) env. Defaults target a local Postgres;
# override in your shell beforehand. Without GS_DATABASE_URL the engine starts,
# but /api/v1/* (tenant auth, license, members, audit) is not registered.
export GS_DATABASE_URL="${GS_DATABASE_URL:-postgres://garuda:GarudaShield2026Pg@127.0.0.1:5432/garudashield?sslmode=disable}"
export GS_SESSION_SECRET="${GS_SESSION_SECRET:-gsec-session-9f2c1a7b4e6d8f0a1b2c3d4e5f607182}"
export GS_LICENSE_SECRET="${GS_LICENSE_SECRET:-gsec-license-A1B2C3D4E5F60718293a4b5c6d7e8f90}"
# GS_ADMIN_SECRET gates operator endpoints; keep it identical to GS_API_KEY.
export GS_ADMIN_SECRET="${GS_ADMIN_SECRET:-855d71cafc1438088ef25ac8f40e1028ab80a943685503a5e03a2c8ad741087a}"
"$ROOT/bin/gs" &
ENGINE_PID=$!

cleanup() { kill "$ENGINE_PID" 2>/dev/null || true; }
trap cleanup EXIT

echo "== start web (npm run dev) — Ctrl+C untuk berhenti =="
cd "$ROOT/dashboard"
export GS_API_URL="${GS_API_URL:-http://localhost:8080}"
export GS_API_KEY="${GS_API_KEY:-}"
npm run dev
