import type { FieldKey, Fields, Industry, Readiness, Score } from './types'

export const industries: { value: Industry; label: string; short: string; color: string }[] = [
  { value: 'retail', label: 'Ритейл и торговля', short: 'Ритейл', color: 'blue' },
  { value: 'education', label: 'Образование', short: 'Образование', color: 'purple' },
  { value: 'services', label: 'Сервисы и услуги', short: 'Сервисы', color: 'orange' },
  { value: 'logistics', label: 'Логистика', short: 'Логистика', color: 'teal' },
  { value: 'agriculture', label: 'Сельское хозяйство', short: 'Агро', color: 'green' },
]
export const industryInfo = (id: string) => industries.find((x) => x.value === id) ?? industries[0]!
export const readinessLabels: Record<Readiness, string> = {
  draft: 'Нужно уточнение',
  working: 'Рабочая',
  ready: 'Готова к работе',
  priority: 'Приоритетная',
}
export const groups = [
  { key: 'context', label: 'Контекст и потребность', fields: ['context', 'need'] },
  { key: 'data', label: 'Данные и материалы', fields: ['dataSource', 'dataFormat', 'dataAccess'] },
  { key: 'result', label: 'Ожидаемый результат', fields: ['deliverable', 'deliveryFormat'] },
  {
    key: 'success',
    label: 'Критерии успеха',
    fields: ['successMetric', 'successTarget', 'acceptanceMethod'],
  },
  { key: 'limits', label: 'Ограничения', fields: ['deadline', 'constraints'] },
  { key: 'users', label: 'Пользователи', fields: ['users', 'usageScenario'] },
  { key: 'contact', label: 'Связь с бизнесом', fields: ['contact', 'feedbackFormat'] },
] as const
export const fieldMeta: Record<FieldKey, { label: string; weight: number; placeholder: string }> = {
  context: {
    label: 'Что происходит сейчас',
    weight: 10,
    placeholder: 'Как устроен процесс и в чём проблема?',
  },
  need: {
    label: 'Что нужно изменить',
    weight: 10,
    placeholder: 'Какое изменение поможет вашему бизнесу?',
  },
  dataSource: {
    label: 'Доступные данные',
    weight: 10,
    placeholder: 'Какие данные, примеры или материалы у вас есть?',
  },
  dataFormat: {
    label: 'Формат данных',
    weight: 5,
    placeholder: 'Например: CSV с продажами за 6 месяцев',
  },
  dataAccess: {
    label: 'Доступ к материалам',
    weight: 5,
    placeholder: 'Как и когда команда получит данные?',
  },
  deliverable: {
    label: 'Результат работы',
    weight: 10,
    placeholder: 'Что конкретно должна создать команда?',
  },
  deliveryFormat: {
    label: 'Формат результата',
    weight: 5,
    placeholder: 'Например: веб-прототип с импортом CSV',
  },
  successMetric: {
    label: 'Что измеряем',
    weight: 5,
    placeholder: 'Например: время подготовки списка закупок',
  },
  successTarget: {
    label: 'Критерий успешности',
    weight: 5,
    placeholder: 'Например: не более 10 минут на один список',
  },
  acceptanceMethod: {
    label: 'Как проверим результат',
    weight: 5,
    placeholder: 'Кто, на каких данных и как проведёт проверку?',
  },
  deadline: { label: 'Срок', weight: 5, placeholder: 'Например: 14 дней после выбора команды' },
  constraints: {
    label: 'Ограничения',
    weight: 5,
    placeholder: 'Технологии, доступы, границы первого прототипа',
  },
  users: {
    label: 'Для кого решение',
    weight: 5,
    placeholder: 'Кто будет пользоваться результатом?',
  },
  usageScenario: {
    label: 'Сценарий использования',
    weight: 5,
    placeholder: 'Когда и для чего человек откроет решение?',
  },
  contact: {
    label: 'Контакт для связи',
    weight: 5,
    placeholder: 'Email, телефон или ссылка для связи',
  },
  feedbackFormat: {
    label: 'Порядок обратной связи',
    weight: 5,
    placeholder: 'Например: созвон с управляющим два раза в неделю',
  },
}
export const fieldKeys = Object.keys(fieldMeta) as FieldKey[]
export const emptyFields = (): Fields =>
  Object.fromEntries(
    fieldKeys.map((key) => [key, { value: null, confirmed: false, source: null }]),
  ) as Fields
export const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T
export const getReadiness = (score: number): Readiness =>
  score < 40 ? 'draft' : score < 70 ? 'working' : score < 90 ? 'ready' : 'priority'
export function fieldValid(key: FieldKey, value: string | null): boolean {
  const text = value?.trim() ?? ''
  if (!text || /^(?:не знаю|позже|неизвестно|нет данных|n\/?a|[-—.]+)$/i.test(text)) return false
  if (key === 'contact')
    return (
      /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(text) ||
      /^https?:\/\/\S+$/i.test(text) ||
      /^[+\d][\d\s()-]{6,}$/.test(text)
    )
  if (key === 'deadline')
    return /\d/.test(text) && /(?:дн|день|недел|месяц|час|минут|\d[./-]\d)/i.test(text)
  if (key === 'successTarget')
    return (
      (/\d/.test(text) && /[а-яa-z%]/i.test(text)) ||
      /(?:все|кажд[а-я]+).*(?:прош|принят|соответств|выполн|отображ)/i.test(text)
    )
  return text.length >= 3
}
/** Browser estimate / local demo only. The API snapshot is authoritative in API mode. */
export function scoreFields(fields: Fields, preview = false): Score {
  const eligible = (key: FieldKey) =>
    fieldValid(key, fields[key].value) && (preview || fields[key].confirmed)
  const earns = (key: FieldKey) =>
    eligible(key) &&
    (!['successTarget', 'acceptanceMethod'].includes(key) || eligible('successMetric'))
  const breakdown = groups.map((group) => ({
    key: group.key,
    label: group.label,
    max: group.fields.reduce((total, key) => total + fieldMeta[key].weight, 0),
    earned: group.fields.reduce(
      (total, key) => total + (earns(key) ? fieldMeta[key].weight : 0),
      0,
    ),
    missingFields: group.fields.filter((key) => !earns(key)) as FieldKey[],
  }))
  const total = breakdown.reduce((sum, group) => sum + group.earned, 0)
  return { total, breakdown, readiness: getReadiness(total) }
}
export function safeUrl(value: string): string | null {
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol) ? url.href : null
  } catch {
    return null
  }
}
