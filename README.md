# XKCD Search

Микросервисная система для поиска комиксов [xkcd.com](https://xkcd.com) по фразе.

Комиксы загружаются из API xkcd, хранятся в PostgreSQL и индексируются для быстрого поиска. Индекс перестраивается автоматически через NATS при каждом обновлении базы. Результаты поиска кешируются в Redis.

## Архитектура

```
  браузер → ┌──────────┐
            │ frontend │ ← HTTP (порт 28090)
            └────┬─────┘
                 ▼
           ┌─────────┐
  клиент → │   api   │ ← HTTP REST (порт 28080)
           └────┬────┘
     ┌──────────┼──────────┐
     ▼          ▼          ▼
  ┌──────┐ ┌────────┐ ┌────────┐
  │words │ │ update │ │ search │  ← gRPC
  └──────┘ └───┬────┘ └───┬────┘
               │    NATS   │        ┌───────┐
               └─────┬─────┘  ← ────│ Redis │
                     ▼              └───────┘
                ┌──────────┐
                │ postgres │
                └──────────┘
```

| Сервис     | Роль |
|------------|------|
| `frontend` | Веб-интерфейс — поиск, регистрация и логин пользователей |
| `api`      | HTTP-шлюз — аутентификация (JWT), rate limiting, конкурентность, Swagger UI |
| `update`   | Загружает комиксы с xkcd.com, сохраняет в PostgreSQL, публикует события в NATS |
| `search`   | Строит поисковый индекс, кеширует результаты в Redis, подписан на NATS |
| `words`    | Токенизация и нормализация фраз (стоп-слова, стемминг) |

## Быстрый старт

```bash
docker compose up -d
```

| Адрес | Описание |
|-------|----------|
| `http://localhost:28090` | Веб-интерфейс (поиск) |
| `http://localhost:28090/register` | Регистрация пользователя |
| `http://localhost:28090/admin` | Панель администратора |
| `http://localhost:28080/swagger/index.html` | Swagger UI — интерактивная документация API |

Полный справочник — в [DOCS.md](DOCS.md).

## Запуск тестов

```bash
# Интеграционные тесты (поднимает все сервисы)
docker-compose run --rm tests

# Юнит-тесты + отчёт о покрытии
make unit
xdg-open cover.html
```

Артефакты CI после каждого прогона:
- **Actions → unit → Artifacts → coverage-report** — `cover.html`
- **Actions → integration → Artifacts → swagger-docs** — `swagger.json`, `swagger.yaml`

## Мониторинг

- **Grafana**: `http://localhost:3000`
- **VictoriaMetrics**: `http://localhost:8428`
- **pgAdmin**: `http://localhost:18888`

Готовый дашборд Grafana лежит в `metrics/dashboard.json` — импортируйте через **Dashboards → Import → Upload JSON file**.

## Стек технологий

**Backend**
- Go 1.25
- gRPC — межсервисное взаимодействие
- NATS — брокер событий (pub/sub)
- PostgreSQL — основное хранилище комиксов и пользователей
- Redis — кеш поисковых запросов (TTL 10 минут)
- JWT — аутентификация (admin и user роли)
- bcrypt — хеширование паролей
- golang-migrate — миграции БД

**API**
- REST HTTP — публичный API
- Swagger / OpenAPI 2.0 — документация и интерактивный UI

**Инфраструктура**
- Docker / Docker Compose
- GitHub Actions CI — lint, unit, coverage, integration
- VictoriaMetrics + Grafana — сбор метрик в Prometheus-формате и дашборды
