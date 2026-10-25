package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"harbor-scout-adapter/internal/converter"
	"harbor-scout-adapter/internal/scanner"
)

const (
	metadataMIME     = "application/vnd.scanner.adapter.metadata+json; version=1.0"
	scanResponseMIME = "application/vnd.scanner.adapter.scan.response+json; version=1.0"
)

type Handler struct {
	scanner *scanner.Service
}

func New(s *scanner.Service) http.Handler {
	h := &Handler{scanner: s}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/metadata", h.metadata)
	mux.HandleFunc("/api/v1/scan", h.scan)
	mux.HandleFunc("/api/v1/scan/", h.report)

	return mux
}

type scanRequest struct {
	Registry struct {
		URL string `json:"url"`
	} `json:"registry"`
	Artifact converter.Artifact `json:"artifact"`
}

func (h *Handler) metadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", metadataMIME)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"scanner": map[string]string{
			"name":    "Docker Scout",
			"vendor":  "Docker",
			"version": "1.27.0",
		},
		"capabilities": []interface{}{
			map[string]interface{}{
				"type": "vulnerability",
				"consumes_mime_types": []string{
					"application/vnd.oci.image.manifest.v1+json",
					"application/vnd.docker.distribution.manifest.v2+json",
				},
				"produces_mime_types": []string{
					converter.ReportMimeType,
				},
			},
		},
	})
}

func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid scan request", http.StatusBadRequest)
		return
	}

	if req.Registry.URL == "" ||
		req.Artifact.Repository == "" ||
		req.Artifact.Digest == "" {
		http.Error(w, "missing registry or artifact fields", http.StatusBadRequest)
		return
	}

	image, err := scanner.ImageReference(
		req.Registry.URL,
		req.Artifact,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.scanner.Submit(req.Artifact, image)

	if err != nil {
		http.Error(w, "scan simulation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", scanResponseMIME)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"id": id,
	})
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const prefix = "/api/v1/scan/"
	const suffix = "/report"

	path := strings.TrimPrefix(r.URL.Path, prefix)
	if !strings.HasSuffix(path, suffix) {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimSuffix(path, suffix)
	id = strings.TrimSuffix(id, "/")

	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	accept := r.Header.Get("Accept")
	if accept != "" &&
		accept != "*/*" &&
		!strings.Contains(accept, converter.ReportMimeType) {
		http.Error(w, "unsupported report format", http.StatusNotAcceptable)
		return
	}

	job, ok := h.scanner.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch job.Status {
	case scanner.Pending, scanner.Running:
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusFound)

	case scanner.Failed:
		http.Error(
			w,
			"scan failed",
			http.StatusInternalServerError,
		)

	case scanner.Completed:
		w.Header().Set(
			"Content-Type",
			converter.ReportMimeType,
		)
		writeJSON(w, http.StatusOK, job.Report)

	default:
		http.Error(
			w,
			"unknown job state",
			http.StatusInternalServerError,
		)
	}
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "JSON encoding failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, _ = io.WriteString(w, string(data))
}
