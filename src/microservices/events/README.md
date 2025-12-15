# Events Service

MVP-сервис для обработки событий с использованием Apache Kafka в системе "Кинобездна".

## Возможности

- **Kafka Producer** - публикация событий в топики Kafka
- **Kafka Consumer** - чтение и обработка событий из топиков
- **Три типа событий**:
  - Movie Events (события фильмов)
  - User Events (события пользователей)
  - Payment Events (события платежей)
- **Логирование** - все события логируются при публикации и обработке

## Архитектура

```
API Request -> Events Service (Producer) -> Kafka Topic
                                              ↓
Events Service (Consumer) <- Kafka Topic (читает и обрабатывает)
```

## Переменные окружения

- `PORT` - порт для запуска сервиса (по умолчанию: 8082)
- `KAFKA_BROKERS` - список Kafka брокеров через запятую (по умолчанию: localhost:9092)

## API Endpoints

### Health Check
```bash
GET /api/events/health
```

### Movie Event
```bash
POST /api/events/movie
Content-Type: application/json

{
  "movie_id": 1,
  "title": "Inception",
  "action": "viewed",
  "user_id": 123,
  "rating": 8.5,
  "genres": ["Sci-Fi", "Action"],
  "description": "A mind-bending thriller"
}
```

### User Event
```bash
POST /api/events/user
Content-Type: application/json

{
  "user_id": 1,
  "username": "john_doe",
  "email": "john@example.com",
  "action": "registered",
  "timestamp": "2023-01-15T14:30:00Z"
}
```

### Payment Event
```bash
POST /api/events/payment
Content-Type: application/json

{
  "payment_id": 1,
  "user_id": 1,
  "amount": 9.99,
  "status": "completed",
  "timestamp": "2023-01-15T14:30:00Z",
  "method_type": "credit_card"
}
```

## Kafka Topics

Сервис работает с тремя топиками:
- `movie-events` - события фильмов
- `user-events` - события пользователей
- `payment-events` - события платежей

## Запуск

### Локально
```bash
export PORT=8082
export KAFKA_BROKERS=localhost:9092

go run main.go
```

### Docker Compose
```bash
docker-compose up events-service
```

## Тестирование

### Создание события фильма
```bash
curl -X POST http://localhost:8000/api/events/movie \
  -H "Content-Type: application/json" \
  -d '{
    "movie_id": 1,
    "title": "Inception",
    "action": "viewed",
    "user_id": 123
  }'
```

### Создание события пользователя
```bash
curl -X POST http://localhost:8000/api/events/user \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "action": "registered",
    "timestamp": "2023-01-15T14:30:00Z"
  }'
```

### Создание события платежа
```bash
curl -X POST http://localhost:8000/api/events/payment \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": 1,
    "user_id": 1,
    "amount": 9.99,
    "status": "completed",
    "timestamp": "2023-01-15T14:30:00Z"
  }'
```

## Логирование

Сервис логирует:
- **При публикации**: `✓ Published event to topic 'movie-events': ID=movie-1-viewed-1234567890, Type=movie`
- **При обработке**: 
  ```
  ✓ Consumed event from topic 'movie-events': ID=movie-1-viewed-1234567890, Type=movie, Partition=0, Offset=42
  Processing event: ID=movie-1-viewed-1234567890, Type=movie, Timestamp=2023-01-15T14:30:00Z
  Payload:
  {
    "movie_id": 1,
    "title": "Inception",
    "action": "viewed"
  }
  ```

## Мониторинг Kafka

Для просмотра топиков и сообщений используйте Kafka UI:
```
http://localhost:8090
```

## Примеры ответов

### Успешная публикация события
```json
{
  "status": "success",
  "partition": 0,
  "offset": 42,
  "event": {
    "id": "movie-1-viewed-1234567890",
    "type": "movie",
    "timestamp": "2023-01-15T14:30:00Z",
    "payload": {
      "movie_id": 1,
      "title": "Inception",
      "action": "viewed"
    }
  }
}
```

## Graceful Shutdown

Сервис корректно завершает работу при получении SIGINT/SIGTERM:
- Останавливает consumers
- Закрывает Kafka writers и readers
- Завершает обработку текущих сообщений
