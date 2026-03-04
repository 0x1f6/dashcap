// Package api provides the dashcap REST API server.
package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"dashcap/internal/buffer"
	"dashcap/internal/config"
	"dashcap/internal/trigger"
)

// Server serves the dashcap REST API over HTTP.
type Server struct {
	cfg        *config.Config
	ring       *buffer.RingManager
	dispatcher *trigger.Dispatcher
	startTime  time.Time
	srv        *http.Server
}

// New creates a Server. Call ListenAndServe to start accepting requests.
func New(cfg *config.Config, ring *buffer.RingManager, dispatcher *trigger.Dispatcher) *Server {
	s := &Server{
		cfg:        cfg,
		ring:       ring,
		dispatcher: dispatcher,
		startTime:  time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	mux.HandleFunc("POST /api/v1/trigger", s.handleTrigger)
	mux.HandleFunc("GET /api/v1/triggers", s.handleTriggers)
	mux.HandleFunc("GET /api/v1/ring", s.handleRing)

	var handler = logMiddleware(mux)
	if cfg.APIToken != "" {
		handler = authMiddleware(cfg.APIToken, handler)
	}

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.APIPort),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s
}

// ListenAndServe starts the HTTP server. If TLS cert/key are configured,
// it serves HTTPS; otherwise plain HTTP. Blocks until Shutdown is called.
func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("api listen %s: %w", s.srv.Addr, err)
	}
	if s.cfg.TLSCert != "" && s.cfg.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(s.cfg.TLSCert, s.cfg.TLSKey)
		if err != nil {
			return fmt.Errorf("load tls cert/key: %w", err)
		}
		s.srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		return s.srv.ServeTLS(l, "", "")
	}
	return s.srv.Serve(l)
}

// Serve accepts connections on the provided listener. Useful for testing.
func (s *Server) Serve(l net.Listener) error {
	return s.srv.Serve(l)
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// handleHealth responds with a simple liveness check.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleStatus returns instance status and ring buffer summary.
func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	segs := s.ring.Segments()
	var totalPackets, totalBytes int64
	for _, seg := range segs {
		totalPackets += seg.Packets
		totalBytes += seg.Bytes
	}
	resp := map[string]any{
		"interface":     s.cfg.Interface,
		"uptime":        time.Since(s.startTime).Round(time.Second).String(),
		"segment_count": len(segs),
		"total_packets": totalPackets,
		"total_bytes":   totalBytes,
	}
	writeJSON(w, http.StatusOK, resp)
}

// triggerRequest is the optional JSON body for POST /api/v1/trigger.
type triggerRequest struct {
	Duration string `json:"duration,omitempty"`
	Since    string `json:"since,omitempty"`
}

// handleTrigger initiates a save of the capture window.
func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	var req triggerRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
			return
		}
	}

	if req.Duration != "" && req.Since != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "duration and since are mutually exclusive"})
		return
	}

	var opts trigger.TriggerOpts

	if req.Duration != "" {
		d, err := time.ParseDuration(req.Duration)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid duration: " + err.Error()})
			return
		}
		if d <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "duration must be positive"})
			return
		}
		opts.Duration = &d
	}

	if req.Since != "" {
		t, err := time.Parse(time.RFC3339, req.Since)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid since timestamp (expected RFC 3339): " + err.Error()})
			return
		}
		if t.After(time.Now()) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "since must not be in the future"})
			return
		}
		opts.Since = &t
	}

	rec, err := s.dispatcher.Trigger("api", opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, rec)
}

// handleTriggers lists recent trigger records.
func (s *Server) handleTriggers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.dispatcher.History())
}

// handleRing returns per-segment metadata for the ring buffer.
func (s *Server) handleRing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.ring.Segments())
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}

// logMiddleware logs each API request at info level.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("api request", "method", r.Method, "path", r.URL.Path, "status", sw.code)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
