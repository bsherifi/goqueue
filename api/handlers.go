package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/betim/goqueue/queue"
)

// Handlers holds the dependencies needed by HTTP handlers.
type Handlers struct {
	Manager *queue.Manager
}

// Health handles GET /api/health.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Stats handles GET /api/stats.
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, h.Manager.Stats())
}

// ListJobs handles GET /api/jobs with optional ?status= filter.
func (h *Handlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	status := queue.Status(r.URL.Query().Get("status"))
	jobs := h.Manager.ListJobs(status)
	JSON(w, http.StatusOK, jobs)
}

// CreateJob handles POST /api/jobs.
func (h *Handlers) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Type == "" {
		ErrorResponse(w, http.StatusBadRequest, "type is required")
		return
	}

	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage(`{}`)
	}

	job, err := h.Manager.AddJob(req.Type, req.Payload)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusCreated, job)
}

// GetJob handles GET /api/jobs/{id}.
func (h *Handlers) GetJob(w http.ResponseWriter, r *http.Request) {
	id := extractJobID(r.URL.Path)
	if id == "" {
		ErrorResponse(w, http.StatusBadRequest, "job ID is required")
		return
	}

	job, err := h.Manager.GetJob(id)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	JSON(w, http.StatusOK, job)
}

// DeleteJob handles DELETE /api/jobs/{id}.
func (h *Handlers) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id := extractJobID(r.URL.Path)
	if id == "" {
		ErrorResponse(w, http.StatusBadRequest, "job ID is required")
		return
	}

	if err := h.Manager.DeleteJob(id); err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// RetryJob handles POST /api/jobs/{id}/retry.
func (h *Handlers) RetryJob(w http.ResponseWriter, r *http.Request) {
	// Path is /api/jobs/{id}/retry — strip the /retry suffix first
	path := strings.TrimSuffix(r.URL.Path, "/retry")
	id := extractJobID(path)
	if id == "" {
		ErrorResponse(w, http.StatusBadRequest, "job ID is required")
		return
	}

	job, err := h.Manager.RetryJob(id)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, job)
}

// JobByID routes GET and DELETE /api/jobs/{id} to the right handler.
func (h *Handlers) JobByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetJob(w, r)
	case http.MethodDelete:
		h.DeleteJob(w, r)
	default:
		ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Jobs routes GET and POST /api/jobs to the right handler.
func (h *Handlers) Jobs(w http.ResponseWriter, r *http.Request) {
	// Check if this is a /api/jobs/{id} or /api/jobs/{id}/retry request
	if id := extractJobID(r.URL.Path); id != "" {
		if strings.HasSuffix(r.URL.Path, "/retry") {
			h.RetryJob(w, r)
			return
		}
		h.JobByID(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.ListJobs(w, r)
	case http.MethodPost:
		h.CreateJob(w, r)
	default:
		ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// extractJobID pulls the job ID from a path like /api/jobs/{id}.
func extractJobID(path string) string {
	path = strings.TrimPrefix(path, "/api/jobs/")
	path = strings.TrimRight(path, "/")
	if path == "" || path == "/api/jobs" {
		return ""
	}
	return path
}
