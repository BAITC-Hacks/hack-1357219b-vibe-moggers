import { afterEach, beforeEach, expect, it, vi } from 'vitest'
beforeEach(() => vi.resetModules())
afterEach(() => vi.unstubAllGlobals())
const response = (data: unknown, status = 200) => new Response(JSON.stringify({ data }), { status })
it('bootstraps CSRF then logs in with cookies and server-owned identity', async () => {
  const user = { id: 'u1', actorId: 't1', name: 'Team', email: 'team@example.test', role: 'team' }
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce(response({ user: null, csrfToken: 'first' }))
    .mockResolvedValueOnce(response({ user, csrfToken: 'rotated' }))
    .mockResolvedValueOnce(response(null))
  vi.stubGlobal('fetch', fetcher)
  const { authenticate, account, logout } = await import('./session')
  await authenticate('login', { email: user.email, password: 'not-a-real-password' })
  expect(account.value).toEqual(user)
  expect(fetcher.mock.calls[1]?.[1]).toMatchObject({
    credentials: 'include',
    headers: { 'X-CSRF-Token': 'first' },
  })
  await logout()
  expect(fetcher.mock.calls[2]?.[1]).toMatchObject({ headers: { 'X-CSRF-Token': 'rotated' } })
  expect(account.value).toBeNull()
})
it('does not pretend logout succeeded when the server failed', async () => {
  const user = {
    id: 'u1',
    actorId: 'b1',
    name: 'Business',
    email: 'b@example.test',
    role: 'business',
  }
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValueOnce(response({ user, csrfToken: 'csrf' }))
      .mockResolvedValueOnce(response(null, 503)),
  )
  const { restoreSession, logout, account } = await import('./session')
  await restoreSession()
  await expect(logout()).rejects.toMatchObject({ status: 503 })
  expect(account.value?.actorId).toBe('b1')
})
it('clears the account on session expiry without replaying writes', async () => {
  const fetcher = vi.fn().mockResolvedValue(response(null, 401))
  vi.stubGlobal('fetch', fetcher)
  const { request, account } = await import('./session')
  account.value = { id: 'u', actorId: 'b', name: 'B', email: 'b@example.test', role: 'business' }
  await expect(request('/ai/chat', 'POST', {})).rejects.toMatchObject({ status: 401 })
  expect(account.value).toBeNull()
  expect(fetcher).toHaveBeenCalledTimes(1)
})
