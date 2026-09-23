<script setup lang="ts">
import { computed, ref } from 'vue'
import { useWorkspace } from '../stores/workspace'
import { useTask } from '../composables/useTask'
import AppIcon from '../components/AppIcon.vue'
import ProposalCard from '../components/ProposalCard.vue'
import ScoreBadge from '../components/ScoreBadge.vue'
const store = useWorkspace()
const { task, proposals, loading, busy, error, load, action } = useTask()
const filter = ref('all')
const filtered = computed(() =>
  proposals.value.filter((proposal) => filter.value === 'all' || proposal.status === filter.value),
)
const selected = computed(
  () => proposals.value.filter((proposal) => proposal.status === 'accepted').length,
)
const waiting = computed(
  () => proposals.value.filter((proposal) => proposal.status === 'pending').length,
)
function decide(id: string, decision: 'accepted' | 'rejected') {
  return action(
    () => store.api.decideProposal(id, decision),
    decision === 'accepted'
      ? 'Команда выбрана. Можно принять и другие предложения.'
      : 'Предложение отклонено.',
  )
}
function confirm(id: string) {
  return action(
    () => store.api.confirmMilestone(id),
    'Этап подтверждён. Команда получила 10 баллов за результат.',
  )
}
</script>
<template>
  <RouterLink to="/business" class="back-link"
    ><AppIcon name="ArrowLeft" :size="15" /> Мои задачи</RouterLink
  >
  <div v-if="loading" class="empty-state">
    <AppIcon name="LoaderCircle" class="spin" :size="28" />
    <p>Загружаем предложения…</p>
  </div>
  <div v-else-if="error || !task" class="empty-state">
    <h2>Не удалось загрузить предложения</h2>
    <p>{{ error }}</p>
    <button class="button secondary" @click="load">Повторить</button>
  </div>
  <div v-else-if="task.ownerId !== store.actorId" class="empty-state">
    <AppIcon name="ShieldCheck" :size="30" />
    <h2>Решение принимает владелец задачи</h2>
    <p>Управлять откликами может только заказчик, разместивший задачу.</p>
  </div>
  <template v-else>
    <div class="page-title-row">
      <div>
        <span class="eyebrow">ОТКЛИКИ И СОТРУДНИЧЕСТВО</span>
        <h1>Ваш следующий шаг —<br /><span class="text-blue">выбрать команду.</span></h1>
        <p>Сравните подходы и начните работу с теми, кто понимает вашу задачу.</p>
      </div>
      <RouterLink :to="`/tasks/${task.id}`" class="button secondary"
        >Карточка задачи <AppIcon name="ArrowUpRight" :size="16"
      /></RouterLink>
    </div>
    <div class="business-task-summary panel">
      <span class="soft-icon blue"><AppIcon name="ClipboardList" /></span>
      <div>
        <h3>{{ task.title }}</h3>
        <span class="muted small">{{
          task.publishedAt ? 'Опубликована в открытом каталоге' : 'Частный черновик'
        }}</span>
      </div>
      <ScoreBadge :score="task.confirmedSnapshot?.total ?? 0" />
    </div>
    <div class="business-stats">
      <div class="panel">
        <span class="soft-icon blue"><AppIcon name="MessageSquare" /></span
        ><strong>{{ proposals.length }}</strong
        ><span>предложений</span>
      </div>
      <div class="panel">
        <span class="soft-icon orange"><AppIcon name="Clock3" /></span><strong>{{ waiting }}</strong
        ><span>ожидают решения</span>
      </div>
      <div class="panel">
        <span class="soft-icon green"><AppIcon name="Users" /></span><strong>{{ selected }}</strong
        ><span>команд выбрано</span>
      </div>
    </div>
    <div class="inline-alert info selection-alert">
      <AppIcon name="ShieldCheck" :size="19" /><span
        ><b>Решение за вами.</b> Можно выбрать одну, несколько или ни одной команды. AI не назначает
        исполнителей.</span
      >
    </div>
    <div class="catalog-tabs" role="group" aria-label="Статус предложения">
      <button
        v-for="item in [
          { value: 'all', label: 'Все предложения' },
          { value: 'pending', label: 'На рассмотрении' },
          { value: 'accepted', label: 'Выбраны' },
          { value: 'rejected', label: 'Отклонены' },
        ]"
        :key="item.value"
        :class="{ active: filter === item.value }"
        :aria-pressed="filter === item.value"
        @click="filter = item.value"
      >
        {{ item.label }}
      </button>
    </div>
    <div v-if="filtered.length" class="proposal-grid">
      <ProposalCard
        v-for="proposal in filtered"
        :key="proposal.id"
        :proposal="proposal"
        business
        :busy="busy"
        @decide="decide"
        @confirm="confirm"
      />
    </div>
    <div v-else class="empty-state">
      <span class="empty-icon"><AppIcon name="MessageSquare" :size="29" /></span>
      <h3>
        {{ proposals.length ? 'В этой группе пока нет предложений' : 'Здесь появятся идеи команд' }}
      </h3>
      <p>
        {{
          proposals.length
            ? 'Переключите фильтр, чтобы увидеть остальные отклики.'
            : 'Студенты смогут откликнуться на вашу опубликованную задачу.'
        }}
      </p>
      <RouterLink v-if="!task.publishedAt" :to="`/tasks/${task.id}/edit`" class="button primary"
        >Подготовить к публикации</RouterLink
      >
    </div>
  </template>
</template>
