# CU Points — Архитектура проекта

> Документ для команды. Обновляется по мере принятия архитектурных решений.
> Последнее обновление: апрель 2026.

---

## Концепция

Поинтовая система лояльности для студентов ЦУ — аналог Innopolis Club.

**Ценность для студента:** зарабатывай поинты за учёбу, трать на кофе и еду рядом с кампусом.
**Ценность для партнёра:** гарантированный поток студентов, компенсация от ЦУ.
**Ценность для ЦУ:** инструмент мотивации студенческой активности.

---

## Роли пользователей

| Роль | Что может |
|------|-----------|
| `student` | смотреть баланс, историю, показывать QR для оплаты |
| `partner` | сканировать QR студентов, списывать поинты |
| `admin` | начислять поинты, смотреть статистику, управлять партнёрами |

---

## Архитектура (высокий уровень)

```
┌────────────────────────────────────────────────┐
│               КЛИЕНТЫ (браузер)                │
│   Студент (web)   Партнёр (web)   Админ (web)  │
└──────────────────────┬─────────────────────────┘
                       │ HTTPS / REST API
                       │
┌──────────────────────▼─────────────────────────┐
│              Go REST API (:8080)               │
│                                                │
│  /auth    /points    /partners    /admin       │
│                                                │
│  middleware: JWT проверка, role guard          │
└────────┬───────────────────────┬───────────────┘
         │                       │
┌────────▼────────┐    ┌─────────▼──────┐
│  PostgreSQL 16  │    │    Redis 7     │
│  (source of     │    │  - QR-токены   │
│   truth)        │    │  - сессии      │
└─────────────────┘    └────────────────┘
```

---

## Ключевые флоу

### Флоу 1: Студент тратит поинты у партнёра (QR)

```
Студент (браузер)        Кассир партнёра          Go API
       │                        │                    │
       │ GET /me/qr             │                    │
       │───────────────────────────────────────────▶ │
       │◀─────────────────────────────────────────── │
       │ {qr_token: JWT 5min}   │                    │
       │                        │                    │
       │  [показывает QR]       │                    │
       │ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─▶ │                    │
       │                        │ POST /partner/spend│
       │                        │ {qr_token, amount} │
       │                        │───────────────────▶│
       │                        │                    │ 1. валидировать JWT
       │                        │                    │ 2. проверить одноразовость (Redis)
       │                        │                    │ 3. проверить баланс ≥ amount
       │                        │                    │ 4. BEGIN TRANSACTION
       │                        │                    │    UPDATE users SET balance -= amount
       │                        │                    │    INSERT INTO transactions
       │                        │                    │ 5. COMMIT
       │                        │                    │ 6. записать jti в Redis (TTL 5min)
       │                        │◀───────────────────│
       │                        │ {success, new_bal} │
```

**Почему QR, а не интеграция с кассой:**
Федеральные сети (Додо, Дринкит) имеют собственное кассовое ПО и не дадут интеграцию
стартапу без длительных переговоров. QR-флоу — самодостаточное решение,
работает с любым партнёром у которого есть смартфон.

### Флоу 2: Начисление поинтов студенту

Источники поинтов (финальный список определят аналитики, здесь — возможные варианты):

| Триггер | Кто инициирует | Примерное количество |
|---------|----------------|----------------------|
| Посещение занятия | Администратор / интеграция с LMS | 5–10 поинтов |
| Сдача задания вовремя | Администратор / интеграция с LMS | 10–20 поинтов |
| Реферал (пригласил друга) | Автоматически | 50–100 поинтов |
| Ручное начисление (победа в конкурсе и т.д.) | Администратор | любое |

**MVP:** только ручное начисление через дашборд администратора.
**V1:** интеграция с LMS/системой посещаемости ЦУ.

---

## База данных (полная схема)

```sql
-- users: студенты, партнёры, администраторы
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    student_id  TEXT UNIQUE,
    role        TEXT NOT NULL CHECK (role IN ('student', 'partner', 'admin')),
    balance     INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- partners: метаданные точек-партнёров
CREATE TABLE partners (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    name            TEXT NOT NULL,
    address         TEXT NOT NULL,
    max_spend_pct   INTEGER NOT NULL DEFAULT 50,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- transactions: append-only лог всех операций с поинтами
CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    partner_id  UUID REFERENCES partners(id),
    amount      INTEGER NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('earn', 'spend', 'admin_grant', 'expire')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- earning_rules: настраиваемые правила начисления
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

**Почему `balance` в таблице `users`, а не вычисляется из транзакций:**
Вычислять баланс через `SUM(transactions)` — медленно при большой истории.
Храним `balance` как денормализованное поле, обновляем атомарно вместе с транзакцией.
PostgreSQL `CHECK (balance >= 0)` — последняя линия защиты от отрицательного баланса.

---

## Структура Go-бэкенда (детально)

```
backend/internal/points/service.go  ←  самый важный файл в проекте

