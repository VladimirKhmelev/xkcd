package handlers

import (
	"bytes"
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

func (h *Handler) AdminPage(w http.ResponseWriter, r *http.Request) {
	token := h.getToken(r)
	data := map[string]any{
		"LoggedIn": token != "",
	}

	if token != "" {
		stats, err := h.getStats(r, token)
		if err != nil {
			data["StatsError"] = err.Error()
		} else {
			data["Stats"] = stats
		}
		status, err := h.getStatus(r, token)
		if err != nil {
			data["StatusError"] = err.Error()
		} else {
			data["Status"] = status
		}
	}

	if msg := r.URL.Query().Get("msg"); msg != "" {
		data["Message"] = msg
	}
	if errMsg := r.URL.Query().Get("err"); errMsg != "" {
		data["Error"] = errMsg
	}

	h.render(w, "admin.html", data)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	password := r.FormValue("password")

	token, err := h.login(name, password)
	if err != nil {
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Login failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})
	http.Redirect(w, r, "/admin?msg="+url.QueryEscape("Logged in successfully"), http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	token := h.getToken(r)
	if token == "" {
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Not authenticated"), http.StatusSeeOther)
		return
	}

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, h.apiAddress+"/api/db/update", nil)
	req.Header.Set("Authorization", "Token "+token)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Update request failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		http.Redirect(w, r, "/admin?msg="+url.QueryEscape("Update already running"), http.StatusSeeOther)
		return
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Update failed: "+strings.TrimSpace(string(body))), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?msg="+url.QueryEscape("Update started"), http.StatusSeeOther)
}

func (h *Handler) Drop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	token := h.getToken(r)
	if token == "" {
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Not authenticated"), http.StatusSeeOther)
		return
	}

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodDelete, h.apiAddress+"/api/db", nil)
	req.Header.Set("Authorization", "Token "+token)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Drop request failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Redirect(w, r, "/admin?err="+url.QueryEscape("Drop failed: "+strings.TrimSpace(string(body))), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?msg="+url.QueryEscape("Database dropped"), http.StatusSeeOther)
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

func (h *Handler) login(name, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name, "password": password})
	resp, err := h.httpClient.Post(h.apiAddress+"/api/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid credentials")
	}
	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(token)), nil
}

func (h *Handler) getStats(r *http.Request, token string) (map[string]int, error) {
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, h.apiAddress+"/api/db/stats", nil)
	req.Header.Set("Authorization", "Token "+token)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func (h *Handler) getStatus(r *http.Request, token string) (string, error) {
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, h.apiAddress+"/api/db/status", nil)
	req.Header.Set("Authorization", "Token "+token)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Status, nil
}

func (h *Handler) getToken(r *http.Request) string {
	c, err := r.Cookie("token")
	if err != nil {
		return ""
	}
	return c.Value
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		h.log.Error("template render error", "template", name, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
