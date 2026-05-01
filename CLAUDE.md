# CU Points — CLAUDE.md

## Что это за проект

Поинтовая система лояльности для студентов Центрального Университета (Москва).
Студенты зарабатывают поинты за активность в ЦУ и тратят их у партнёров
(локальные кофейни, столовые, магазины рядом с кампусом).

Аналог Innopolis Club, адаптированный для Москвы. Партнёры — небольшой локальный
бизнес, не федеральные сети.

**Статус:** проект с нуля, чистый репозиторий.

---

## Команда

| Роль | Описание |
|------|----------|
| Тимлид / Продакт / Разраб | Emil — основной пользователь Claude Code |
| Backend-разработчик | разный уровень Go |
| Frontend-разработчик | разный уровень Next.js |
| Аналитик × 2 | требования, исследования, метрики |
| Дизайнер | UI/UX, Figma → компоненты |
| Экономист | бизнес-модель, партнёрские условия |

> **Важно для Claude Code:** в команде разный уровень Go и Next.js.
> Код должен быть хорошо прокомментирован, структура — предсказуемой,
> сложные паттерны — объяснены в комментарии над функцией.

---

## Стек

| Слой | Технология |
|------|-----------|
| Backend | Go 1.22+ |
| Frontend | Next.js 14 (App Router) + TypeScript |
| Стили | Tailwind CSS |
| БД | PostgreSQL 16 |
| Кэш / сессии | Redis 7 |
| Миграции | goose |
| Контейнеры | Docker + Docker Compose |
| Репозиторий | GitHub |
| CI/CD | GitHub Actions |
| Деплой | Yandex Cloud |
| Трекер задач | Kanban (Notion) |

---

## Структура репозитория

```
cu-points/
├── backend/
│   ├── cmd/api/
│   │   └── main.go              # точка входа: инициализация, DI, запуск
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go        # конфиг из env-переменных
│   │   ├── auth/
│   │   │   ├── handler.go       # HTTP-хендлеры (только парсинг запроса/ответа)
│   │   │   ├── service.go       # бизнес-логика аутентификации
│   │   │   ├── repository.go    # SQL-запросы
│   │   │   └── jwt.go           # генерация и валидация JWT
│   │   ├── points/
│   │   │   ├── handler.go
│   │   │   ├── service.go       # earn/spend — самый критичный слой
│   │   │   └── repository.go
│   │   ├── users/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── partners/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── admin/
│   │   │   ├── handler.go
│   │   │   └── service.go
│   │   └── middleware/
│   │       ├── auth.go          # проверка JWT, прокидывание user_id в контекст
│   │       ├── role.go          # проверка роли (student/partner/admin)
│   │       └── logger.go        # структурированное логирование запросов
│   └── pkg/
│       ├── db/postgres.go       # инициализация pgx pool
│       ├── cache/redis.go       # инициализация Redis клиента
│       └── response/json.go     # стандартные JSON-ответы (success/error)
├── frontend/
│   ├── app/
│   │   ├── (auth)/login/page.tsx
│   │   ├── (student)/
│   │   │   ├── dashboard/page.tsx   # баланс + последние операции
│   │   │   ├── history/page.tsx     # полная история транзакций
│   │   │   ├── partners/page.tsx    # список партнёров
│   │   │   └── qr/page.tsx          # QR-код для оплаты
│   │   ├── (partner)/
│   │   │   └── scan/page.tsx        # сканер QR + форма суммы
│   │   └── (admin)/
│   │       ├── dashboard/page.tsx
│   │       └── grant/page.tsx       # ручное начисление поинтов
│   ├── components/
│   │   ├── ui/                      # атомарные компоненты (Button, Input...)
│   │   ├── BalanceCard.tsx
│   │   ├── TransactionList.tsx
│   │   ├── QRDisplay.tsx
│   │   └── PartnerCard.tsx
│   ├── lib/
│   │   ├── api.ts                   # fetch-обёртка с baseURL и токеном
│   │   ├── store.ts                 # Zustand: глобальный стейт
│   │   ├── types.ts                 # типы для API-ответов
│   │   └── utils.ts                 # форматирование дат, чисел
│   └── middleware.ts                # защита роутов по роли
├── migrations/
│   ├── 00001_init_users.sql
│   ├── 00002_init_partners.sql
│   ├── 00003_init_transactions.sql
│   └── 00004_init_earning_rules.sql
├── .github/workflows/
│   ├── backend-ci.yml
│   └── frontend-ci.yml
├── docker-compose.yml
├── Makefile
├── .env.example
└── CLAUDE.md
```

