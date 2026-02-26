package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
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
	mux.HandleFunc("/v1/loyalty/merchant-routing", h.merchantRouting)
	mux.HandleFunc("/v1/admin/loyalty/merchant-routing", h.adminSetMerchantRouting)
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
			"healthz":                  "/healthz",
			"network":                  "/v1/network",
			"status":                   "/v1/status",
			"merchant_routing":         "/v1/loyalty/merchant-routing",
			"admin_merchant_routing":   "/v1/admin/loyalty/merchant-routing",
			"version":                  "/v1/version",
			"faucet":                   "see tokenchain-faucet service",
			"openapi":                  "n/a",
			"base_url":                 "",
			"source_chain_rest_status": h.cfg.RESTAddr,
		},
	})
}

type chainVerifiedToken struct {
	Denom                        string `json:"denom"`
	Name                         string `json:"name"`
	Symbol                       string `json:"symbol"`
	Verified                     bool   `json:"verified"`
	MaxSupply                    string `json:"max_supply"`
	MintedSupply                 string `json:"minted_supply"`
	MerchantIncentiveStakersBps  string `json:"merchant_incentive_stakers_bps"`
	MerchantIncentiveTreasuryBps string `json:"merchant_incentive_treasury_bps"`
}

type chainVerifiedTokenListResponse struct {
	Verifiedtoken []chainVerifiedToken `json:"verifiedtoken"`
	Pagination    struct {
		NextKey string `json:"next_key"`
	} `json:"pagination"`
}

type merchantRoutingItem struct {
	Denom                        string `json:"denom"`
	Name                         string `json:"name"`
	Symbol                       string `json:"symbol"`
	Verified                     bool   `json:"verified"`
	MaxSupply                    string `json:"max_supply"`
	MintedSupply                 string `json:"minted_supply"`
	MerchantIncentiveStakersBps  uint64 `json:"merchant_incentive_stakers_bps"`
	MerchantIncentiveTreasuryBps uint64 `json:"merchant_incentive_treasury_bps"`
	MerchantIncentiveStakersPct  string `json:"merchant_incentive_stakers_pct"`
	MerchantIncentiveTreasuryPct string `json:"merchant_incentive_treasury_pct"`
}

func (h *Handler) merchantRouting(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 25, 100)
	verifiedOnly := true
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("verified_only")), "false") {
		verifiedOnly = false
	}

	tokens, err := h.fetchVerifiedTokens(limit, verifiedOnly)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":     false,
			"error":  "failed to fetch merchant routing from chain rest",
			"rest":   h.cfg.RESTAddr,
			"detail": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"chain_id":    h.cfg.ChainID,
		"network":     h.cfg.Network,
		"count":       len(tokens),
		"items":       tokens,
		"generated":   time.Now().UTC().Format(time.RFC3339),
		"source_rest": h.cfg.RESTAddr,
	})
}

func (h *Handler) fetchVerifiedTokens(limit int, verifiedOnly bool) ([]merchantRoutingItem, error) {
	baseURL := strings.TrimRight(h.cfg.RESTAddr, "/") + "/tokenchain/loyalty/v1/verifiedtoken"
	client := &http.Client{Timeout: 6 * time.Second}
	nextKey := ""
	items := make([]merchantRoutingItem, 0, limit)

	for len(items) < limit {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}

		q := u.Query()
		q.Set("pagination.limit", "100")
		if nextKey != "" {
			q.Set("pagination.key", nextKey)
		}
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var out chainVerifiedTokenListResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&out)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("chain rest returned status %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		for _, token := range out.Verifiedtoken {
			if verifiedOnly && !token.Verified {
				continue
			}
			items = append(items, toMerchantRoutingItem(token))
			if len(items) >= limit {
				break
			}
		}

		if out.Pagination.NextKey == "" {
			break
		}
		nextKey = out.Pagination.NextKey
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Denom < items[j].Denom
	})

	return items, nil
}

func toMerchantRoutingItem(t chainVerifiedToken) merchantRoutingItem {
	stakers := parseBPS(t.MerchantIncentiveStakersBps)
	treasury := parseBPS(t.MerchantIncentiveTreasuryBps)

	// Backward compatibility for tokens created before routing fields existed.
	if stakers == 0 && treasury == 0 {
		stakers = 5000
		treasury = 5000
	}

	return merchantRoutingItem{
		Denom:                        t.Denom,
		Name:                         t.Name,
		Symbol:                       t.Symbol,
		Verified:                     t.Verified,
		MaxSupply:                    t.MaxSupply,
		MintedSupply:                 t.MintedSupply,
		MerchantIncentiveStakersBps:  stakers,
		MerchantIncentiveTreasuryBps: treasury,
		MerchantIncentiveStakersPct:  bpsToPercent(stakers),
		MerchantIncentiveTreasuryPct: bpsToPercent(treasury),
	}
}

