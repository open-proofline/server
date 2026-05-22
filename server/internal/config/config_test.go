package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLoadDefaultBindAddrs(t *testing.T) {
	cfg := loadConfigForTest(t, nil)

	assertStringsEqual(t, cfg.PrivateBindAddrs, []string{"127.0.0.1:8080"})
	assertStringsEqual(t, cfg.PublicBindAddrs, []string{"127.0.0.1:8081"})
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