---

## Соглашения по коду

### Go (Backend)

- **Архитектура:** строго `handler → service → repository`. Бизнес-логика **только в service**.
- **Именование файлов:** `snake_case`. Пакеты: короткие, без подчёркиваний.
- **Ошибки:** всегда явные, никогда не игнорировать `err`. Оборачивать через `fmt.Errorf("context: %w", err)`.
- **HTTP-роутер:** стандартная библиотека `net/http` + `chi`. Не использовать gin/echo/fiber без обсуждения с тимлидом.
- **БД:** только `pgx/v5` или `sqlx` с raw SQL. ORM (gorm и т.п.) — **запрещены**.
- **Логирование:** `log/slog` (стандартная библиотека Go 1.21+), структурированные поля.
- **Конфиг:** только через `internal/config/config.go`. Никаких magic strings по всему коду.
- **Комментарии:** обязательны над каждой экспортируемой функцией и над нетривиальной логикой. На английском.

```go
// SpendPoints debits the given amount from the user's balance
// and records a transaction atomically in a single DB transaction.
// Returns ErrInsufficientBalance if balance < amount.
func (s *Service) SpendPoints(ctx context.Context, req SpendRequest) error {
```

### TypeScript / Next.js (Frontend)

- **Роутер:** App Router (не Pages Router).
- **Стейт:** Zustand — глобальный (user, balance). `useState` — локальный UI-стейт.
- **Запросы к API:** только через `lib/api.ts`. **Не использовать axios**.
- **Компоненты:** функциональные, именованные экспорты: `export function BalanceCard(...)`.
- **Типы:** строгая типизация. **`any` — запрещён**. Типы API-ответов в `lib/types.ts`.
- **Стили:** Tailwind CSS. Кастомный CSS только если Tailwind не покрывает случай.

### Общие правила

- Названия переменных, функций, комментарии — **на английском**.
- Commit messages: `feat:`, `fix:`, `chore:`, `refactor:`, `docs:` + описание на английском.
  Пример: `feat: add QR token generation endpoint`
- `TODO` только с ссылкой на задачу: `// TODO(notion:TASK-42): handle expired tokens`
- Все API-эндпоинты с префиксом `/api/v1/`.

---

## Git-стратегия

Trunk-based flow — простой и подходящий для команды нашего размера:

```
main  ←  всегда стабильная, деплоится автоматически
  └── feature/auth-jwt
  └── feature/points-spend-qr
  └── fix/balance-negative-edge-case
```

- **Никогда не пушить напрямую в `main`.**
- Каждая задача из Notion — отдельная ветка: `feature/<название>` или `fix/<название>`.
- Перед мержем — PR с ревью минимум одного человека.
- CI (lint + tests) должен быть зелёным перед мержем.

---

## Модель данных

```sql
-- Пользователи (студенты, партнёры, администраторы)
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    student_id  TEXT UNIQUE,  -- только для студентов ЦУ
    role        TEXT NOT NULL CHECK (role IN ('student', 'partner', 'admin')),
    balance     INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Партнёры (кофейни, столовые и т.д.)
CREATE TABLE partners (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),   -- аккаунт кассира
    name            TEXT NOT NULL,
    address         TEXT NOT NULL,
    max_spend_pct   INTEGER NOT NULL DEFAULT 50,  -- макс. % покупки оплатить поинтами
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Транзакции — append-only лог, НИКОГДА не удалять записи
CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    partner_id  UUID REFERENCES partners(id),  -- NULL для earn-транзакций
    amount      INTEGER NOT NULL,  -- > 0 earn, < 0 spend
    type        TEXT NOT NULL CHECK (type IN ('earn', 'spend', 'admin_grant', 'expire')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Правила начисления (настраиваются администратором)
CREATE TABLE earning_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    points_amount   INTEGER NOT NULL,
    trigger_type    TEXT NOT NULL CHECK (trigger_type IN ('attendance', 'assignment', 'referral', 'admin')),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON transactions(user_id, created_at DESC);
CREATE INDEX ON transactions(partner_id, created_at DESC);
```

