# gosdk-postgres-core

`gosdk-postgres-core` — пакет для удобной и безопасной работы с PostgreSQL через **GORM** в рамках  `gosdk-db-core` `gosdk-core` приложения.

- [Документация GOSDK-CORE](https://github.com/exgamer/gosdk-core)
- [Документация GOSDK-DB-CORE](https://github.com/exgamer/gosdk-db-core)


- 🧩 **Dependency Injection**
    - [Что доступно в DI из коробки](pkg/di/DI_FUNCTIONS_README.MD)
- 🧱 **Миграции**
    - [Как подключить и использовать](MIGRATIONS.md)

## Возможности

- Registry подключений `PostgresGormRegistry`
- Ленивое создание `*gorm.DB`
- Singleflight при конкурентном доступе
- Корректное закрытие всех подключений
- Kernel для жизненного цикла приложения
- Миграции схемы (`Migrator`, ручной накат/откат через консоль проекта, см. [MIGRATIONS.md](MIGRATIONS.md))
- Helper-функции для бизнес-кода

## Конфигурация

Переменные `POSTGRES_DB_*` (`config.PostgresDbConfig`) читаются из окружения процесса и из `.env`;
окружение главнее `.env`. Неверное число или флаг в окружении - ошибка старта с именем переменной.

TLS - `POSTGRES_DB_SSLMODE`: `prefer` (по умолчанию: TLS, если сервер умеет, иначе без него), `disable`, `allow`,
`require`, `verify-ca`, `verify-full`; другое значение - ошибка подключения. Значения DSN (пароль с пробелом,
кавычкой, `\`) экранируются. Подключение ждёт сервер не дольше 5 с (`connect_timeout=5`; `PGCONNECT_TIMEOUT` в
окружении - его значение).

Пул: `POSTGRES_DB_MAX_OPEN_CONNECTIONS` (не задан - 25), `POSTGRES_DB_MAX_IDLE_CONNECTIONS` (не задан - 10), см.
`gosdk-db-core`.

При `disable`, `allow`, `prefer` (и пустом `POSTGRES_DB_SSLMODE`) и `APP_ENV` не `local` (пустой и неизвестный - тоже
не `local`) один раз пишется WARN (при старте `PostgresKernel` или первом подключении): соединение может быть без
шифрования. Вне local задать `require`, `verify-ca` или `verify-full`.

Пароль не выводится: `fmt` (`%v`, `%+v`, `%#v`) и `json.Marshal` конфига показывают `POSTGRES_DB_PASSWORD` как `***`
(пустой - пустым).

## Логирование запросов

`POSTGRES_DB_LOG_LEVEL` и `POSTGRES_DB_LOGGING` не применяются: лог запросов GORM задаёт `LOG_LEVEL`
(`gosdk-db-core`). Значения ниже - для совместимости конфигов
- "info"
- "errors"
- "warnings"


## Установка

```bash
go get github.com/exgamer/gosdk-postgres-core
```

## Быстрый старт

### Подключение kernel

```go
a := app.NewApp()
_ = a.RegisterKernel(&app.PostgresKernel{})
```

### Получение подключения

```go
db, err := app.GetDefaultPostgresConnection(a)
if err != nil {
    return err
}
```

### Именованное подключение

```go
db, err := app.GetPostgresConnection(a, "analytics")
```

### Добавление подключения

```go
cfg := &config.PostgresDbConfig{}
_ = app.AddPostgresConnection(a, "analytics", cfg)
```

## Shutdown

При остановке приложения автоматически вызывается `CloseAll()` и все соединения закрываются.

## License

MIT
