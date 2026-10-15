// Command garudashield is the v3 edge shield entrypoint.
//
// Phases 0-2: boots, loads config, instantiates the full defense pipeline
// (shield.Shield) and an access gate (access.Gate), serves a reverse-proxy that
// runs Evaluate on every request with request-queue backpressure, a circuit
// breaker, and CAPTCHA gating. Plus /health, /stats, /_captcha endpoints.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ahkurdev/garudashield/internal/transport"
	"golang.org/x/net/http2"

	"github.com/ahkurdev/garudashield/internal/access"
	"github.com/ahkurdev/garudashield/internal/admin"
	"github.com/ahkurdev/garudashield/internal/alerts"
	"github.com/ahkurdev/garudashield/internal/auth"
	"github.com/ahkurdev/garudashield/internal/cluster"
	"github.com/ahkurdev/garudashield/internal/config"
	"github.com/ahkurdev/garudashield/internal/db"
	"github.com/ahkurdev/garudashield/internal/harden"
	"github.com/ahkurdev/garudashield/internal/license"
	"github.com/ahkurdev/garudashield/internal/mailer"
	"github.com/ahkurdev/garudashield/internal/metrics"
	"github.com/ahkurdev/garudashield/internal/netguard"
	"github.com/ahkurdev/garudashield/internal/offload"
	"github.com/ahkurdev/garudashield/internal/policy"
	"github.com/ahkurdev/garudashield/internal/quicguard"
	"github.com/ahkurdev/garudashield/internal/quota"
	"github.com/ahkurdev/garudashield/internal/reqview"
	"github.com/ahkurdev/garudashield/internal/shield"
	"github.com/ahkurdev/garudashield/internal/status"
	"github.com/ahkurdev/garudashield/internal/tenant"
	"github.com/ahkurdev/garudashield/internal/udpproxy"
)

