import { afterEach, describe, expect, it, vi } from 'vitest'
import { HttpApi } from './http-api'
import { emptyFields } from '../domain/scoring'

afterEach(() => vi.unstubAllGlobals())
describe('REST adapter', () => {
  it('uses the agreed envelope and selected demo actor', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ data: [] }), { status: 200 }))
    vi.stubGlobal('fetch', fetcher)
    const api = new HttpApi(() => 'team-2', '/api')
    expect(await api.listTasks({ industry: 'retail' })).toEqual([])
    expect(fetcher).toHaveBeenCalledWith(
      '/api/tasks?industry=retail',
      expect.objectContaining({ headers: expect.objectContaining({ 'X-Demo-Actor': 'team-2' }) }),
    )
  })
  it('reports server errors instead of silently switching to synthetic data', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { message: 'Сервис временно недоступен' } }), {
          status: 503,
        }),
      ),
    )
    const api = new HttpApi(() => 'demo-business-1', '/api')
    await expect(api.listTasks()).rejects.toMatchObject({
      status: 503,
      message: 'Сервис временно недоступен',
    })
    expect(api.mode).toBe('api')
  })
  it('rejects malformed JSON from AI without losing the provided input', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: { mode: 'live', questions: 'not-an-array' } }), {
          status: 200,
        }),
      ),
    )
    const input = {
      stage: 'clarify' as const,
      sources: [{ id: 'draft', text: 'Нужен помощник для магазина.' }],
      currentFields: emptyFields(),
    }
    await expect(new HttpApi(() => 'demo-business-1', '/api').analyze(input)).rejects.toMatchObject(
      { status: 502 },
    )
    expect(input.sources[0]?.text).toBe('Нужен помощник для магазина.')
  })
})
