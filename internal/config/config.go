package config

import "os"

type Config struct {
	ListenAddr string
	ChainID    string
	Network    string
	RPCAddr    string
	RESTAddr   string
}

func FromEnv() Config {
	return Config{
		ListenAddr: getenv("LISTEN_ADDR", ":3321"),
		ChainID:    getenv("CHAIN_ID", "tokenchain-testnet-1"),
		Network:    getenv("NETWORK", "testnet"),
		RPCAddr:    getenv("RPC_ADDR", "http://127.0.0.1:26657"),
		RESTAddr:   getenv("REST_ADDR", "http://127.0.0.1:1317"),
	}
}

func getenv(k, fallback string) string {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	return v
}