const (
	captchaSVGPath   = "/_captcha"
	captchaVerifyURL = "/_captcha/verify"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file (defaults built-in)")
	printLicense := flag.Bool("print-self-license", false, "print host fingerprint and exit")
	flag.Parse()

	if *printLicense {
		fmt.Printf("host_fingerprint: %s\n", license.FingerprintHash())
		os.Exit(0)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}
	if cfg.Version == "" {
		cfg = config.Default()
	}

	// Security gate: a real deployment must set secrets. Running with empty
	// secrets means the admin API is loopback-only and the access gate uses an
	// insecure default token — acceptable for local dev, dangerous in prod.
	// Fail-closed: a non-loopback bind with an empty admin secret refuses to
	// start (GS_ALLOW_INSECURE=1 overrides for deliberate staging setups).
	insecureOverride := os.Getenv("GS_ALLOW_INSECURE") == "1"
	if cfg.Admin.Secret == "" && !isLoopbackOnly(cfg) {
		if insecureOverride {
			logger.Warn("ADMIN SECRET IS EMPTY (GS_ALLOW_INSECURE=1) — admin API is open to anyone on the network")
		} else {
			logger.Error("admin.secret is empty while binding a non-loopback address — refusing to start. Set admin.secret or bind to 127.0.0.1, or set GS_ALLOW_INSECURE=1 to override")
			os.Exit(1)
		}
	}
	if cfg.AccessSecret == "" && !isLoopbackOnly(cfg) {
		if insecureOverride {
			logger.Warn("ACCESS SECRET IS EMPTY (GS_ALLOW_INSECURE=1) — access gate uses insecure default token")
		} else {
			logger.Error("accessSecret is empty while binding a non-loopback address — refusing to start. Set accessSecret or bind to 127.0.0.1, or set GS_ALLOW_INSECURE=1 to override")
			os.Exit(1)
		}
	}
	if cfg.Server.TLS.Enabled && (cfg.Server.TLS.CertFile == "" || cfg.Server.TLS.KeyFile == "") {
		if cfg.Server.TLS.AutoSelfSigned {
			logger.Warn("TLS enabled without cert files — minting ephemeral self-signed cert (clients will NOT trust it; use real certs or Caddy for public traffic)")
		} else {
			logger.Warn("TLS enabled but certFile/keyFile missing — falling back to plaintext HTTP")
			cfg.Server.TLS.Enabled = false
		}
	}
	if !cfg.Server.TLS.Enabled {
		logger.Warn("TLS disabled — edge serves plaintext HTTP; the admin key travels in cleartext. Terminate TLS at a front proxy or set server.tls")
	}

	// L4 offload seam. Under `-tags ebpf` with offload.enabled + a valid
	// interface, this attaches a kernel XDP program that drops blocked IPs before
	// they reach userspace. Otherwise it is a userspace no-op and volumetric L4
	// protection relies on an upstream scrubbing center / anycast.
	off := offload.New(cfg.Offload)
	if !off.SupportsXDP() {
		logger.Info("L4 offload: userspace-only (no XDP/eBPF); large volumetric L4 floods must be scrubbed upstream")
	} else {
		if err := off.Offload(context.Background()); err != nil {
			logger.Error("xdp offload attach failed, falling back to userspace", "err", err)
		} else {
			logger.Info("L4 offload: XDP/eBPF attached", "interface", cfg.Offload.Iface)
		}
	}
	defer off.Close()

	sh := shield.New(cfg)
	defer sh.Close()

	// Keep the kernel drop list in sync with the engine blocklist (XDP offload).
	if off.SupportsXDP() {
		sh.SetBlocklistSync(off)
	}

	gate := access.New(cfg.Access, cfg.AccessSecret)

	var proxy *httputil.ReverseProxy
	if cfg.Upstream.Host != "" {
		target, e := url.Parse(fmt.Sprintf("http://%s:%d", cfg.Upstream.Host, cfg.Upstream.Port))
		if e != nil {
			logger.Error("invalid upstream address", "err", e)
			os.Exit(1)
		}
		proxy = httputil.NewSingleHostReverseProxy(target)
	}

	// Per-site reverse proxies (multi-subdomain routing). A request whose Host
	// matches a configured site of type "proxy" is forwarded to that site's
	// upstream; otherwise the global upstream above is used as the fallback.
	siteProxies := map[string]*httputil.ReverseProxy{}
	for name, site := range cfg.Sites {
		if site.Type != "proxy" {
			continue
		}
		target, ok := site.SiteProxyTarget()
		if !ok {
			continue
		}
		u, e := url.Parse("http://" + target)
		if e != nil {
			logger.Error("invalid site upstream, skipping", "site", name, "err", e)
			continue
		}
		siteProxies[hostKeyOf(name)] = httputil.NewSingleHostReverseProxy(u)
	}
	if len(siteProxies) > 0 {
		logger.Info("multi-site routing enabled", "sites", len(siteProxies))
	}

	// Request queue: reject with 503 when the edge is saturated (backpressure).
	sem := make(chan struct{}, maxInt(cfg.Server.MaxConcurrency, 1))

	breaker := newBreaker(5*time.Second, 10)

	// Health monitor (5 components, uptime, incidents).
	mon := status.New(cfg.Version, osHostname())
	mon.Register("Shield", nil)
	mon.Register("Proxy", nil)
	mon.Register("RedisStore", func() (status.ComponentStatus, string) {
		if sh.StoreHealthy() {
			return status.OK, ""
		}
		return status.Degraded, "state layer L2 unreachable"
	})
	mon.Register("Upstream", func() (status.ComponentStatus, string) {
		if proxy == nil {
			return status.OK, "no upstream configured"
		}
		if breaker.Open() {
			return status.Degraded, "circuit breaker open"
		}
		return status.OK, ""
	})
	mon.Register("Dashboard", func() (status.ComponentStatus, string) {
		return status.OK, "not served by this process"
	})
	mon.Start(10 * time.Second)
	defer mon.Close()

	// Alerts sink, wired into the engine as its notifier.
	al := alerts.New(cfg.Alerts)
	sh.SetNotifier(al)

	// Sync topology (baseline: this engine is the only active sync node).
	// Cluster.Nodes drives the whole model: one node needs no bus, optional
	// peers may be offline without degrading protection.
	ps := cluster.New(sh.Store(), cfg.Cluster)
	ps.Start()
	defer ps.Stop()
	sh.SetPeerBroadcaster(ps)
	syncMode := "off"
	if ps.Enabled() {
		syncMode = "on"
	}
	logger.Info("cluster: node=" + cfg.Cluster.NodeID + " topology=" + cluster.Label(cfg.Cluster) + " sync=" + syncMode)
	if cfg.Cluster.NodeCount() > 1 {
		logger.Warn("cluster: multi-node topology active; unavailable peers are reported as offline, never fatal")
	}

	mux := http.NewServeMux()
	// Prometheus metrics endpoint (operator scraping for Grafana). Cheap
	// fixed-window per-IP cap: scraping at 1 Hz is plenty for Grafana.
	metricsLimiter := admin.NewPublicRateLimiter(60)
	metricsWrapped := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !metricsLimiter.Allow(clientIP(r, cfg.Proxy)) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	mux.Handle("/metrics", metricsWrapped(metrics.Handler()))

	// /health and /stats are public too: cap them the same way (they are
	// cheap JSON dumps, but unthrottled they are an amplification vector).
	healthLimiter := admin.NewPublicRateLimiter(120)
	healthWrapped := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !healthLimiter.Allow(clientIP(r, cfg.Proxy)) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}

	mux.HandleFunc("/health", healthWrapped(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":     cfg.Name,
			"version":  cfg.Version,
			"status":   "ok",
			"phase":    "3-control-plane",
			"unixTime": time.Now().Unix(),
		})
	}))
	mux.HandleFunc("/stats", healthWrapped(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sh.GetStats())
	}))
	// Browser verification. `/_captcha` serves a self-contained HTML page whose
	// inline JS solves an HMAC-bound SHA-256 proof-of-work and reports a light
	// fingerprint; `/_captcha/verify` validates the PoW + client metadata and
	// issues the signed access cookie. This replaces the human-eye SVG CAPTCHA
	// so that only a real JS-capable browser can pass.
	mux.HandleFunc(captchaSVGPath, func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, cfg.Proxy)
		if !gate.CaptchaAllowed(ip) {
			http.Error(w, "rate-limited", http.StatusTooManyRequests)
			return
		}
		gate.RecordCaptcha(ip)
		writeChallengePage(w, r, gate, ip)
	})
	mux.HandleFunc(captchaVerifyURL, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Fetch Metadata: real browsers always attach Sec-Fetch-Site to a
		// same-origin fetch(). Raw HTTP clients (curl, most bots) do not.
		if r.Header.Get("Sec-Fetch-Site") == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			ID    string `json:"id"`
			Nonce string `json:"nonce"`
			FP    struct {
				WebDriver bool `json:"wd"`
			} `json:"fp"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		ip := clientIP(r, cfg.Proxy)
		if req.FP.WebDriver {
			http.Error(w, "automation detected", http.StatusForbidden)
			return
		}
		if !gate.VerifyChallenge(req.ID, req.Nonce, ip) {
			http.Error(w, "challenge failed", http.StatusForbidden)
			return
		}
		gate.ResetHold(ip)
		// A client that just solved the proof-of-work is a real browser: clear
		// any accumulated soft enforcement (reputation, cached verdicts,
		// auto-blocks) so it is not immediately re-blocked on the next request.
		sh.ClearSoftState(ip)
		gate.SetCookie(w, gate.Token("ok:"+ip))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	// Commercial store (PostgreSQL). Optional: only wired when GS_DATABASE_URL
	// is set. Enables tenant isolation, license issuance, and customer auth.
	// `ts`/`licSecret` are declared here (before the edge handler) so the edge
	// can enforce per-tenant license gating; the PG connection is established
	// just below, and the closure sees the assigned value at request time.
	var ts *tenant.Store
	// en is a *quota.Enforcer: per-plan request budget enforced at the edge when
	// quotas.enabled and Postgres are available. Nil disables quota (default).
	var en *quota.Enforcer
	licSecret := os.Getenv("GS_LICENSE_SECRET")
	// Fail-closed host binding: if GS_LICENSE_KEY is set and carries a host
	// fingerprint that does not match this machine, refuse to start (the
	// self-hosted license cannot be copied to another server). A host-less
	// license (managed/SaaS) is unaffected.
	if envLic := os.Getenv("GS_LICENSE_KEY"); envLic != "" && licSecret != "" {
		if l, perr := license.ParseKey(envLic, licSecret); perr == nil {
			if err := license.Validate(l, licSecret, time.Now()); err != nil {
				if err == license.ErrHostMismatch {
					logger.Error("GS_LICENSE_KEY is bound to a different host; refusing to start", "tenant", l.TenantID)
					os.Exit(1)
				}
				logger.Error("GS_LICENSE_KEY invalid", "err", err)
				os.Exit(1)
			}
		}
	}
	if os.Getenv("GS_DATABASE_URL") != "" {
		pool, err := db.Connect()
		if err != nil {
			logger.Error("postgres connect failed", "err", err)
		} else {
			if err := db.Migrate(context.Background(), pool); err != nil {
				logger.Error("postgres migrate failed", "err", err)
			} else {
				ts = tenant.New(pool)
				authh := auth.New(pool, os.Getenv("GS_SESSION_SECRET"))
				// Transactional email + public URL power password reset. Both
				// degrade gracefully when unset (token still created, not sent).
				authh.SetMailer(mailer.New(cfg.Alerts))
				pubURL := os.Getenv("GS_PUBLIC_URL")
				if pubURL == "" {
					pubURL = fmt.Sprintf("http://localhost:%d", cfg.Server.DashboardPort)
				}
				authh.SetPublicURL(pubURL)
				mux.HandleFunc("/api/auth/login", authh.Login)
				mux.HandleFunc("/api/auth/logout", authh.Logout)
				// Versioned aliases (/api/v1) for the commercial API.
				mux.HandleFunc("/api/v1/auth/login", authh.Login)
				mux.HandleFunc("/api/v1/auth/logout", authh.Logout)
				mux.HandleFunc("/api/v1/auth/register", authh.Register)
				mux.HandleFunc("/api/v1/auth/mfa/verify", authh.LoginMFA)
				mux.HandleFunc("/api/v1/auth/password/reset-request", authh.PasswordResetRequest)
				mux.HandleFunc("/api/v1/auth/password/reset", authh.PasswordReset)
				mux.HandleFunc("/api/v1/account/password", authh.ChangePassword)
				mux.HandleFunc("/api/v1/account/logout-all", authh.LogoutAll)
				mux.HandleFunc("/api/v1/account", authh.Account)
				mux.HandleFunc("/api/v1/account/mfa/setup", authh.MFASetup)
				mux.HandleFunc("/api/v1/account/mfa/enable", authh.MFAEnable)
				mux.HandleFunc("/api/v1/account/mfa/disable", authh.MFADisable)
				admin.RegisterTenant(mux, ts, authh)
				// Per-domain protection policies (customer site settings). The
				// engine reconciles them from Postgres on a short interval so
				// both edge nodes converge without a dedicated bus.
				admin.RegisterPolicies(mux, ts, authh, sh)
				sh.StartPolicyWatcher(policy.New(pool), 8*time.Second)
				if cfg.Quotas.Enabled {
					qs := quota.New(pool)
					en = quota.NewEnforcer(qs, cfg.Quotas.PlanLimits, cfg.Quotas.FallbackLimit)
					admin.RegisterQuotaV1(mux, qs, ts, authh, cfg.Quotas.PlanLimits, cfg.Quotas.FallbackLimit)
					logger.Info("usage quotas enabled at the edge",
						"planLimits", cfg.Quotas.PlanLimits, "fallback", cfg.Quotas.FallbackLimit)
				}
				logger.Info("commercial store connected (postgres)")
			}
		}
	}

	edge := makeEdgeHandler(sh, gate, proxy, siteProxies, breaker, logger, cfg.AllowedHosts, cfg.Proxy, ts, en, licSecret)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			edge(w, r)
		default:
			http.Error(w, "edge saturated", http.StatusServiceUnavailable)
		}
	})

	// Control-plane API (v2-compatible /api/* + SSE at /api/stats/stream).
	// With server.adminBind set, the admin API mounts on a DEDICATED listener
	// and the public edge mux rejects /api/* entirely: the control plane is
	// then only reachable on the bind address (e.g. loopback / private NIC).
	adminMux := http.NewServeMux()
	if cfg.Server.AdminBind != "" {
		admin.Register(adminMux, sh, mon, al, cfg.Admin, gate, ps, &cfg)
		mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "admin API moved to "+cfg.Server.AdminBind, http.StatusNotFound)
		})
	} else {
		admin.Register(mux, sh, mon, al, cfg.Admin, gate, ps, &cfg)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort)
	handler := harden.Headers(harden.CSRF(cfg.AllowedHosts)(mux))
	idle := time.Duration(cfg.Server.ConnIdleTimeoutMs) * time.Millisecond
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second, // anti-slowloris
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
		// ConnContext applies the per-connection idle deadline (set by the
		// netguard listener) right as the request begins, closing sockets that
		// stay silent longer than ConnIdleTimeoutMs.
		ConnContext: func(c context.Context, conn net.Conn) context.Context {
			if tc, ok := conn.(*netguard.TrackedConn); ok && idle > 0 {
				tc.SetIdleTimeout()
			}
			return c
		},
	}

	// Harden HTTP/2 (only meaningful when TLS is enabled; H2 needs TLS by
	// default). Caps concurrent streams per connection and max frame size to
	// blunt multiplexing-based slowloris and HPACK/decompression bombs.
	if cfg.Server.TLS.Enabled {
		maxStreams := cfg.Server.HTTP2.MaxConcurrentStreams
		if maxStreams <= 0 {
			maxStreams = 100
		}
		maxFrame := cfg.Server.HTTP2.MaxReadFrameSize
		if maxFrame <= 0 {
			maxFrame = 1 << 16
		}
		h2 := &http2.Server{
			MaxConcurrentStreams: uint32(maxStreams),
			MaxReadFrameSize:     uint32(maxFrame),
		}
		if err := http2.ConfigureServer(srv, h2); err != nil {
			logger.Error("http2 configure failed", "err", err)
		} else {
			logger.Info("http/2 hardened", "maxConcurrentStreams", h2.MaxConcurrentStreams,
				"maxReadFrameSize", h2.MaxReadFrameSize)
		}
	}

	// Bind the raw listener ourselves so we can wrap it with per-IP connection
	// caps + idle reaping (netguard) before the HTTP server sees the socket.
	rawLn, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("listen failed", "addr", addr, "err", err)
		os.Exit(1)
	}
	guard := netguard.New(rawLn, cfg.Server.MaxConnsPerIP, idle, int64(cfg.Server.MaxBytesPerSecPerIP))
	logger.Info("listener wrapped", "maxConnsPerIP", cfg.Server.MaxConnsPerIP,
		"connIdleMs", cfg.Server.ConnIdleTimeoutMs)
	if idle > 0 {
		srv.IdleTimeout = idle
	}

	// HTTP/3 (QUIC) listener — only when TLS is configured and http3.enabled.
	// QUIC runs over UDP and is itself a DDoS surface, so we cap per-connection
	// streams (MaxIncomingStreams) to blunt connection-amplification / stream
	// exhaustion. It runs alongside the TCP (H1/H2) listener.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Server.TLS.Enabled && cfg.Server.HTTP3.Enabled {
		h3Port := cfg.Server.HTTP3.Port
		if h3Port == 0 {
			h3Port = cfg.Server.HTTPPort + 1
		}
		maxStreams := cfg.Server.HTTP3.MaxIncomingStreams
		if maxStreams <= 0 {
			maxStreams = 100
		}
		idleMs := cfg.Server.HTTP3.MaxIdleTimeoutMs
		idle := 30 * time.Second
		if idleMs > 0 {
			idle = time.Duration(idleMs) * time.Millisecond
		}
		udpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, h3Port)
		tc := tlsConfigFor(cfg.Server.TLS)
		if tc == nil {
			logger.Error("http3 enabled but TLS cert/key missing; skipping QUIC listener")
		} else if pc, err := transport.StartH3(udpAddr, handler, tc, maxStreams, idle); err != nil {
			logger.Error("http3 listener failed", "addr", udpAddr, "err", err)
		} else {
			// QUIC bypasses netguard (TCP-only), so apply the packet-level
			// per-IP byte budget here. Fail-closed: over-budget IPs lose
			// packets silently for the penalty window.
			qg := quicguard.New(int64(cfg.Server.MaxBytesPerSecPerIP))
			guardedPC := qg.Wrap(pc)
			logger.Info("http/3 (QUIC) listener starting", "addr", udpAddr, "maxStreams", maxStreams,
				"perIPBytesPerSec", cfg.Server.MaxBytesPerSecPerIP)
			go func() {
				<-ctx.Done()
				guardedPC.Close()
			}()
		}
	}

	// UDP guard proxy (udpGuard): fronts one UDP service (DNS/game/NTP)
	// with the layer4 amplification heuristics. Abusive senders are
	// mirrored into the shield blocklist so they die on TCP/HTTP too.
	if ug := udpproxy.New(udpproxy.Config{
		Enabled:     cfg.UDPGuard.Enabled,
		Listen:      cfg.UDPGuard.Listen,
		Upstream:    cfg.UDPGuard.Upstream,
		ServicePort: cfg.UDPGuard.ServicePort,
		MaxRPSPerIP: cfg.UDPGuard.MaxRPSPerIP,
		TimeoutMs:   cfg.UDPGuard.TimeoutMs,
	}, sh.L4); ug != nil {
		ug.OnAbuse = func(ip, reason string) {
			sh.BlockIP(ip, 300000, reason)
		}
		go func() {
			logger.Info("udp guard listening", "listen", cfg.UDPGuard.Listen,
				"upstream", cfg.UDPGuard.Upstream, "servicePort", cfg.UDPGuard.ServicePort)
			if err := ug.Run(ctx); err != nil {
				logger.Error("udp guard failed", "err", err)
			}
		}()
	}

	// Dedicated control-plane listener (server.adminBind). The admin server
	// gets the same anti-slowloris timeouts but NO netguard edge guard: it is
	// bound to a trusted network, and its availability must not compete with
	// edge traffic.
	if cfg.Server.AdminBind != "" {
		adminSrv := &http.Server{
			Addr:              cfg.Server.AdminBind,
			Handler:           harden.Headers(harden.CSRF(cfg.AllowedHosts)(adminMux)),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		go func() {
			logger.Info("admin control plane listening (dedicated)", "addr", cfg.Server.AdminBind)
			if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("admin listener failed", "err", err)
				os.Exit(1)
			}
		}()
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = adminSrv.Shutdown(shutdownCtx)
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		if cfg.Server.TLS.Enabled {
			logger.Info("garudashield v3 starting (TLS)", "addr", addr, "version", cfg.Version)
			errCh <- srv.ServeTLS(guard, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			logger.Info("garudashield v3 starting (plaintext HTTP)", "addr", addr, "version", cfg.Version)
			errCh <- srv.Serve(guard)
		}
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "err", err)
		}
	}
	logger.Info("bye")
}

func makeEdgeHandler(sh *shield.Shield, gate *access.Gate, defaultProxy *httputil.ReverseProxy, siteProxies map[string]*httputil.ReverseProxy, breaker *breaker, logger *slog.Logger, allowedHosts []string, proxyCfg config.Proxy, ts *tenant.Store, en *quota.Enforcer, licSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered in request handler", "err", rec, "path", r.URL.Path)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()

		// 0. host allowlist (empty list = allow all)
		if !hostAllowed(allowedHosts, r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}

		// 0b. tenant license gate (commercial). If this Host maps to a tenant
		// and that tenant's license is missing/expired, refuse with 402 — the
		// customer must renew via WhatsApp. Free plan (no license) passes.
		// The resolved tenant is reused below by the usage-quota rail.
		var tnt *tenant.Tenant
		if ts != nil {
			t, err := ts.ResolveByDomain(r.Context(), r.Host)
			if err == nil {
				tnt = t
				if t.Plan != "free" {
					// Commercial status gate: a suspended/revoked tenant is
					// refused regardless of license key (operator revoked it).
					if t.Status == "suspended" || t.Status == "revoked" {
						http.Error(w, "subscription "+t.Status+" — renew via WhatsApp", http.StatusPaymentRequired)
						return
					}
					if t.LicenseKey == "" {
						http.Error(w, "license required — renew via WhatsApp", http.StatusPaymentRequired)
						return
					}
					if licSecret != "" {
						if l, perr := license.ParseKey(t.LicenseKey, licSecret); perr != nil {
							http.Error(w, "license invalid — contact support", http.StatusPaymentRequired)
							return
						} else if err := license.Validate(l, licSecret, time.Now()); err != nil {
							msg := "license invalid — contact support"
							if err == license.ErrHostMismatch {
								msg = "license bound to another host — renew via WhatsApp"
							} else if l.Expiry != 0 && time.Now().Unix() > l.Expiry {
								msg = "license expired — renew via WhatsApp"
							}
							http.Error(w, msg, http.StatusPaymentRequired)
							return
						}
					}
				}
			}
		}

		body, _ := readCapped(r, 1<<20)
		v := reqview.New(r, body)
		ip := clientIP(r, proxyCfg)
		// Align the view's client IP with the resolved (X-Forwarded-For aware)
		// IP so per-IP state (WAF fast-fail, reputation) is keyed consistently
		// even when the engine runs behind a proxy.
		v.RemoteAddr = ip

		// Track the request as an in-flight connection for this client IP for
		// the whole handler (released on return). The rate limiter's conn-limit
		// is concurrency-based, so a normal page load (many sequential asset
		// requests) is never blocked — only genuinely concurrent ones are.
		sh.Rate.ConnOpen(ip)
		defer sh.Rate.ConnClose(ip)

		// 1. defense pipeline
		d := sh.Evaluate(v, ip)
		switch d.Action {
		case shield.ActionBlock:
			// Soft/automated signals (reputation, rate-shaping) must never
			// hard-block a human. A verified visitor (valid access cookie) is
			// waved through, and an unverified one is handed to the
			// human-verification gate instead of a bare 403 so a legitimate
			// client behind a shared/NAT IP can recover. While flood mode is
			// active we keep the cheap 403 (fail-closed) rather than feeding
			// every shaped request into the gate. Definitive hostile signals
			// (honeypot, manual blocklist, geo, emergency, WAF/L7 attack
			// signatures) always stay a hard 403.
			if soft := isSoftBlockReason(d.Reason) && !sh.FloodOn(); soft {
				if hasValidAccess(gate, r, ip) {
					metrics.RequestsTotal.WithLabelValues("pass").Inc()
					break
				}
				if gate.ShouldGate(v.Host, r.URL.Path) {
					metrics.RequestsTotal.WithLabelValues("challenge").Inc()
					break
				}
			}
			metrics.RequestsTotal.WithLabelValues("block").Inc()
			http.Error(w, "Blocked: "+d.Reason, http.StatusForbidden)
			return
		case shield.ActionChallenge:
			metrics.RequestsTotal.WithLabelValues("challenge").Inc()
			// When the human-verification gate covers this host/path, defer to
			// it: gated clients get the static, cacheable proof-of-work page
			// (zero origin round-trip) and its signed cookie is honoured on the
			// next request. The shield's JSON challenge is only for non-gated
			// API clients. This is what keeps Next.js untouched during a flood.
			if gate.ShouldGate(v.Host, r.URL.Path) {
				break
			}
			// v2 verify-browser / score challenge: hand back a challenge id
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"action":      "challenge",
				"challengeId": d.ChallengeID,
				"reason":      d.Reason,
			})
			return
		}

		// 2. access gate (CAPTCHA / latency hold)
		if gate.ShouldGate(v.Host, r.URL.Path) {
			if hasValidAccess(gate, r, ip) {
				// pass through
			} else {
				// exponential latency hold (anti-automation)
				if hold := gate.Hold(ip); hold > 0 {
					time.Sleep(hold)
				}
				serveChallenge(w, r, gate, ip)
				return
			}
		}

		// 2b. usage-quota rail. Only requests that survived the DDoS pipeline
		// and the access gate consume the tenant's daily budget, so an attack
		// cannot burn a customer's quota. Counter is in Postgres and shared by
		// every engine instance on the same DB. A counter outage fails OPEN
		// (paid traffic must never be bricked by a billing-rail blip).
		if en != nil && tnt != nil {
			if ok, qerr := en.Check(r.Context(), tnt.ID, tnt.Plan); qerr != nil {
				logger.Warn("quota counter unavailable; allowing request", "tenant", tnt.ID, "err", qerr)
			} else if !ok {
				metrics.QuotaRejected.WithLabelValues(tnt.Plan).Inc()
				w.Header().Set("Retry-After", "86400")
				http.Error(w, "usage quota for today exceeded — upgrade plan via WhatsApp", http.StatusTooManyRequests)
				return
			}
		}

		// 3. upstream proxy (with circuit breaker). Route by Host: a matching
		// site proxy wins, otherwise the global upstream is the fallback.
		upstream := proxyForRequest(defaultProxy, siteProxies, r.Host)
		if upstream != nil {
			if breaker.Open() {
				http.Error(w, "upstream unavailable", http.StatusBadGateway)
				return
			}
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			upstream.ServeHTTP(rec, r)
			breaker.Record(rec.status >= 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		metrics.RequestsTotal.WithLabelValues("pass").Inc()
		_ = json.NewEncoder(w).Encode(map[string]any{"action": "pass", "ip": ip})
	}
}

// isSoftBlockReason reports whether a block reason comes from an automated,
// heuristic signal rather than a definitive hostile action. Soft reasons may be
// re-verified by the human-verification gate; hard reasons (honeypot, manual
// blocklist, geo, emergency, WAF/L7 attack signatures) are never softened.
func isSoftBlockReason(reason string) bool {
	switch reason {
	case "reputation-block", "score",
		"rate-exceeded", "burst-exceeded", "throttled", "conn-limit",
		"path-rule-limit", "red-drop", "red-ip-drop", "token-bucket",
		"global-limit":
		return true
	}
	return false
}

// hasValidAccess checks the signed access token (cookie or query param) and
// binds it to the client IP: the token payload is "ok:<ip>", so a cookie
// copied to another client is worthless. Latency hold is applied by the caller.
func hasValidAccess(gate *access.Gate, r *http.Request, ip string) bool {
	if c, err := r.Cookie(gate.CookieName()); err == nil && gate.Check(c.Value, "ok:"+ip) {
		return true
	}
	if q := r.URL.Query().Get(gate.QueryParam()); q != "" && gate.Check(q, "ok:"+ip) {
		return true
	}
	return false
}

// serveChallenge renders the browser-verification page for a gated request.
func serveChallenge(w http.ResponseWriter, r *http.Request, gate *access.Gate, ip string) {
	writeChallengePage(w, r, gate, ip)
}

// writeChallengePage issues a fresh challenge and renders the self-contained
// HTML page (inline JS only, no external assets) whose script solves the
// SHA-256 proof-of-work and reports a light fingerprint. Status 403 keeps it
// out of shared caches and signals a pending human check.
func writeChallengePage(w http.ResponseWriter, r *http.Request, gate *access.Gate, ip string) {
	id := gate.IssueChallenge(ip)
	page := strings.NewReplacer(
		"__CID__", id,
		"__DIFF__", strconv.Itoa(access.ChallengeDifficulty),
	).Replace(challengePageHTML)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusForbidden)
	_, _ = io.WriteString(w, page)
}

// challengePageHTML is the static shell; __CID__ (signed challenge id) and
// __DIFF__ (leading hex zeros required) are substituted per request. The script
// has no external dependencies so it runs under a strict CSP.
const challengePageHTML = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex">
<title>Verifikasi Browser - GarudaShield</title>
<style>
:root{--navy:#0A192F;--navy2:#122440;--gold:#D4AF37;--ink:#0B1220;--mut:#93A4BF}
*{box-sizing:border-box}
html,body{height:100%;margin:0}
body{background:linear-gradient(160deg,#0A192F 0%,#0E2138 55%,#122440 100%);color:#E6EDF7;
font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;display:flex;align-items:center;justify-content:center;padding:24px}
.card{background:#F5F7FA;color:var(--ink);max-width:440px;width:100%;border-radius:16px;
padding:32px;box-shadow:0 24px 60px rgba(2,10,25,.45);text-align:center}
.logo{display:inline-flex;align-items:center;gap:10px;font-weight:800;letter-spacing:.5px;color:var(--navy);margin-bottom:18px}
.dot{width:12px;height:12px;border-radius:3px;background:var(--gold);display:inline-block}
h1{font-size:20px;margin:0 0 8px;color:var(--navy)}
p{margin:0 0 20px;color:#465063;font-size:14px;line-height:1.5}
.track{height:8px;background:#E1E7F0;border-radius:999px;overflow:hidden;margin:0 0 14px}
.bar{height:100%;width:4%;background:linear-gradient(90deg,var(--gold),#C9A227);border-radius:999px;transition:width .18s ease}
#s{font-size:13px;color:#465063;min-height:18px}
#r{display:none;margin-top:16px;background:var(--navy);color:#fff;border:0;border-radius:10px;
padding:11px 22px;font-size:14px;font-weight:600;cursor:pointer}
#r:hover{background:var(--navy2)}
.foot{margin-top:20px;font-size:11px;color:#8894A6}
</style>
</head>
<body>
<div class="card">
<div class="logo"><span class="dot"></span> GARUDA SHIELD</div>
<h1>Verifikasi Browser</h1>
<p>Kami memeriksa apakah permintaan ini berasal dari browser asli. Proses ini berjalan otomatis dan hanya butuh beberapa saat.</p>
<div class="track"><div class="bar" id="b"></div></div>
<div id="s">Menyiapkan verifikasi&hellip;</div>
<button id="r" type="button">Coba lagi</button>
<div class="foot">Proteksi DDoS multi-layer &middot; GarudaShield v3</div>
</div>
<script>
(function(){
var CID="__CID__",DIFF=__DIFF__,VERIFY="/_captcha/verify";
var s=document.getElementById("s"),b=document.getElementById("b"),r=document.getElementById("r");
var enc=new TextEncoder();
function st(t){s.textContent=t}
function fail(m){st(m||"Verifikasi gagal. Silakan coba lagi.");r.style.display="inline-block"}
function fp(){
  var tz="";try{tz=Intl.DateTimeFormat().resolvedOptions().timeZone||""}catch(e){}
  var np=0;try{np=navigator.plugins?navigator.plugins.length:0}catch(e){}
  return {wd:navigator.webdriver===true,hc:navigator.hardwareConcurrency||0,
    lang:navigator.language||"",langs:(navigator.languages||[]).join(","),tz:tz,
    scr:(screen.width||0)+"x"+(screen.height||0)+"x"+(screen.colorDepth||0),
    tzo:new Date().getTimezoneOffset(),pl:np,mtp:navigator.maxTouchPoints||0,
    ce:navigator.cookieEnabled===true};
}
var zb=Math.floor(DIFF/2),zr=DIFF%2;
function ok(buf){
  var u=new Uint8Array(buf);
  for(var i=0;i<zb;i++){if(u[i]!==0)return false}
  if(zr===1&&(u[zb]>>4)!==0)return false;
  return true;
}
function submit(n){
  st("Memverifikasi hasil\u2026");
  fetch(VERIFY,{method:"POST",headers:{"Content-Type":"application/json"},credentials:"same-origin",
    body:JSON.stringify({id:CID,nonce:String(n),fp:fp()})})
  .then(function(x){return x.json()})
  .then(function(j){if(j&&j.ok){b.style.width="100%";st("Berhasil. Mengalihkan\u2026");location.reload()}
    else{fail()}})
  .catch(function(){fail("Koneksi gagal. Periksa jaringan lalu coba lagi.")});
}
r.addEventListener("click",function(){location.reload()});
(function solve(){
  if(navigator.webdriver===true){fail("Otomasi terdeteksi. Gunakan browser standar.");return}
  if(!(window.crypto&&crypto.subtle)){fail("Browser tidak mendukung verifikasi ini.");return}
  st("Memverifikasi browser\u2026");
  var nonce=0,BATCH=512,MAX=120000000,done=false;
  function step(){
    if(done)return;
    var base=nonce,jobs=[];
    for(var k=0;k<BATCH;k++){jobs.push(crypto.subtle.digest("SHA-256",enc.encode(CID+"."+(base+k))))}
    nonce+=BATCH;
    var pct=Math.min(95,Math.round(nonce/65536*60));
    b.style.width=(pct<5?5:pct)+"%";
    Promise.all(jobs).then(function(ds){
      for(var k=0;k<ds.length;k++){if(ok(ds[k])){done=true;submit(base+k);return}}
      if(nonce>MAX){fail("Verifikasi terlalu lama. Coba lagi.");return}
      setTimeout(step,0);
    }).catch(function(){fail()});
  }
  step();
})();
})();
</script>
</body>
</html>`

// ---------- circuit breaker ----------

// statusRecorder captures the upstream response status for the breaker.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

type breaker struct {
	mu        sync.Mutex
	open      bool
	failures  int
	threshold int
	cooldown  time.Duration
	openedAt  time.Time
}

func newBreaker(cooldown time.Duration, threshold int) *breaker {
	return &breaker{cooldown: cooldown, threshold: threshold}
}

func (b *breaker) Open() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.open && time.Since(b.openedAt) > b.cooldown {
		b.open = false
		b.failures = 0
	}
	return b.open
}

