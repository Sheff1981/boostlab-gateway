package config

import (
	"os"
	"strconv"
)

type Config struct {
	NodeID             string
	Region             string
	HTTPAddr           string
	UDPAddr            string
	WireGuardPublicKey string
	WireGuardPort      int
}

func Load() Config {
	return Config{
		NodeID:             envOr("BOOSTLAB_NODE_ID", "dev-node"),
		Region:             envOr("BOOSTLAB_REGION", "local"),
		HTTPAddr:           envOr("BOOSTLAB_HTTP_ADDR", ":8080"),
		UDPAddr:            envOr("BOOSTLAB_UDP_ADDR", ":51821"),
		WireGuardPublicKey: os.Getenv("BOOSTLAB_WG_PUBLIC_KEY"),
		WireGuardPort:      envIntOr("BOOSTLAB_WG_PORT", 51820),
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
