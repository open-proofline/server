package config

import (
	"fmt"
	"os"
	"strings"
)

func bindAddrsFromEnv(pluralName, singularName, fallback string) ([]string, error) {
	if raw, ok := os.LookupEnv(pluralName); ok {
		addrs, err := parseBindAddrs(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", pluralName, err)
		}
		return addrs, nil
	}
	if raw, ok := os.LookupEnv(singularName); ok {
		addrs, err := parseBindAddrs(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", singularName, err)
		}
		return addrs, nil
	}
	return []string{fallback}, nil
}

func parseBindAddrs(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	addrs := make([]string, 0, len(parts))
	for index, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			return nil, fmt.Errorf("bind address list contains empty entry at position %d", index+1)
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("bind address list must contain at least one address")
	}
	return addrs, nil
}
