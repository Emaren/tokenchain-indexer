package faucet

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var addrPattern = regexp.MustCompile(`^[a-z0-9]{20,100}$`)

type Server struct {
	cfg Config

	mu         sync.Mutex
	nextByAddr map[string]time.Time
	nextByIP   map[string]time.Time
	dailyCount map[string]int
}

func NewServer(cfg Config) *Server {
	return &Server{
		cfg:        cfg,
		nextByAddr: make(map[string]time.Time),
		nextByIP:   make(map[string]time.Time),
		dailyCount: make(map[string]int),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v1/faucet/policy", s.policy)
	mux.HandleFunc("/v1/faucet/request", s.request)
	return s.withCORS(mux)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                   true,
		"enabled":              s.cfg.Enabled,
		"chain_id":             s.cfg.ChainID,
		"denom":                s.cfg.Denom,
		"dispense_amount":      s.cfg.DispenseAmount,
		"max_requests_per_day": s.cfg.MaxPerDay,
	})
}

func (s *Server) policy(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":                  s.cfg.Enabled,
		"chain_id":                 s.cfg.ChainID,
		"denom":                    s.cfg.Denom,
		"dispense_amount":          s.cfg.DispenseAmount,
		"address_cooldown_minutes": int(s.cfg.AddressCooldown / time.Minute),
		"ip_cooldown_minutes":      int(s.cfg.IPCooldown / time.Minute),
		"max_requests_per_day":     s.cfg.MaxPerDay,
	})
}

type requestBody struct {
	Address string `json:"address"`
}

func (s *Server) request(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method_not_allowed"})
		return
	}
	if !s.cfg.Enabled {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "faucet_disabled"})
		return
	}

	var body requestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}
	address := strings.TrimSpace(body.Address)
	if err := validateAddress(s.cfg.Bech32Prefix, address); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_address", "detail": err.Error()})
		return
	}
	clientIP := extractClientIP(r)
	now := time.Now().UTC()
	dayKey := now.Format("2006-01-02")

	s.mu.Lock()
	defer s.mu.Unlock()

	if retryAt, ok := s.nextByAddr[address]; ok && now.Before(retryAt) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":       "address_rate_limited",
			"retry_after": retryAt.Format(time.RFC3339),
		})
		return
	}
	if retryAt, ok := s.nextByIP[clientIP]; ok && now.Before(retryAt) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":       "ip_rate_limited",
			"retry_after": retryAt.Format(time.RFC3339),
		})
		return
	}
	dailyKey := address + "|" + dayKey
	if s.dailyCount[dailyKey] >= s.cfg.MaxPerDay {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error": "daily_limit_reached",
		})
		return
	}

	txHash, err := s.sendTokens(r.Context(), address)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":  "faucet_send_failed",
			"detail": err.Error(),
		})
		return
	}

	s.nextByAddr[address] = now.Add(s.cfg.AddressCooldown)
	s.nextByIP[clientIP] = now.Add(s.cfg.IPCooldown)
	s.dailyCount[dailyKey]++

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"chain_id":       s.cfg.ChainID,
		"recipient":      address,
		"amount":         s.cfg.DispenseAmount,
		"denom":          s.cfg.Denom,
		"tx_hash":        txHash,
		"next_eligible":  s.nextByAddr[address].Format(time.RFC3339),
		"daily_requests": s.dailyCount[dailyKey],
	})
}

func (s *Server) sendTokens(ctx context.Context, to string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	amount := s.cfg.DispenseAmount + s.cfg.Denom
	cmd := exec.CommandContext(
		ctx,
		"/usr/local/bin/tokenchaind",
		"tx", "bank", "send",
		s.cfg.FromKey,
		to,
		amount,
		"--home", s.cfg.Home,
		"--keyring-backend", s.cfg.KeyringBackend,
		"--chain-id", s.cfg.ChainID,
		"--node", s.cfg.NodeRPC,
		"--yes",
		"--broadcast-mode", "sync",
		"--fees", s.cfg.Fees,
		"--output", "json",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(strings.TrimSpace(string(out)))
	}

	var resp struct {
		Code   uint32 `json:"code"`
		RawLog string `json:"raw_log"`
		TxHash string `json:"txhash"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", errors.New("could not decode tx response")
	}
	if resp.Code != 0 {
		if strings.TrimSpace(resp.RawLog) == "" {
			return "", errors.New("chain rejected transaction")
		}
		return "", errors.New(resp.RawLog)
	}
	if strings.TrimSpace(resp.TxHash) == "" {
		return "", errors.New("missing tx hash in response")
	}
	return resp.TxHash, nil
}

func validateAddress(prefix, address string) error {
	if !strings.HasPrefix(address, prefix+"1") {
		return errors.New("address prefix mismatch")
	}
	if !addrPattern.MatchString(address) {
		return errors.New("address format is invalid")
	}
	return nil
}

func extractClientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	xri := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return "unknown"
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
