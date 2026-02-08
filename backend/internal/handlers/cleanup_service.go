package handlers

import (
	"context"
	"log"
	"sync"
	"time"
)

// CleanupService runs session cleanup on a schedule.
type CleanupService struct {
	cleaner  *SessionCleaner
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
}

// NewCleanupService creates a new cleanup service.
func NewCleanupService(cleaner *SessionCleaner, interval time.Duration) *CleanupService {
	if interval <= 0 {
		interval = time.Hour
	}

	return &CleanupService{
		cleaner:  cleaner,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the cleanup loop in a background goroutine.
func (s *CleanupService) Start() {
	go s.run()
}

// Stop signals the cleanup loop to stop and waits for completion.
func (s *CleanupService) Stop(ctx context.Context) {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})

	select {
	case <-s.doneCh:
	case <-ctx.Done():
	}
}

func (s *CleanupService) run() {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	if err := s.cleaner.RunPeriodicCleanup(); err != nil {
		log.Printf("WARNING: Initial session cleanup failed: %v", err)
	}

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.cleaner.RunPeriodicCleanup(); err != nil {
				log.Printf("WARNING: Periodic session cleanup failed: %v", err)
			}
		}
	}
}
