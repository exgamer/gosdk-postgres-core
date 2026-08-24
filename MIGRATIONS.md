# Миграции

`Migrator` (`pkg/app/migrator.go`) применяет и откатывает версионированные
миграции схемы поверх переданного `*gorm.DB`. Это **ручная** операция —
`PostgresKernel` про миграции ничего не знает и сам их не трогает: он только
открывает соединение и регистрирует его в DI при старте приложения. Накат и
откат — отдельная консольная команда проекта, вызываемая явно (деплой-пайплайн,
oncall), а не часть обычного запуска сервиса.

Сам механизм прогона (`Migration`, `Run`, `RollbackLast`, `RollbackTo`) — в
[`gosdk-db-core/pkg/migration`](https://github.com/exgamer/gosdk-db-core) и
полностью не знает про Postgres. Всё, что специфично для Postgres (advisory
lock от гонки при параллельном вызове), — здесь, в `Migrator`.

`Migrator` не открывает и не закрывает соединение сам — принимает готовый
`*gorm.DB` в `NewMigrator(db, migrations...)`. Осознанно: в проекте может
быть несколько Postgres-клиентов, и какой из них передать (а если мигрируют
несколько БД — под каждую свой `Migrator`), решает вызывающий код.

## Быстрый старт

**1. Сгенерировать миграцию** (через
[`gosdk-generator`](https://github.com/exgamer/go-sdk-generator)):

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@latest \
    migration add handbook/city create_city
```

Создаёт `internal/migrations/<timestamp>_handbook_city_create_city.go` и
создаёт/дополняет `internal/migrations/registry.go` (`All()` — список всех
миграций проекта по порядку). Повторные вызовы только дописывают в `All()`,
уже существующие файлы не трогают.

**2. Заполнить тело миграции** — генератор оставляет заготовку:

```go
func migration20260824120000() *migration.Migration {
	return &migration.Migration{
		ID: "20260824120000_handbook_city_create_city",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`CREATE TABLE city (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100),
				status INT DEFAULT 0
			)`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP TABLE city`).Error
		},
	}
}
```

`tx` — обычный `*gorm.DB`: можно писать сырой SQL (`tx.Exec(...)`) или
использовать GORM-стиль (`tx.AutoMigrate(&model{})`). Пишите `Rollback`
всегда, даже если не планируете откатывать конкретную миграцию — без него
`RollbackTo` не сможет пройти дальше неё.

**3. Подключить в консольную команду проекта** — один раз, например
`cmd/console/main.go` (если консоль построена на
[`gosdk-console-core`](https://github.com/exgamer/gosdk-console-core)):

```go
import "myproject/internal/migrations"

dbConfig, err := database.InitPostgresDbConfig()
...
db, err := database.InitPostgresGormConnection(dbConfig)
...
defer closeConnection(db) // sqlDB, _ := db.DB(); sqlDB.Close()

migrator := postgres.NewMigrator(db, migrations.All()...)
```

Дальше при вызове:

```go
migrator.Up()               // применить все ещё не применённые миграции
migrator.RollbackLast()     // откатить последнюю применённую
migrator.RollbackTo(id)     // откатить всё после id (саму id не откатывает)
```

каждый вызов:

1. берёт `pg_advisory_lock` — безопасно, если несколько вызовов пересекутся
   по времени (два CI job'а, два оператора и т.д.): выполнит только один,
   остальные дождутся и увидят уже готовый результат;
2. применяет/откатывает миграции по порядку (таблица `migrations` в БД
   хранит историю — уже применённые повторно не выполняются);
3. снимает лок.

Соединением (`db`) `Migrator` не управляет — открыть и закрыть его должен
вызывающий код (`main.go`).

## Добавление следующей миграции

Просто повторить шаг 1 (`codegen migration add ...`) и шаг 2 (заполнить
тело) — код консольной команды трогать не нужно, `migrations.All()` уже
содержит новую запись.
