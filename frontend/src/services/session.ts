import { ref } from 'vue'
import { ApiError } from './errors'
export interface Account {
  id: string
  actorId: string
  name: string
  email: string
  role: 'business' | 'team'
}
export const account = ref<Account | null>(null)
let csrf = ''
let restored: Promise<void> | undefined
const base = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '')
export async function request<T>(
  path: string,
  method = 'GET',
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const controller = new AbortController()
  const abort = () => controller.abort()
  signal?.addEventListener('abort', abort, { once: true })
  if (signal?.aborted) controller.abort()
  const timer = setTimeout(abort, 25_000)
  try {
    const response = await fetch(`${base}${path}`, {
      method,
      credentials: 'include',
      signal: controller.signal,
      headers: {
        Accept: 'application/json',
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(method === 'GET' ? {} : { 'X-CSRF-Token': csrf }),
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    })
    const payload = await response.json().catch(() => null)
    if (response.status === 401) account.value = null
    if (!response.ok)
      throw new ApiError(
        response.status === 401
          ? 'Войдите в аккаунт, чтобы продолжить.'
          : payload?.error?.message || 'Сервис недоступен. Попробуйте позже.',
        response.status,
      )
    if (!payload || !('data' in payload))
      throw new ApiError('Сервис ещё не подключён или вернул неверный ответ.', 502)
    return payload.data as T
  } catch (error) {
    if (error instanceof ApiError) throw error
    throw new ApiError(
      controller.signal.aborted
        ? 'Запрос прерван. Попробуйте ещё раз.'
        : 'Сервис недоступен. Попробуйте позже.',
      503,
    )
  } finally {
    clearTimeout(timer)
    signal?.removeEventListener('abort', abort)
  }
}
function acceptSession(value: { user: Account | null; csrfToken: string }) {
  if (
    !value ||
    typeof value.csrfToken !== 'string' ||
    !value.csrfToken ||
    (value.user &&
      (!['business', 'team'].includes(value.user.role) ||
        !value.user.actorId ||
        !value.user.email ||
        !value.user.name))
  )
    throw new ApiError('Неверный ответ сервиса входа.', 502)
  csrf = value.csrfToken
  account.value = value.user
}
export async function restoreSession() {
  if (!restored)
    restored = request<{ user: Account | null; csrfToken: string }>('/auth/session')
      .then(acceptSession)
      .catch(() => {
        /* Guest browsing stays available. Login retries bootstrap. */
      })
  return restored
}
export async function authenticate(
  kind: 'login' | 'register',
  input: { email: string; password: string; name?: string; role?: 'business' | 'team' },
) {
  acceptSession(await request('/auth/session'))
  acceptSession(await request(`/auth/${kind}`, 'POST', input))
  if (!account.value) throw new ApiError('Сервер не подтвердил вход.', 502)
}
export async function logout() {
  await request('/auth/logout', 'POST', {})
  account.value = null
  csrf = ''
  restored = undefined
}
export function csrfToken() {
  return csrf
}
