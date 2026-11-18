package app

import (
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/exgamer/gosdk-postgres-core/pkg/config"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestRegistryGet_FactoryPanicReturnsError(t *testing.T) {
	reg := NewPostgresGormRegistry(func(*config.PostgresDbConfig) (*gorm.DB, error) {
		panic("boom")
	})
	reg.AddDefaultConnection(&config.PostgresDbConfig{})

	db, err := reg.GetDefaultConnection()
	if db != nil || err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Get = (%v, %v), want (nil, panic error)", db, err)
	}
}

func TestRegistryGet_FactoryPanicWaitersGetError(t *testing.T) {
	release := make(chan struct{})

	reg := NewPostgresGormRegistry(func(*config.PostgresDbConfig) (*gorm.DB, error) {
		<-release
		panic("boom")
	})
	reg.AddDefaultConnection(&config.PostgresDbConfig{})

	const callers = 8

	var wg sync.WaitGroup

	errs := make([]error, callers)
	dbs := make([]*gorm.DB, callers)

	for i := range callers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			dbs[i], errs[i] = reg.GetDefaultConnection()
		}()
	}

	close(release)
	wg.Wait()

	for i := range callers {
		if dbs[i] != nil || errs[i] == nil {
			t.Fatalf("caller %d: (%v, %v), want (nil, error)", i, dbs[i], errs[i])
		}
	}
}

func TestRegistryGet_FactoryNilWithoutError(t *testing.T) {
	reg := NewPostgresGormRegistry(func(*config.PostgresDbConfig) (*gorm.DB, error) {
		return nil, nil
	})
	reg.AddDefaultConnection(&config.PostgresDbConfig{})

	if db, err := reg.GetDefaultConnection(); db != nil || err == nil {
		t.Fatalf("Get = (%v, %v), want (nil, error)", db, err)
	}
}

func TestRegistryGet_ClosedDuringCreateReturnsError(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})

	reg := NewPostgresGormRegistry(func(*config.PostgresDbConfig) (*gorm.DB, error) {
		close(started)
		<-release

		return gorm.Open(tests.DummyDialector{}, &gorm.Config{})
	})
	reg.AddDefaultConnection(&config.PostgresDbConfig{})

	type res struct {
		db  *gorm.DB
		err error
	}

	got := make(chan res, 1)

	go func() {
		db, err := reg.GetDefaultConnection()
		got <- res{db, err}
	}()

	<-started

	closed := make(chan error, 1)
	go func() { closed <- reg.CloseAll() }()

	for !reg.IsClosing() {
		runtime.Gosched()
	}

	close(release)

	r := <-got
	if r.db != nil || r.err == nil {
		t.Fatalf("Get = (%v, %v), want (nil, closing error)", r.db, r.err)
	}

	if err := <-closed; err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
}
