package app

import (
	"context"
	"fmt"

	"github.com/exgamer/gosdk-core/pkg/app"
	"github.com/exgamer/gosdk-core/pkg/di"
	"github.com/exgamer/gosdk-db-core/pkg/migration"
	database "github.com/exgamer/gosdk-postgres-core/pkg/helpers"
	"gorm.io/gorm"
)

const DbKernelName = "postgres"

// migrationLockID — идентификатор Postgres advisory lock, которым
// runMigrations оборачивает прогон миграций. Защищает от гонки при
// параллельном старте нескольких инстансов сервиса (rolling deploy): держит
// лок инстанс, который применяет миграции, остальные ждут на этом же
// SELECT и после разблокировки видят уже применённое состояние.
// Postgres-специфично — поэтому живёт здесь, а не в dialect-agnostic
// gosdk-db-core/pkg/migration.
const migrationLockID = 72261 // произвольное, но стабильное число для этого kernel'а

type PostgresKernel struct {
	migrations []*migration.Migration
	reg        *PostgresGormRegistry
}

// WithMigrations регистрирует миграции, которые Init прогонит сразу после
// открытия соединения, до того как оно попадёт в DI. Опционально — без
// вызова ничего не меняется, PostgresKernel{} ведёт себя как раньше.
func (m *PostgresKernel) WithMigrations(migrations ...*migration.Migration) *PostgresKernel {
	m.migrations = migrations

	return m
}

func (m *PostgresKernel) Name() string {
	return DbKernelName
}

func (m *PostgresKernel) Init(a *app.App) error {
	dbConfig, err := database.InitPostgresDbConfig()

	if err != nil {
		return err
	}

	reg := NewPostgresGormRegistry(database.InitPostgresGormConnection)
	reg.AddDefaultConnection(dbConfig)

	if len(m.migrations) > 0 {
		db, err := reg.GetDefaultConnection()
		if err != nil {
			return err
		}

		if err := runMigrations(db, m.migrations); err != nil {
			return err
		}
	}

	di.Register(a.Container, reg)
	m.reg = reg

	return nil
}

func (m *PostgresKernel) Start(a *app.App) error {

	return nil
}

func (m *PostgresKernel) Stop(ctx context.Context) error {
	if m.reg == nil {
		return nil
	}

	return m.reg.CloseAll()
}

// runMigrations берёт advisory lock на db и применяет migrations через
// migration.Run. Пустой список — no-op, лок не берётся вообще. Вынесено из
// Init отдельной функцией, чтобы связку "lock → Run → unlock" можно было
// читать саму по себе.
func runMigrations(db *gorm.DB, migrations []*migration.Migration) error {
	if len(migrations) == 0 {
		return nil
	}

	if err := db.Exec("SELECT pg_advisory_lock(?)", migrationLockID).Error; err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer db.Exec("SELECT pg_advisory_unlock(?)", migrationLockID)

	if err := migration.Run(db, migrations); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
