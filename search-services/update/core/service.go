package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	publisher   Publisher
	concurrency int
	isRunning   atomic.Bool
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, publisher Publisher, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		publisher:   publisher,
		concurrency: concurrency,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	if !s.isRunning.CompareAndSwap(false, true) {
		return ErrUpdateRunning
	}
	defer s.isRunning.Store(false)

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last ID: %w", err)
	}

	existingIDs, err := s.db.IDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing IDs: %w", err)
	}

	existingSet := make(map[int]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}

	missingIDs := make([]int, 0)
	for id := 1; id <= lastID; id++ {
		if id == 404 {
			continue
		}
		if _, exists := existingSet[id]; !exists {
			missingIDs = append(missingIDs, id)
		}
	}

	if len(missingIDs) == 0 {
		return nil
	}

	jobs := make(chan int)
	go func() {
		for _, id := range missingIDs {
			jobs <- id
		}
		close(jobs)
	}()

	var wg sync.WaitGroup

	for i := 0; i < s.concurrency; i++ {
		wg.Go(func() {
			for id := range jobs {
				if ctx.Err() != nil {
					return
				}
				comic, err := s.xkcd.Get(ctx, id)
				if err != nil {
					s.log.Warn("Failed to fetch comic", "id", id, "error", err)
					continue
				}

				keywords, err := s.words.Norm(ctx, comic.Description)
				if err != nil {
					s.log.Warn("Failed to normalize words", "id", id, "error", err)
					continue
				}

				if err := s.db.Add(ctx, Comics{
					ID:    comic.ID,
					URL:   comic.URL,
					Words: keywords,
				}); err != nil {
					s.log.Warn("Failed to save comic", "id", id, "error", err)
				}
			}
		})
	}

	wg.Wait()
	s.publisher.PublishUpdated()
	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, err
	}

	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, err
	}

	comicsTotal := lastID
	if lastID >= 404 {
		comicsTotal--
	}
	return ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: comicsTotal,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	if s.isRunning.Load() {
		return StatusRunning
	}
	return StatusIdle
}

func (s *Service) Drop(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		return err
	}
	s.publisher.PublishDropped()
	return nil
}
