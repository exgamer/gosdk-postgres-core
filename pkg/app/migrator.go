package app

import (
	"fmt"

	"github.com/exgamer/gosdk-db-core/pkg/migration"
	"gorm.io/gorm"
)

// migrationLockID — идентификатор Postgres advisory lock, которым Migrator
// оборачивает накат/откат миграций. Защищает от гонки при параллельном
// вызове (например, если кто-то запустил накат и откат одновременно из
// разных мест) — держит лок тот, кто выполняет операцию, остальные ждут на
// этом же SELECT. Postgres-специфично — поэтому живёт здесь, а не в
// dialect-agnostic gosdk-db-core/pkg/migration.
const migrationLockID = 72261 // произвольное, но стабильное число

// Migrator управляет накатом/откатом миграций схемы БД поверх переданного
// db. Не открывает и не закрывает соединение сам — какое именно подключение
// использовать (в проекте их может быть несколько), решает вызывающая
// сторона. Не связан с PostgresKernel.Init/Start/Stop и его DI-жизненным
// циклом. Предназначен для вызова из отдельной консольной команды (см.
// gosdk-console-core, MIGRATIONS.md в go-sdk-rest-template), а не из
// работающего сервиса.
type Migrator struct {
	db         *gorm.DB
	migrations []*migration.Migration
}

// NewMigrator создаёт Migrator для db (уже открытое соединение — например,
// database.InitPostgresGormConnection(dbConfig)) со списком миграций проекта
// (обычно — результат <project>/internal/migrations.All()).
func NewMigrator(db *gorm.DB, migrations ...*migration.Migration) *Migrator {
	return &Migrator{db: db, migrations: migrations}
}

// Up применяет все ещё не применённые миграции — то же самое, что раньше
// делал PostgresKernel.Init при старте сервиса.
func (m *Migrator) Up() error {
	if len(m.migrations) == 0 {
		return nil
	}

	err := withMigrationLock(m.db, func() error {
		return migration.Run(m.db, m.migrations)
	})
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// RollbackLast откатывает последнюю применённую миграцию.
func (m *Migrator) RollbackLast() error {
	if len(m.migrations) == 0 {
		return nil
	}

	err := withMigrationLock(m.db, func() error {
		return migration.RollbackLast(m.db, m.migrations)
	})
	if err != nil {
		return fmt.Errorf("rollback last migration: %w", err)
	}

	return nil
}

// RollbackTo откатывает все применённые миграции после migrationID (саму
// migrationID не откатывает — семантика gormigrate.RollbackTo).
func (m *Migrator) RollbackTo(migrationID string) error {
	if len(m.migrations) == 0 {
		return nil
	}

	err := withMigrationLock(m.db, func() error {
		return migration.RollbackTo(m.db, m.migrations, migrationID)
	})
	if err != nil {
		return fmt.Errorf("rollback to %q: %w", migrationID, err)
	}

	return nil
}

// withMigrationLock берёт Postgres advisory lock на db, выполняет fn и
// снимает лок по завершении.
func withMigrationLock(db *gorm.DB, fn func() error) error {
	if err := db.Exec("SELECT pg_advisory_lock(?)", migrationLockID).Error; err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer db.Exec("SELECT pg_advisory_unlock(?)", migrationLockID)

	return fn()
}
