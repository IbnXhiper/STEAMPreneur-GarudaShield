package main

import (
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ahkurdev/garudashield/internal/access"
	"github.com/ahkurdev/garudashield/internal/config"
	"github.com/ahkurdev/garudashield/internal/shield"
)

func u(q string) string {
	return "/?" + url.Values{"q": {q}}.Encode()
}

func newTestEdge(t *testing.T, sh *shield.Shield) http.HandlerFunc {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	t.Cleanup(upstream.Close)
	u, _ := url.Parse(upstream.URL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	gate := access.New(config.Access{Enabled: false}, "")
	br := newBreaker(5*time.Second, 10)
	return makeEdgeHandler(sh, gate, proxy, nil, br, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, config.Proxy{}, nil, nil, "")
}

func edgeAttackConfig() config.Config {
	return config.Config{
		Version: "3.0.0",
		Shield:  config.Shield{ModeName: "normal", BudgetMs: 5},
		Defense: config.Defense{
			WAF: config.WAFConfig{
				Enabled:          true,
				SQLi:             true,
				XSS:              true,
				CommandInjection: true,
				PathTraversal:    true,
				LFI:              true,
				RFI:              true,
				Shellshock:       true,
				Scanners:         true,
			},
		},
	}
}

func TestEdgeBlocksSQLiAndPassesBenign(t *testing.T) {
	sh := shield.New(edgeAttackConfig())
	h := newTestEdge(t, sh)

	req := httptest.NewRequest("GET", u("1' UNION SELECT * FROM users--"), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sqli should be 403, got %d", rec.Code)
	}

	req2 := httptest.NewRequest("GET", u("cats"), nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("benign should proxy 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "upstream-ok") {
		t.Fatalf("benign did not reach upstream: %q", rec2.Body.String())
	}
}

func TestEdgeBlocksXSSViaHeader(t *testing.T) {
	sh := shield.New(edgeAttackConfig())
	h := newTestEdge(t, sh)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "() { :; }; /bin/bash -c 'cat /etc/passwd'")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("shellshock header should be 403, got %d", rec.Code)
	}
}

// TestEdgeFastFailIsolatedPerClientIP guards against a WAF fast-fail cache that
// keys on the connection RemoteAddr instead of the resolved client IP. When the
// engine runs behind a proxy, every request shares the proxy's RemoteAddr, so a
// single clean request would otherwise poison the cache and let attacks through.
func TestEdgeFastFailIsolatedPerClientIP(t *testing.T) {
	sh := shield.New(edgeAttackConfig())
	h := newTestEdge(t, sh)

	// clean request from one client IP (connection IP distinguishes clients)
	b := httptest.NewRequest("GET", "/", nil)
	b.RemoteAddr = "192.0.2.1:1111"
	rb := httptest.NewRecorder()
	h.ServeHTTP(rb, b)
	if rb.Code != http.StatusOK {
		t.Fatalf("benign should be 200, got %d", rb.Code)
	}

	// attack from a DIFFERENT client IP must still be hard-blocked
	a := httptest.NewRequest("GET", u("1' UNION SELECT * FROM users--"), nil)
	a.RemoteAddr = "192.0.2.2:2222"
	ra := httptest.NewRecorder()
	h.ServeHTTP(ra, a)
	if ra.Code != http.StatusForbidden {
		t.Fatalf("sqli from a different client IP should be 403, got %d", ra.Code)
	}
}

// TestEdgeWAFModuleToggle proves the WAF produces a real block when enabled and
// lets the request through when the SQLi module is disabled (v2 parity of the
// module toggle), exercising the full shield.Evaluate pipeline end-to-end.
func TestEdgeWAFModuleToggle(t *testing.T) {
	// WAF enabled -> SQLi is hard-blocked (403).
	shOn := shield.New(edgeAttackConfig())
	defer shOn.Close()
	hOn := newTestEdge(t, shOn)
	reqOn := httptest.NewRequest("GET", u("1' UNION SELECT * FROM users--"), nil)
	recOn := httptest.NewRecorder()
	hOn.ServeHTTP(recOn, reqOn)
	if recOn.Code != http.StatusForbidden {
		t.Fatalf("sqli with WAF on should be 403, got %d", recOn.Code)
	}

	// WAF disabled -> SQLi passes (200).
	off := edgeAttackConfig()
	off.Defense.WAF.Enabled = false
	off.Defense.WAF.SQLi = false
	shOff := shield.New(off)
	defer shOff.Close()
	hOff := newTestEdge(t, shOff)
	reqOff := httptest.NewRequest("GET", u("1' UNION SELECT * FROM users--"), nil)
	recOff := httptest.NewRecorder()
	hOff.ServeHTTP(recOff, reqOff)
	if recOff.Code != http.StatusOK {
		t.Fatalf("sqli with WAF off should be 200, got %d", recOff.Code)
	}
}

