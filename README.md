# gosdk-postgres-core

`gosdk-postgres-core` — пакет для удобной и безопасной работы с PostgreSQL через **GORM** в рамках `gosdk-core` приложения.

## Возможности

- Registry подключений `PostgresGormRegistry`
- Ленивое создание `*gorm.DB`
- Singleflight при конкурентном доступе
- Корректное закрытие всех подключений
- Kernel для жизненного цикла приложения
- Helper-функции для бизнес-кода

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
