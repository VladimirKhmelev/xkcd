package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"context"

	"github.com/VictoriaMetrics/metrics"
	"yadro.com/course/api/core"
)

func NewMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	}
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		replies := make(map[string]string)
		for name, pinger := range pingers {
			if err := pinger.Ping(r.Context()); err != nil {
				replies[name] = "error: " + err.Error()
			} else {
				replies[name] = "ok"
			}
		}
		writeJSON(w, map[string]any{"replies": replies})
	}
}

type Authenticator interface {
	Login(user, password string) (string, error)
	LoginFromDB(name, password string, checkPassword func(name, password string) error) (string, error)
}

type UserChecker interface {
	CheckPassword(ctx context.Context, username, password string) error
}

type UserRegistrar interface {
	CreateUser(ctx context.Context, username, password string) error
}

func NewRegisterHandler(log *slog.Logger, storage UserRegistrar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Password == "" {
			http.Error(w, "name and password are required", http.StatusBadRequest)
			return
		}
		if err := storage.CreateUser(r.Context(), req.Name, req.Password); err != nil {
			log.Error("register failed", "error", err)
			http.Error(w, "user already exists or internal error", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func NewUserLoginHandler(log *slog.Logger, auth Authenticator, checker UserChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		token, err := auth.LoginFromDB(req.Name, req.Password, func(name, password string) error {
			return checker.CheckPassword(r.Context(), name, password)
		})
		if err != nil {
			log.Error("user login failed", "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(token)); err != nil {
			log.Error("failed to write token", "error", err)
		}
	}
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		token, err := auth.Login(req.Name, req.Password)
		if err != nil {
			log.Error("login failed", "error", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(token)); err != nil {
			log.Error("failed to write token", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Update(r.Context())
		if errors.Is(err, core.ErrAlreadyExists) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if err != nil {
			log.Error("update failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("stats failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]int{
			"words_total":    stats.WordsTotal,
			"words_unique":   stats.WordsUnique,
			"comics_fetched": stats.ComicsFetched,
			"comics_total":   stats.ComicsTotal,
		})
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := updater.Status(r.Context())
		if err != nil {
			log.Error("status failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": string(status)})
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Drop(r.Context()); err != nil {
			log.Error("drop failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func parsePageParams(r *http.Request) (limit, page int, err error) {
	limit = 10
	if ls := r.URL.Query().Get("limit"); ls != "" {
		limit, err = strconv.Atoi(ls)
		if err != nil || limit < 1 {
			return 0, 0, errors.New("invalid limit")
		}
	}
	page = 1
	if ps := r.URL.Query().Get("page"); ps != "" {
		page, err = strconv.Atoi(ps)
		if err != nil || page < 1 {
			return 0, 0, errors.New("invalid page")
		}
	}
	return limit, page, nil
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "phrase is required", http.StatusBadRequest)
			return
		}

		limit, page, err := parsePageParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		all, err := searcher.Search(r.Context(), phrase, 0)
		if err != nil {
			log.Error("search failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		total := len(all)
		pages := (total + limit - 1) / limit
		if pages == 0 {
			pages = 1
		}
		offset := (page - 1) * limit
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}
		comics := all[offset:end]

		type comicsItem struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		}
		items := make([]comicsItem, 0, len(comics))
		for _, c := range comics {
			items = append(items, comicsItem{ID: c.ID, URL: c.URL})
		}
		writeJSON(w, map[string]any{
			"comics": items,
			"total":  total,
			"page":   page,
			"pages":  pages,
		})
	}
}

func NewSearchIndexHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Error(w, "phrase is required", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			var err error
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit < 0 {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
		}

		comics, err := searcher.SearchIndex(r.Context(), phrase, limit)
		if err != nil {
			log.Error("isearch failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type comicsItem struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		}
		items := make([]comicsItem, 0, len(comics))
		for _, c := range comics {
			items = append(items, comicsItem{ID: c.ID, URL: c.URL})
		}
		writeJSON(w, map[string]any{
			"comics": items,
			"total":  len(items),
		})
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
