<script setup lang="ts">
import { computed } from 'vue'
import type { Score } from '../domain/types'
import { fieldMeta, readinessLabels } from '../domain/scoring'
import AppIcon from './AppIcon.vue'
const props = defineProps<{ score: Score; confirmed?: number; preview?: boolean }>()
defineEmits<{ jump: [key: string] }>()
const next = computed(() => props.score.breakdown.find((group) => group.missingFields.length))
</script>
<template>
  <aside class="score-panel panel">
    <div class="panel-eyebrow"><AppIcon name="TrendingUp" :size="16" /> Готовность задачи</div>
    <div class="score-orbit" :class="{ 'is-preview': preview }">
      <svg viewBox="0 0 160 160" aria-hidden="true">
        <circle cx="80" cy="80" r="66" class="orbit-track" />
        <circle
          cx="80"
          cy="80"
          r="66"
          class="orbit-value"
          :style="{ strokeDashoffset: 414.7 * (1 - score.total / 100) }"
        />
      </svg>
      <div>
        <strong>{{ score.total }}<span>/100</span></strong
        ><small>{{ preview ? 'после подтверждения' : 'подтверждено' }}</small>
      </div>
    </div>
    <div class="score-level">{{ readinessLabels[score.readiness] }}</div>
    <p v-if="preview && confirmed !== undefined" class="confirmed-caption">
      Подтверждённый рейтинг: <b>{{ confirmed }}</b
      ><span
        v-if="score.total !== confirmed"
        :class="score.total > confirmed ? 'positive' : 'negative'"
        >{{ score.total > confirmed ? '+' : '' }}{{ score.total - confirmed }}</span
      >
    </p>
    <div class="score-breakdown">
      <button
        v-for="group in score.breakdown"
        :key="group.key"
        type="button"
        class="score-row"
        @click="$emit('jump', group.key)"
      >
        <span class="score-row-label"
          >{{ group.label
          }}<strong
            >{{ group.earned }}<span>/{{ group.max }}</span></strong
          ></span
        >
        <span class="mini-progress"
          ><span :style="{ width: `${(group.earned / group.max) * 100}%` }"
        /></span>
      </button>
    </div>
    <div v-if="next" class="next-step">
      <span class="soft-icon blue"><AppIcon name="Lightbulb" :size="17" /></span>
      <div>
        <b>Следующий шаг</b>
        <p>{{ fieldMeta[next.missingFields[0]!].label }}</p>
        <button type="button" class="text-link" @click="$emit('jump', next.key)">
          Дополнить сведения <AppIcon name="ArrowRight" :size="13" />
        </button>
      </div>
    </div>
    <div v-else class="all-complete">
      <AppIcon name="CheckCheck" :size="18" /> Все сведения заполнены
    </div>
    <p class="score-footnote">
      <AppIcon name="ShieldCheck" :size="15" /> Баллы за сведения, подтверждённые бизнесом.
    </p>
  </aside>
</template>
