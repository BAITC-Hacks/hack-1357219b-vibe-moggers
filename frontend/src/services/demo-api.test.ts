import { beforeEach, describe, expect, it } from 'vitest'
import { DemoApi, validateSuggestions } from './demo-api'
import { clone, emptyFields, fieldKeys, getReadiness, scoreFields } from '../domain/scoring'
import { demo40, demo95, demoDraft, demoTitle } from '../data/demo'
import type { FieldKey, Task } from '../domain/types'

describe('demo workflow invariants', () => {
  let actor: string
  let saved: Map<string, string>
  let api: DemoApi
  beforeEach(() => {
    actor = 'demo-business-1'
    saved = new Map()
    api = new DemoApi(() => actor, {
      getItem: (key) => saved.get(key) ?? null,
      setItem: (key, value) => {
        saved.set(key, value)
      },
    })
  })
  async function saveExample(task: Task, values: Record<string, string>) {
    const fields = emptyFields()
    for (const [key, value] of Object.entries(values)) fields[key as FieldKey].value = value
    return api.saveCard(task.id, {
      revision: task.revision,
      title: task.title,
      industry: task.industry,
      answers: [],
      fields,
    })
  }
  it('keeps guest browsing public and gives a new customer only their own tasks', async () => {
    actor = 'guest'
    expect(await api.listTasks()).toHaveLength(5)
    expect(await api.listProposals('seed-task-1')).toHaveLength(0)
    await expect(
      api.createTask({ title: demoTitle, draft: demoDraft, industry: 'retail' }),
    ).rejects.toMatchObject({ status: 403 })
    const owner = api.registerProfile('business', 'Мой магазин')
    actor = owner.id
    expect(await api.listOwnTasks()).toHaveLength(0)
    const task = await api.createTask({ title: demoTitle, draft: demoDraft, industry: 'retail' })
    expect(task.ownerId).toBe(owner.id)
    expect(task.ownerName).toBe('Мой магазин')
    expect(await api.listOwnTasks()).toHaveLength(1)
    await expect(api.decideProposal('proposal-1', 'accepted')).rejects.toMatchObject({
      status: 403,
    })
    actor = api.registerProfile('business', 'Другая компания').id
    await expect(api.getTask(task.id)).rejects.toMatchObject({ status: 404 })
  })
  it('creates an independent team, preserves it after reload and submits under its own name', async () => {
    const profile = api.registerProfile('team', 'Новая команда')
    actor = profile.id
    const proposal = await api.createProposal('seed-task-5', {
      idea: 'Соберём рабочий прототип для кооператива',
      plan: ['Уточним доступные данные'],
      durationDays: 10,
      prototypeUrl: 'https://example.org/prototype',
    })
    expect(proposal.teamId).toBe(profile.id)
    expect((await api.listTeams()).find((team) => team.id === profile.id)?.name).toBe(
      'Новая команда',
    )
    const reloaded = new DemoApi(() => actor, {
      getItem: (key) => saved.get(key) ?? null,
      setItem: (key, value) => {
        saved.set(key, value)
      },
    })
    expect(reloaded.getProfile(profile.id)).toEqual(profile)
    expect(reloaded.savedProfile('team')).toEqual(profile)
    expect(reloaded.getProfile('team-1')).toBeNull()
    expect(await reloaded.listProposals('seed-task-5')).toHaveLength(1)
    await expect(
      reloaded.createTask({ title: demoTitle, draft: demoDraft, industry: 'retail' }),
    ).rejects.toMatchObject({ status: 403 })
  })
  it('does not create a profile when persistence fails', () => {
    const failing = new DemoApi(() => 'guest', {
      getItem: () => null,
      setItem: () => {
        throw new Error('Quota')
      },
    })
    expect(() => failing.registerProfile('team', 'Команда')).toThrow('Не удалось сохранить')
    expect(failing.savedProfile('team')).toBeNull()
  })
  it('moves a manually confirmed task from 4th at 40 to 1st at 95; unconfirmed edits stay private', async () => {
    let task = await api.createTask({ title: demoTitle, draft: demoDraft, industry: 'retail' })
    task = await saveExample(task, demo40)
    expect(scoreFields(task.workingFields, true).total).toBe(40)
    await expect(api.publishTask(task.id, task.revision)).rejects.toMatchObject({ status: 422 })
    task = await api.confirmTask(task.id, task.revision, fieldKeys)
    task = await api.publishTask(task.id, task.revision)
    expect((await api.listTasks()).findIndex((item) => item.id === task.id)).toBe(3)
    task = await saveExample(task, demo95)
    actor = 'team-1'
    const publicTask = await api.getTask(task.id)
    expect(publicTask.confirmedSnapshot?.total).toBe(40)
    expect(publicTask.workingFields.contact.value).toBeNull()
    actor = 'demo-business-1'
    task = await api.confirmTask(task.id, task.revision, fieldKeys)
    expect(task.confirmedSnapshot?.total).toBe(95)
    expect((await api.listTasks())[0]?.id).toBe(task.id)
  })
  it('invalidates dependent success fields and can lower a published score', async () => {
    let task = await api.createTask({ title: demoTitle, draft: demoDraft, industry: 'retail' })
    task = await saveExample(task, demo95)
    task = await api.confirmTask(task.id, task.revision, fieldKeys)
    const fields = clone(task.workingFields)
    fields.successMetric.value = null
    task = await api.saveCard(task.id, {
      revision: task.revision,
      title: task.title,
      industry: task.industry,
      answers: [],
      fields,
    })
    expect(task.workingFields.successTarget.confirmed).toBe(false)
    expect(task.workingFields.acceptanceMethod.confirmed).toBe(false)
    task = await api.confirmTask(task.id, task.revision, fieldKeys)
    expect(task.confirmedSnapshot?.total).toBe(80)
    expect(task.confirmedSnapshot?.fields.successTarget.value).toBeNull()
  })
  it('allows any team to propose on a score-20 task without a total proposal cap', async () => {
    const low = (await api.listTasks()).find((task) => task.confirmedSnapshot?.total === 20)!
    for (let i = 0; i < 7; i++) {
      actor = `team-${(i % 5) + 1}`
      await api.createProposal(low.id, {
        idea: 'Уточним задачу и предложим рабочий прототип.',
        plan: ['Уточним данные'],
        durationDays: 7,
        prototypeUrl: 'https://example.org/demo',
      })
    }
    actor = 'demo-business-1'
    expect((await api.listProposals(low.id)).length).toBe(8)
  })
  it('allows multiple accepted teams and awards a confirmed milestone only once, including after reload', async () => {
    await api.decideProposal('proposal-1', 'accepted')
    await api.decideProposal('proposal-2', 'accepted')
    expect(
      (await api.listProposals('seed-task-1')).filter((proposal) => proposal.status === 'accepted'),
    ).toHaveLength(2)
    actor = 'team-1'
    const proposal = await api.submitMilestone('proposal-1', {
      resultText: 'Подготовлен макет таблицы и структура данных.',
      evidenceUrl: 'https://example.org/demo',
    })
    await expect(api.confirmMilestone(proposal.milestone!.id)).rejects.toMatchObject({
      status: 403,
    })
    actor = 'demo-business-1'
    await Promise.all([
      api.confirmMilestone(proposal.milestone!.id),
      api.confirmMilestone(proposal.milestone!.id),
    ])
    expect((await api.listTeams()).find((team) => team.id === 'team-1')?.points).toBe(10)
    const reloaded = new DemoApi(() => actor, {
      getItem: (key) => saved.get(key) ?? null,
      setItem: (key, value) => {
        saved.set(key, value)
      },
    })
    expect((await reloaded.listTeams()).find((team) => team.id === 'team-1')?.points).toBe(10)
  })
  it('allows declining all proposals and prevents milestones from an unselected team', async () => {
    await api.decideProposal('proposal-1', 'rejected')
    await api.decideProposal('proposal-2', 'rejected')
    expect(
      (await api.listProposals('seed-task-1')).every((proposal) => proposal.status === 'rejected'),
    ).toBe(true)
    actor = 'team-1'
    await expect(
      api.submitMilestone('proposal-1', {
        resultText: 'Недопустимый результат этапа.',
        evidenceUrl: 'https://example.org/demo',
      }),
    ).rejects.toMatchObject({ status: 403 })
  })
  it('rejects a stale edit and does not overwrite the latest value', async () => {
    const task = await api.getTask('seed-task-1')
    await saveExample(task, demo95)
    await expect(saveExample(task, demo40)).rejects.toMatchObject({ status: 409 })
    expect(scoreFields((await api.getTask(task.id)).workingFields, true).total).toBe(95)
  })
  it('drops unsupported facts and unknown sources from AI suggestions', () => {
    const input = {
      stage: 'assemble' as const,
      sources: [{ id: 'answer-1', text: 'Формат: CSV за 6 месяцев.' }],
      currentFields: emptyFields(),
    }
    const result = validateSuggestions(
      {
        mode: 'live',
        questions: [],
        warnings: [],
        fieldSuggestions: [
          { field: 'dataFormat', value: 'CSV за 6 месяцев', sourceId: 'answer-1' },
          { field: 'contact', value: 'invented@example.org', sourceId: 'answer-1' },
          { field: 'dataSource', value: 'CSV за 6 месяцев', sourceId: 'missing' },
        ],
      },
      input,
    )
    expect(result.fieldSuggestions).toHaveLength(1)
    expect(result.warnings).toHaveLength(1)
  })
  it.each([
    [39, 'draft'],
    [40, 'working'],
    [69, 'working'],
    [70, 'ready'],
    [89, 'ready'],
    [90, 'priority'],
  ] as const)('classifies %i as %s', (score, readiness) => {
    expect(getReadiness(score)).toBe(readiness)
  })
})
