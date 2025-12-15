# Развертывание CinemaAbyss с использованием Helm

Полное руководство по развертыванию системы «Кинобездна» с использованием Helm чартов.

## Преимущества Helm

- **Упрощенное управление** - один команда для установки/обновления/удаления
- **Версионирование** - откат к предыдущим версиям одной командой
- **Шаблонизация** - переиспользование конфигураций для разных окружений
- **Канареечные релизы** - постепенное обновление сервисов
- **Управление зависимостями** - автоматическая установка всех компонентов

## Предварительные требования

- Kubernetes кластер (Minikube, Kind, или облачный)
- Helm 3.x
- kubectl
- GitHub Personal Access Token с правами `read:packages`

## Часть 1: Подготовка

### 1.1. Установка Helm

**Windows (PowerShell):**
```powershell
choco install kubernetes-helm
```

**Linux:**
```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

**macOS:**
```bash
brew install helm
```

Проверка установки:
```bash
helm version
```

### 1.2. Настройка Docker Registry Secret

#### Создание Personal Access Token

1. Перейдите на https://github.com/settings/tokens
2. Создайте токен с правами `read:packages`
3. Скопируйте токен

#### Создание base64 для dockerconfigjson

**Windows (PowerShell):**
```powershell
# Создайте auth строку
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
# Создайте auth строку
echo -n "ваш_username:ваш_токен" | base64

# Создайте config.json
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

### 1.3. Обновление values.yaml

Отредактируйте `src/kubernetes/helm/values.yaml`:

#### Обновите пути к образам

Замените `ghcr.io/db-exp/cinemaabysstest` на ваш путь:

```yaml
monolith:
  image:
    repository: ghcr.io/ваш_username/ваш_репозиторий/monolith
    tag: latest

proxyService:
  image:
    repository: ghcr.io/ваш_username/ваш_репозиторий/proxy-service
    tag: latest

moviesService:
  image:
    repository: ghcr.io/ваш_username/ваш_репозиторий/movies-service
    tag: latest

eventsService:
  image:
    repository: ghcr.io/ваш_username/ваш_репозиторий/events-service
    tag: latest
```

#### Обновите imagePullSecrets

```yaml
imagePullSecrets:
  dockerconfigjson: ваш_base64_результат_из_шага_1.2
```

## Часть 2: Развертывание с Helm

### 2.1. Запуск Minikube

```bash
minikube start --cpus=4 --memory=8192
```

### 2.2. Включение Ingress

```bash
minikube addons enable ingress
```

### 2.3. Установка Helm чарта

```bash
# Из корня проекта
helm install cinemaabyss ./src/kubernetes/helm --namespace cinemaabyss --create-namespace
```

**Ожидаемый вывод:**
```
NAME: cinemaabyss
LAST DEPLOYED: ...
NAMESPACE: cinemaabyss
STATUS: deployed
REVISION: 1
```

### 2.4. Проверка статуса установки

```bash
# Проверка релиза
helm list -n cinemaabyss

# Проверка подов
kubectl -n cinemaabyss get pods

# Ожидаемый вывод (все поды должны быть Running):
NAME                              READY   STATUS    RESTARTS   AGE
events-service-xxx                1/1     Running   0          2m
kafka-0                           1/1     Running   0          2m
monolith-xxx                      1/1     Running   0          2m
movies-service-xxx                1/1     Running   0          2m
postgres-0                        1/1     Running   0          2m
proxy-service-xxx                 1/1     Running   0          2m
zookeeper-0                       1/1     Running   0          2m
```

### 2.5. Настройка hosts

**Windows:** `C:\Windows\System32\drivers\etc\hosts`  
**Linux/macOS:** `/etc/hosts`

Добавьте:
```
127.0.0.1 cinemaabyss.example.com
```

### 2.6. Запуск Minikube tunnel

В отдельном терминале:
```bash
minikube tunnel
```

## Часть 3: Тестирование

### 3.1. Проверка API

```bash
curl http://cinemaabyss.example.com/api/movies
```

Должен вернуться список фильмов в JSON формате.

### 3.2. Проверка health endpoints

```bash
curl http://cinemaabyss.example.com/health
curl http://cinemaabyss.example.com/api/movies/health
curl http://cinemaabyss.example.com/api/events/health
```

### 3.3. Создание события

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

### 3.4. Просмотр логов

```bash
# Логи proxy-service
kubectl -n cinemaabyss logs -f deployment/proxy-service

# Логи events-service
kubectl -n cinemaabyss logs -f deployment/events-service
```

### 3.5. Запуск Postman тестов

```bash
cd tests/postman
npm install
npm run test:kubernetes
```

## Часть 4: Управление конфигурацией

### 4.1. Изменение процента миграции

Отредактируйте `values.yaml`:
```yaml
config:
  moviesMigrationPercent: "100"  # 100% трафика на микросервис
```

Обновите релиз:
```bash
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss
```

### 4.2. Масштабирование сервисов

Отредактируйте `values.yaml`:
```yaml
proxyService:
  replicas: 3  # Увеличить до 3 реплик
```

Обновите:
```bash
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss
```

### 4.3. Изменение ресурсов

```yaml
proxyService:
  resources:
    limits:
      cpu: "500m"
      memory: "512Mi"
    requests:
      cpu: "200m"
      memory: "256Mi"
```

## Часть 5: Управление релизами

### 5.1. Просмотр истории релизов

```bash
helm history cinemaabyss -n cinemaabyss
```

### 5.2. Откат к предыдущей версии

```bash
helm rollback cinemaabyss -n cinemaabyss
```

Откат к конкретной ревизии:
```bash
helm rollback cinemaabyss 1 -n cinemaabyss
```

