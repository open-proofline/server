package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultBindAddrs(t *testing.T) {
	cfg := loadConfigForTest(t, nil)

	assertStringsEqual(t, cfg.PrivateBindAddrs, []string{"127.0.0.1:8080"})
	assertStringsEqual(t, cfg.PublicBindAddrs, []string{"127.0.0.1:8081"})
}

func TestLoadDefaultHTTPTimeouts(t *testing.T) {
	cfg := loadConfigForTest(t, nil)

	assertTimeoutsEqual(t, cfg.PrivateTimeouts, HTTPTimeouts{
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	})
	assertTimeoutsEqual(t, cfg.PublicTimeouts, HTTPTimeouts{
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	})
}

func TestLoadHTTPTimeoutsFromEnv(t *testing.T) {
	cfg := loadConfigForTest(t, map[string]string{
		"SAFE_PRIVATE_READ_HEADER_TIMEOUT": "11s",
		"SAFE_PRIVATE_READ_TIMEOUT":        "0",
		"SAFE_PRIVATE_WRITE_TIMEOUT":       "0s",
		"SAFE_PRIVATE_IDLE_TIMEOUT":        "2m",
		"SAFE_PUBLIC_READ_HEADER_TIMEOUT":  "12s",
		"SAFE_PUBLIC_READ_TIMEOUT":         "31s",
		"SAFE_PUBLIC_WRITE_TIMEOUT":        "5m",
		"SAFE_PUBLIC_IDLE_TIMEOUT":         "3m",
	})

	assertTimeoutsEqual(t, cfg.PrivateTimeouts, HTTPTimeouts{
		ReadHeaderTimeout: 11 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       2 * time.Minute,
	})
	assertTimeoutsEqual(t, cfg.PublicTimeouts, HTTPTimeouts{
		ReadHeaderTimeout: 12 * time.Second,
		ReadTimeout:       31 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       3 * time.Minute,
	})
}

func TestLoadRejectsInvalidHTTPTimeouts(t *testing.T) {
	tests := map[string]map[string]string{
		"negative": {
			"SAFE_PRIVATE_READ_TIMEOUT": "-1s",
		},
		"invalid": {
			"SAFE_PUBLIC_WRITE_TIMEOUT": "soon",
		},
		"empty": {
			"SAFE_PUBLIC_IDLE_TIMEOUT": "",
		},
	}

	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := loadConfigForTestErr(t, env)
			if err == nil {
				t.Fatal("expected timeout config error")
			}
			if !strings.Contains(err.Error(), "parse SAFE_") {
				t.Fatalf("expected env var parse context, got %v", err)
			}
		})
	}
}

func TestLoadSingularBindAddrs(t *testing.T) {
	cfg := loadConfigForTest(t, map[string]string{
		"SAFE_PRIVATE_BIND_ADDR": "10.66.0.1:8080",
		"SAFE_PUBLIC_BIND_ADDR":  "192.168.1.20:8081",
	})

	assertStringsEqual(t, cfg.PrivateBindAddrs, []string{"10.66.0.1:8080"})
	assertStringsEqual(t, cfg.PublicBindAddrs, []string{"192.168.1.20:8081"})
}

func TestLoadPluralBindAddrs(t *testing.T) {
	cfg := loadConfigForTest(t, map[string]string{
		"SAFE_PRIVATE_BIND_ADDRS": "127.0.0.1:8080,10.66.0.1:8080",
		"SAFE_PUBLIC_BIND_ADDRS":  "127.0.0.1:8081,192.168.1.20:8081",
	})

	assertStringsEqual(t, cfg.PrivateBindAddrs, []string{"127.0.0.1:8080", "10.66.0.1:8080"})
	assertStringsEqual(t, cfg.PublicBindAddrs, []string{"127.0.0.1:8081", "192.168.1.20:8081"})
}

func TestLoadPluralBindAddrsTakePrecedenceOverSingular(t *testing.T) {
	cfg := loadConfigForTest(t, map[string]string{
		"SAFE_PRIVATE_BIND_ADDR":  "10.0.0.1:8080",
		"SAFE_PRIVATE_BIND_ADDRS": "127.0.0.1:8080,10.66.0.1:8080",
		"SAFE_PUBLIC_BIND_ADDR":   "10.0.0.2:8081",
		"SAFE_PUBLIC_BIND_ADDRS":  "127.0.0.1:8081",
	})

	assertStringsEqual(t, cfg.PrivateBindAddrs, []string{"127.0.0.1:8080", "10.66.0.1:8080"})
	assertStringsEqual(t, cfg.PublicBindAddrs, []string{"127.0.0.1:8081"})
}

func TestLoadBindAddrsTrimWhitespace(t *testing.T) {
	cfg := loadConfigForTest(t, map[string]string{
		"SAFE_PRIVATE_BIND_ADDRS": " 127.0.0.1:8080 , 10.66.0.1:8080 ",
		"SAFE_PUBLIC_BIND_ADDRS":  " 127.0.0.1:8081 ",
	})

	assertStringsEqual(t, cfg.PrivateBindAddrs, []string{"127.0.0.1:8080", "10.66.0.1:8080"})
	assertStringsEqual(t, cfg.PublicBindAddrs, []string{"127.0.0.1:8081"})
}

