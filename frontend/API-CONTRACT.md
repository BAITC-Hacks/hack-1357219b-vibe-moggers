> Для пятичасового MVP используется переключатель демонстрационных профилей и
> `X-Demo-Actor`. Регистрация не нужна. Голос, AI-вопросы и чат описаны в
> [AI-CONTRACT.md](AI-CONTRACT.md); уточнение принимает только `mode: live`.

# Подключение frontend к Go

Источник DTO: [src/domain/types.ts](src/domain/types.ts).
HTTP-реализация: [src/services/http-api.ts](src/services/http-api.ts).
Этот файл конкретизирует `docs/SPEC.md` для уже реализованного frontend.

## Демонстрационные роли

В верхней панели доступен переключатель «Бизнес / Студенческая команда» с готовыми
профилями `demo-business-1` и `team-1`…`team-5`. Выбор сохраняется локально и
преобразуется адаптером в `X-Demo-Actor: business:UUID` или `team:UUID`.
Для гостя заголовок отсутствует, каталог открыт.
Необязательный `Task.ownerName` используется для имени заказчика; если он отсутствует,
для известного seed-владельца показано AI Sana, для остальных — «Заказчик».

## Общие правила

- Префикс `/api`; JSON. Успех **всегда** `{ "data": ... }`.
- Ошибка `{ "error": { "code": "...", "message": "Понятное описание", "fields": {} } }`.
- `X-Demo-Actor: business:10000000-0000-4000-8000-000000000001` либо
  `team:00000000-0000-4000-8000-000000000001` … `...000005`.
  Frontend не отправляет доверенный ownerId/teamId в теле.
- Demo actor — временная демонстрационная идентичность. Go сам проверяет владельца действий.
- camelCase, строковые ID, время UTC ISO 8601, неизвестные значения `null`.
- Изменяющие методы возвращают полную обновлённую сущность, не только `{success:true}`.
- Максимум запроса frontend — 20 секунд; AI timeout backend должен быть короче.
- Публичный рейтинг — `confirmedSnapshot.total`; frontend не заменяет его прогнозом.

## Адаптация формата Go

1. **`GET /business/tasks`** — приватные и опубликованные задачи выбранного бизнеса.
   Без этого запроса после перезагрузки нельзя восстановить список его частных черновиков.
   Основной `GET /tasks` остаётся общим каталогом только опубликованных задач.
2. **`Proposal.milestone`** — вложенный этап или `null`. Создание/подтверждение этапа
   возвращает `Milestone`; после действия frontend перечитывает предложения. `GET /teams` возвращает `points`
   (сумму подтверждённых наград), чтобы показать баллы команды после перезагрузки.

Списки Go возвращают `{items,total,limit,offset}` внутри `data`. Адаптер проходит все
страницы по 100 элементов. `api-contract.ts` превращает плоский публичный снимок
в `Task.confirmedSnapshot`, `scoreBreakdown` в `breakdown`, `timestamp` в `confirmedAt`,
группу `constraints` в `limits`. Числовой рейтинг остаётся серверным.
Публичный снимок не содержит владельца и приватного черновика — адаптер их не выдумывает.

## Методы

| Метод | Вход | `data` в ответе |
| --- | --- | --- |
| `GET /tasks` | необязательные `industry`, `readiness` | страница публичных снимков |
| `GET /business/tasks` | actor бизнеса | страница `Task` (включая рабочие поля/черновики) |
| `GET /tasks/:id` | необязательный actor | `Task` владельцу, публичный снимок остальным |
| `POST /tasks` | `{draft,title,industry}` | `Task` |
| `PUT /tasks/:id/card` | `{revision,title,industry,answers,fields}` | `Task` |
| `POST /tasks/:id/confirm` | `{revision,fieldKeys,reviewed:true}` | `Task` |
| `POST /tasks/:id/publish` | `{revision}` | `Task` |
| `POST /ai/analyze` | `AnalyzeInput` | `AnalyzeResult` |
| `GET /teams` | — | страница `Team`, включая `points` |
| `GET /tasks/:id/proposals` | actor | страница `Proposal`; владельцу все, команде только свои |
| `POST /tasks/:id/proposals` | `{idea,plan:string[],durationDays,prototypeUrl}` | `Proposal` |
| `POST /proposals/:id/decision` | `{decision:"accepted"\|"rejected"}` | `Proposal` |
| `POST /proposals/:id/milestone` | `{resultText,evidenceUrl}` | `Milestone` |
| `POST /milestones/:id/confirm` | `{confirmed:true}` | подтверждённый `Milestone` |

