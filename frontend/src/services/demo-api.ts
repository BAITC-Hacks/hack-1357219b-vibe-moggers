import { initialData } from '../data/demo'
import {
  clone,
  emptyFields,
  fieldKeys,
  fieldMeta,
  fieldValid,
  safeUrl,
  scoreFields,
} from '../domain/scoring'
import type {
  AnalyzeInput,
  AnalyzeResult,
  CardInput,
  CatalogFilters,
  FieldKey,
  Gateway,
  Industry,
  Proposal,
  ProposalInput,
  Task,
} from '../domain/types'
import { ApiError } from './errors'

export const STORAGE_KEY = 'qadam:demo:v1'
interface StorageLike {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
}
type Database = ReturnType<typeof initialData>
const now = () => new Date().toISOString()
const id = (prefix: string) => `${prefix}-${crypto.randomUUID()}`
function requireText(text: string, label: string, min = 3) {
  if (text.trim().length < min) throw new ApiError(`Заполните поле «${label}»`, 422)
}

export class DemoApi implements Gateway {
  readonly mode = 'demo' as const
  private db: Database
  constructor(
    private actor: () => string,
    private storage: StorageLike,
  ) {
    const saved = storage.getItem(STORAGE_KEY)
    let parsed: Database | null = null
    try {
      if (saved) parsed = JSON.parse(saved) as Database
    } catch {
      /* Recover a damaged demo file with synthetic fixtures. */
    }
    this.db =
      parsed &&
      Array.isArray(parsed.tasks) &&
      Array.isArray(parsed.teams) &&
      Array.isArray(parsed.proposals)
        ? parsed
        : initialData()
  }
  private write<T>(mutate: (next: Database) => T): T {
    const next = clone(this.db)
    const result = mutate(next)
    try {
      this.storage.setItem(STORAGE_KEY, JSON.stringify(next))
    } catch {
      throw new ApiError(
        'Не удалось сохранить демо в браузере. Проверьте доступность локального хранилища.',
        500,
      )
    }
    this.db = next
    return clone(result)
  }
  private task(db: Database, taskId: string) {
    const task = db.tasks.find((item) => item.id === taskId)
    if (!task) throw new ApiError('Задача не найдена', 404)
    return task
  }
  private own(task: Task) {
    if (task.ownerId !== this.actor())
      throw new ApiError('Это действие доступно владельцу задачи', 403)
  }
  private version(task: Task, revision: number) {
    if (task.revision !== revision)
      throw new ApiError('Задача уже обновлена. Перезагрузите страницу перед сохранением.', 409)
  }
  private publicView(task: Task): Task {
    if (!task.publishedAt || !task.confirmedSnapshot)
      throw new ApiError('Задача ещё не опубликована', 404)
    return {
      ...clone(task),
      title: task.confirmedSnapshot.title,
      industry: task.confirmedSnapshot.industry,
      workingFields: clone(task.confirmedSnapshot.fields),
      draft: '',
      answers: [],
    }
  }
  async listTasks(filters: CatalogFilters = {}) {
    return this.db.tasks
      .filter((task) => task.publishedAt && task.confirmedSnapshot)
      .map((task) => this.publicView(task))
      .filter(
        (task) =>
          (!filters.industry || task.industry === filters.industry) &&
          (!filters.readiness || task.confirmedSnapshot?.readiness === filters.readiness),
      )
      .sort(
        (a, b) =>
          (b.confirmedSnapshot?.total ?? 0) - (a.confirmedSnapshot?.total ?? 0) ||
          a.publishedAt!.localeCompare(b.publishedAt!) ||
          a.id.localeCompare(b.id),
      )
  }
  async listOwnTasks() {
    return clone(
      this.db.tasks
        .filter((task) => task.ownerId === this.actor())
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
    )
  }
  async getTask(taskId: string) {
    const task = this.task(this.db, taskId)
    return task.ownerId === this.actor() ? clone(task) : this.publicView(task)
  }
  async createTask(input: { draft: string; title: string; industry: Industry }) {
    if (this.actor() !== 'demo-business-1')
      throw new ApiError('Создание задачи доступно в режиме бизнеса', 403)
    requireText(input.title, 'Название')
    requireText(input.draft, 'Описание', 10)
    return this.write((db) => {
      const task: Task = {
        ...input,
        id: id('task'),
        ownerId: this.actor(),
        answers: [],
        workingFields: emptyFields(),
        confirmedSnapshot: null,
        revision: 1,
        confirmedRevision: null,
        publishedAt: null,
        createdAt: now(),
      }
      db.tasks.push(task)
      return task
    })
  }
  async saveCard(taskId: string, input: CardInput) {
    return this.write((db) => {
      const task = this.task(db, taskId)
      this.own(task)
      this.version(task, input.revision)
      requireText(input.title, 'Название')
      const previous = task.workingFields
      task.workingFields = clone(input.fields)
      for (const key of fieldKeys) {
        const field = task.workingFields[key]
        field.value = field.value?.trim() || null
        field.confirmed = previous[key].value === field.value && previous[key].confirmed
        if (previous[key].value !== field.value && field.source?.quote !== field.value)
          field.source = null
      }
      if (previous.successMetric.value !== task.workingFields.successMetric.value) {
        task.workingFields.successTarget.confirmed = false
        task.workingFields.acceptanceMethod.confirmed = false
      }
      task.title = input.title.trim()
      task.industry = input.industry
      task.answers = clone(input.answers)
      task.revision++
      return task
    })
  }
  async confirmTask(taskId: string, revision: number, keys: FieldKey[]) {
    return this.write((db) => {
      const task = this.task(db, taskId)
      this.own(task)
      this.version(task, revision)
      for (const key of keys)
        if (fieldKeys.includes(key))
          task.workingFields[key].confirmed = fieldValid(key, task.workingFields[key].value)
      if (!task.workingFields.successMetric.confirmed) {
        task.workingFields.successTarget.confirmed = false
        task.workingFields.acceptanceMethod.confirmed = false
      }
      const fields = clone(task.workingFields)
      for (const key of fieldKeys)
        if (!fields[key].confirmed) fields[key] = { value: null, confirmed: false, source: null }
      task.confirmedSnapshot = {
        title: task.title,
        industry: task.industry,
        fields,
        ...scoreFields(fields),
        confirmedAt: now(),
      }
      task.revision++
      task.confirmedRevision = task.revision
      return task
    })
  }
  async publishTask(taskId: string, revision: number) {
    return this.write((db) => {
      const task = this.task(db, taskId)
      this.own(task)
      this.version(task, revision)
      if (!task.confirmedSnapshot || task.confirmedRevision !== task.revision)
        throw new ApiError('Сначала подтвердите текущую версию карточки', 422)
      task.publishedAt ||= now()
      return task
    })
  }
  async analyze(input: AnalyzeInput): Promise<AnalyzeResult> {
    const subject = /магазин|закуп|остатк/i.test(input.sources[0]?.text ?? '')
      ? 'магазине'
      : 'вашем процессе'
    return {
      mode: 'fallback',
      fieldSuggestions: [],
      warnings: ['Локальные вопросы по шаблону. Реальный AI доступен после подключения Go API.'],
      questions: [
        {
          id: 'q1',
          fieldKeys: ['context', 'need', 'users', 'usageScenario'],
          text: `Что сейчас происходит в ${subject}, что нужно изменить и кто будет пользоваться решением?`,
        },
        {
          id: 'q2',
          fieldKeys: ['dataSource', 'dataFormat', 'dataAccess'],
          text: 'Какие данные доступны, в каком формате и как команда сможет их получить?',
        },
        {
          id: 'q3',
          fieldKeys: [
            'deliverable',
            'deliveryFormat',
            'successMetric',
            'successTarget',
            'acceptanceMethod',
          ],
          text: 'Что должна создать команда и по какому измеримому результату вы примете работу?',
        },
        {
          id: 'q4',
          fieldKeys: ['deadline', 'constraints', 'contact', 'feedbackFormat'],
          text: 'Какие есть сроки и ограничения? Как команда будет получать обратную связь?',
        },
      ],
    }
  }
  async listTeams() {
    return this.db.teams.map((team) => ({
      ...clone(team),
      points:
        this.db.proposals.filter(
          (proposal) => proposal.teamId === team.id && proposal.milestone?.status === 'confirmed',
        ).length * 10,
    }))
  }
  async listProposals(taskId: string) {
    const task = this.task(this.db, taskId)
    if (!task.publishedAt && task.ownerId !== this.actor())
      throw new ApiError('Задача не найдена', 404)
    return clone(
      this.db.proposals.filter(
        (proposal) =>
          proposal.taskId === taskId &&
          (task.ownerId === this.actor() || proposal.teamId === this.actor()),
      ),
    )
  }
  async createProposal(taskId: string, input: ProposalInput) {
    if (!this.db.teams.some((team) => team.id === this.actor()))
      throw new ApiError('Выберите студенческую команду, чтобы отправить предложение', 403)
    requireText(input.idea, 'Идея', 10)
    if (!input.plan.length || input.plan.some((step) => step.trim().length < 3))
      throw new ApiError('Добавьте хотя бы один конкретный шаг плана', 422)
    if (!Number.isInteger(input.durationDays) || input.durationDays < 1 || input.durationDays > 365)
      throw new ApiError('Укажите срок от 1 до 365 дней', 422)
    if (!safeUrl(input.prototypeUrl))
      throw new ApiError('Укажите ссылку на прототип с http:// или https://', 422)
    return this.write((db) => {
      const task = this.task(db, taskId)
      if (!task.publishedAt) throw new ApiError('Задача ещё не опубликована', 422)
      const proposal: Proposal = {
        ...input,
        id: id('proposal'),
        taskId,
        teamId: this.actor(),
        status: 'pending',
        createdAt: now(),
        milestone: null,
      }
      db.proposals.push(proposal)
      return proposal
    })
  }
  async decideProposal(proposalId: string, decision: 'accepted' | 'rejected') {
    return this.write((db) => {
      const proposal = this.proposal(db, proposalId)
      this.own(this.task(db, proposal.taskId))
      if (proposal.status !== 'pending' && proposal.status !== decision)
        throw new ApiError('Решение по этому отклику уже принято', 409)
      proposal.status = decision
      return proposal
    })
  }
  private proposal(db: Database, proposalId: string) {
    const proposal = db.proposals.find((item) => item.id === proposalId)
    if (!proposal) throw new ApiError('Предложение не найдено', 404)
    return proposal
  }
  async submitMilestone(proposalId: string, input: { resultText: string; evidenceUrl: string }) {
    requireText(input.resultText, 'Результат этапа', 10)
    if (!safeUrl(input.evidenceUrl)) throw new ApiError('Укажите http(s) ссылку на результат', 422)
    return this.write((db) => {
      const proposal = this.proposal(db, proposalId)
      if (proposal.teamId !== this.actor() || proposal.status !== 'accepted')
        throw new ApiError('Этап доступен только выбранной команде', 403)
      if (proposal.milestone) throw new ApiError('Результат этапа уже отправлен', 409)
      proposal.milestone = {
        ...input,
        id: id('milestone'),
        proposalId,
        status: 'submitted',
        confirmedAt: null,
      }
      return proposal
    })
  }
  async confirmMilestone(milestoneId: string) {
    return this.write((db) => {
      const proposal = db.proposals.find((item) => item.milestone?.id === milestoneId)
      if (!proposal?.milestone) throw new ApiError('Этап не найден', 404)
      this.own(this.task(db, proposal.taskId))
      if (proposal.status !== 'accepted') throw new ApiError('Команда не выбрана', 403)
      proposal.milestone.status = 'confirmed'
      proposal.milestone.confirmedAt ||= now()
      return proposal
    })
  }
}

export function validateSuggestions(result: AnalyzeResult, input: AnalyzeInput): AnalyzeResult {
  const normalize = (value: string) => value.replace(/\s+/g, ' ').trim()
  const valid = result.fieldSuggestions.filter((suggestion) => {
    const source = input.sources.find((item) => item.id === suggestion.sourceId)
    return (
      Object.hasOwn(fieldMeta, suggestion.field) &&
      typeof suggestion.value === 'string' &&
      suggestion.value.trim() &&
      source &&
      normalize(source.text).includes(normalize(suggestion.value))
    )
  })
  return {
    ...result,
    fieldSuggestions: valid,
    warnings: [
      ...result.warnings,
      ...(valid.length < result.fieldSuggestions.length
        ? ['Некоторые предложения не имеют подтверждающего источника и были пропущены.']
        : []),
    ],
  }
}
