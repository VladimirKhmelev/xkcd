package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	log        *slog.Logger
	apiAddress string
	templates  *template.Template
	httpClient *http.Client
}

func New(log *slog.Logger, apiAddress string, templates *template.Template) *Handler {
	return &Handler{
		log:        log,
		apiAddress: apiAddress,
		templates:  templates,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Handler) SearchPage(w http.ResponseWriter, r *http.Request) {
	phrase := r.URL.Query().Get("phrase")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	data := map[string]any{
		"Phrase": phrase,
		"Limit":  limit,
	}

	if phrase != "" {
		comics, total, err := h.search(r, phrase, limit)
		if err != nil {
			data["Error"] = err.Error()
		} else {
			data["Comics"] = comics
			data["Total"] = total
		}
	}

	h.render(w, "search.html", data)
}

type comicsItem struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

func (h *Handler) search(r *http.Request, phrase string, limit int) ([]comicsItem, int, error) {
	apiURL := fmt.Sprintf("%s/api/search?phrase=%s&limit=%d", h.apiAddress, url.QueryEscape(phrase), limit)
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, apiURL, nil)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("search error: %s", strings.TrimSpace(string(body)))
	}

	var result struct {
		Comics []comicsItem `json:"comics"`
		Total  int          `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse response: %w", err)
	}
	return result.Comics, result.Total, nil
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		h.log.Error("template render error", "template", name, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
