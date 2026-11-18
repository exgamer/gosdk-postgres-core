package app

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/exgamer/gosdk-db-core/pkg/migration"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockPostgres(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	return db, mock
}

func shortLockWait(t *testing.T, wait time.Duration) {
	t.Helper()

	prevWait, prevPoll := migrationLockWait, migrationLockPoll
	migrationLockWait, migrationLockPoll = wait, 10*time.Millisecond

	t.Cleanup(func() { migrationLockWait, migrationLockPoll = prevWait, prevPoll })
}

func lockRows(v bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"locked"}).AddRow(v)
}

func TestWithMigrationLock_RetriesThenUnlocks(t *testing.T) {
	shortLockWait(t, time.Second)

	db, mock := newMockPostgres(t)

	mock.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(migrationLockID).WillReturnRows(lockRows(false))
	mock.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(migrationLockID).WillReturnRows(lockRows(true))
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT pg_advisory_unlock").WithArgs(migrationLockID).WillReturnRows(lockRows(true))

	err := withMigrationLock(db, func(conn *gorm.DB) error { return conn.Exec("SELECT 1").Error })
	if err != nil {
		t.Fatalf("withMigrationLock: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithMigrationLock_UnlockResultChecked(t *testing.T) {
	db, mock := newMockPostgres(t)

	mock.ExpectQuery("SELECT pg_try_advisory_lock").WillReturnRows(lockRows(true))
	mock.ExpectQuery("SELECT pg_advisory_unlock").WillReturnRows(lockRows(false))

	err := withMigrationLock(db, func(*gorm.DB) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "release migration lock") {
		t.Fatalf("err = %v, want release error", err)
	}
}

func TestWithMigrationLock_FnErrorAndUnlockError(t *testing.T) {
	db, mock := newMockPostgres(t)

	fnErr := errors.New("migration failed")

	mock.ExpectQuery("SELECT pg_try_advisory_lock").WillReturnRows(lockRows(true))
	mock.ExpectQuery("SELECT pg_advisory_unlock").WillReturnError(errors.New("conn lost"))

	err := withMigrationLock(db, func(*gorm.DB) error { return fnErr })
	if !errors.Is(err, fnErr) || !strings.Contains(err.Error(), "conn lost") {
		t.Fatalf("err = %v, want both errors", err)
	}
}

func TestWithMigrationLock_WaitDeadline(t *testing.T) {
	shortLockWait(t, 50*time.Millisecond)

	db, mock := newMockPostgres(t)
	mock.MatchExpectationsInOrder(false)

	for range 100 {
		mock.ExpectQuery("SELECT pg_try_advisory_lock").WillReturnRows(lockRows(false))
	}

	called := false

	err := withMigrationLock(db, func(*gorm.DB) error {
		called = true

		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "held by another session") || called {
		t.Fatalf("err = %v, called = %v", err, called)
	}
}

// TestMigrator_PostgresLockOnOneSession Накат на настоящем Postgres: лок снят (на том же соединении), миграции
// применены, параллельный накат ждёт, занятый лок — ошибка по дедлайну. POSTGRES_TEST_DSN — DSN пустой тестовой БД;
// без него тест пропускается
func TestMigrator_PostgresLockOnOneSession(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set")
	}

	shortLockWait(t, 2*time.Second)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() {
		_ = db.Exec("DROP TABLE IF EXISTS migrator_probe, migrations").Error
		_ = sqlDB.Close()
	})

	migrations := []*migration.Migration{{
		ID: "001_probe",
		Migrate: func(tx *gorm.DB) error {
			time.Sleep(200 * time.Millisecond)

			return tx.Exec("CREATE TABLE migrator_probe (id int)").Error
		},
		Rollback: func(tx *gorm.DB) error { return tx.Exec("DROP TABLE migrator_probe").Error },
	}}

	var wg sync.WaitGroup

	errs := make([]error, 3)

	for i := range errs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			errs[i] = NewMigrator(db, migrations...).Up()
		}()
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("Up %d: %v", i, err)
		}
	}

	if held := advisoryLocksHeld(t, db); held != 0 {
		t.Fatalf("advisory locks held after Up: %d", held)
	}

	if err := db.Exec("SELECT 1 FROM migrator_probe").Error; err != nil {
		t.Fatalf("migration not applied: %v", err)
	}

	// лок держит другая сессия — Up ждёт дедлайн и возвращает ошибку
	err = db.Connection(func(other *gorm.DB) error {
		if err := other.Exec("SELECT pg_advisory_lock(?)", migrationLockID).Error; err != nil {
			return err
		}

		defer other.Exec("SELECT pg_advisory_unlock(?)", migrationLockID)

		upErr := NewMigrator(db, migrations...).RollbackLast()
		if upErr == nil || !strings.Contains(upErr.Error(), "held by another session") {
			t.Errorf("RollbackLast with foreign lock: %v", upErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}

	if err := NewMigrator(db, migrations...).RollbackLast(); err != nil {
		t.Fatalf("RollbackLast: %v", err)
	}

	if held := advisoryLocksHeld(t, db); held != 0 {
		t.Fatalf("advisory locks held after RollbackLast: %d", held)
	}
}

func advisoryLocksHeld(t *testing.T, db *gorm.DB) int64 {
	t.Helper()

	var n int64
	if err := db.Raw("SELECT count(*) FROM pg_locks WHERE locktype = 'advisory' AND objid = ?", migrationLockID).Scan(&n).Error; err != nil {
		t.Fatalf("pg_locks: %v", err)
	}

	return n
}
