# Qadam UX contract

## Canonical UI map

| Capability | Canonical owner | Source of truth | Allowed variants | Verification |
|---|---|---|---|---|
| Select/Listbox | Native `<select>` in `RoleSwitcher.vue`, `Catalog.vue`, `TaskEditor.vue` | This contract; OS popup is accepted | Demo role, industry, readiness | Keyboard selection, open popup, mobile viewport |
| Demo identity | `RoleSwitcher.vue`, `Profile.vue`, workspace store | `frontend/API-CONTRACT.md` | Guest, business, five preset teams | Switch roles and reload in browser |
| Form | `TaskEditor.vue`, `formControls.ts` | This contract | Task and proposal forms | Invalid, pending, success and retry states |
| Scrollbar | Global baseline in `frontend/src/styles.css` | `DESIGN.md` | Document, sidebar and modal internal scroll | Narrow and short viewport browser checks |
| Toast | Workspace store + `App.vue` | This contract | Success, error and info | Live region and visible stacking |
| CRUD | Gateway + router + workspace store | `docs/SPEC.md`, `frontend/API-CONTRACT.md` | Business task and team proposal flows | Unit tests and complete demo flow |
| Voice capture | `VoiceRecorder.vue`, `ai-api.ts` | `frontend/AI-CONTRACT.md` | Description and question answer | Permission denial, success mock, unmount cleanup |
| AI clarification | `HttpApi.analyze` | `frontend/AI-CONTRACT.md` | Clarify and assemble stages | Malformed/fallback response tests |
| AI chat | `AiAssistant.vue`, `ai-api.ts` | `frontend/AI-CONTRACT.md` | Current page and optional task context | Success, retry, Escape and narrow viewport |
| Dialog | Native dialog in `AiAssistant.vue` | This contract | Modal assistant drawer | Focus, Escape, backdrop, reduced motion |

## Workflow ledger

| Operation | Trigger | Pending | Success | Failure recovery |
|---|---|---|---|---|
| Select demo role | header select or `/start` | disable selector/action | route to role workspace | keep selection, inline/toast error |
| Record voice | microphone button | elapsed timer, stop action | upload recording | stop tracks, show retry and keep existing text |
| Transcribe | recording stops | stable transcription status | append editable text | preserve current field and allow another recording |
| Generate questions | task description submit | disable duplicate submit | 3–5 relevant questions | retain draft, retry or fill manually |
| Ask Qadam AI | chat submit | busy state | append plain text response | retain question for retry |

The public catalog works with `guest`. Creating/editing tasks requires the business demo
profile; proposals and results require one of the pre-created team profiles. Client role
checks guide the demo flow; a production system would still enforce permissions server-side.

## Voice and privacy

Microphone access starts only after an explicit click. The browser stops every media track
after recording, on error and on component unmount. The recording is sent once to the Go
backend and is not placed in localStorage. UI copy tells users not to include confidential
data. The backend contract requires deletion after transcription and forbids audio logging.

## Responsive and accessibility

Controls use native buttons, labels and selects. Recording, transcription, success and error
states have live status text. Color is never the only state cue. Keyboard focus remains visible.
At narrow widths, team rows and voice callouts reflow; the AI drawer stays within `100dvh`.
