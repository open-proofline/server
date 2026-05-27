package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/open-proofline/server/internal/config"
)

func TestNewHTTPServersCreatesOneServerPerBindAddress(t *testing.T) {
	privateHandler := http.NewServeMux()
	publicHandler := http.NewServeMux()
	cfg := config.Config{
		PrivateBindAddrs: []string{"127.0.0.1:8080", "10.66.0.1:8080"},
		PublicBindAddrs:  []string{"127.0.0.1:8081", "192.168.1.20:8081"},
	}

	servers := newHTTPServers(cfg, privateHandler, publicHandler)

	if len(servers) != 4 {
		t.Fatalf("got %d servers, want 4", len(servers))
	}
	assertServer(t, servers[0], "private api", "127.0.0.1:8080", privateHandler)
	assertServer(t, servers[1], "private api", "10.66.0.1:8080", privateHandler)
	assertServer(t, servers[2], "public incident viewer", "127.0.0.1:8081", publicHandler)
	assertServer(t, servers[3], "public incident viewer", "192.168.1.20:8081", publicHandler)
}

func TestNewHTTPServersAppliesPrivateAndPublicTimeouts(t *testing.T) {
	privateHandler := http.NewServeMux()
	publicHandler := http.NewServeMux()
	cfg := config.Config{
		PrivateBindAddrs: []string{"127.0.0.1:8080"},
		PublicBindAddrs:  []string{"127.0.0.1:8081"},
		PrivateTimeouts: config.HTTPTimeouts{
			ReadHeaderTimeout: 11 * time.Second,
			ReadTimeout:       0,
			WriteTimeout:      0,
			IdleTimeout:       121 * time.Second,
		},
		PublicTimeouts: config.HTTPTimeouts{
			ReadHeaderTimeout: 12 * time.Second,
			ReadTimeout:       31 * time.Second,
			WriteTimeout:      301 * time.Second,
			IdleTimeout:       122 * time.Second,
		},
	}

	servers := newHTTPServers(cfg, privateHandler, publicHandler)

	assertServerTimeouts(t, servers[0].server, cfg.PrivateTimeouts)
	assertServerTimeouts(t, servers[1].server, cfg.PublicTimeouts)
}

func assertServer(t *testing.T, got namedServer, name, addr string, handler http.Handler) {
	t.Helper()
	if got.name != name {
		t.Fatalf("server name = %q, want %q", got.name, name)
	}
	if got.server.Addr != addr {
		t.Fatalf("server addr = %q, want %q", got.server.Addr, addr)
	}
	if got.server.Handler != handler {
		t.Fatal("server handler did not match expected shared handler")
	}
}

func assertServerTimeouts(t *testing.T, server *http.Server, want config.HTTPTimeouts) {
	t.Helper()
	if server.ReadHeaderTimeout != want.ReadHeaderTimeout ||
		server.ReadTimeout != want.ReadTimeout ||
		server.WriteTimeout != want.WriteTimeout ||
		server.IdleTimeout != want.IdleTimeout {
		t.Fatalf("server timeouts = read_header %s read %s write %s idle %s, want %+v",
			server.ReadHeaderTimeout,
			server.ReadTimeout,
			server.WriteTimeout,
			server.IdleTimeout,
			want,
		)
	}
}
