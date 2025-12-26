package app

import (
	"fmt"
	"sync"

	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/gorm"
)

const DefaultPostgresConnectionKey = "default"

type GormFactory func(cfg *config.PostgresDbConfig) (*gorm.DB, error)

type PostgresGormRegistry struct {
	mu      sync.RWMutex
	configs map[string]*config.PostgresDbConfig
	clients map[string]*gorm.DB
	factory GormFactory

	// singleflight per key (hand-rolled)
	inflight map[string]chan struct{}
	errs     map[string]error

	// shutdown gate
	closing bool
}

func NewPostgresGormRegistry(factory GormFactory) *PostgresGormRegistry {
	return &PostgresGormRegistry{
		configs:  make(map[string]*config.PostgresDbConfig),
		clients:  make(map[string]*gorm.DB),
		inflight: make(map[string]chan struct{}),
		errs:     make(map[string]error),
		factory:  factory,
	}
}

func (r *PostgresGormRegistry) AddDefaultConnection(cfg *config.PostgresDbConfig) {
	r.Add(DefaultPostgresConnectionKey, cfg)
}

func (r *PostgresGormRegistry) GetDefaultConnection() (*gorm.DB, error) {
	return r.Get(DefaultPostgresConnectionKey)
}

// Add registers/overrides config.
// You can choose stricter behavior: forbid overriding when client already exists.
func (r *PostgresGormRegistry) Add(name string, cfg *config.PostgresDbConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.configs[name] = cfg
}

// IsClosing indicates CloseAll() was called.
func (r *PostgresGormRegistry) IsClosing() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.closing
}

func (r *PostgresGormRegistry) Get(name string) (*gorm.DB, error) {
	// ---------- fast path (read lock) ----------
	r.mu.RLock()
	if r.closing {
		r.mu.RUnlock()
		return nil, fmt.Errorf("postgres registry is closing")
	}
	if db, ok := r.clients[name]; ok && db != nil {
		r.mu.RUnlock()
		return db, nil
	}
	// someone is already creating this connection — wait
	if ch, ok := r.inflight[name]; ok {
		r.mu.RUnlock()
		<-ch

		r.mu.RLock()
		db := r.clients[name]
		err := r.errs[name]
		closing := r.closing
		r.mu.RUnlock()

		if closing {
			return nil, fmt.Errorf("postgres registry is closing")
		}
		if err != nil {
			return nil, err
		}
		if db == nil {
			return nil, fmt.Errorf("postgres connection not created: %s", name)
		}
		return db, nil
	}
	r.mu.RUnlock()

	// ---------- upgrade to write lock to register inflight ----------
	r.mu.Lock()

	// re-check under write lock
	if r.closing {
		r.mu.Unlock()
		return nil, fmt.Errorf("postgres registry is closing")
	}
	if db, ok := r.clients[name]; ok && db != nil {
		r.mu.Unlock()
		return db, nil
	}
	if ch, ok := r.inflight[name]; ok {
		// someone raced with us; wait without holding locks
		r.mu.Unlock()
		<-ch

		r.mu.RLock()
		db := r.clients[name]
		err := r.errs[name]
		closing := r.closing
		r.mu.RUnlock()

		if closing {
			return nil, fmt.Errorf("postgres registry is closing")
		}
		if err != nil {
			return nil, err
		}
		if db == nil {
			return nil, fmt.Errorf("postgres connection not created: %s", name)
		}
		return db, nil
	}

	cfg, ok := r.configs[name]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("postgres connection config not found: %s", name)
	}

	ch := make(chan struct{})
	r.inflight[name] = ch
	r.mu.Unlock()

	// ---------- IMPORTANT: create without holding the registry lock ----------
	var (
		db  *gorm.DB
		err error
	)

	// Ensure inflight is always resolved (even on panic).
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic in gorm factory for %s: %v", name, rec)
		}

		// publish result (or close immediately if closing)
		r.mu.Lock()
		if r.closing {
			// do not publish; close created db if any
			if db != nil {
				if sqlDB, e := db.DB(); e == nil {
					_ = sqlDB.Close()
				}
			}
			r.errs[name] = fmt.Errorf("postgres registry is closing")
		} else if err != nil {
			r.errs[name] = err
		} else {
			r.clients[name] = db
			delete(r.errs, name)
		}

		close(ch)
		delete(r.inflight, name)
		r.mu.Unlock()
	}()

	db, err = r.factory(cfg)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (r *PostgresGormRegistry) CloseAll() error {
	// 1) flip closing gate (blocks new Get creations/returns)
	r.mu.Lock()
	r.closing = true

	// take inflight snapshot while locked
	waits := make([]chan struct{}, 0, len(r.inflight))
	for _, ch := range r.inflight {
		waits = append(waits, ch)
	}
	r.mu.Unlock()

	// 2) wait all inflight to resolve (so no one publishes after we snapshot)
	for _, ch := range waits {
		<-ch
	}

	// 3) snapshot clients + clear maps under write lock
	r.mu.Lock()
	clients := make(map[string]*gorm.DB, len(r.clients))
	for name, db := range r.clients {
		clients[name] = db
	}

	// clear; future Get will return "closing" until you create a new registry
	r.clients = make(map[string]*gorm.DB)
	r.errs = make(map[string]error)

	// (configs keep as is; optional)
	r.mu.Unlock()

	// 4) close connections without locks
	var firstErr error
	for name, gdb := range clients {
		if gdb == nil {
			continue
		}

		sqlDB, err := gdb.DB()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", name, err)
			}
			continue
		}

		if err := sqlDB.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", name, err)
		}
	}

	return firstErr
}
