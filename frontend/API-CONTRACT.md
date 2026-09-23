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
передаётся в `X-Demo-Actor`. Каталог остаётся доступным с actor `guest`.
Необязательный `Task.ownerName` используется для имени заказчика; если он отсутствует,
для известного seed-владельца показано AI Sana, для остальных — «Заказчик».

## Общие правила

- Префикс `/api`; JSON. Успех **всегда** `{ "data": ... }`.
- Ошибка `{ "error": { "code": "...", "message": "Понятное описание", "fields": {} } }`.
- `X-Demo-Actor: demo-business-1` либо `team-1` … `team-5`.
  Frontend не отправляет доверенный ownerId/teamId в теле.
- Demo actor — временная демонстрационная идентичность. Go сам проверяет владельца действий.
- camelCase, строковые ID, время UTC ISO 8601, неизвестные значения `null`.
- Изменяющие методы возвращают полную обновлённую сущность, не только `{success:true}`.
- Максимум запроса frontend — 20 секунд; AI timeout backend должен быть короче.
- Публичный рейтинг — `confirmedSnapshot.total`; frontend не заменяет его прогнозом.

## Два дополнения, которые нужно учесть

1. **`GET /tasks?scope=owned`** — приватные и опубликованные задачи выбранного бизнеса.
   Без этого запроса после перезагрузки нельзя восстановить список его частных черновиков.
   Основной `GET /tasks` остаётся общим каталогом только опубликованных задач.
2. **`Proposal.milestone`** — вложенный этап или `null`. Создание/подтверждение этапа
   возвращает обновлённый `Proposal` с этим полем. `GET /teams` возвращает `points`
   (сумму подтверждённых наград), чтобы показать баллы команды после перезагрузки.

Если Go уже возвращает другой формат — достаточно согласованно адаптировать
`http-api.ts`. Страницы не должны отдельно знать несколько версий API.

## Методы

| Метод | Вход | `data` в ответе |
| --- | --- | --- |
| `GET /tasks` | необязательные `industry`, `readiness` | `Task[]` (подтверждённые публичные сведения) |
| `GET /tasks?scope=owned` | actor бизнеса | `Task[]` (включая рабочие поля/черновики) |
| `GET /tasks/:id` | actor | `Task` (рабочие поля только владельцу) |
| `POST /tasks` | `{draft,title,industry}` | `Task` |
| `PUT /tasks/:id/card` | `{revision,title,industry,answers,fields}` | `Task` |
| `POST /tasks/:id/confirm` | `{revision,fieldKeys,reviewed:true}` | `Task` |
| `POST /tasks/:id/publish` | `{revision}` | `Task` |
| `POST /ai/analyze` | `AnalyzeInput` | `AnalyzeResult` |
| `GET /teams` | — | `Team[]`, включая `points` |
| `GET /tasks/:id/proposals` | actor | `Proposal[]`; владельцу все, команде только свои |
| `POST /tasks/:id/proposals` | `{idea,plan:string[],durationDays,prototypeUrl}` | `Proposal` |
| `POST /proposals/:id/decision` | `{decision:"accepted"\|"rejected"}` | `Proposal` |
| `POST /proposals/:id/milestone` | `{resultText,evidenceUrl}` | `Proposal` с `milestone` |
| `POST /milestones/:id/confirm` | `{confirmed:true}` | `Proposal` с подтверждённым `milestone` |

## Task: форма данных

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

Эти пункты требуют настоящего backend и пока не отмечены проверенными.
