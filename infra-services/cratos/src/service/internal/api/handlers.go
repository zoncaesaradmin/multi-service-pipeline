package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"servicegomodule/internal/processing"
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
	ErrorMethodNotAllowed  = "Method not allowed"
	ErrorFailedToEncode    = "Failed to encode response"
)

// Constants for headers
const (
	HeaderContentType = "Content-Type"
	ContentTypeJSON   = "application/json"
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
	logger    logging.Logger
	collector *metrics.MetricsCollector
	pipeline  *processing.Pipeline
}

// NewHandler creates a new Handler instance
func NewHandler(logger logging.Logger, collector *metrics.MetricsCollector, pipeline *processing.Pipeline) *Handler {
	return &Handler{
		logger:    logger,
		collector: collector,
		pipeline:  pipeline,
	}
}

// SetupRoutes sets up the API routes
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("/health", h.HealthCheck)

	mux.HandleFunc("/api/v1/stats", h.GetStats)
	mux.HandleFunc("/api/v1/config/", h.HandleConfigs)

	// Add metrics routes if metrics collector is available
	mux.HandleFunc("/metrics", h.handleMetrics)
	mux.HandleFunc("/metrics/pipeline", h.handlePipelineMetrics)
	mux.HandleFunc("/metrics/summaries", h.handleSummaries)
	mux.HandleFunc("/metrics/events", h.handleEvents)
	mux.HandleFunc("/metrics/health", h.handleHealth)

	mux.HandleFunc("/api/debug/rules", h.GetRules)
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

	var stats map[string]interface{}
	if h.pipeline != nil {
		stats = h.pipeline.GetStats()
	} else {
		stats = map[string]interface{}{
			"error": "pipeline not initialized",
		}
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{
		Message: MsgStatsRetrieved,
		Data:    stats,
	})
}

// GetStats handles statistics requests
func (h *Handler) GetRules(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("GetRules handler entry", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)
	defer h.logger.Infow("GetRules handler exit", "method", r.Method, "path", r.URL.Path)

	var rBytes []byte
	if h.pipeline != nil {
		rBytes = h.pipeline.GetRuleInfo()
	} else {
		rBytes = []byte("pipeline not initialized")
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{
		Message: MsgStatsRetrieved,
		Data:    string(rBytes),
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

// handleMetrics returns all metrics data
func (api *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	metricsData := api.collector.GetMetrics()
	summaries := api.collector.GetSummaries()

	response := map[string]interface{}{
		"pipeline":  metricsData,
		"summaries": summaries,
		"timestamp": time.Now(),
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, ErrorFailedToEncode, http.StatusInternalServerError)
		return
	}
}

// handlePipelineMetrics returns pipeline-specific metrics
func (api *Handler) handlePipelineMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	metricsData := api.collector.GetMetrics()

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(metricsData); err != nil {
		http.Error(w, ErrorFailedToEncode, http.StatusInternalServerError)
		return
	}
}

// handleSummaries returns metric summaries
func (api *Handler) handleSummaries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	summaries := api.collector.GetSummaries()

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		http.Error(w, ErrorFailedToEncode, http.StatusInternalServerError)
		return
	}
}

// handleEvents returns recent metric events
func (api *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	events := api.collector.GetRecentEvents(limit)

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(events); err != nil {
		http.Error(w, ErrorFailedToEncode, http.StatusInternalServerError)
		return
	}
}

// handleHealth returns metrics system health
func (api *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, ErrorMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	summaries := api.collector.GetSummaries()
	metricsData := api.collector.GetMetrics()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"events":    len(api.collector.GetRecentEvents(1000)), // Get count via length
		"summaries": len(summaries),
		"pipeline":  metricsData,
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, ErrorFailedToEncode, http.StatusInternalServerError)
		return
	}
}