### 5.3. Обновление релиза

```bash
# После изменения values.yaml
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss

# С дополнительными параметрами
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss \
  --set config.moviesMigrationPercent=75 \
  --set proxyService.replicas=2
```

### 5.4. Проверка изменений перед применением

```bash
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss --dry-run --debug
```

## Часть 6: Канареечные релизы

### 6.1. Установка канареечной версии

```bash
# Установка основного релиза
helm install cinemaabyss ./src/kubernetes/helm -n cinemaabyss

# Установка канареечной версии с новым образом
helm install cinemaabyss-canary ./src/kubernetes/helm -n cinemaabyss \
  --set proxyService.image.tag=v2.0.0 \
  --set proxyService.replicas=1
```

### 6.2. Постепенное переключение трафика

Используйте Ingress weights или Service Mesh (Istio/Linkerd) для управления трафиком между версиями.

## Часть 7: Мониторинг и отладка

### 7.1. Просмотр всех ресурсов

```bash
kubectl -n cinemaabyss get all
```

### 7.2. Описание релиза

```bash
helm get all cinemaabyss -n cinemaabyss
```

### 7.3. Просмотр values

```bash
helm get values cinemaabyss -n cinemaabyss
```

### 7.4. Просмотр манифестов

```bash
helm get manifest cinemaabyss -n cinemaabyss
```

### 7.5. Отладка подов

```bash
# Описание пода
kubectl -n cinemaabyss describe pod pod_name

# Логи
kubectl -n cinemaabyss logs pod_name

# Exec в под
kubectl -n cinemaabyss exec -it pod_name -- /bin/sh
```

## Часть 8: Удаление

### 8.1. Удаление релиза

```bash
helm uninstall cinemaabyss -n cinemaabyss
```

### 8.2. Удаление namespace

```bash
kubectl delete namespace cinemaabyss
```

### 8.3. Остановка Minikube

```bash
minikube stop
```

## Troubleshooting

### Проблема: ImagePullBackOff

**Причина:** Неправильный dockerconfigjson или отсутствие доступа к образам.

**Решение:**
1. Проверьте правильность токена
2. Убедитесь, что образы существуют в GitHub Container Registry
3. Пересоздайте secret:
```bash
kubectl -n cinemaabyss delete secret dockerconfigjson
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss
```

### Проблема: Kafka не запускается

**Решение:**
```bash
kubectl -n cinemaabyss delete pod kafka-0 zookeeper-0
kubectl -n cinemaabyss delete pvc --all
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss
```

### Проблема: Ingress не работает

**Решение:**
1. Проверьте, что minikube tunnel запущен
2. Проверьте ingress:
```bash
kubectl -n cinemaabyss get ingress
kubectl -n cinemaabyss describe ingress cinemaabyss-ingress
```

### Проблема: Поды не запускаются

**Решение:**
```bash
# Проверьте события
kubectl -n cinemaabyss get events --sort-by='.lastTimestamp'

# Проверьте ресурсы
kubectl top nodes
kubectl top pods -n cinemaabyss
```

## Полезные команды Helm

```bash
# Валидация чарта
helm lint ./src/kubernetes/helm

# Шаблонизация без установки
helm template cinemaabyss ./src/kubernetes/helm

# Установка с отладкой
helm install cinemaabyss ./src/kubernetes/helm -n cinemaabyss --debug

# Обновление с ожиданием
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss --wait --timeout 5m

# Удаление с сохранением истории
helm uninstall cinemaabyss -n cinemaabyss --keep-history
```

## Структура Helm чарта

```
src/kubernetes/helm/
├── Chart.yaml                    # Метаданные чарта
├── values.yaml                   # Конфигурационные значения
├── templates/                    # Шаблоны Kubernetes манифестов
│   ├── _helpers.tpl             # Вспомогательные функции
│   ├── namespace.yaml           # Namespace
│   ├── configmap.yaml           # ConfigMap
│   ├── secret.yaml              # Secrets
│   ├── dockerconfigsecret.yaml  # Docker registry secret
│   ├── ingress.yaml             # Ingress
│   ├── services/                # Сервисы
│   │   ├── postgres.yaml
│   │   ├── monolith.yaml
│   │   ├── movies-service.yaml
│   │   ├── proxy-service.yaml   # ✓ Заполнен
│   │   └── events-service.yaml  # ✓ Заполнен
│   └── kafka/                   # Kafka компоненты
└── README.md                    # Документация
```

## Параметры конфигурации

### Основные параметры в values.yaml

```yaml
# Namespace и домен
global:
  namespace: cinemaabyss
  domain: cinemaabyss.example.com

# Proxy Service
proxyService:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/username/repo/proxy-service
    tag: latest
  resources:
    limits:
      cpu: 300m
      memory: 256Mi

# Events Service
eventsService:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/username/repo/events-service
    tag: latest

# Конфигурация приложения
config:
  gradualMigration: "true"
  moviesMigrationPercent: "50"
```

## Результаты

После успешной установки:

✅ Все сервисы развернуты одной командой  
✅ API доступен через https://cinemaabyss.example.com/api/movies  
✅ Легкое управление конфигурацией через values.yaml  
✅ Возможность отката к предыдущим версиям  
✅ Упрощенное обновление сервисов  
✅ Готовность к канареечным релизам

## Следующие шаги

1. Настройте CI/CD для автоматического деплоя через Helm
2. Добавьте мониторинг (Prometheus + Grafana)
3. Настройте Service Mesh для advanced traffic management
4. Реализуйте канареечные релизы с автоматическим rollback
5. Добавьте HorizontalPodAutoscaler для автомасштабирования
