# Инструкция по деплою в Kubernetes

Пошаговое руководство по развертыванию системы «Кинобездна» в Kubernetes.

## Предварительные требования

- Установленный Minikube
- Установленный kubectl
- Docker
- GitHub Personal Access Token с правами `read:packages`

## Часть 1: Настройка CI/CD

### 1.1. Обновление путей к образам

Отредактируйте следующие файлы, заменив `studenttsu/architecture-cinemaabyss` на ваш GitHub username и название репозитория:

**`src/kubernetes/monolith.yaml:25`**
```yaml
image: ghcr.io/ваш_username/ваш_репозиторий/monolith:latest
```

**`src/kubernetes/movies-service.yaml:25`**
```yaml
image: ghcr.io/ваш_username/ваш_репозиторий/movies-service:latest
```

**`src/kubernetes/proxy-service.yaml:25`**
```yaml
image: ghcr.io/ваш_username/ваш_репозиторий/proxy-service:latest
```

**`src/kubernetes/events-service.yaml:25`**
```yaml
image: ghcr.io/ваш_username/ваш_репозиторий/events-service:latest
```

### 1.2. Запуск CI/CD

Закоммитьте и запушьте изменения:
```bash
git add .
git commit -m "Add proxy and events services with CI/CD"
git push origin main
```

Проверьте статус сборки в GitHub Actions:
- Перейдите в раздел **Actions** вашего репозитория
- Дождитесь завершения workflow **Docker Build and Push**
- Убедитесь, что все шаги зеленые ✓

Проверьте наличие образов в GitHub Container Registry:
- Перейдите в **Packages** вашего репозитория
- Должны быть видны 4 пакета: monolith, movies-service, proxy-service, events-service

## Часть 2: Настройка Docker Registry Secret

### 2.1. Создание Personal Access Token

1. Перейдите на https://github.com/settings/tokens
2. Нажмите **Generate new token (classic)**
3. Выберите права: `read:packages`
4. Скопируйте созданный токен

### 2.2. Логин в Docker Registry

```bash
echo "ваш_токен" | docker login ghcr.io -u ваш_username --password-stdin
```

### 2.3. Создание base64 для dockerconfigjson

**Windows (PowerShell):**
```powershell
# Создайте строку в формате username:token
$auth = "ваш_username:ваш_токен"
$authBytes = [System.Text.Encoding]::UTF8.GetBytes($auth)
$authBase64 = [System.Convert]::ToBase64String($authBytes)

# Создайте JSON конфигурацию
$config = @{
    auths = @{
        "ghcr.io" = @{
            auth = $authBase64
        }
    }
} | ConvertTo-Json -Compress

# Конвертируйте в base64
$configBytes = [System.Text.Encoding]::UTF8.GetBytes($config)
$configBase64 = [System.Convert]::ToBase64String($configBytes)
Write-Host $configBase64
```

**Linux/macOS:**
```bash
# Создайте base64 для auth
echo -n "ваш_username:ваш_токен" | base64

# Создайте полный config.json
cat <<EOF > config.json
{
  "auths": {
    "ghcr.io": {
      "auth": "результат_предыдущей_команды"
    }
  }
}
EOF

# Конвертируйте в base64
cat config.json | base64 -w 0
```

### 2.4. Обновление dockerconfigsecret.yaml

