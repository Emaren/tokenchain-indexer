package config

import "os"

type Config struct {
	ListenAddr string
	ChainID    string
	Network    string
	RPCAddr    string
	RESTAddr   string

	RelayerService string
	HermesBinary   string
	HermesConfig   string

	AdminAPIToken string
	AdminFromKey  string
	ChainHome     string
	Keyring       string
	TxFees        string
	TxGas         string
	Tokenchaind   string
}

func FromEnv() Config {
	return Config{
		ListenAddr: getenv("LISTEN_ADDR", ":3321"),
		ChainID:    getenv("CHAIN_ID", "tokenchain-testnet-1"),
		Network:    getenv("NETWORK", "testnet"),
		RPCAddr:    getenv("RPC_ADDR", "http://127.0.0.1:26657"),
		RESTAddr:   getenv("REST_ADDR", "http://127.0.0.1:1317"),

		RelayerService: getenv("RELAYER_SERVICE", "tokenchain-relayer.service"),
		HermesBinary:   getenv("HERMES_BIN", "/usr/local/bin/hermes"),
		HermesConfig:   getenv("HERMES_CONFIG", "/etc/tokenchain/hermes.toml"),

		AdminAPIToken: getenv("ADMIN_API_TOKEN", ""),
		AdminFromKey:  getenv("ADMIN_FROM_KEY", "founder"),
		ChainHome:     getenv("CHAIN_HOME", "/var/lib/tokenchain-testnet"),
		Keyring:       getenv("KEYRING_BACKEND", "test"),
		TxFees:        getenv("TX_FEES", "5000utoken"),
		TxGas:         getenv("TX_GAS", "200000"),
		Tokenchaind:   getenv("TOKENCHAIND_BIN", "/usr/local/bin/tokenchaind"),
	}
}

func getenv(k, fallback string) string {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	return v
}
