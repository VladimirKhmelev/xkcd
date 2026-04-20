# XKCD Search

Микросервисная система для поиска комиксов [xkcd.com](https://xkcd.com) по фразе.

Комиксы загружаются из API xkcd, хранятся в PostgreSQL и индексируются для быстрого поиска. Индекс перестраивается автоматически через NATS при каждом обновлении базы.

## Архитектура

```
           ┌─────────┐
  клиент → │   api   │ ← HTTP REST (порт 28080)
           └────┬────┘
     ┌──────────┼──────────┐
     ▼          ▼          ▼
  ┌──────┐ ┌────────┐ ┌────────┐
  │words │ │ update │ │ search │  ← gRPC
  └──────┘ └───┬────┘ └───┬────┘
               │    NATS   │
               └─────┬─────┘
                     ▼
                ┌──────────┐
                │ postgres │
                └──────────┘
```

| Сервис   | Роль |
|----------|------|
| `api`    | HTTP-шлюз — аутентификация, rate limiting, ограничение конкурентности |
| `update` | Загружает комиксы с xkcd.com, сохраняет в PostgreSQL, публикует события в NATS |
| `search` | Строит и опрашивает поисковый индекс; подписан на NATS для перестройки индекса |
| `words`  | Токенизация и нормализация фраз (стоп-слова, стемминг) |

## Быстрый старт

```bash
docker compose up -d
```

API доступен по адресу `http://localhost:28080`. Полный справочник — в [DOCS.md](DOCS.md).

## Запуск тестов

```bash
# Быстрые smoke-тесты (без побочных эффектов)
docker compose run --rm tests go test -v -run "TestPreflight|TestPing|TestPingAllServices" ./...

# Полный интеграционный прогон
docker compose run --rm tests go test -v -timeout 30m ./...
```

## Мониторинг

- **Grafana**: `http://localhost:3000`
- **VictoriaMetrics**: `http://localhost:8428`
- **pgAdmin**: `http://localhost:18888`
