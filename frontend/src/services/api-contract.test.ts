import { afterEach, expect, it, vi } from 'vitest'
import { actorHeaders, taskFromApi, teamFromApi, proposalFromApi } from './api-contract'
import { HttpApi } from './http-api'
import { emptyFields } from '../domain/scoring'
import type { Proposal } from '../domain/types'

afterEach(() => vi.unstubAllGlobals())

it('normalizes public snapshots and preserves server scoring', () => {
  const task = taskFromApi({
    id: 'public-id',
    title: 'Поставка',
    industry: 'logistics',
    fields: emptyFields(),
    total: 40,
    readiness: 'working',
    timestamp: '2026-09-23',
    scoreBreakdown: [
      {
        key: 'constraints',
        label: 'Constraints',
        earned: 5,
        max: 10,
        missingFields: ['constraints'],
      },
    ],
    publishedAt: '2026-09-23',
  })
  expect(task.confirmedSnapshot?.total).toBe(40)
  expect(task.confirmedSnapshot?.breakdown[0]).toMatchObject({
    key: 'limits',
    label: 'Ограничения',
    earned: 5,
  })
  expect(task.ownerId).toBe('')
  expect(task.workingFields.context).toEqual(emptyFields().context)
})

it('keeps private tasks editable only by the matching local actor', () => {
  expect(
    taskFromApi({ id: 'private', ownerId: '10000000-0000-4000-8000-000000000001', revision: 3 })
      .ownerId,
  ).toBe('demo-business-1')
  expect(actorHeaders('guest')).toEqual({})
  expect(actorHeaders('demo-business-1')).toEqual({
    'X-Demo-Actor': 'business:10000000-0000-4000-8000-000000000001',
  })
})

it('maps teams and proposals to the same selected identity', () => {
  const id = '00000000-0000-4000-8000-000000000003'
  expect(teamFromApi({ id, name: 'Team', interests: [], skills: [], technologies: [] }).id).toBe(
    'team-3',
  )
  expect(proposalFromApi({ teamId: id } as Proposal).teamId).toBe('team-3')
})

it('reads all pages and uses the actual owned-tasks route', async () => {
  const response = (items: unknown[], total: number) =>
    new Response(JSON.stringify({ data: { items, total, limit: 100 } }))
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce(response([{ id: 'one' }], 2))
    .mockResolvedValueOnce(response([{ id: 'two' }], 2))
  vi.stubGlobal('fetch', fetcher)
  expect((await new HttpApi(() => 'demo-business-1').listOwnTasks()).map((t) => t.id)).toEqual([
    'one',
    'two',
  ])
  expect(fetcher.mock.calls.map((call) => call[0])).toEqual([
    '/api/business/tasks?limit=100&offset=0',
    '/api/business/tasks?limit=100&offset=1',
  ])
})
