# Миграции

`PostgresKernel` умеет прогонять версионированные миграции схемы при старте
приложения — до того, как соединение попадёт в DI и до того, как остальные
kernels (HTTP, Rabbit) начнут принимать трафик.

Сам механизм прогона (`Migration`, `Run`) — в
[`gosdk-db-core/pkg/migration`](https://github.com/exgamer/gosdk-db-core)
и полностью не знает про Postgres. Всё, что специфично для Postgres (advisory
lock от гонки при параллельном старте нескольких инстансов), — здесь, в
`PostgresKernel`.

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
использовать GORM-стиль (`tx.AutoMigrate(&model{})`).

**3. Подключить в `app.go`** — один раз на проект:

```go
import "myproject/internal/migrations"

appInstance.RegisterAndInitKernels(
	(&postgres.PostgresKernel{}).WithMigrations(migrations.All()...),
	&http.HttpKernel{},
	...
)
```

Дальше при каждом старте приложения `PostgresKernel.Init()`:

1. открывает соединение;
2. берёт `pg_advisory_lock` — безопасно, если поднимается несколько
   инстансов сразу (rolling deploy): миграции применит только один, остальные
   дождутся и увидят уже готовый результат;
3. применяет все ещё не применённые миграции по порядку (таблица
   `migrations` в БД хранит историю — уже применённые повторно не выполняются);
4. снимает лок, регистрирует соединение в DI.

Только после этого стартуют остальные kernels — миграции гарантированно
готовы раньше, чем в приложение придёт первый запрос.

## Добавление следующей миграции

Просто повторить шаг 1 (`codegen migration add ...`) и шаг 2 (заполнить
тело) — `app.go` трогать не нужно, `migrations.All()` уже содержит новую
запись.
