##  Что доступно из DI после инициализации приложения

Основной коннект postgres:

```go
postgresConncetion, err := app.GetDefaultPostgresConnection(c *di.Container) (*gorm.DB, error)
```

Добавить свой коннет к постгресу:

```go
err := app.AddPostgresConnection(c *di.Containerp, name string, config *config.PostgresDbConfig) error
```

Получить свой коннет к постгресу:

```go
postgresConncetion, err := := GetPostgresConnection(c *di.Container, name string) (*gorm.DB, error) 
```