---

## Бизнес-правила (критически важно)

1. **Баланс ≥ 0 всегда** — двойная защита: `CHECK (balance >= 0)` в PostgreSQL + проверка в `points.Service` перед списанием.
2. **Транзакции атомарны** — списание баланса и запись транзакции — одна SQL-транзакция (`BEGIN / COMMIT`).
3. **Поинты не конвертируются в рубли** — студент не может вывести поинты деньгами.
4. **Курс:** 1 поинт = 1 рубль у партнёра. Партнёр получает компенсацию от ЦУ по договору.
5. **Лимит списания:** не более `max_spend_pct`% (по умолчанию 50%) от суммы покупки.
6. **QR-токен одноразовый** — JWT с TTL 5 минут. После использования — записать в Redis (ключ = `used_qr:<jti>`, TTL 5 минут) для защиты от повторного использования.
7. **Публичная оферта** — правила программы доступны без авторизации (требование НК РФ п.68 ст.217).

---

## API эндпоинты (MVP)

```
POST  /api/v1/auth/login           # email + password → {access_token, refresh_token}
POST  /api/v1/auth/refresh         # {refresh_token} → {access_token}

GET   /api/v1/me                   # профиль + текущий баланс
GET   /api/v1/me/transactions      # история (query: limit, offset)
GET   /api/v1/me/qr                # сгенерировать QR-токен (TTL 5 мин)

GET   /api/v1/partners             # список активных партнёров (публично)

POST  /api/v1/partner/spend        # {qr_token, amount} → списать поинты
                                   # доступно только роли 'partner'

POST  /api/v1/admin/points/grant   # {user_id, amount, description}
GET   /api/v1/admin/transactions   # все транзакции системы
GET   /api/v1/admin/users          # список студентов + балансы
GET   /api/v1/admin/stats          # агрегированная статистика
```

---

## Переменные окружения

Файл `.env.example` в корне репозитория:

```env
# Backend
DATABASE_URL=postgres://user:password@localhost:5432/cupoints?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key-minimum-32-characters
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
PORT=8080
ENV=development

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
```

**Никогда не коммитить `.env`** — только `.env.example`.

---

## Makefile

```makefile
make docker-up        # поднять postgres + redis
make docker-down      # остановить
make migrate-up       # применить все миграции
make migrate-down     # откатить последнюю миграцию
make run-backend      # запустить Go API (hot reload через air)
make run-frontend     # запустить Next.js dev
make test             # go test ./... + jest
make test-coverage    # coverage report
make lint             # golangci-lint + eslint + tsc
```

---

## Тестирование

- **Unit-тесты** обязательны для `internal/points/service.go` и `internal/auth/service.go`.
- **Integration-тесты** для критических путей: earn, spend, граничный случай (баланс = 0).
- Тестовая БД: `cupoints_test` в docker-compose.
- **Целевое покрытие:** ≥ 70% для пакетов `points` и `transactions`.

---

## CI (GitHub Actions)

На каждый PR в `main`:
1. `golangci-lint`
2. `go test ./...`
3. `tsc --noEmit` + `eslint`
4. Docker build (проверка что образы собираются)

Мерж только при зелёном CI.

---

## Чего НЕ делать

- Не хранить баланс только в Redis — PostgreSQL source of truth.
- Не писать бизнес-логику в хендлерах.
- Не использовать ORM.
- Не удалять записи из `transactions`.
- Не конвертировать поинты в рубли напрямую студенту.
- Не пушить в `main` напрямую.
- Не коммитить `.env`, ключи, пароли.
- Не использовать `any` в TypeScript.

---

## Приоритет задач (MVP)

1. `docker-compose.yml` + `Makefile` + `.env.example`
2. Миграции (все 4 таблицы)
3. Auth: login, JWT, middleware проверки роли
4. Points: earn/spend с атомарностью и проверкой баланса
5. QR: генерация токена + валидация (одноразовость через Redis)
6. REST API студента (me, transactions, qr)
7. REST API партнёра (spend)
8. REST API администратора (grant, stats)
9. Next.js: кабинет студента (баланс, история, QR)
10. Next.js: интерфейс партнёра (сканер + форма суммы)
11. Next.js: дашборд администратора
