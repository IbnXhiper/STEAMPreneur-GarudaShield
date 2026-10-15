package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ahkurdev/garudashield/internal/access"
	"github.com/ahkurdev/garudashield/internal/config"
	"github.com/ahkurdev/garudashield/internal/db"
	"github.com/ahkurdev/garudashield/internal/license"
	"github.com/ahkurdev/garudashield/internal/shield"
	"github.com/ahkurdev/garudashield/internal/tenant"
	"log/slog"
)

// TestEdgeLicenseGateEnforces402 verifies the multi-tenant commercial gate: a
// Host that resolves to a paid tenant with no/expired license is refused with
// 402 Payment Required, while a free tenant or unknown Host passes through.
// Uses the real Postgres store when GS_DATABASE_URL is set; skipped otherwise.
func TestEdgeLicenseGateEnforces402(t *testing.T) {
	dsn := os.Getenv("GS_DATABASE_URL")
	if dsn == "" {
		t.Skip("GS_DATABASE_URL not set")
	}
	pool, err := db.Connect()
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id IN ('paid','free')`)
	_, _ = pool.Exec(ctx, `INSERT INTO tenants (id,name,plan,domain) VALUES ('paid','Paid','pro','paid.example.com') ON CONFLICT (id) DO NOTHING`)
	_, _ = pool.Exec(ctx, `INSERT INTO tenants (id,name,plan,domain) VALUES ('free','Free','free','free.example.com') ON CONFLICT (id) DO NOTHING`)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id IN ('paid','free')`) }()

	ts := tenant.New(pool)
	licSecret := "test-license-secret"

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer up.Close()
	u, _ := url.Parse(up.URL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	gate := access.New(config.Access{Enabled: false}, "")
	br := newBreaker(5*time.Second, 10)
	sh := shield.New(edgeAttackConfig())

	h := makeEdgeHandler(sh, gate, proxy, nil, br, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, config.Proxy{}, ts, nil, licSecret)

	// 1. paid tenant, no license -> 402.
	if rec := serveHost(h, "paid.example.com"); rec.Code != http.StatusPaymentRequired {
		t.Fatalf("paid without license should be 402, got %d", rec.Code)
	}

	// 2. issue a valid license -> 200.
	now := time.Now().Unix()
	key := license.Sign(license.License{
		TenantID: "paid", Plan: license.PlanPro, Expiry: now + 86400, Issued: now,
		Features: []string{"waf"},
	}, licSecret).Key()
	if err := ts.SetLicense(ctx, "paid", key, now+86400); err != nil {
		t.Fatal(err)
	}
	if rec := serveHost(h, "paid.example.com"); rec.Code != http.StatusOK {
		t.Fatalf("paid with valid license should be 200, got %d", rec.Code)
	}

	// 3. free tenant -> 200 (no license needed).
	if rec := serveHost(h, "free.example.com"); rec.Code != http.StatusOK {
		t.Fatalf("free tenant should pass, got %d", rec.Code)
	}

	// 4. unknown host -> pass-through (200 from upstream).
	if rec := serveHost(h, "nope.example.com"); rec.Code != http.StatusOK {
		t.Fatalf("unknown host should pass, got %d", rec.Code)
	}

	// 5. expired license -> 402.
	expiredKey := license.Sign(license.License{
		TenantID: "paid", Plan: license.PlanPro, Expiry: now - 1, Issued: now,
		Features: []string{"waf"},
	}, licSecret).Key()
	if err := ts.SetLicense(ctx, "paid", expiredKey, now-1); err != nil {
		t.Fatal(err)
	}
	if rec := serveHost(h, "paid.example.com"); rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expired license should be 402, got %d", rec.Code)
	}
}

func serveHost(h http.HandlerFunc, host string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
