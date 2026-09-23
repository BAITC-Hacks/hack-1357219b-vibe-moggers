import seed from '../../../docs/fixtures/seed.json'
import { clone, scoreFields } from '../domain/scoring'
import type { Fields, Industry, Proposal, Task, Team } from '../domain/types'

export function initialData(): { tasks: Task[]; teams: Team[]; proposals: Proposal[] } {
  return {
    tasks: seed.cards.map((card) => {
      const fields = clone(card.fields) as Fields
      return {
        id: card.id,
        ownerId: card.ownerId,
        title: card.title,
        industry: card.industry as Industry,
        draft: fields.context.value ?? '',
        answers: [],
        workingFields: fields,
        confirmedSnapshot: {
          title: card.title,
          industry: card.industry as Industry,
          fields: clone(fields),
          ...scoreFields(fields),
          confirmedAt: card.publishedAt,
        },
        revision: 1,
        confirmedRevision: 1,
        publishedAt: card.publishedAt,
        createdAt: card.publishedAt,
      }
    }),
    teams: clone(seed.teams).map((team) => ({ ...team, points: 0 })),
    proposals: clone(seed.proposals) as Proposal[],
  }
}

export const demoDraft = 'У нас небольшой магазин. Хотим ИИ, чтобы лучше управлять остатками.'
export const demoTitle = 'Помощник закупок для магазина Arman'
export const demo40 = {
  context:
    'Управляющий вручную проверяет остатки в Excel, и отдельные товары заканчиваются до следующей поставки.',
  need: 'Помогать управляющему составлять список товаров для следующей закупки.',
  users: 'Управляющий магазина.',
  usageScenario: 'Каждое утро проверяет остатки и решает, что заказать у поставщика.',
  dataSource: 'Выгрузка продаж и остатков по 50 товарам за последние 6 месяцев.',
}
export const demo95 = {
  ...demo40,
  dataFormat: 'CSV: date, sku, units_sold, stock_units.',
  dataAccess: 'Обезличенную выгрузку передадим выбранной команде ссылкой после согласования.',
  deliverable: 'Веб-прототип со списком рекомендуемых к закупке товаров.',
  deliveryFormat: 'Веб-страница с импортом CSV и таблицей рекомендаций.',
  successMetric: 'Время составления списка закупки для 50 товаров.',
  successTarget: 'Не более 10 минут на один список закупки для 50 товаров.',
  acceptanceMethod: 'Управляющий выполняет 3 контрольных сценария, мы фиксируем время каждого.',
  deadline: '14 дней после выбора команды.',
  constraints: 'Только обезличенные данные, без интеграции с кассой в первом прототипе.',
  contact: 'arman-demo@example.org',
}
