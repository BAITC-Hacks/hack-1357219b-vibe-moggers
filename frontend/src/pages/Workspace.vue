<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useWorkspace } from '../stores/workspace'
import type { Proposal } from '../domain/types'
import { industryInfo } from '../domain/scoring'
import AppIcon from '../components/AppIcon.vue'
import ScoreBadge from '../components/ScoreBadge.vue'
const store = useWorkspace()
const applications = ref<Proposal[]>([])
const applicationError = ref('')
const loadingApplications = ref(false)
const filter = ref('all')
const filtered = computed(() =>
  store.mine.filter(
    (task) =>
      filter.value === 'all' ||
      (filter.value === 'published' ? task.publishedAt : !task.publishedAt),
  ),
)
let request = 0
async function loadApplications() {
  const token = ++request
  applications.value = []
  applicationError.value = ''
  if (store.isBusiness) return
  loadingApplications.value = true
  try {
    const rows = await Promise.all(store.catalog.map((task) => store.api.listProposals(task.id)))
    if (token === request)
      applications.value = rows.flat().filter((proposal) => proposal.teamId === store.actorId)
  } catch (err) {
    if (token === request)
      applicationError.value = err instanceof Error ? err.message : 'Не удалось загрузить отклики'
  } finally {
    if (token === request) loadingApplications.value = false
  }
}
watch(() => [store.actorId, store.catalog], loadApplications, { immediate: true })
const statuses = { pending: 'На рассмотрении', accepted: 'Команда выбрана', rejected: 'Отклонено' }
</script>
<template>
  <div class="page-title-row">
    <div>
      <span class="eyebrow">ВАШЕ РАБОЧЕЕ ПРОСТРАНСТВО</span>
      <h1>
        {{ store.isBusiness ? 'Идеи, которые движутся вперёд.' : 'Ваши идеи уже на шаг ближе.' }}
      </h1>
      <p>
        {{
          store.isBusiness
            ? 'Управляйте задачами, улучшайте готовность и выбирайте команды.'
            : 'Следите за откликами и передавайте результаты выбранных проектов.'
        }}
      </p>
    </div>
    <RouterLink v-if="store.isBusiness" to="/tasks/new" class="button primary"
      ><AppIcon name="Plus" :size="17" /> Новая задача</RouterLink
    ><span v-else class="team-points"
      ><AppIcon name="Zap" :size="21" /><strong>{{ store.currentTeam?.points ?? 0 }}</strong> баллов
      за результат</span
    >
  </div>
  <div v-if="store.error || applicationError" class="inline-alert error" role="alert">
    <AppIcon name="CircleAlert" :size="18" />{{ store.error || applicationError
    }}<button class="text-link" @click="store.refresh">Повторить</button>
  </div>
  <template v-if="store.isBusiness"
    ><div class="catalog-tabs" role="group" aria-label="Публикация задач">
      <button
        v-for="item in [
          { value: 'all', label: 'Все задачи' },
          { value: 'published', label: 'Опубликованные' },
          { value: 'draft', label: 'Черновики' },
        ]"
        :key="item.value"
        :class="{ active: filter === item.value }"
        :aria-pressed="filter === item.value"
        @click="filter = item.value"
      >
        {{ item.label }}
      </button>
    </div>
    <div v-if="filtered.length" class="workspace-list">
      <article v-for="task in filtered" :key="task.id" class="workspace-row panel">
        <span class="soft-icon" :class="industryInfo(task.industry).color"
          ><AppIcon :name="task.publishedAt ? 'ClipboardList' : 'Pencil'" :size="21"
        /></span>
        <div class="workspace-task-info">
          <span class="overline"
            >{{ industryInfo(task.industry).short }} ·
            {{ task.publishedAt ? 'ОПУБЛИКОВАНА' : 'ЧЕРНОВИК' }}</span
          >
          <h3>
            <RouterLink :to="`/tasks/${task.id}${task.publishedAt ? '' : '/edit'}`">{{
              task.title
            }}</RouterLink>
          </h3>
        </div>
        <ScoreBadge :score="task.confirmedSnapshot?.total ?? 0" compact />
        <div class="workspace-row-actions">
          <RouterLink
            :to="`/tasks/${task.id}/edit`"
            class="icon-button"
            :aria-label="`Редактировать: ${task.title}`"
            ><AppIcon name="Pencil" :size="17" /></RouterLink
          ><RouterLink :to="`/business/tasks/${task.id}`" class="button secondary"
            >Отклики <AppIcon name="ArrowRight" :size="15"
          /></RouterLink>
        </div>
      </article>
    </div>
    <div v-else-if="!store.loading" class="empty-state">
      <span class="empty-icon"><AppIcon name="FolderOpen" :size="29" /></span>
      <h3>Начните с одной идеи</h3>
      <p>Создайте задачу и расскажите, что хотите изменить в своём бизнесе.</p>
      <RouterLink to="/tasks/new" class="button primary"
        >Создать задачу <AppIcon name="Plus" :size="17"
      /></RouterLink></div
  ></template>
  <template v-else
    ><div class="inline-alert info">
      <AppIcon name="Flag" :size="18" /><span
        >Баллы начисляются за результат этапа, который подтвердил бизнес. Отправка отклика сама по
        себе баллов не добавляет.</span
      >
    </div>
    <div v-if="loadingApplications" class="empty-state">
      <AppIcon name="LoaderCircle" class="spin" :size="28" />
    </div>
    <div v-else-if="applications.length" class="workspace-list">
      <article v-for="proposal in applications" :key="proposal.id" class="workspace-row panel">
        <span class="soft-icon" :class="proposal.status === 'accepted' ? 'green' : 'blue'"
          ><AppIcon :name="proposal.status === 'accepted' ? 'CheckCheck' : 'Send'"
        /></span>
        <div class="workspace-task-info">
          <span class="proposal-status" :class="proposal.status">{{
            statuses[proposal.status]
          }}</span>
          <h3>
            {{
              store.catalog.find((task) => task.id === proposal.taskId)?.title || 'Бизнес-задача'
            }}
          </h3>
          <p class="muted small line-clamp">{{ proposal.idea }}</p>
        </div>
        <RouterLink :to="`/tasks/${proposal.taskId}`" class="button secondary"
          >{{ proposal.status === 'accepted' ? 'К работе' : 'Открыть'
          }}<AppIcon name="ArrowRight" :size="15"
        /></RouterLink>
      </article>
    </div>
    <div v-else class="empty-state">
      <span class="empty-icon"><AppIcon name="Send" :size="28" /></span>
      <h3>Какую задачу решит ваша команда?</h3>
      <p>Выберите интересную задачу и предложите свой подход.</p>
      <RouterLink to="/catalog" class="button primary"
        >Найти задачу <AppIcon name="ArrowRight" :size="17"
      /></RouterLink></div
  ></template>
</template>
