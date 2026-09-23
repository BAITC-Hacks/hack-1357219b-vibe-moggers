<script setup lang="ts">
import { getReadiness, readinessLabels } from '../domain/scoring'
import AppIcon from './AppIcon.vue'
defineProps<{ score: number; compact?: boolean }>()
</script>
<template>
  <span class="score-badge" :class="getReadiness(score)">
    <AppIcon
      :name="
        score >= 90 ? 'Zap' : score >= 70 ? 'CheckCircle2' : score >= 40 ? 'Clock3' : 'CircleHelp'
      "
      :size="13"
    />
    <span v-if="!compact">{{ readinessLabels[getReadiness(score)] }}</span
    ><strong v-else>{{ score }}/100</strong>
  </span>
</template>
