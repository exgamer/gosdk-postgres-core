# Changelog

## [1.0.23]

`POSTGRES_DB_*` из переменных окружения процесса (Docker, k8s, `docker run -e`) читаются
и без `.env`. Типы, поля и сигнатуры не изменились.

### Исправлено

- У полей `config.PostgresDbConfig` есть `json`-теги с теми же именами, что `mapstructure`
  (`json:"POSTGRES_DB_HOST"` и т. д.). Без них `InitConfig` из `gosdk-core` находил переменную
  в окружении, но не мог положить её в поле: применялись только ключи, которые есть в `.env`.
  Требуется `gosdk-core` v1.0.27+ (уже в `go.mod`).

### ⚠️ Изменения поведения

- Переменная `POSTGRES_DB_*`, которой нет в `.env`, но которая задана в окружении процесса,
  теперь применяется (раньше молча игнорировалась, поле оставалось пустым). Если `.env` нет совсем,
  конфиг целиком берётся из окружения. Для ключей, которые есть в `.env`, окружение было главнее
  и раньше.
- Неверное число или флаг в окружении (`POSTGRES_DB_MAX_OPEN_CONNECTIONS=abc`,
  `POSTGRES_DB_LOGGING=yes`) - ошибка `InitPostgresDbConfig` с именем переменной, `PostgresKernel`
  и консоль не стартуют (раньше ошибки не было: значение игнорировалось, для ключей из `.env`
  viper только печатал сообщение в stdout).
- Пустая переменная в окружении (`POSTGRES_DB_PASSWORD=`) теперь перекрывает значение из `.env`
  пустой строкой / `0` / `false` (раньше бралось значение из `.env`). Не задавайте пустые
  `POSTGRES_DB_*` в окружении, если значение должно прийти из `.env`.
- `json.Marshal(PostgresDbConfig)` выдаёт ключи `POSTGRES_DB_*` вместо имён полей (`Host`, `Port`…).
  В SDK и `go-sdk-rest-template` конфиг в JSON не сериализуется.
- Затронуты все места, где конфиг читается через `InitPostgresDbConfig` / `InitConfig`:
  `app.PostgresKernel` (через `newPostgresRegistry`), консоль `go-sdk-rest-template`
  (`internal/console/app.go`) и сервисы, созданные `gosdk-generator` с kernel `postgres`.
