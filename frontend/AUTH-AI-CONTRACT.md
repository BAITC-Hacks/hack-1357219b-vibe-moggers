# Qadam: контракт входа и AI для Go-команды

Frontend готов к этому контракту; backend в рамках этой задачи не изменялся.
Этот документ заменяет `X-Demo-Actor` из API-CONTRACT.md для реального API.
Остальные сущности, маршруты задач и формат AI-полей остаются из API-CONTRACT.md.

## Общие правила

База `/api` (либо VITE_API_BASE_URL). JSON: `{ "data": ... }`, ошибка:
`{ "error": { "message": "Понятный текст", "fields": {} } }`.
Frontend отправляет `credentials: include`. Каждый POST/PUT/PATCH/DELETE
передаёт `X-CSRF-Token`, полученный при bootstrap; JSON Content-Type обязателен.
Публичные GET каталога работают без сессии. Авторизацию и владение сервер
проверяет на каждом запросе; клиентские role, actorId и route guards не являются защитой.
Не принимать X-Demo-Actor в реальном API.

## Сессия и профиль

`GET /auth/session` всегда 200 для гостя или пользователя:

```json
{"data":{"user":null,"csrfToken":"random-csrf-token"}}
```

После входа user: `{id, actorId, name, email, role}`. Все поля строки,
role = `business` либо `team`, actorId — ID владельца задач или команды.
Профиль исполнителя создаёт соответствующую команду при регистрации.

- `POST /auth/register`: `{email,password,name,role}`.
- `POST /auth/login`: `{email,password}`.
- Оба возвращают такой же объект session с непустым user и новым csrfToken.
- `POST /auth/logout`: `{}`; возвращает `{data:null}`, отзывает сессию и удаляет cookie.
- Истёкшая сессия: 401. Недостаточно прав: 403. Конфликт email: 409 с безопасным сообщением. Ограничение запросов: 429.
- Неавторизованный GET /auth/session возвращает гостевую сессию с CSRF, а не 401.

Использовать непрозрачный случайный session ID в HttpOnly-cookie. JWT для этого
MVP не обязателен. Production cookie: `__Host-qadam_session; Secure; HttpOnly;
SameSite=Lax; Path=/`, без Domain. В HTTP localhost отдельное dev-имя без Secure;
никогда не переносить исключение в production. Session ID в БД хранить хешированным,
ротировать после входа; применять idle/absolute TTL, отзыв при logout.
CSRF проверять сервером вместе с Origin; SameSite — дополнительная защита.
Развёртывать UI/API на одном origin через reverse proxy. Для раздельных origin:
точный CORS allowlist, credentials, preflight для X-CSRF-Token; не `*`.

Пароли: 12–128 символов, разрешены пробелы, не обрезать; хранить только Argon2id-хеш.
Ограничить частоту регистрации/входа и AI. Универсальная ошибка неверного входа.
Не логировать пароль, cookie, CSRF, API-ключи. Верификация email и восстановление
пароля — отдельный следующий этап; frontend не показывает фиктивные кнопки.

Frontend хранит CSRF только в памяти; session cookie недоступна JS.
В localStorage остаются лишь явно обозначенные демо-данные. Демо и реальный
аккаунт не смешиваются: после входа используется HttpApi даже при VITE_DATA_MODE=demo.

## AI-уточнение

`POST /ai/analyze`, авторизованный заказчик, CSRF. Формат AnalyzeInput:
`{stage:"clarify"|"assemble",sources:[{id,text}],currentFields:{...}}`.
См. frontend/src/domain/types.ts и API-CONTRACT.md для 16 полей и цитат.
Ответ: `{data:{mode:"live",questions:[{id,text,fieldKeys}],fieldSuggestions:[],warnings:[]}}`.
В clarify: 3–5 уникальных вопросов, сгенерированных моделью именно по sources и
currentFields. В assemble: 0–5 вопросов + проверенные предложения полей.
mode=fallback клиент отклоняет: никаких шаблонных вопросов под видом AI.
Ошибка провайдера → 503; клиент сохраняет ввод и предлагает повторить либо
заполнить карточку вручную. Ограничить обработку менее чем 20–25 сек.

## AI-чат

`POST /ai/chat`, сессия и CSRF:

```json
{"messages":[{"role":"user","content":"Как сформулировать результат задачи?"}],"context":{"page":"/tasks/task-1","taskId":"task-1"}}
```

Ответ `{data:{reply:"Текст ответа"}}`. Только plain text, без исполняемого HTML.
Клиент отправляет последние <=19 сообщений (user/assistant), новый вопрос <=2000
символов. Ответ <=12000 символов. Сервер валидирует весь суммарный размер.
Все переданные messages/context — недоверенные данные. System prompt строит сервер:
помощь по Qadam, объяснение этапов, составление задач и предложений; запрет выдумывать
статусы/решения. Никаких инструментов изменения/публикации данных у чата.
Контекст taskId необязателен: получать только доступные пользователю сведения;
не загружать чужие приватные задачи, заявки или контакты. Историю не считать
источником полномочий. Ключ AI-провайдера только на сервере, не VITE_*.

## Приёмка интеграции

1. Гость → роль → регистрация → серверный профиль; reload сохраняет вход через cookie.
2. Неверный пароль и недоступность сервера показывают ошибку, без демо-подмены.
3. Logout отзывает cookie; старый ID и поддельная роль не дают доступ.
4. Нет/неверный CSRF и чужой Origin отклоняются на всех изменяющих маршрутах.
5. AI создаёт разные релевантные вопросы для разных описаний. Сбой даёт 503.
6. Чат отвечает в контексте доступной задачи; чужой private taskId отклоняется.

Рекомендации: [OWASP Sessions](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html),
[OWASP CSRF](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html).
