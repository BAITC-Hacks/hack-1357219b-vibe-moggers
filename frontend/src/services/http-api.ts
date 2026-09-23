import type {
  AnalyzeInput,
  AnalyzeResult,
  CardInput,
  CatalogFilters,
  FieldKey,
  Gateway,
  Industry,
  Milestone,
  Proposal,
  ProposalInput,
  Team,
} from '../domain/types'
import { ApiError } from './errors'
import { validateSuggestions } from './demo-api'
import { fieldKeys } from '../domain/scoring'
import {
  actorHeaders,
  taskFromApi,
  teamFromApi,
  proposalFromApi,
  type WireTask,
} from './api-contract'

export class HttpApi implements Gateway {
  readonly mode = 'api' as const
  constructor(
    private _actor: () => string,
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
          ...actorHeaders(this._actor()),
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
  private async list<T>(path: string): Promise<T[]> {
    const items: T[] = []
    let offset = 0
    for (;;) {
      const page = await this.request<{ items: T[]; total: number; limit: number }>(
        `${path}${path.includes('?') ? '&' : '?'}limit=100&offset=${offset}`,
      )
      if (!page || !Array.isArray(page.items) || !Number.isInteger(page.total))
        throw new ApiError('Ответ списка API не соответствует контракту.', 502)
      items.push(...page.items)
      offset += page.items.length
      if (offset >= page.total) return items
      if (!page.items.length) throw new ApiError('API вернул неполный список.', 502)
    }
  }
  async listTasks(filters: CatalogFilters = {}) {
    const params = new URLSearchParams(Object.entries(filters).filter(([, value]) => !!value))
    return (await this.list<WireTask>(`/tasks${params.size ? `?${params}` : ''}`)).map(taskFromApi)
  }
  async listOwnTasks() {
    return (await this.list<WireTask>('/business/tasks')).map(taskFromApi)
  }
  async getTask(id: string) {
    return taskFromApi(await this.request<WireTask>(`/tasks/${encodeURIComponent(id)}`))
  }
  async createTask(input: { draft: string; title: string; industry: Industry }) {
    return taskFromApi(await this.request<WireTask>('/tasks', 'POST', input))
  }
  async saveCard(id: string, input: CardInput) {
    return taskFromApi(
      await this.request<WireTask>(`/tasks/${encodeURIComponent(id)}/card`, 'PUT', input),
    )
  }
  async confirmTask(id: string, revision: number, fieldKeys: FieldKey[]) {
    return taskFromApi(
      await this.request<WireTask>(`/tasks/${encodeURIComponent(id)}/confirm`, 'POST', {
        revision,
        fieldKeys,
        reviewed: true,
      }),
    )
  }
  async publishTask(id: string, revision: number) {
    return taskFromApi(
      await this.request<WireTask>(`/tasks/${encodeURIComponent(id)}/publish`, 'POST', {
        revision,
      }),
    )
  }
  async analyze(input: AnalyzeInput) {
    const result = await this.request<AnalyzeResult>('/ai/analyze', 'POST', input)
    if (
      !result ||
      result.mode !== 'live' ||
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
  async listTeams() {
    return (await this.list<Team>('/teams')).map(teamFromApi)
  }
  async listProposals(taskId: string) {
    return (await this.list<Proposal>(`/tasks/${encodeURIComponent(taskId)}/proposals`)).map(
      proposalFromApi,
    )
  }
  async createProposal(taskId: string, input: ProposalInput) {
    return proposalFromApi(
      await this.request<Proposal>(`/tasks/${encodeURIComponent(taskId)}/proposals`, 'POST', input),
    )
  }
  async decideProposal(id: string, decision: 'accepted' | 'rejected') {
    return proposalFromApi(
      await this.request<Proposal>(`/proposals/${encodeURIComponent(id)}/decision`, 'POST', {
        decision,
      }),
    )
  }
  submitMilestone(id: string, input: { resultText: string; evidenceUrl: string }) {
    return this.request<Milestone>(`/proposals/${encodeURIComponent(id)}/milestone`, 'POST', input)
  }
  confirmMilestone(id: string) {
    return this.request<Milestone>(`/milestones/${encodeURIComponent(id)}/confirm`, 'POST', {
      confirmed: true,
    })
  }
}
