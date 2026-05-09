# Документация

## Веб-интерфейс

Фронтенд доступен по адресу `http://localhost:28090`.

### Поиск

Введите фразу, выберите количество результатов (10 / 20 / 50 / 100) и нажмите **Search**. Результаты отображаются в виде сетки картинок со ссылками на оригинальные страницы xkcd.com.

### Регистрация и вход

Доступны по адресам `http://localhost:28090/register` и `http://localhost:28090/login`.

Зарегистрированный пользователь может выполнять поиск через веб-интерфейс. Пароли хранятся в PostgreSQL в виде bcrypt-хешей.

### Панель администратора

Доступна по адресу `http://localhost:28090/admin`. Требует авторизации (логин и пароль задаются через переменные `ADMIN_USER` / `ADMIN_PASSWORD` в docker-compose, по умолчанию `admin` / `password`).

После входа доступны:

- **Stats** — количество загруженных комиксов и слов в базе
- **Status** — текущий статус обновления (`idle` / `running`)
- **Update Database** — запустить загрузку новых комиксов с xkcd.com
- **Drop Database** — очистить базу (с подтверждением)

---

## Запуск системы

```bash
docker compose up -d
```

Все сервисы поднимаются автоматически. API доступен по адресу `http://localhost:28080`.

---

## Аутентификация

В системе два типа пользователей:

| Тип | Как создать | Может |
|-----|------------|-------|
| Admin | Через env `ADMIN_USER` / `ADMIN_PASSWORD` | Всё — update, drop, search |
| User | Через `POST /api/register` | Только поиск |

### Логин администратора

```bash
curl -s -X POST http://localhost:28080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"name":"admin","password":"password"}'
```

### Регистрация и логин пользователя

```bash
# Регистрация
curl -s -X POST http://localhost:28080/api/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"vasya","password":"mypassword"}'

# Логин
curl -s -X POST http://localhost:28080/api/user/login \
  -H 'Content-Type: application/json' \
  -d '{"name":"vasya","password":"mypassword"}'
```

Полученный токен передаётся в заголовке:

```
Authorization: Token <токен>
```

TTL токена — 2 минуты по умолчанию (настраивается через переменную `TOKEN_TTL`).

---

## Эндпоинты

### Статус сервисов

```
GET /api/ping
```

Возвращает состояние всех внутренних сервисов.

```json
{
  "replies": {
    "words":  "ok",
    "update": "ok",
    "search": "ok"
  }
}
```

---

### База данных

#### Обновить базу комиксов

Загружает новые комиксы с xkcd.com и сохраняет их в базу. Требует токен.

```bash
curl -X POST http://localhost:28080/api/db/update \
  -H "Authorization: Token <токен>"
```

Возвращает `200 OK` по завершении, `202 Accepted` если обновление уже выполняется.

#### Статистика

```
GET /api/db/stats
```

```json
{
  "comics_total":   3000,
  "comics_fetched": 3000,
  "words_total":    120000,
  "words_unique":   8000
}
```

#### Статус обновления

```
GET /api/db/status
```

```json
{"status": "idle"}
```

Возможные значения: `idle` (ожидание), `running` (идёт обновление).

#### Очистить базу

Удаляет все комиксы из базы. Требует токен.

```bash
curl -X DELETE http://localhost:28080/api/db \
  -H "Authorization: Token <токен>"
```

---

### Поиск

#### Полный перебор

Ищет по всем комиксам в базе. Максимум 10 одновременных запросов.

```
GET /api/search?phrase=<фраза>&limit=<n>
```

| Параметр | Обязателен | По умолчанию | Описание |
|----------|-----------|--------------|----------|
| `phrase` | да        | —            | Поисковая фраза |
| `limit`  | нет       | 10           | Максимальное количество результатов |

```bash
curl "http://localhost:28080/api/search?phrase=linux&limit=5"
```

```json
{
  "comics": [
    {"id": 272, "url": "https://imgs.xkcd.com/comics/supported_features.png"}
  ],
  "total": 1
}
```

#### Поиск по индексу

Ищет по заранее построенному индексу. Значительно быстрее полного перебора. Ограничение — 100 запросов в секунду.

```
GET /api/isearch?phrase=<фраза>&limit=<n>
```

Параметры те же, что у `/api/search`. Индекс перестраивается автоматически после каждого обновления базы через NATS.

#### Кеш поиска

Результаты `/api/search` кешируются в Redis на 10 минут. Ключ кеша — `search:<phrase>:<limit>`. При обновлении или очистке базы кеш сбрасывается автоматически.

---

## Swagger UI

Интерактивная документация всех API эндпоинтов доступна по адресу:

```
http://localhost:28080/swagger/index.html
```

Позволяет отправлять запросы прямо из браузера. Для защищённых эндпоинтов нажмите **Authorize** и введите токен.

Спецификация в формате OpenAPI: `GET /swagger/doc.json`

---

## Метрики

Метрики в формате Prometheus доступны по адресу:

```
GET /metrics
```

Дашборд Grafana: `http://localhost:3000`.

---

## Тестирование

### Интеграционные тесты

Запускают полный прогон на реально поднятых сервисах в Docker:

```bash
docker-compose up -d
docker-compose run --rm tests
```

### Юнит-тесты

Запускаются без зависимостей, покрывают бизнес-логику и адаптеры:

```bash
make unit
```

После выполнения в корне проекта появится `cover.html` — откройте в браузере чтобы посмотреть покрытие по строкам:

```bash
xdg-open cover.html   # Linux
open cover.html       # macOS
```

### CI

При каждом пуше и pull request запускаются три джоба параллельно:

- **lint** — protolint + golangci-lint
- **unit** — юнит-тесты, проверка порога покрытия (≥50%), артефакт `coverage-report`
- **integration** — поднимает все сервисы, smoke тесты frontend и swagger, интеграционные тесты, артефакт `swagger-docs`

`integration` запускается только после успешных `lint` и `unit`.

Артефакты доступны во вкладке **Actions → нужный запуск → Artifacts**.