Отредактируйте `src/kubernetes/dockerconfigsecret.yaml`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dockerconfigjson
  namespace: cinemaabyss
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ваш_base64_результат_из_предыдущего_шага
```

## Часть 3: Развертывание в Kubernetes

### 3.1. Запуск Minikube

```bash
minikube start --cpus=4 --memory=8192
```

### 3.2. Создание namespace

```bash
kubectl apply -f src/kubernetes/namespace.yaml
```

### 3.3. Создание секретов и конфигураций

```bash
kubectl apply -f src/kubernetes/configmap.yaml
kubectl apply -f src/kubernetes/secret.yaml
kubectl apply -f src/kubernetes/dockerconfigsecret.yaml
kubectl apply -f src/kubernetes/postgres-init-configmap.yaml
```

### 3.4. Развертывание PostgreSQL

```bash
kubectl apply -f src/kubernetes/postgres.yaml
```

Проверьте статус:
```bash
kubectl -n cinemaabyss get pod
```

Ожидайте статус `Running` для пода `postgres-0`.

### 3.5. Развертывание Kafka

```bash
kubectl apply -f src/kubernetes/kafka/kafka.yaml
```

Проверьте статус:
```bash
kubectl -n cinemaabyss get pod
```

Должны быть запущены: `postgres-0`, `kafka-0`, `zookeeper-0`.

### 3.6. Развертывание сервисов

```bash
kubectl apply -f src/kubernetes/monolith.yaml
kubectl apply -f src/kubernetes/movies-service.yaml
kubectl apply -f src/kubernetes/events-service.yaml
kubectl apply -f src/kubernetes/proxy-service.yaml
```

### 3.7. Проверка статуса подов

```bash
kubectl -n cinemaabyss get pod
```

Ожидаемый вывод:
```
NAME                              READY   STATUS    RESTARTS   AGE
events-service-xxx                1/1     Running   0          2m
kafka-0                           1/1     Running   0          5m
monolith-xxx                      1/1     Running   0          3m
movies-service-xxx                1/1     Running   0          3m
postgres-0                        1/1     Running   0          8m
proxy-service-xxx                 1/1     Running   0          2m
zookeeper-0                       1/1     Running   0          5m
```

Если поды не запускаются, проверьте логи:
```bash
kubectl -n cinemaabyss logs pod_name
kubectl -n cinemaabyss describe pod pod_name
```

### 3.8. Настройка Ingress

Включите Ingress addon:
```bash
minikube addons enable ingress
```

Примените Ingress конфигурацию:
```bash
kubectl apply -f src/kubernetes/ingress.yaml
```

Проверьте Ingress:
```bash
kubectl -n cinemaabyss get ingress
```

### 3.9. Настройка hosts файла

**Windows:** Отредактируйте `C:\Windows\System32\drivers\etc\hosts` (от имени администратора)

**Linux/macOS:** Отредактируйте `/etc/hosts`

Добавьте строку:
```
127.0.0.1 cinemaabyss.example.com
```

### 3.10. Запуск Minikube tunnel

**Важно:** Эта команда должна работать постоянно в отдельном терминале!

```bash
minikube tunnel
```

На Windows может потребоваться права администратора.

## Часть 4: Тестирование

### 4.1. Проверка доступности API

```bash
curl http://cinemaabyss.example.com/api/movies
```

Должен вернуться список фильмов в формате JSON.

### 4.2. Проверка health endpoints

```bash
curl http://cinemaabyss.example.com/health
curl http://cinemaabyss.example.com/api/movies/health
curl http://cinemaabyss.example.com/api/events/health
```

### 4.3. Тестирование событий

```bash
curl -X POST http://cinemaabyss.example.com/api/events/movie \
  -H "Content-Type: application/json" \
  -d '{
    "movie_id": 1,
    "title": "Test Movie",
    "action": "viewed",
    "user_id": 123
  }'
