import { ApiError } from './errors'

const base = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/$/, '')

async function parseResponse<T>(response: Response): Promise<T> {
  const payload = await response.json().catch(() => null)
  if (!response.ok)
    throw new ApiError(
      payload?.error?.message || `AI-сервис вернул ошибку ${response.status}.`,
      response.status,
    )
  if (!payload || !('data' in payload))
    throw new ApiError('Ответ AI-сервиса не соответствует контракту.', 502)
  return payload.data as T
}

export async function aiRequest<T>(
  actorId: string,
  path: string,
  body: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const controller = new AbortController()
  const abort = () => controller.abort()
  signal?.addEventListener('abort', abort, { once: true })
  if (signal?.aborted) controller.abort()
  const timer = globalThis.setTimeout(abort, 25_000)
  try {
    const response = await fetch(`${base}${path}`, {
      method: 'POST',
      signal: controller.signal,
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-Demo-Actor': actorId,
      },
      body: JSON.stringify(body),
    })
    return await parseResponse<T>(response)
  } catch (error) {
    if (error instanceof ApiError) throw error
    if (error instanceof Error && error.name === 'AbortError')
      throw new ApiError('Запрос остановлен. Можно повторить его.', 499)
    throw new ApiError('AI-сервис недоступен. Введённые данные сохранены.', 503)
  } finally {
    globalThis.clearTimeout(timer)
    signal?.removeEventListener('abort', abort)
  }
}

export async function transcribeAudio(
  actorId: string,
  audio: Blob,
  context: string,
  signal?: AbortSignal,
) {
  if (audio.size > 15 * 1024 * 1024)
    throw new ApiError('Запись слишком большая. Запишите более короткий ответ.', 413)
  const controller = new AbortController()
  const abort = () => controller.abort()
  signal?.addEventListener('abort', abort, { once: true })
  if (signal?.aborted) controller.abort()
  const timer = globalThis.setTimeout(abort, 70_000)
  const type = audio.type || 'audio/webm'
  const extension = type.includes('mp4') ? 'm4a' : type.includes('ogg') ? 'ogg' : 'webm'
  const form = new FormData()
  form.append('audio', audio, `qadam-recording.${extension}`)
  form.append('context', context.slice(0, 1000))
  form.append('languages', 'ru,kk')
  try {
    const response = await fetch(`${base}/ai/transcribe`, {
      method: 'POST',
      signal: controller.signal,
      headers: { Accept: 'application/json', 'X-Demo-Actor': actorId },
      body: form,
    })
    const result = await parseResponse<{ text: string; language?: string }>(response)
    if (!result || typeof result.text !== 'string' || !result.text.trim())
      throw new ApiError('Не удалось распознать речь. Запишите ответ ещё раз.', 422)
    if (result.text.length > 6000) throw new ApiError('Запись получилась слишком длинной.', 422)
    return result
  } catch (error) {
    if (error instanceof ApiError) throw error
    if (error instanceof Error && error.name === 'AbortError')
      throw new ApiError('Распознавание остановлено. Можно записать ответ ещё раз.', 499)
    throw new ApiError('Не удалось отправить запись. Проверьте Go API и повторите.', 503)
  } finally {
    globalThis.clearTimeout(timer)
    signal?.removeEventListener('abort', abort)
  }
}
