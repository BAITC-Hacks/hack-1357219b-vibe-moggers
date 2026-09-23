import type { Fields, Industry, Proposal, ScoreGroup, Snapshot, Task, Team } from '../domain/types'
import { emptyFields, groups } from '../domain/scoring'

const businessId = '10000000-0000-4000-8000-000000000001'
const teamPrefix = '00000000-0000-4000-8000-00000000000'

export function actorHeaders(actor: string): Record<string, string> {
  if (actor === 'guest') return {}
  if (actor === 'demo-business-1') return { 'X-Demo-Actor': `business:${businessId}` }
  if (/^team-[1-5]$/.test(actor)) return { 'X-Demo-Actor': `team:${teamPrefix}${actor.slice(-1)}` }
  return { 'X-Demo-Actor': `team:${actor}` }
}

function teamId(id: string) {
  return id.startsWith(teamPrefix) && /^[1-5]$/.test(id.slice(teamPrefix.length))
    ? `team-${id.slice(-1)}`
    : id
}

type WireSnapshot = Omit<Snapshot, 'breakdown' | 'confirmedAt'> & {
  scoreBreakdown: ScoreGroup[]
  timestamp: string
}
export type WireTask = Partial<Omit<Task, 'confirmedSnapshot'>> & {
  id: string
  confirmedSnapshot?: WireSnapshot | null
} & Partial<WireSnapshot>

function snapshot(value: WireSnapshot): Snapshot {
  return {
    title: value.title,
    industry: value.industry,
    fields: { ...emptyFields(), ...value.fields },
    total: value.total,
    readiness: value.readiness,
    confirmedAt: value.timestamp,
    breakdown: value.scoreBreakdown.map((row) => {
      const key = row.key === 'constraints' ? 'limits' : row.key
      return { ...row, key, label: groups.find((group) => group.key === key)?.label || row.label }
    }),
  }
}

export function taskFromApi(value: WireTask): Task {
  const confirmed = value.confirmedSnapshot
    ? snapshot(value.confirmedSnapshot)
    : value.fields && value.scoreBreakdown
      ? snapshot(value as WireSnapshot)
      : null
  return {
    id: value.id,
    ownerId: value.ownerId === businessId ? 'demo-business-1' : value.ownerId || '',
    title: value.title || '',
    industry: value.industry || ('services' as Industry),
    draft: value.draft || '',
    answers: value.answers || [],
    workingFields: { ...emptyFields(), ...(value.workingFields || confirmed?.fields) } as Fields,
    confirmedSnapshot: confirmed,
    revision: value.revision ?? 0,
    confirmedRevision: value.confirmedRevision ?? null,
    publishedAt: value.publishedAt || null,
    createdAt: value.createdAt || value.publishedAt || '',
  }
}

export function teamFromApi(team: Team): Team {
  return {
    ...team,
    id: teamId(team.id),
    name: team.name === 'Vibe Moggers' ? 'Команда веб-разработки' : team.name,
  }
}

export function proposalFromApi(proposal: Proposal): Proposal {
  return { ...proposal, teamId: teamId(proposal.teamId) }
}