// TestClientIPTrustProxy verifies that proxy headers are only honored when
// TrustProxy is enabled AND the connection originates from a trusted proxy.
// This prevents a client from spoofing its IP via X-Forwarded-For / X-Real-IP.
func TestClientIPTrustProxy(t *testing.T) {
	base := func() *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "203.0.113.7:55555"
		return r
	}
	rp := func(p config.Proxy) string { return clientIP(base(), p) }

	// 1. TrustProxy disabled: header ignored, connection IP wins.
	if got := rp(config.Proxy{}); got != "203.0.113.7" {
		t.Fatalf("trustProxy=false should ignore XFF, got %q", got)
	}

	// 2. TrustProxy enabled but connection not a trusted proxy: still ignored.
	if got := rp(config.Proxy{TrustProxy: true, ForwardedForHeader: "X-Forwarded-For"}); got != "203.0.113.7" {
		t.Fatalf("untrusted connection should be used, got %q", got)
	}

	// 3. Trusted connection + X-Forwarded-For: first hop is returned.
	trusted := config.Proxy{
		TrustProxy:         true,
		ForwardedForHeader: "X-Forwarded-For",
		TrustedProxies:     []string{"203.0.113.0/24"},
	}
	r3 := base()
	r3.Header.Set("X-Forwarded-For", "198.51.100.23, 203.0.113.7")
	if got := clientIP(r3, trusted); got != "198.51.100.23" {
		t.Fatalf("trusted proxy should return first XFF hop, got %q", got)
	}

	// 4. X-Real-IP is preferred over X-Forwarded-For when trusted.
	trustedReal := config.Proxy{
		TrustProxy:     true,
		RealIPHeader:   "X-Real-IP",
		TrustedProxies: []string{"203.0.113.7"},
	}
	r4 := base()
	r4.Header.Set("X-Real-IP", "198.51.100.99")
	r4.Header.Set("X-Forwarded-For", "198.51.100.23")
	if got := clientIP(r4, trustedReal); got != "198.51.100.99" {
		t.Fatalf("X-Real-IP should win when trusted, got %q", got)
	}

	// 5. Untrusted X-Forwarded-For must NOT leak into the resolved IP.
	r5 := base()
	r5.Header.Set("X-Forwarded-For", "198.51.100.23")
	if got := clientIP(r5, config.Proxy{TrustProxy: true}); got != "203.0.113.7" {
		t.Fatalf("spoofed XFF from untrusted conn must be ignored, got %q", got)
	}
}

// TestEdgeMultiSiteRouting verifies per-subdomain proxy selection: each Host
// is forwarded to its configured site upstream, an unknown Host falls back to
// the global upstream (nil here -> pass-through JSON), and matching is
// case-insensitive and port-agnostic (v2 _hostKey behavior).
func TestEdgeMultiSiteRouting(t *testing.T) {
	sh := shield.New(edgeAttackConfig())
	gate := access.New(config.Access{Enabled: false}, "")
	br := newBreaker(5*time.Second, 10)

	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("site-a"))
	}))
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("site-b"))
	}))
	t.Cleanup(a.Close)
	t.Cleanup(b.Close)

	siteProxies := map[string]*httputil.ReverseProxy{}
	ua, _ := url.Parse(a.URL)
	siteProxies["a.example.com"] = httputil.NewSingleHostReverseProxy(ua)
	ub, _ := url.Parse(b.URL)
	siteProxies["b.example.com"] = httputil.NewSingleHostReverseProxy(ub)

	h := makeEdgeHandler(sh, gate, nil, siteProxies, br, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, config.Proxy{}, nil, nil, "")

	reqTo := func(host string) string {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = host
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return fmt.Sprintf("%d:%s", rec.Code, rec.Body.String())
	}

	if got := reqTo("a.example.com:80"); !strings.Contains(got, "site-a") {
		t.Fatalf("a.example.com should hit site-a, got %q", got)
	}
	if got := reqTo("B.Example.COM"); !strings.Contains(got, "site-b") {
		t.Fatalf("B.Example.COM should hit site-b (case-insensitive), got %q", got)
	}
	if got := reqTo("unknown.example.com"); !strings.Contains(got, "pass") {
		t.Fatalf("unknown host with no global upstream should pass, got %q", got)
	}
}

