package httpapi

import (
	"encoding/json"
	"net/http"
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
