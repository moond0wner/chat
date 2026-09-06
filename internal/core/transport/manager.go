package core_transport

import (
	"context"
	"sync"
	"time"

	core_logger "github.com/moond0wner/chat/internal/core/logger"
	"go.uber.org/zap"
)

type Manager struct {
	servers []Server
	logger  *core_logger.Logger
	wg      sync.WaitGroup
}

func NewManager(logger *core_logger.Logger, servers ...Server) *Manager {
	return &Manager{
		servers: servers,
		logger:  logger,
		wg:      sync.WaitGroup{},
	}
}

func (m *Manager) StartAll(ctx context.Context) error {
	errCh := make(chan error, len(m.servers))

	for _, srv := range m.servers {
		m.wg.Add(1)
		go func(s Server) {
			defer m.wg.Done()
			if err := s.Start(ctx); err != nil {
				errCh <- err
			}
		}(srv)
	}
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return m.ShutdownAll()
	}
}

func (m *Manager) ShutdownAll() error {
	m.logger.Warn("Shutting down all servers...")
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	for _, srv := range m.servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			m.logger.Error("Shutdown error", zap.Error(err))
		}
	}
	m.wg.Wait()
	m.logger.Info("All servers stopped")
	return nil
}
