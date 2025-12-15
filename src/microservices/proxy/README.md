# Proxy Service (API Gateway)

API Gateway для системы "Кинобездна" с поддержкой паттерна Strangler Fig для постепенной миграции с монолита на микросервисы.

## Возможности

- **Единая точка входа** - все клиентские запросы проходят через API Gateway
- **Паттерн Strangler Fig** - постепенная миграция трафика с монолита на микросервисы
- **Feature Flag** - управление процентом трафика через переменные окружения
- **Маршрутизация запросов** - автоматическая маршрутизация на соответствующие сервисы

## Архитектура

```
Клиент -> API Gateway (Proxy) -> [Монолит | Movies Service | Events Service]
```

## Переменные окружения

- `PORT` - порт для запуска сервиса (по умолчанию: 8000)
- `MONOLITH_URL` - URL монолитного приложения
- `MOVIES_SERVICE_URL` - URL микросервиса movies
- `EVENTS_SERVICE_URL` - URL микросервиса events
- `GRADUAL_MIGRATION` - включение/выключение постепенной миграции (true/false)
- `MOVIES_MIGRATION_PERCENT` - процент трафика для микросервиса movies (0-100)

## Маршрутизация

### Movies Service (с Feature Flag)
- `GET /api/movies` - получение списка фильмов
- `GET /api/movies?id=X` - получение фильма по ID
- `POST /api/movies` - создание фильма

При `GRADUAL_MIGRATION=true` и `MOVIES_MIGRATION_PERCENT=50`:
- 50% запросов идут в Movies Microservice
- 50% запросов идут в Monolith

### Events Service
- `POST /api/events/*` - все запросы к событиям

### Monolith (остальные запросы)
- `GET /api/users` - пользователи
- `GET /api/payments` - платежи
- `GET /api/subscriptions` - подписки

## Запуск

### Локально
```bash
export PORT=8000
export MONOLITH_URL=http://localhost:8080
export MOVIES_SERVICE_URL=http://localhost:8081
export EVENTS_SERVICE_URL=http://localhost:8082
export GRADUAL_MIGRATION=true
export MOVIES_MIGRATION_PERCENT=50

go run main.go
```

### Docker Compose
```bash
docker-compose up proxy-service
```

## Тестирование

### Проверка health
```bash
curl http://localhost:8000/health
```

### Получение списка фильмов
```bash
curl http://localhost:8000/api/movies
```

### Изменение процента миграции

Отредактируйте `docker-compose.yml`:
```yaml
environment:
  MOVIES_MIGRATION_PERCENT: "100"  # 100% трафика на микросервис
```

Перезапустите сервис:
```bash
docker-compose restart proxy-service
```

## Логирование

Сервис логирует каждый проксируемый запрос с указанием целевого сервиса:
```
Proxying GET /api/movies to http://movies-service:8081/api/movies
Routing to Movies Microservice (migration: 50%)
```

## Примеры использования

### 0% миграции (весь трафик на монолит)
```yaml
GRADUAL_MIGRATION: "true"
MOVIES_MIGRATION_PERCENT: "0"
```

### 50% миграции (канареечное развертывание)
```yaml
GRADUAL_MIGRATION: "true"
MOVIES_MIGRATION_PERCENT: "50"
```

### 100% миграции (полный переход на микросервис)
```yaml
GRADUAL_MIGRATION: "true"
MOVIES_MIGRATION_PERCENT: "100"
```

### Отключение миграции (только монолит)
```yaml
GRADUAL_MIGRATION: "false"
```
