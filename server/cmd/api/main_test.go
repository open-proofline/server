package main

import (
	"net/http"
	"testing"

	"safety-recorder/server/internal/config"
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
	assertServer(t, servers[2], "public emergency viewer", "127.0.0.1:8081", publicHandler)
	assertServer(t, servers[3], "public emergency viewer", "192.168.1.20:8081", publicHandler)
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
