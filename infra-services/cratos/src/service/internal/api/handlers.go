package api

import (
	"encoding/json"
	"net/http"
	"time"

	"servicegomodule/internal/models"
	"sharedgomodule/logging"
)

// Error message constants
const (
	ErrInvalidRequestBody  = "Invalid request body"
	ErrMissingSearchQuery  = "Missing search query"
	ErrSearchFailed        = "Search failed"
	ErrMethodNotAllowed    = "Method not allowed"
	ErrNotImplemented      = "Not implemented"
	ErrServiceNotAvailable = "User service not available"
)

// Success message constants
const (
	MsgStatsRetrieved  = "Statistics retrieved successfully"
	MsgConfigRetrieved = "Configuration retrieved successfully"
)

// API route constants
const (
	APIUsersPath = "/api/v1/users/"
)

// Handler holds the dependencies for API handlers
type Handler struct {
	logger logging.Logger
	// Any implementation specific variables to be added
}

// NewHandler creates a new Handler instance
func NewHandler(logger logging.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// SetupRoutes sets up the API routes
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", h.HealthCheck)

	mux.HandleFunc("/api/v1/stats", h.GetStats)
	mux.HandleFunc("/api/v1/config/", h.HandleConfigs)
}

// Helper functions for JSON responses and middleware

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// corsMiddleware handles CORS headers
func (h *Handler) corsMiddleware(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HealthCheck handler entry", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
	defer h.logger.Infow("HealthCheck handler exit", "method", r.Method, "path", r.URL.Path)

	// Apply CORS middleware
	h.corsMiddleware(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	health := &models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}
	writeJSON(w, http.StatusOK, health)
}

// GetStats handles statistics requests
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("GetStats handler entry", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
	defer h.logger.Infow("GetStats handler exit", "method", r.Method, "path", r.URL.Path)

	stats := map[string]interface{}{
		"total_messages": 0, // Stub implementation
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{
		Message: MsgStatsRetrieved,
		Data:    stats,
	})
}

// handles config related requests
func (h *Handler) HandleConfigs(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandleConfigs handler entry", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
	defer h.logger.Infow("HandleConfigs handler exit", "method", r.Method, "path", r.URL.Path)

	// Apply CORS middleware
	h.corsMiddleware(w, r)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case "GET":
		// Stub implementation for reading info
		data := []interface{}{} // Empty list for now
		writeJSON(w, http.StatusOK, models.SuccessResponse{
			Message: MsgConfigRetrieved,
			Data:    data,
		})
	default:
		h.logger.Warnw("Method not allowed", "method", r.Method, "path", r.URL.Path)
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{
			Error: ErrMethodNotAllowed,
		})
	}
}