func TestTLSConfigForSelfSigned(t *testing.T) {
	if got := tlsConfigFor(config.TLSConfig{}); got != nil {
		t.Fatal("empty TLS config must yield nil")
	}
	tc := tlsConfigFor(config.TLSConfig{Enabled: true, AutoSelfSigned: true})
	if tc == nil || len(tc.Certificates) != 1 {
		t.Fatal("autoSelfSigned must yield a tls.Config with one cert")
	}
	leaf, err := x509.ParseCertificate(tc.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	foundLocalhost := false
	for _, n := range leaf.DNSNames {
		if n == "localhost" {
			foundLocalhost = true
		}
	}
	if !foundLocalhost {
		t.Fatalf("cert DNSNames = %v, want localhost", leaf.DNSNames)
	}
	if time.Until(leaf.NotAfter) < 80*24*time.Hour {
		t.Fatalf("cert validity too short: %v", leaf.NotAfter)
	}
}

func newTestEdgeWithGate(t *testing.T, sh *shield.Shield, gate *access.Gate) http.HandlerFunc {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	t.Cleanup(upstream.Close)
	u, _ := url.Parse(upstream.URL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	br := newBreaker(5*time.Second, 10)
	return makeEdgeHandler(sh, gate, proxy, nil, br, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, config.Proxy{}, nil, nil, "")
}

// TestEdgeSoftReputationBlockRoutesToGate is the real-user false-positive
// guard: an automated reputation block must NOT hard-403 a human. An unverified
// client is handed the proof-of-work gate, and a client holding a valid access
// cookie (i.e. already passed verification) is waved straight through.
func TestEdgeSoftReputationBlockRoutesToGate(t *testing.T) {
	sh := shield.New(edgeAttackConfig())
	gate := access.New(config.Access{
		Enabled:    true,
		ProtectAll: true,
		CookieName: "gs_access",
		QueryParam: "gs_t",
		TokenTTLMs: 3600000,
	}, "test-secret")
	h := newTestEdgeWithGate(t, sh, gate)

	const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36"
	ip := "192.0.2.55"
	sh.Rep.Adjust(ip, 95, "attack") // own score over the block threshold (80)

	// 1. Unverified soft-blocked client is sent to the PoW gate, not a flat 403.
	req := httptest.NewRequest("GET", "/login", nil)
	req.RemoteAddr = ip + ":1234"
	req.Header.Set("User-Agent", ua)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "Blocked: reputation-block") {
		t.Fatalf("soft block must not hard-403 a human, body=%q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Verifikasi Browser") {
		t.Fatalf("soft block should serve the PoW gate, code=%d body=%q", rec.Code, rec.Body.String())
	}

	// 2. A client that already passed verification passes through to upstream.
	req2 := httptest.NewRequest("GET", "/login", nil)
	req2.RemoteAddr = ip + ":1234"
	req2.Header.Set("User-Agent", ua)
	req2.AddCookie(&http.Cookie{Name: "gs_access", Value: gate.Token("ok:" + ip)})
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !strings.Contains(rec2.Body.String(), "upstream-ok") {
		t.Fatalf("verified client must reach upstream, code=%d body=%q", rec2.Code, rec2.Body.String())
	}
}

// TestEdgeHardBlockStaysBlocked verifies that definitive hostile signals are
// NOT softened by the gate: a honeypot hit stays a flat 403 even with a valid
// access cookie.
func TestEdgeHardBlockStaysBlocked(t *testing.T) {
	cfg := edgeAttackConfig()
	cfg.Defense.Honeypot = config.Honeypot{Enabled: true}
	sh := shield.New(cfg)
	gate := access.New(config.Access{Enabled: true, ProtectAll: true, CookieName: "gs_access"}, "test-secret")
	h := newTestEdgeWithGate(t, sh, gate)

	ip := "192.0.2.56"
	req := httptest.NewRequest("GET", "/.env", nil)
	req.RemoteAddr = ip + ":1234"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36")
	req.AddCookie(&http.Cookie{Name: "gs_access", Value: gate.Token("ok:" + ip)})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Blocked: honeypot") {
		t.Fatalf("hard hostile block must stay 403, code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestIsSoftBlockReason(t *testing.T) {
	soft := []string{"reputation-block", "score", "rate-exceeded", "burst-exceeded", "throttled", "conn-limit"}
	for _, r := range soft {
		if !isSoftBlockReason(r) {
			t.Fatalf("%q should be soft", r)
		}
	}
	hard := []string{"blocklisted", "honeypot", "honeypot:/.git", "emergency-mode", "country-block",
		"blocked-ua", "sql-injection", "path-traversal", "layer4", "manual"}
	for _, r := range hard {
		if isSoftBlockReason(r) {
			t.Fatalf("%q must stay hard", r)
		}
	}
}
