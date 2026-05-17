package core

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
	cache Cache

	mu    sync.RWMutex
	index map[string][]Comics
}

func NewService(log *slog.Logger, db DB, words Words, cache Cache) *Service {
	return &Service{
		log:   log,
		db:    db,
		words: words,
		cache: cache,
		index: make(map[string][]Comics),
	}
}

func (s *Service) Search(ctx context.Context, phrase string, limit, page int) ([]Comics, int, error) {
	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to normalize phrase: %w", err)
	}
	if len(keywords) == 0 {
		return []Comics{}, 0, nil
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	cacheKey := fmt.Sprintf("search:%s:%d:%d", phrase, limit, page)
	if cached, ok, err := s.cache.Get(ctx, cacheKey); err == nil && ok {
		s.log.Debug("cache hit", "key", cacheKey)
		return cached.Comics, cached.Total, nil
	}

	comics, total, err := s.db.Search(ctx, keywords, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if err := s.cache.Set(ctx, cacheKey, CachedResult{Comics: comics, Total: total}); err != nil {
		s.log.Warn("cache set failed", "error", err)
	}
	return comics, total, nil
}

func (s *Service) ResetIndex() {
	s.mu.Lock()
	s.index = make(map[string][]Comics)
	s.mu.Unlock()
	if err := s.cache.Flush(context.Background()); err != nil {
		s.log.Warn("cache flush failed", "error", err)
	}
	s.log.Info("index reset")
}

func (s *Service) BuildIndex(ctx context.Context) error {
	all, err := s.db.AllComics(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch comics for index: %w", err)
	}
	newIndex := make(map[string][]Comics, len(all))
	for _, ic := range all {
		c := Comics{ID: ic.ID, URL: ic.URL}
		for _, kw := range ic.Keywords {
			newIndex[kw] = append(newIndex[kw], c)
		}
	}
	s.mu.Lock()
	s.index = newIndex
	s.mu.Unlock()
	s.log.Info("index built", "keywords", len(newIndex))
	return nil
}

func (s *Service) ISearch(ctx context.Context, phrase string, limit int) ([]Comics, error) {
	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize phrase: %w", err)
	}
	if len(keywords) == 0 {
		return []Comics{}, nil
	}

	s.mu.RLock()
	counts := make(map[int]int)
	seen := make(map[int]Comics)
	for _, kw := range keywords {
		for _, c := range s.index[kw] {
			counts[c.ID]++
			seen[c.ID] = c
		}
	}
	s.mu.RUnlock()

	type scored struct {
		c Comics
		n int
	}
	result := make([]scored, 0, len(counts))
	for id, n := range counts {
		result = append(result, scored{seen[id], n})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].n > result[j].n })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	comics := make([]Comics, len(result))
	for i, r := range result {
		comics[i] = r.c
	}
	return comics, nil
}