func (b *breaker) Record(failed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if failed {
		b.failures++
		if b.failures >= b.threshold {
			b.open = true
			b.openedAt = time.Now()
		}
	} else {
		b.failures = 0
	}
}

// ---------- helpers ----------

// clientIP resolves the originating client IP. Proxy-provided headers
// (X-Real-IP / X-Forwarded-For) are only honored when TrustProxy is enabled
// AND the immediate connection originates from a trusted proxy; otherwise a
// client could spoof its own IP via those headers. When untrusted, the
// connection IP is used.
func clientIP(r *http.Request, p config.Proxy) string {
	connIP := remoteIP(r)
	if !p.TrustProxy || !isTrustedProxy(connIP, p.TrustedProxies) {
		return connIP
	}
	if p.RealIPHeader != "" {
		if v := r.Header.Get(p.RealIPHeader); v != "" {
			return trimSpace(v)
		}
	}
	if p.ForwardedForHeader != "" {
		if v := r.Header.Get(p.ForwardedForHeader); v != "" {
			if i := indexByte(v, ','); i >= 0 {
				return trimSpace(v[:i])
			}
			return trimSpace(v)
		}
	}
	return connIP
}

// remoteIP extracts the connection IP from the request's RemoteAddr.
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isTrustedProxy reports whether ip belongs to the trusted proxy set. Entries
// may be a plain IP or a CIDR. An empty set means nothing is trusted.
func isTrustedProxy(ip string, trusted []string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, c := range trusted {
		if c == "" {
			continue
		}
		if ip == c {
			return true
		}
		if _, network, err := net.ParseCIDR(c); err == nil && network.Contains(parsed) {
			return true
		}
	}
	return false
}

