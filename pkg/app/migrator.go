package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/exgamer/gosdk-db-core/pkg/migration"
	"gorm.io/gorm"
)

// migrationLockID — идентификатор Postgres advisory lock, которым Migrator
// оборачивает накат/откат миграций. Защищает от гонки при параллельном
// вызове (например, если кто-то запустил накат и откат одновременно из
// разных мест) — держит лок тот, кто выполняет операцию, остальные ждут его
// до migrationLockWait. Postgres-специфично — поэтому живёт здесь, а не в
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

	err := withMigrationLock(m.db, func(conn *gorm.DB) error {
		return migration.Run(conn, m.migrations)
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

	err := withMigrationLock(m.db, func(conn *gorm.DB) error {
		return migration.RollbackLast(conn, m.migrations)
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

	err := withMigrationLock(m.db, func(conn *gorm.DB) error {
		return migration.RollbackTo(conn, m.migrations, migrationID)
	})
	if err != nil {
		return fmt.Errorf("rollback to %q: %w", migrationID, err)
	}

	return nil
}

// migrationLockWait — сколько ждать лок миграций, который держит другой
// процесс (накат на другом инстансе), прежде чем вернуть ошибку;
// migrationLockPoll — пауза между попытками. Переменные — для тестов.
var (
	migrationLockWait = 5 * time.Minute
	migrationLockPoll = 500 * time.Millisecond
)

// withMigrationLock берёт Postgres advisory lock и выполняет fn на одном и
// том же соединении (advisory lock принадлежит сессии: unlock на другом
// соединении пула лок не снимает), снимает лок и проверяет, что он был
// снят. Лок, занятый другим процессом, ждёт до migrationLockWait через
// pg_try_advisory_lock — без вечного ожидания.
func withMigrationLock(db *gorm.DB, fn func(conn *gorm.DB) error) error {
	return db.Connection(func(pinned *gorm.DB) (err error) {
		// новая сессия на закреплённом соединении: каждый запрос — со своим
		// Statement (экземпляр из Connection копит состояние между запросами)
		conn := pinned.Session(&gorm.Session{NewDB: true})

		if err := acquireMigrationLock(conn); err != nil {
			return err
		}

		defer func() {
			var unlocked bool

			unlockErr := conn.Raw("SELECT pg_advisory_unlock(?)", migrationLockID).Scan(&unlocked).Error
			if unlockErr == nil && !unlocked {
				unlockErr = errors.New("lock was not held by this session")
			}

			if unlockErr != nil {
				err = errors.Join(err, fmt.Errorf("release migration lock: %w", unlockErr))
			}
		}()

		return fn(conn)
	})
}

// acquireMigrationLock pg_try_advisory_lock с повтором раз в
// migrationLockPoll, пока не истечёт migrationLockWait.
func acquireMigrationLock(conn *gorm.DB) error {
	deadline := time.Now().Add(migrationLockWait)

	for {
		var locked bool

		if err := conn.Raw("SELECT pg_try_advisory_lock(?)", migrationLockID).Scan(&locked).Error; err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}

		if locked {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("acquire migration lock: held by another session for more than %s", migrationLockWait)
		}

		time.Sleep(migrationLockPoll)
	}
}