```

### 4.4. Запуск Postman тестов

```bash
cd tests/postman
npm install
npm run test:kubernetes
```

### 4.5. Просмотр логов events-service

```bash
kubectl -n cinemaabyss logs -f deployment/events-service
```

Вы должны увидеть:
```
✓ Published event to topic 'movie-events': ID=movie-1-viewed-...
✓ Consumed event from topic 'movie-events': ID=movie-1-viewed-...
Processing event: ID=movie-1-viewed-...
```

**Сделайте скриншот логов!**

### 4.6. Тестирование миграции трафика

Измените процент миграции в `src/kubernetes/configmap.yaml`:
```yaml
MOVIES_MIGRATION_PERCENT: "0"    # Весь трафик на монолит
MOVIES_MIGRATION_PERCENT: "50"   # 50/50
MOVIES_MIGRATION_PERCENT: "100"  # Весь трафик на микросервис
```

Примените изменения:
```bash
kubectl apply -f src/kubernetes/configmap.yaml
kubectl -n cinemaabyss rollout restart deployment/proxy-service
```

Проверьте логи прокси:
```bash
kubectl -n cinemaabyss logs -f deployment/proxy-service
```

## Часть 5: Получение скриншотов

### 5.1. Скриншот вывода /api/movies

Откройте в браузере или выполните:
```bash
curl http://cinemaabyss.example.com/api/movies
```

**Сделайте скриншот!**

### 5.2. Скриншот логов events-service

```bash
kubectl -n cinemaabyss logs deployment/events-service --tail=50
```

**Сделайте скриншот обработки событий!**

### 5.3. Скриншот статуса подов

```bash
kubectl -n cinemaabyss get pods -o wide
```

**Сделайте скриншот!**

## Часть 6: Очистка ресурсов

После завершения тестирования:

```bash
# Удалить все ресурсы
kubectl delete all --all -n cinemaabyss

# Удалить namespace
kubectl delete namespace cinemaabyss

# Остановить Minikube
minikube stop
```

## Troubleshooting

### Проблема: Поды в статусе ImagePullBackOff

**Решение:**
1. Проверьте, что образы существуют в GitHub Container Registry
2. Проверьте правильность dockerconfigsecret
3. Убедитесь, что пути к образам корректны

```bash
kubectl -n cinemaabyss describe pod pod_name
```

### Проблема: Kafka не запускается

**Решение:**
```bash
kubectl -n cinemaabyss delete pod kafka-0 zookeeper-0
kubectl -n cinemaabyss delete pvc --all
kubectl apply -f src/kubernetes/kafka/kafka.yaml
```

### Проблема: Ingress не работает

**Решение:**
1. Проверьте, что minikube tunnel запущен
2. Проверьте статус ingress:
```bash
kubectl -n cinemaabyss get ingress
kubectl -n cinemaabyss describe ingress cinemaabyss-ingress
```

### Проблема: События не обрабатываются

**Решение:**
1. Проверьте логи events-service
2. Проверьте, что Kafka работает:
```bash
kubectl -n cinemaabyss logs kafka-0
```

## Полезные команды

```bash
# Просмотр всех ресурсов
kubectl -n cinemaabyss get all

# Просмотр логов всех подов
kubectl -n cinemaabyss logs -l app=proxy-service --tail=100

# Перезапуск deployment
kubectl -n cinemaabyss rollout restart deployment/proxy-service

# Масштабирование
kubectl -n cinemaabyss scale deployment/proxy-service --replicas=2

# Проброс портов (альтернатива ingress)
kubectl -n cinemaabyss port-forward service/proxy-service 8000:8000
```

## Архитектура в Kubernetes

```
Internet
    ↓
Ingress (cinemaabyss.example.com)
    ↓
Proxy Service (API Gateway)
    ↓
┌─────────────┬──────────────────┬─────────────────┐
│             │                  │                 │
Monolith   Movies Service   Events Service    
│             │                  │
└─────────────┴──────────────────┘
              ↓
          PostgreSQL
              
Events Service ←→ Kafka ←→ Zookeeper
```

## Результаты

После успешного деплоя у вас должно быть:

✅ Все сервисы запущены в Kubernetes  
✅ API доступен через https://cinemaabyss.example.com  
✅ Proxy-service маршрутизирует запросы  
✅ Events-service обрабатывает события через Kafka  
✅ Можно управлять миграцией трафика через ConfigMap  
✅ Postman тесты проходят успешно  
✅ Скриншоты логов и API ответов готовы
