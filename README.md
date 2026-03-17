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