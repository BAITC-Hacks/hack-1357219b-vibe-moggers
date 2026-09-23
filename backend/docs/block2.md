# Блок 2: команды, отклики и награды

Реализация использует существующие Go `net/http`, `database/sql` и PostgreSQL
с `github.com/lib/pq`. Новых зависимостей нет. Миграции запускаются при старте API.

## Маршруты

| Метод и путь | Результат | Demo-профиль |
| --- | --- | --- |
| `POST /api/teams` | Создание команды с 0 баллов, 201 | Не требуется |
| `GET /api/teams` | Список команд, 200 | Не требуется |
| `GET /api/teams/{team_id}` | Профиль и текущие баллы, 200 | Не требуется |
| `POST /api/tasks/{task_id}/offers` | Новый отклик `pending`, 201 | Команда |
| `GET /api/tasks/{task_id}/offers` | Отклики на свою задачу, 200 | Бизнес-владелец |
| `PATCH /api/offers/{offer_id}/status` | Решение и начисление при принятии, 200 | Бизнес-владелец |

Списки: `{"items":[],"total":0,"limit":20,"offset":0}`. Параметры:
`limit=1..100`, `offset>=0`. Список откликов поддерживает
`status=pending|accepted|rejected`, сортируется по `created_at DESC, id ASC`.
Пустой список содержит `[]`, а не `null`; `total` сохраняется за пределами страницы.

## Выбор профиля

Используется `X-Demo-Actor` из исходного `docs/SPEC.md` с уточнённым форматом:

- `X-Demo-Actor: team:<UUID команды>` — отправка отклика.
- `X-Demo-Actor: business:<UUID владельца>` — просмотр всех откликов и решение.

Это выбор участника локального демо, **не настоящая аутентификация**. Произвольный
клиент может поменять заголовок. Для публичного сервера нужен проверенный пользователь
из сессии/токена вместо `httpapi.Actor`. Внутри выбранного профиля сервер проверяет
роль и совпадение UUID бизнеса с `tasks.owner_id`; тело запроса не задаёт владельца.
`team_id` в теле необязателен, но при наличии обязан совпадать с выбранной командой.

Нет заголовка/неверный формат: 401. Чужая задача, роль или подмена команды: 403.
Нет ресурса: 404. Задача не опубликована: 409. Неправильный JSON/параметры списка: 400.
Невалидные поля: 422. Неожиданная ошибка: 500 без SQL/внутренних подробностей.
Ошибки: `{"error":{"code":"...","message":"...","fields":{"field":"..."}}}`.

## Тела запросов

`POST /api/teams`:

```json
{"name":"Vibe Moggers"}
```

Название: 1–200 символов после trim. Баллы нельзя передать с клиента.
Ответ `{"team":{...}}` содержит `id`, `name`, `points`, `interests`, `skills`,
`technologies`, `created_at`. У новой команды списки пустые, `points=0`.
UUID пяти начальных команд: `00000000-0000-4000-8000-000000000001` … `005`.

`POST /api/tasks/{task_id}/offers`:

```json
{
  "solution_idea": "Сервис сравнения услуг",
  "plan": "1. Собрать примеры\n2. Сделать прототип\n3. Проверить с пользователями",
  "timeline": "Одна неделя",
  "prototype_link": "https://example.org/prototype"
}
```

Сохранены snake_case-поля текущего контракта партнёра. Идея и план:
1–10000 символов; срок: 1–500. План — текст с шагами, не массив.
Прототип необязателен: пустая строка или абсолютная HTTP(S)-ссылка до 2048 символов.
Сервер не загружает содержимое ссылки. Имя команды берётся из профиля.

Ответ `{"offer":{...}}` содержит `id`, `task_id`, `team_id`, `team_name`, поля
заявки, `status`, `created_at`, `decided_at` (изначально `null`).

`PATCH /api/offers/{offer_id}/status`:

```json
{"status":"accepted"}
```

Разрешены только `accepted` и `rejected`. Ответ — актуальный `{"offer":{...}}`.
Баллы после решения читаются через `GET /api/teams/{team_id}`.

## Правила решений и баллов

1. Отклик разрешён на любую опубликованную задачу, независимо от рейтинга.
2. Можно принять несколько команд или отклонить все. Другие отклики не меняются.
3. В одной SQL-транзакции: блокировка задачи/отклика, проверка владельца,
   изменение решения, `INSERT point_awards ON CONFLICT DO NOTHING`, прибавка
   10 только при новой записи награды, обновление `tasks.work_status`.
4. `point_awards.offer_id` — первичный ключ; повторные/одновременные запросы
   и повторное принятие после смены решения не дают дополнительных баллов.
5. Как в текущем контракте партнёра, решение можно исправить. Уже выданная
   награда остаётся: это однократная награда за первое принятие отклика.
6. Есть хотя бы один `accepted` → `work_status=in_progress` («В работе»).
   После отклонения последнего принятого отклика → `work_status=open`.
   Публикация сохраняется, новые отклики доступны в обоих состояниях.
