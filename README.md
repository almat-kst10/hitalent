## Запуск

```bash
# 1. Клонировать репозиторий
git clone https://github.com/ВАШ_НИКНЕЙМ/orgapi.git
cd orgapi

# 2. Запустить
docker-compose up --build
```

## Тесты

```bash
go test ./internal/handler/... -v
```

## Структура проекта

```
.
├── cmd/api/main.go              # точка входа — только инициализация logger и config
├── config/                      # конфигурация из env
├── internal/
│   ├── app/app.go               # запуск сервера, wire зависимостей, graceful shutdown
│   ├── handler/                 # HTTP handlers (net/http)
│   ├── service/                 # бизнес-логика, валидация, интерфейсы
│   ├── repository/              # GORM, работа с БД, интерфейсы
│   ├── middleware/              # logger, recover
│   └── models/                  # структуры и DTO
├── migrations/                  # SQL миграции (goose)
├── Dockerfile
└── docker-compose.yml
```

## API

| Метод  | Путь                              | Описание                        |
|--------|-----------------------------------|---------------------------------|
| POST   | /departments/                     | Создать подразделение           |
| GET    | /departments/{id}                 | Детейл + сотрудники + поддерево |
| PATCH  | /departments/{id}                 | Обновить / переместить          |
| DELETE | /departments/{id}                 | Удалить                         |
| POST   | /departments/{id}/employees/      | Создать сотрудника              |

### Query параметры GET /departments/{id}

| Параметр            | По умолчанию | Описание                        |
|---------------------|--------------|---------------------------------|
| depth               | 1            | Глубина дерева (макс 5)         |
| include_employees   | true         | Включать сотрудников            |

### Query параметры DELETE /departments/{id}

| Параметр                   | Описание                                          |
|----------------------------|---------------------------------------------------|
| mode                       | `cascade` или `reassign` (обязателен)             |
| reassign_to_department_id  | Куда перевести сотрудников (обязателен при reassign) |

## Примеры

```bash
# Создать подразделение
curl -X POST http://localhost:8080/departments/ \
  -H "Content-Type: application/json" \
  -d '{"name": "Engineering"}'

# Создать дочернее
curl -X POST http://localhost:8080/departments/ \
  -H "Content-Type: application/json" \
  -d '{"name": "Backend", "parent_id": 1}'

# Дерево глубиной 3
curl "http://localhost:8080/departments/1?depth=3"

# Переместить
curl -X PATCH http://localhost:8080/departments/2 \
  -H "Content-Type: application/json" \
  -d '{"parent_id": 5}'

# Удалить каскадно
curl -X DELETE "http://localhost:8080/departments/1?mode=cascade"

# Удалить с переносом сотрудников
curl -X DELETE "http://localhost:8080/departments/1?mode=reassign&reassign_to_department_id=2"

# Создать сотрудника
curl -X POST http://localhost:8080/departments/1/employees/ \
  -H "Content-Type: application/json" \
  -d '{"full_name": "Иван Иванов", "position": "Senior Developer"}'
```