func readCapped(r *http.Request, max int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, max))
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// hostKeyOf normalizes a Host header into the site-matching key: lowercase and
// with the port stripped (matching v2's _hostKey behavior).
func hostKeyOf(h string) string {
	h = strings.ToLower(h)
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return h
}

// proxyForRequest returns the reverse proxy for the request Host: a matching
// site proxy if configured, otherwise the global default (which may be nil).
func proxyForRequest(def *httputil.ReverseProxy, sites map[string]*httputil.ReverseProxy, host string) *httputil.ReverseProxy {
	if p, ok := sites[hostKeyOf(host)]; ok {
		return p
	}
	return def
}

// hostAllowed reports whether the incoming Host is permitted. An empty
// allowlist permits everything (v2 default). Matching strips the port and
// supports a single wildcard "*".
func hostAllowed(allowed []string, host string) bool {
	if len(allowed) == 0 {
		return true
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	for _, a := range allowed {
		if a == "*" || a == host {
			return true
		}
	}
	return false
}

func osHostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}

// isLoopbackOnly reports whether the engine is bound to a loopback address and
// therefore only reachable from the local machine. Used to relax the empty-
// secret warning: a loopback-only dev instance cannot be reached from the
// network, so an empty admin secret is tolerable.

// tlsConfigFor loads the edge TLS certificate into a *tls.Config for use by the
// HTTP/3 (QUIC) listener. The TCP TLS listener is handled by ServeTLS directly;
// QUIC needs the parsed config because http3.Server does not take file paths.
func tlsConfigFor(t config.TLSConfig) *tls.Config {
	if t.CertFile != "" && t.KeyFile != "" {
		c, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil
		}
		return &tls.Config{
			Certificates: []tls.Certificate{c},
			NextProtos:   []string{"h3"},
		}
	}
	if t.AutoSelfSigned {
		c, err := selfSignedCert()
		if err != nil {
			return nil
		}
		return &tls.Config{
			Certificates: []tls.Certificate{c},
			NextProtos:   []string{"h3"},
		}
	}
	return nil
}

// selfSignedCert mints an ephemeral ECDSA P-256 cert (90 days) for boot
// paths where TLS is enabled but no files exist (dev, QUIC smoke, internal
// links). Never silently used: callers enable it explicitly via
// tls.autoSelfSigned and must log that clients will see an untrusted cert.
func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "garudashield"
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost", host},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
}

func isLoopbackOnly(cfg config.Config) bool {
	h := cfg.Server.Host
	if h == "" || h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}