7. Несколько разных принятых откликов одной команды дают по 10 за каждый.
   Рейтинг готовности задачи от этих баллов не меняется.

## Совместимость с Postman и frontend

Работают прежние маршруты:

- `POST/GET /api/tasks/{task_id}/proposals`;
- `POST /api/proposals/{proposal_id}/accept`;
- `POST /api/proposals/{proposal_id}/reject`.

Они используют те же таблицы и транзакции. Отличия представления: обёртка
`proposal` вместо `offer`, статус `submitted` вместо `pending` (в том числе
фильтр списка). Установленные `accepted/rejected` одинаковы. Нужны те же
`X-Demo-Actor` заголовки. В Postman задайте переменную `business_id` равной
`tasks.owner_id`; `team_id` по-прежнему заполняется запросом списка команд.

## Граница с блоком 1

В исходном checkout был только `/health`: таблицы и API задач ещё не существовали.
Поэтому `001_task_contract.sql` создаёт минимальную общую таблицу `tasks`:

| Поле | Контракт |
| --- | --- |
| `id` | UUID, первичный ключ |
| `owner_id` | UUID выбранного бизнес-профиля, обязателен |
| `status` | `draft`, `confirmed` или `published` |
| `published_at` | TIMESTAMPTZ; заполнен только для `published` |
| `work_status` | `open` / `in_progress`, отдельно от готовности и публикации |
| `created_at`, `updated_at` | TIMESTAMPTZ |

Партнёр расширяет эту таблицу полями черновика, AI, карточки и рейтинга через
**новую миграцию**, сохраняя перечисленные поля. Если схема задач изменится,
адаптер находится в `internal/tasks/access.go`: `Lock` и `RefreshWorkStatus`
принимают **ту же `*sql.Tx`**, что изменение отклика и награда.
В ответ задачи/каталога блок 1 должен добавить `work_status`, а число откликов
считать из `offers` по `task_id`. Публикацию определять по `status/published_at`.

`002_teams_offers_awards.sql` принадлежит блоку 2. Миграции встроены через
`go:embed`, применяются по имени один раз с записью в `schema_migrations`.
Уже применённые файлы не редактировать — добавлять следующий номер.

Подключение маршрутов сделано в `cmd/api/main.go` через `Register` каждого
модуля. Блок 1 подключается таким же аргументом к `server.New`.

## Проверка вручную до готовности API задач

Сначала запустите API по `backend/README.md`, чтобы применились миграции.
В **локальной тестовой БД** через pgAdmin выполните явную фикстуру:

```sql
INSERT INTO tasks (id, owner_id, status, published_at)
VALUES (
  '20000000-0000-4000-8000-000000000001',
  '10000000-0000-4000-8000-000000000001',
  'published', CURRENT_TIMESTAMP
)
ON CONFLICT (id) DO NOTHING;
```

Затем в PowerShell:

```powershell
$apiBase = 'http://localhost:8080'
$demoTaskId = '20000000-0000-4000-8000-000000000001'
$businessHeaders = @{ 'X-Demo-Actor' = 'business:10000000-0000-4000-8000-000000000001' }
$demoTeams = (Invoke-RestMethod "$apiBase/api/teams?limit=2").items
$createdOffers = @()
foreach ($demoTeam in $demoTeams) {
    $teamHeaders = @{ 'X-Demo-Actor' = "team:$($demoTeam.id)" }
    $offerBody = @{
        solution_idea = 'Service comparison'
        plan = '1. Import samples; 2. Build prototype; 3. Test'
        timeline = 'One week'
        prototype_link = 'https://example.org/demo'
    } | ConvertTo-Json
    $createdOffers += (Invoke-RestMethod "$apiBase/api/tasks/$demoTaskId/offers" `
        -Method Post -Headers $teamHeaders -ContentType 'application/json' -Body $offerBody).offer
}
foreach ($demoOffer in $createdOffers) {
    $decisionUrl = "$apiBase/api/offers/$($demoOffer.id)/status"
    Invoke-RestMethod $decisionUrl -Method Patch -Headers $businessHeaders `
        -ContentType 'application/json' -Body '{"status":"accepted"}'
    # The repeated request must not add another 10 points.
    Invoke-RestMethod $decisionUrl -Method Patch -Headers $businessHeaders `
        -ContentType 'application/json' -Body '{"status":"accepted"}'
    Invoke-RestMethod "$apiBase/api/teams/$($demoOffer.team_id)"
}
Invoke-RestMethod "$apiBase/api/tasks/$demoTaskId/offers" -Headers $businessHeaders
```

У каждой команды прибавится ровно 10 за её новый отклик. Повторный запуск всего
скрипта создаёт новые отклики и новые награды. Для чистой проверки без накопления
используйте `TEST_DATABASE_URL` и интеграционный тест из README: он изолирует
данные в отдельной схеме и удаляет только свою схему после завершения.
