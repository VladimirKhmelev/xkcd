package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
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

		comics, err := searcher.Search(r.Context(), phrase, limit)
		if err != nil {
			log.Error("search failed", "error", err)
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