## Task: внутренняя форма frontend после адаптации

```typescript
interface Task {
  id: string
  ownerId: string
  title: string
  industry: 'retail' | 'education' | 'services' | 'logistics' | 'agriculture'
  draft: string
  answers: { id: string; text: string }[]
  workingFields: Fields
  confirmedSnapshot: Snapshot | null
  revision: number
  confirmedRevision: number | null
  publishedAt: string | null
  createdAt: string
}

type Fields = Record<FieldKey, {
  value: string | null
  confirmed: boolean
  source: { id: string; quote: string } | null
}>

interface Snapshot {
  title: string
  industry: Task['industry']
  fields: Fields
  total: number
  readiness: 'draft' | 'working' | 'ready' | 'priority'
  breakdown: {
    key: 'context' | 'data' | 'result' | 'success' | 'limits' | 'users' | 'contact'
    label: string
    earned: number
    max: number
    missingFields: FieldKey[]
  }[]
  confirmedAt: string
}
```

Все 16 ключей `Fields` присутствуют всегда. Их список и веса — в `src/domain/scoring.ts`.
Неизвестные поля: `{value:null,confirmed:false,source:null}`. Порядок `breakdown`:
контекст, данные, результат, успех, ограничения, пользователи, контакт.

`PUT card` содержит и флаги confirmed из формы, но сервер **не должен им доверять**:
сохранение снимает подтверждение изменённых значений. Только `POST confirm` создаёт
подтверждённый снимок. При изменении successMetric снимается подтверждение зависимых
successTarget/acceptanceMethod. Повторный confirm изменяет текущую публичную версию,
если задача уже опубликована. Рейтинг может и уменьшаться.

Успешный confirm возвращает `confirmedRevision === revision`. Save после него
делает значения разными до следующего подтверждения. Устаревшая revision → 409.
Публикация не требует рейтинга 40/70/90. Пустая, но просмотренная и подтверждённая
карточка с названием/темой может иметь 0 баллов и принимать отклики.

## AI

Вход: `{stage:"clarify"|"assemble", sources:[{id,text}], currentFields:Fields}`.
Первый источник имеет id `draft`, ответы сохраняют id вопроса. Выход:

```json
{
  "data": {
    "mode": "live",
    "questions": [
      {"id":"q1","fieldKeys":["context","need"],"text":"Что происходит сейчас и что хотите изменить?"},
      {"id":"q2","fieldKeys":["dataSource","dataFormat","dataAccess"],"text":"Какие данные доступны и как их получить?"},
      {"id":"q3","fieldKeys":["deliverable","successMetric","successTarget"],"text":"Какой результат нужен и как измерить успех?"}
    ],
    "fieldSuggestions": [],
    "warnings": []
  }
}
```

`clarify`: 3–5 вопросов. `assemble`: до 5 оставшихся вопросов и предложения полей.
`fieldSuggestions`: `{field:FieldKey,value:string,sourceId:string}`.
Значение должно встречаться в переданном источнике после нормализации пробелов.
Frontend дополнительно отбрасывает предложения без такого источника.
Backend всё равно обязан валидировать AI-ответ самостоятельно.

При недоступной модели backend возвращает `503`. Ответ `mode:"fallback"` frontend считает
неуспешным и предлагает заполнить карточку вручную: заранее заготовленные AI-вопросы в демо
не используются. Ключ и название модели находятся только в окружении Go.

## Проверка интеграции

1. `VITE_DATA_MODE=api`, запустить Go и frontend, каталог загружает реальные JSON-данные.
2. Переключение участника меняет `X-Demo-Actor`; чужое действие возвращает 403.
3. Создать и сохранить частную задачу; она есть в owned, отсутствует в общем каталоге.
4. Confirm → publish → после перезагрузки задача видна всем; низкий балл не блокирует отклик.
5. Save правок не меняет публичный снимок; confirm меняет рейтинг/позицию.
6. Создать два отклика, вручную выбрать/отклонить; принять два в отдельном сценарии.
7. Передать и подтвердить этап; повторный запрос даёт тот же итог, `points` увеличивается один раз.
8. Отключить AI, проверить fallback. Отключить Go, получить понятную ошибку без подмены демо-данными.

Проверены на локальном Go + PostgreSQL и в браузере; детали и границы проверки —
в [отчёте интеграции](../docs/INTEGRATION.md).