func TestLoadBindAddrsRejectEmptyEntries(t *testing.T) {
	tests := map[string]map[string]string{
		"fully empty private list": {
			"SAFE_PRIVATE_BIND_ADDRS": "",
		},
		"comma-only public list": {
			"SAFE_PUBLIC_BIND_ADDRS": ",",
		},
		"middle empty entry": {
			"SAFE_PRIVATE_BIND_ADDRS": "127.0.0.1:8080,,10.66.0.1:8080",
		},
		"singular empty entry": {
			"SAFE_PRIVATE_BIND_ADDR": "",
		},
	}

	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := loadConfigForTestErr(t, env)
			if err == nil {
				t.Fatal("expected config error")
			}
			if !strings.Contains(err.Error(), "empty entry") {
				t.Fatalf("expected empty-entry error, got %v", err)
			}
		})
	}
}

func TestParseBindAddrsKeepsAddressStringsForHTTPValidation(t *testing.T) {
	addrs, err := parseBindAddrs("not-a-net-addr")
	if err != nil {
		t.Fatalf("parseBindAddrs returned error: %v", err)
	}
	assertStringsEqual(t, addrs, []string{"not-a-net-addr"})
}

func TestParseBytesAcceptsUnitValues(t *testing.T) {
	tests := map[string]int64{
		"42":    42,
		"1B":    1,
		"0.5KB": 512,
		"1.5MB": 1572864,
		"2 G":   2 * 1024 * 1024 * 1024,
	}

	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			got, err := parseBytes(raw)
			if err != nil {
				t.Fatalf("parseBytes returned error: %v", err)
			}
			if got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		})
	}
}

func TestParseBytesRejectsUnsafeUnitValues(t *testing.T) {
	tests := map[string]string{
		"0B":                    "positive",
		"0.0001B":               "at least one byte",
		"9223372036853727232":   "too large",
		"9223372036853727232B":  "too large",
		"9999999999999999999GB": "too large",
		"NaNMB":                 "invalid byte value",
		"InfMB":                 "invalid byte value",
	}

	for raw, wantError := range tests {
		t.Run(raw, func(t *testing.T) {
			_, err := parseBytes(raw)
			if err == nil {
				t.Fatal("expected parseBytes error")
			}
			if !strings.Contains(err.Error(), wantError) {
				t.Fatalf("expected error containing %q, got %v", wantError, err)
			}
		})
	}
}

func TestLoadRejectsUnsafeMaxUploadBytes(t *testing.T) {
	_, err := loadConfigForTestErr(t, map[string]string{
		"SAFE_MAX_UPLOAD_BYTES": "0.0001B",
	})
	if err == nil {
		t.Fatal("expected Load error")
	}
	if !strings.Contains(err.Error(), "parse SAFE_MAX_UPLOAD_BYTES") {
		t.Fatalf("expected SAFE_MAX_UPLOAD_BYTES context, got %v", err)
	}
}

func loadConfigForTest(t *testing.T, env map[string]string) Config {
	t.Helper()
	cfg, err := loadConfigForTestErr(t, env)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func loadConfigForTestErr(t *testing.T, env map[string]string) (Config, error) {
	t.Helper()
	names := []string{
		"SAFE_PRIVATE_BIND_ADDRS",
		"SAFE_PUBLIC_BIND_ADDRS",
		"SAFE_PRIVATE_BIND_ADDR",
		"SAFE_PUBLIC_BIND_ADDR",
		"SAFE_DATA_DIR",
		"SAFE_DB_PATH",
		"SAFE_MAX_UPLOAD_BYTES",
		"SAFE_PRIVATE_READ_HEADER_TIMEOUT",
		"SAFE_PRIVATE_READ_TIMEOUT",
		"SAFE_PRIVATE_WRITE_TIMEOUT",
		"SAFE_PRIVATE_IDLE_TIMEOUT",
		"SAFE_PUBLIC_READ_HEADER_TIMEOUT",
		"SAFE_PUBLIC_READ_TIMEOUT",
		"SAFE_PUBLIC_WRITE_TIMEOUT",
		"SAFE_PUBLIC_IDLE_TIMEOUT",
	}
	restoreEnv(t, names)
	for name, value := range env {
		if err := os.Setenv(name, value); err != nil {
			t.Fatalf("set %s: %v", name, err)
		}
	}
	return Load()
}

func restoreEnv(t *testing.T, names []string) {
	t.Helper()
	originals := make(map[string]string, len(names))
	present := make(map[string]bool, len(names))
	for _, name := range names {
		value, ok := os.LookupEnv(name)
		originals[name] = value
		present[name] = ok
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
	t.Cleanup(func() {
		for _, name := range names {
			if present[name] {
				_ = os.Setenv(name, originals[name])
				continue
			}
			_ = os.Unsetenv(name)
		}
	})
}

func assertStringsEqual(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertTimeoutsEqual(t *testing.T, got, want HTTPTimeouts) {
	t.Helper()
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
