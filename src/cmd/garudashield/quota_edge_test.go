package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ahkurdev/garudashield/internal/access"
	"github.com/ahkurdev/garudashield/internal/config"
	"github.com/ahkurdev/garudashield/internal/db"
	"github.com/ahkurdev/garudashield/internal/license"
	"github.com/ahkurdev/garudashield/internal/quota"
	"github.com/ahkurdev/garudashield/internal/shield"
	"github.com/ahkurdev/garudashield/internal/tenant"

	"net/http/httputil"
)

// TestEdgeQuotaEnforces429 verifies the commercial quota rail end-to-end on a
// real Postgres counter: a paid tenant with a tiny budget gets 200 while under
// budget and 429 once the budget is exceeded. Skips without GS_DATABASE_URL.
func TestEdgeQuotaEnforces429(t *testing.T) {
	dsn := os.Getenv("GS_DATABASE_URL")
	if dsn == "" {
		t.Skip("GS_DATABASE_URL not set")
	}
	pool, err := db.Connect()
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer pool.Close()
	ctx := context.Background()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM usage_quota WHERE tenant_id='quota-edge'`)
	_, _ = pool.Exec(ctx, `INSERT INTO tenants (id,name,plan,domain) VALUES ('quota-edge','QuotaEdge','pro','quota-edge.example.com') ON CONFLICT (id) DO NOTHING`)
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM usage_quota WHERE tenant_id='quota-edge'`)
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id='quota-edge'`)
	}()

	ts := tenant.New(pool)
	licSecret := "test-license-secret"
	now := time.Now().Unix()
	key := license.Sign(license.License{
		TenantID: "quota-edge", Plan: license.PlanPro, Expiry: now + 86400, Issued: now,
	}, licSecret).Key()
	if err := ts.SetLicense(ctx, "quota-edge", key, now+86400); err != nil {
		t.Fatal(err)
	}

	gate := access.New(config.Access{Enabled: false}, "")
	br := newBreaker(5*time.Second, 10)
	sh := shield.New(edgeAttackConfig())

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer up.Close()
	u, _ := url.Parse(up.URL)
	proxy := httputil.NewSingleHostReverseProxy(u)

	en := quota.NewEnforcer(quota.New(pool), map[string]int64{"pro": 2}, 10)
	h := makeEdgeHandler(sh, gate, proxy, nil, br, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, config.Proxy{}, ts, en, licSecret)

	for i := 0; i < 2; i++ {
		if rec := serveHost(h, "quota-edge.example.com"); rec.Code != http.StatusOK {
			t.Fatalf("request %d under budget = %d, want 200", i, rec.Code)
		}
	}
	if rec := serveHost(h, "quota-edge.example.com"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request over budget = %d, want 429", rec.Code)
	}
	if rec := serveHost(h, "unknown.example.com"); rec.Code != http.StatusOK {
		t.Fatalf("unknown host should bypass quota = %d, want 200", rec.Code)
	}
}
