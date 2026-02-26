package faucet

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr string
	Enabled    bool
	StateFile  string

	ChainID        string
	NodeRPC        string
	Home           string
	KeyringBackend string
	FromKey        string
	Bech32Prefix   string

	Denom          string
	DispenseAmount string
	Fees           string

	AddressCooldown time.Duration
	IPCooldown      time.Duration
	MaxPerDay       int
}

func FromEnv() Config {
	return Config{
		ListenAddr:      getenv("LISTEN_ADDR", ":3322"),
		Enabled:         getenvBool("FAUCET_ENABLED", true),
		StateFile:       getenv("STATE_FILE", "/var/lib/tokenchain-testnet/faucet-state.json"),
		ChainID:         getenv("CHAIN_ID", "tokenchain-testnet-1"),
		NodeRPC:         getenv("NODE_RPC", "http://127.0.0.1:26657"),
		Home:            getenv("CHAIN_HOME", "/var/lib/tokenchain-testnet"),
		KeyringBackend:  getenv("KEYRING_BACKEND", "test"),
		FromKey:         getenv("FROM_KEY", "treasury"),
		Bech32Prefix:    getenv("BECH32_PREFIX", "tokenchain"),
		Denom:           getenv("DENOM", "utoken"),
		DispenseAmount:  getenv("DISPENSE_AMOUNT", "1000000"),
		Fees:            getenv("TX_FEES", "500utoken"),
		AddressCooldown: time.Duration(getenvInt("ADDRESS_COOLDOWN_MINUTES", 1440)) * time.Minute,
		IPCooldown:      time.Duration(getenvInt("IP_COOLDOWN_MINUTES", 30)) * time.Minute,
		MaxPerDay:       getenvInt("MAX_REQUESTS_PER_DAY", 1),
	}
}

func getenv(k, fallback string) string {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	return v
}

func getenvBool(k string, fallback bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getenvInt(k string, fallback int) int {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