// Пример структуры сервиса
type Service struct {
    repo   Repository      // интерфейс для моков в тестах
    cache  cache.Client
    db     *pgxpool.Pool
}

func (s *Service) SpendPoints(ctx context.Context, req SpendRequest) error {
    // 1. Проверить QR-токен (одноразовость через Redis)
    // 2. Загрузить пользователя
    // 3. Проверить баланс >= req.Amount
    // 4. Проверить лимит (amount <= purchase_total * max_spend_pct / 100)
    // 5. BEGIN TRANSACTION
    //    UPDATE users SET balance = balance - req.Amount WHERE id = req.UserID
    //    INSERT INTO transactions (...)
    // 6. COMMIT
    // 7. Записать jti в Redis (TTL 5 минут)
}
```

**Паттерн Repository:** каждый пакет определяет интерфейс `Repository`,
что позволяет писать unit-тесты без реальной БД (mock-реализация).

---

## Безопасность

| Угроза | Защита |
|--------|--------|
| Отрицательный баланс | `CHECK (balance >= 0)` + проверка в сервисе |
| Повторное использование QR | Redis: `used_qr:<jti>` с TTL 5 мин |
| Подделка роли | Role guard middleware + JWT claims |
| SQL-инъекции | Только параметризованные запросы (pgx/sqlx) |
| Утечка токенов | Access token TTL = 15 минут |
| Brute force логина | Rate limiting по IP (middleware) |

---

## Деплой (Yandex Cloud)

```
Yandex Cloud
├── Application Load Balancer
│   └── TLS-терминация (сертификат от Let's Encrypt через YC Certificate Manager)
├── Container Registry
│   ├── cu-points-backend:latest
│   └── cu-points-frontend:latest
├── Compute Cloud (или Serverless Containers)
│   ├── backend   (2 реплики, 1 vCPU / 1 GB RAM каждая)
│   └── frontend  (2 реплики)
├── Managed Service for PostgreSQL
│   └── HA-кластер (1 мастер + 1 реплика)
└── Managed Service for Redis
    └── 1 инстанс
```

**Локальная разработка:**
```yaml
# docker-compose.yml (минимальный)
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: cupoints
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: dev
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

volumes:
  pgdata:
```

---

## Правовые требования

Чтобы поинты не облагались НДФЛ у студентов (п.68 ст.217 НК РФ):

1. **Публичная оферта** — правила программы опубликованы на сайте ЦУ, доступны без авторизации.
2. **Срок акцепта ≥ 30 дней** — прописать в условиях программы.
3. **Студенты ≠ сотрудники** — поинты за учёбу, не за трудовые обязательства.
4. **Юрлицо-оператор** — ООО (можно ЦУ или отдельное юрлицо), которое заключает договоры с партнёрами и компенсирует им потраченные студентами поинты.

---

## Дорожная карта

### MVP
- [ ] Инфраструктура (docker-compose, миграции, CI)
- [ ] Auth (JWT, роли)
- [ ] Модель транзакций + баланс
- [ ] QR-флоу списания у партнёра
- [ ] Кабинет студента (баланс, история, QR)
- [ ] Ручное начисление администратором
- [ ] 2–3 партнёра-пилота рядом с кампусом

### V1 (после MVP)
- [ ] Автоначисление через интеграцию с LMS ЦУ
- [ ] Дашборд аналитики (для аналитиков и руководства)
- [ ] Реферальная программа
- [ ] PWA для мобильных (без нативного приложения)

### V2
- [ ] Срок жизни поинтов (expire)
- [ ] Уровни участников (Silver / Gold / Platinum)
- [ ] Push-уведомления
- [ ] API для партнёров (без QR — прямая интеграция для тех кто готов)

---

## Открытые вопросы (для команды)

- [ ] **Откуда студент получает поинты?** Финальный список триггеров — на аналитиках.
- [ ] **Кто оператор программы?** Нужно ли отдельное ООО или достаточно ЦУ?
- [ ] **Как партнёр получает компенсацию?** Ежемесячный акт или автоматически? — на экономисте.
- [ ] **Лимит 50% от суммы покупки** — нужно согласовать с партнёрами.
- [ ] **Дизайн** — Figma-макеты от дизайнера до начала разработки фронтенда.
