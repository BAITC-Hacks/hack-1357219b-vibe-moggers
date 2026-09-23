import { afterEach, describe, expect, it, vi } from 'vitest'
import { aiRequest, transcribeAudio } from './ai-api'

afterEach(() => vi.unstubAllGlobals())

describe('AI adapter', () => {
  it('sends chat context under the selected demonstration actor', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ data: { reply: 'Ответ' } }), { status: 200 }),
      )
    vi.stubGlobal('fetch', fetcher)
    await expect(aiRequest('team-3', '/ai/chat', { messages: [] })).resolves.toEqual({
      reply: 'Ответ',
    })
    expect(fetcher).toHaveBeenCalledWith(
      '/api/ai/chat',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'X-Demo-Actor': 'team-3' }),
      }),
    )
  })

  it('uploads a bounded browser recording as multipart data', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: { text: 'Нужен помощник для магазина.' } }), {
        status: 200,
      }),
    )
    vi.stubGlobal('fetch', fetcher)
    const result = await transcribeAudio(
      'demo-business-1',
      new Blob(['audio-bytes'], { type: 'audio/webm' }),
      'Описание задачи',
    )
    expect(result.text).toContain('магазина')
    const options = fetcher.mock.calls[0]?.[1] as RequestInit
    expect(options.headers).toMatchObject({ 'X-Demo-Actor': 'demo-business-1' })
    expect(options.body).toBeInstanceOf(FormData)
    expect((options.body as FormData).get('context')).toBe('Описание задачи')
  })

  it('keeps a useful error when transcription is unavailable', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { message: 'Транскрибация выключена' } }), {
          status: 503,
        }),
      ),
    )
    await expect(
      transcribeAudio('demo-business-1', new Blob(['audio']), 'Описание'),
    ).rejects.toMatchObject({ status: 503, message: 'Транскрибация выключена' })
  })
})
