package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Emaren/tokenchain-indexer/internal/config"
	"github.com/Emaren/tokenchain-indexer/internal/version"
)

type Handler struct {
	cfg config.Config
}

func NewHandler(cfg config.Config) http.Handler {
	h := &Handler{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/v1/network", h.network)
	mux.HandleFunc("/v1/status", h.status)
	mux.HandleFunc("/v1/endpoints", h.endpoints)
	mux.HandleFunc("/v1/version", h.ver)
	return mux
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) network(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"chain_id": h.cfg.ChainID,
		"network":  h.cfg.Network,
		"rpc":      h.cfg.RPCAddr,
	})
}

type rpcStatusResponse struct {
	Result struct {
		NodeInfo struct {
			Moniker string `json:"moniker"`
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			CatchingUp        bool   `json:"catching_up"`
			LatestBlockHeight string `json:"latest_block_height"`
			LatestBlockTime   string `json:"latest_block_time"`
		} `json:"sync_info"`
	} `json:"result"`
}

func (h *Handler) status(w http.ResponseWriter, _ *http.Request) {
	statusURL := strings.TrimRight(h.cfg.RPCAddr, "/") + "/status"
	req, err := http.NewRequest(http.MethodGet, statusURL, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "failed to create status request",
		})
		return
	}

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": "failed to query chain rpc status",
			"rpc":   h.cfg.RPCAddr,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":         false,
			"error":      fmt.Sprintf("rpc status returned %d", resp.StatusCode),
			"rpc":        h.cfg.RPCAddr,
			"rpc_detail": string(body),
		})
		return
	}

	var out rpcStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":    false,
			"error": "failed to decode rpc status",
			"rpc":   h.cfg.RPCAddr,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                  true,
		"chain_id":            h.cfg.ChainID,
		"network":             h.cfg.Network,
		"rpc":                 h.cfg.RPCAddr,
		"moniker":             out.Result.NodeInfo.Moniker,
		"rpc_network":         out.Result.NodeInfo.Network,
		"catching_up":         out.Result.SyncInfo.CatchingUp,
		"latest_block_height": out.Result.SyncInfo.LatestBlockHeight,
		"latest_block_time":   out.Result.SyncInfo.LatestBlockTime,
	})
}

func (h *Handler) endpoints(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"chain_id": h.cfg.ChainID,
		"network":  h.cfg.Network,
		"rpc":      h.cfg.RPCAddr,
		"api": map[string]string{
			"healthz":  "/healthz",
			"network":  "/v1/network",
			"status":   "/v1/status",
			"version":  "/v1/version",
			"faucet":   "see tokenchain-faucet service",
			"openapi":  "n/a",
			"base_url": "",
		},
	})
}

func (h *Handler) ver(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version": version.Version,
		"commit":  version.Commit,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
