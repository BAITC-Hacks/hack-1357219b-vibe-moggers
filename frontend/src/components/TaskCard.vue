<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../domain/types'
import { industryInfo } from '../domain/scoring'
import AppIcon from './AppIcon.vue'
import ScoreBadge from './ScoreBadge.vue'
const props = defineProps<{ task: Task; index?: number }>()
const info = computed(() => industryInfo(props.task.industry))
const score = computed(() => props.task.confirmedSnapshot?.total ?? 0)
const fields = computed(() => props.task.confirmedSnapshot?.fields ?? props.task.workingFields)
const icons = {
  retail: 'Store',
  education: 'GraduationCap',
  services: 'Wrench',
  logistics: 'Truck',
  agriculture: 'Wheat',
} as const
</script>
<template>
  <article class="task-card" :style="{ '--card-order': index ?? 0 }">
    <div class="task-card-top">
      <span class="industry-icon" :class="info.color"
        ><AppIcon :name="icons[task.industry] || 'Building2'" :size="22" /></span
      ><ScoreBadge :score="score" />
    </div>
    <span class="card-category">{{ info.label }}</span>
    <h3>
      <RouterLink :to="`/tasks/${task.id}`">{{
        task.confirmedSnapshot?.title || task.title
      }}</RouterLink>
    </h3>
    <p class="card-description">
      {{
        fields.need.value ||
        fields.context.value ||
        'Бизнес уточняет детали задачи. Вы можете задать свой подход в отклике.'
      }}
    </p>
    <div class="card-score">
      <span>Готовность задачи</span><strong>{{ score }}<span>/100</span></strong>
    </div>
    <div class="segmented-progress" aria-hidden="true">
      <span
        v-for="n in 10"
        :key="n"
        :class="{ filled: score >= n * 10, partial: score > (n - 1) * 10 && score < n * 10 }"
      />
    </div>
    <div class="task-facts">
      <span
        ><AppIcon :name="fields.dataSource.value ? 'FileText' : 'CircleHelp'" :size="14" />{{
          fields.dataSource.value ? 'Данные описаны' : 'Данные уточняются'
        }}</span
      ><span
        ><AppIcon name="Clock3" :size="14" />{{
          fields.deadline.value
            ?.match(/\d+\s*(?:календарных\s*)?(?:дней|дня|день|недел\w*)/)?.[0]
            ?.replace('календарных ', '') || 'Срок уточняется'
        }}</span
      >
    </div>
    <div class="task-card-footer">
      <span class="business-label"
        ><span class="mini-avatar">{{ info.short.slice(0, 1) }}</span
        >Бизнес · AI Sana</span
      ><RouterLink
        :to="`/tasks/${task.id}`"
        class="card-open"
        :aria-label="`Открыть задачу: ${task.title}`"
        ><AppIcon name="ArrowUpRight" :size="20"
      /></RouterLink>
    </div>
  </article>
</template>
