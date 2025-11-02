package cache

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func resolveValkeyAddrs() []string {
	if nodes := os.Getenv("VALKEY_NODES"); nodes != "" {
		return strings.Split(nodes, ",")
	}

	if svc := os.Getenv("VALKEY_SERVICE"); svc != "" {
		addrs, err := net.LookupHost(svc)
		if err != nil {
			log.Fatalf("failed to resolve %s: %v", svc, err)
		}
		var out []string
		for _, ip := range addrs {
			out = append(out, fmt.Sprintf("%s:6379", ip))
		}
		return out
	}

	log.Fatal(
		"no Valkey discovery env provided (VALKEY_NODES or VALKEY_SERVICE)",
	)
	return nil
}
