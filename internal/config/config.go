package config

import "os"

type Config struct {
	NodeID   string
	Region   string
	HTTPAddr string
	UDPAddr  string
}

func Load() Config {
	return Config{
		NodeID:   envOr("BOOSTLAB_NODE_ID", "dev-node"),
		Region:   envOr("BOOSTLAB_REGION", "local"),
		HTTPAddr: envOr("BOOSTLAB_HTTP_ADDR", ":8080"),
		UDPAddr:  envOr("BOOSTLAB_UDP_ADDR", ":51821"),
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
