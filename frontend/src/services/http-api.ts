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
  Team,
} from '../domain/types'
import { ApiError } from './errors'
import { validateSuggestions } from './demo-api'
import { fieldKeys } from '../domain/scoring'

export class HttpApi implements Gateway {
  readonly mode = 'api' as const
  constructor(
    private actor: () => string,
    private base = import.meta.env.VITE_API_BASE_URL || '/api',
  ) {}
  private async request<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 20_000)
    try {
      const response = await fetch(`${this.base.replace(/\/$/, '')}${path}`, {
        method,
        signal: controller.signal,
        headers: {
          Accept: 'application/json',
          'X-Demo-Actor': this.actor(),
          ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        },
        ...(body === undefined ? {} : { body: JSON.stringify(body) }),
      })
      const payload = await response.json().catch(() => null)
      if (!response.ok)
        throw new ApiError(
          payload?.error?.message || `Сервер вернул ошибку ${response.status}`,
          response.status,
          payload?.error?.fields,
        )
      if (!payload || !('data' in payload))
        throw new ApiError('Ответ API не соответствует контракту: ожидается объект data', 502)
      return payload.data as T
    } catch (error) {
      if (error instanceof ApiError) throw error
      if (error instanceof Error && error.name === 'AbortError')
        throw new ApiError('Сервер не ответил вовремя. Введённые данные сохранены в форме.', 504)
      throw new ApiError(
        'Не удалось связаться с Go API. Проверьте адрес сервера и подключение.',
        503,
      )
    } finally {
      clearTimeout(timer)
    }
  }
  listTasks(filters: CatalogFilters = {}) {
    const params = new URLSearchParams(Object.entries(filters).filter(([, value]) => !!value))
    return this.request<Task[]>(`/tasks${params.size ? `?${params}` : ''}`)
  }
  listOwnTasks() {
    return this.request<Task[]>('/tasks?scope=owned')
  }
  getTask(id: string) {
    return this.request<Task>(`/tasks/${encodeURIComponent(id)}`)
  }
  createTask(input: { draft: string; title: string; industry: Industry }) {
    return this.request<Task>('/tasks', 'POST', input)
  }
  saveCard(id: string, input: CardInput) {
    return this.request<Task>(`/tasks/${encodeURIComponent(id)}/card`, 'PUT', input)
  }
  confirmTask(id: string, revision: number, fieldKeys: FieldKey[]) {
    return this.request<Task>(`/tasks/${encodeURIComponent(id)}/confirm`, 'POST', {
      revision,
      fieldKeys,
      reviewed: true,
    })
  }
  publishTask(id: string, revision: number) {
    return this.request<Task>(`/tasks/${encodeURIComponent(id)}/publish`, 'POST', { revision })
  }
  async analyze(input: AnalyzeInput) {
    const result = await this.request<AnalyzeResult>('/ai/analyze', 'POST', input)
    if (
      !result ||
      !['live', 'fallback'].includes(result.mode) ||
      !Array.isArray(result.questions) ||
      !Array.isArray(result.fieldSuggestions) ||
      !Array.isArray(result.warnings)
    )
      throw new ApiError('AI вернул некорректный ответ. Карточку можно заполнить вручную.', 502)
    if (input.stage === 'clarify' && result.questions.length < 3)
      throw new ApiError(
        'AI вернул недостаточно вопросов. Попробуйте ещё раз или заполните карточку вручную.',
        502,
      )
    if (
      result.questions.length > 5 ||
      new Set(result.questions.map((q) => q?.id)).size !== result.questions.length ||
      result.questions.some(
        (q) =>
          !q ||
          typeof q.text !== 'string' ||
          !q.text.trim() ||
          typeof q.id !== 'string' ||
          !Array.isArray(q.fieldKeys) ||
          q.fieldKeys.some((key) => !fieldKeys.includes(key)),
      )
    )
      throw new ApiError('AI вернул некорректные вопросы. Ваше описание сохранено.', 502)
    if (
      result.fieldSuggestions.some(
        (s) =>
          !s ||
          typeof s.field !== 'string' ||
          typeof s.value !== 'string' ||
          typeof s.sourceId !== 'string',
      )
    )
      throw new ApiError('AI вернул некорректные поля. Заполните карточку вручную.', 502)
    return validateSuggestions(result, input)
  }
  listTeams() {
    return this.request<Team[]>('/teams')
  }
  listProposals(taskId: string) {
    return this.request<Proposal[]>(`/tasks/${encodeURIComponent(taskId)}/proposals`)
  }
  createProposal(taskId: string, input: ProposalInput) {
    return this.request<Proposal>(`/tasks/${encodeURIComponent(taskId)}/proposals`, 'POST', input)
  }
  decideProposal(id: string, decision: 'accepted' | 'rejected') {
    return this.request<Proposal>(`/proposals/${encodeURIComponent(id)}/decision`, 'POST', {
      decision,
    })
  }
  submitMilestone(id: string, input: { resultText: string; evidenceUrl: string }) {
    return this.request<Proposal>(`/proposals/${encodeURIComponent(id)}/milestone`, 'POST', input)
  }
  confirmMilestone(id: string) {
    return this.request<Proposal>(`/milestones/${encodeURIComponent(id)}/confirm`, 'POST', {
      confirmed: true,
    })
  }
}
