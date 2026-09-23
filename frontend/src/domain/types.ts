export type Industry = 'retail' | 'education' | 'services' | 'logistics' | 'agriculture'
export type Readiness = 'draft' | 'working' | 'ready' | 'priority'
export type FieldKey =
  | 'context'
  | 'need'
  | 'dataSource'
  | 'dataFormat'
  | 'dataAccess'
  | 'deliverable'
  | 'deliveryFormat'
  | 'successMetric'
  | 'successTarget'
  | 'acceptanceMethod'
  | 'deadline'
  | 'constraints'
  | 'users'
  | 'usageScenario'
  | 'contact'
  | 'feedbackFormat'
export interface Source {
  id: string
  text: string
}
export interface FieldValue {
  value: string | null
  confirmed: boolean
  source: { id: string; quote: string } | null
}
export type Fields = Record<FieldKey, FieldValue>
export interface ScoreGroup {
  key: string
  label: string
  earned: number
  max: number
  missingFields: FieldKey[]
}
export interface Score {
  total: number
  readiness: Readiness
  breakdown: ScoreGroup[]
}
export interface Snapshot extends Score {
  title: string
  industry: Industry
  fields: Fields
  confirmedAt: string
}
export interface Task {
  id: string
  ownerId: string
  ownerName?: string
  title: string
  industry: Industry
  draft: string
  answers: Source[]
  workingFields: Fields
  confirmedSnapshot: Snapshot | null
  revision: number
  confirmedRevision: number | null
  publishedAt: string | null
  createdAt: string
}
export interface Team {
  id: string
  name: string
  interests: string[]
  skills: string[]
  technologies: string[]
  points?: number
}
export interface LocalProfile {
  id: string
  role: 'business' | 'team'
  name: string
}
export interface ProposalInput {
  idea: string
  plan: string[]
  durationDays: number
  prototypeUrl: string
}
export interface Milestone {
  id: string
  proposalId: string
  resultText: string
  evidenceUrl: string
  status: 'submitted' | 'confirmed'
  confirmedAt: string | null
}
export interface Proposal extends ProposalInput {
  id: string
  taskId: string
  teamId: string
  status: 'pending' | 'accepted' | 'rejected'
  createdAt: string
  milestone?: Milestone | null
}
export interface Question {
  id: string
  fieldKeys: FieldKey[]
  text: string
}
export interface AnalyzeInput {
  stage: 'clarify' | 'assemble'
  sources: Source[]
  currentFields: Fields
}
export interface AnalyzeResult {
  questions: Question[]
  fieldSuggestions: { field: FieldKey; value: string; sourceId: string }[]
  mode: 'live' | 'fallback'
  warnings: string[]
}
export interface CardInput {
  revision: number
  title: string
  industry: Industry
  answers: Source[]
  fields: Fields
}
export interface CatalogFilters {
  industry?: string
  readiness?: string
}
export interface Gateway {
  mode: 'demo' | 'api'
  listTasks(filters?: CatalogFilters): Promise<Task[]>
  listOwnTasks(): Promise<Task[]>
  getTask(id: string): Promise<Task>
  createTask(input: { draft: string; title: string; industry: Industry }): Promise<Task>
  saveCard(id: string, input: CardInput): Promise<Task>
  confirmTask(id: string, revision: number, fieldKeys: FieldKey[]): Promise<Task>
  publishTask(id: string, revision: number): Promise<Task>
  analyze(input: AnalyzeInput): Promise<AnalyzeResult>
  listTeams(): Promise<Team[]>
  listProposals(taskId: string): Promise<Proposal[]>
  createProposal(taskId: string, input: ProposalInput): Promise<Proposal>
  decideProposal(id: string, decision: 'accepted' | 'rejected'): Promise<Proposal>
  submitMilestone(id: string, input: { resultText: string; evidenceUrl: string }): Promise<Proposal>
  confirmMilestone(id: string): Promise<Proposal>
}
