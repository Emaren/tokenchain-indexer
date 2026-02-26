package config

import "os"

type Config struct {
	ListenAddr string
	ChainID    string
	Network    string
}

func FromEnv() Config {
	return Config{
		ListenAddr: getenv("LISTEN_ADDR", ":3321"),
		ChainID:    getenv("CHAIN_ID", "tokenchain-testnet-1"),
		Network:    getenv("NETWORK", "testnet"),
	}
}

func getenv(k, fallback string) string {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	return v
}
