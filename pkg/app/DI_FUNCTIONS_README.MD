##  Что доступно из DI после инициализации приложения

Основной коннект postgres:

```go
postgresConncetion, err := app.GetDefaultPostgresConnection(a *app.App) (*gorm.DB, error)
```

Добавить свой коннет к постгресу:

```go
err := app.AddPostgresConnection(a *app.App, name string, config *config.PostgresDbConfig) error
```

Получить свой коннет к постгресу:

```go
postgresConncetion, err := := GetPostgresConnection(a *app.App, name string) (*gorm.DB, error) 
```