func bpsToPercent(v uint64) string {
	return fmt.Sprintf("%.2f%%", float64(v)/100.0)
}

func parseBPS(raw string) uint64 {
	n, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	if n > 10000 {
		return 0
	}
	return n
}

func parseLimit(raw string, fallback, max int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}

type adminSetMerchantRoutingRequest struct {
	Denom                        string `json:"denom"`
	MerchantIncentiveStakersBps  uint64 `json:"merchant_incentive_stakers_bps"`
	MerchantIncentiveTreasuryBps uint64 `json:"merchant_incentive_treasury_bps"`
}

type txSyncResponse struct {
	Code    uint32 `json:"code"`
	TxHash  string `json:"txhash"`
	RawLog  string `json:"raw_log"`
	Info    string `json:"info"`
	Height  string `json:"height"`
	CodeSpa string `json:"codespace"`
}

func (h *Handler) adminSetMerchantRouting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		return
	}
	if strings.TrimSpace(h.cfg.AdminAPIToken) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "admin_api_disabled"})
		return
	}
	if !h.isAuthorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}

	var req adminSetMerchantRoutingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
		return
	}
	req.Denom = strings.TrimSpace(req.Denom)
	if req.Denom == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "denom_required"})
		return
	}
	if req.MerchantIncentiveStakersBps+req.MerchantIncentiveTreasuryBps != 10000 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":     false,
			"error":  "invalid_routing_split",
			"detail": "merchant incentive bps must total 10000",
		})
		return
	}

	txResp, err := h.submitMerchantRoutingTx(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok":     false,
			"error":  "tx_submission_failed",
			"detail": err.Error(),
		})
		return
	}
	if txResp.Code != 0 {
		detail := strings.TrimSpace(txResp.RawLog)
		if detail == "" {
			detail = "chain rejected transaction"
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":        false,
			"error":     "tx_rejected",
			"code":      txResp.Code,
			"codespace": txResp.CodeSpa,
			"tx_hash":   txResp.TxHash,
			"detail":    detail,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                              true,
		"tx_hash":                         txResp.TxHash,
		"height":                          txResp.Height,
		"denom":                           req.Denom,
		"merchant_incentive_stakers_bps":  req.MerchantIncentiveStakersBps,
		"merchant_incentive_treasury_bps": req.MerchantIncentiveTreasuryBps,
	})
}

func (h *Handler) isAuthorized(r *http.Request) bool {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(raw, "Bearer ") {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if token == "" {
		return false
	}
	expected := strings.TrimSpace(h.cfg.AdminAPIToken)
	if expected == "" {
		return false
	}
	if len(token) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (h *Handler) submitMerchantRoutingTx(ctx context.Context, req adminSetMerchantRoutingRequest) (*txSyncResponse, error) {
	args := []string{
		"tx", "loyalty", "set-merchant-incentive-routing",
		req.Denom,
		strconv.FormatUint(req.MerchantIncentiveStakersBps, 10),
		strconv.FormatUint(req.MerchantIncentiveTreasuryBps, 10),
		"--from", h.cfg.AdminFromKey,
		"--chain-id", h.cfg.ChainID,
		"--node", h.cfg.RPCAddr,
		"--home", h.cfg.ChainHome,
		"--keyring-backend", h.cfg.Keyring,
		"--yes",
		"--broadcast-mode", "sync",
		"--fees", h.cfg.TxFees,
		"--gas", h.cfg.TxGas,
		"-o", "json",
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		cmd := exec.CommandContext(ctx, h.cfg.Tokenchaind, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			lastErr = errors.New(msg)
			if strings.Contains(msg, "account sequence mismatch") && attempt < 2 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return nil, lastErr
		}

		var resp txSyncResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, errors.New("could not decode tx response")
		}

		// Handle chain-level sequence mismatch returned in JSON with non-zero code.
		if resp.Code != 0 && strings.Contains(resp.RawLog, "account sequence mismatch") && attempt < 2 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return &resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("failed to submit transaction")
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
