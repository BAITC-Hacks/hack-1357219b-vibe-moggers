<script setup lang="ts">
import { computed } from 'vue'
import type { Proposal } from '../domain/types'
import { useWorkspace } from '../stores/workspace'
import { safeUrl } from '../domain/scoring'
import AppIcon from './AppIcon.vue'
const props = defineProps<{ proposal: Proposal; business?: boolean; busy?: boolean }>()
defineEmits<{ decide: [id: string, decision: 'accepted' | 'rejected']; confirm: [id: string] }>()
const store = useWorkspace()
const team = computed(() => store.teams.find((item) => item.id === props.proposal.teamId))
const labels = { pending: 'Ожидает решения', accepted: 'Команда выбрана', rejected: 'Отклонено' }
</script>
<template>
  <article class="proposal-card panel" :class="proposal.status">
    <div class="proposal-card-head">
      <span class="team-avatar">{{
        (team?.name || 'К')
          .split(' ')
          .map((word) => word[0])
          .slice(0, 2)
          .join('')
      }}</span>
      <div>
        <h3>{{ team?.name || 'Студенческая команда' }}</h3>
        <span class="muted small">{{ team?.technologies.join(' · ') }}</span>
      </div>
      <span class="proposal-status" :class="proposal.status"
        ><AppIcon v-if="proposal.status === 'accepted'" name="Check" :size="13" />{{
          labels[proposal.status]
        }}</span
      >
    </div>
    <div class="proposal-section">
      <span class="overline">ИДЕЯ РЕШЕНИЯ</span>
      <p>{{ proposal.idea }}</p>
    </div>
    <div class="proposal-section">
      <span class="overline">ПЛАН РАБОТЫ</span>
      <ol class="proposal-plan">
        <li v-for="(step, index) in proposal.plan" :key="index">
          <span>{{ index + 1 }}</span
          >{{ step }}
        </li>
      </ol>
    </div>
    <div class="proposal-meta">
      <span><AppIcon name="Clock3" :size="15" />{{ proposal.durationDays }} дней</span
      ><a
        v-if="safeUrl(proposal.prototypeUrl)"
        :href="safeUrl(proposal.prototypeUrl)!"
        target="_blank"
        rel="noopener noreferrer"
        class="text-link"
        >Посмотреть прототип <AppIcon name="ExternalLink" :size="13" /></a
      ><span v-else class="muted small">Ссылка недоступна</span>
    </div>
    <div v-if="business && proposal.status === 'pending'" class="proposal-decisions">
      <button
        class="button primary"
        :disabled="busy"
        @click="$emit('decide', proposal.id, 'accepted')"
      >
        <AppIcon name="Check" :size="16" /> Выбрать команду</button
      ><button
        class="button secondary"
        :disabled="busy"
        @click="$emit('decide', proposal.id, 'rejected')"
      >
        Отклонить
      </button>
    </div>
    <div v-if="proposal.milestone" class="milestone-summary">
      <div class="inline-title">
        <AppIcon
          :name="proposal.milestone.status === 'confirmed' ? 'CheckCheck' : 'Flag'"
          :size="19"
        /><b>{{
          proposal.milestone.status === 'confirmed'
            ? 'Этап подтверждён · +10 баллов'
            : 'Команда передала результат этапа'
        }}</b>
      </div>
      <p>{{ proposal.milestone.resultText }}</p>
      <a
        v-if="safeUrl(proposal.milestone.evidenceUrl)"
        class="text-link"
        :href="safeUrl(proposal.milestone.evidenceUrl)!"
        target="_blank"
        rel="noopener noreferrer"
        >Открыть результат <AppIcon name="ExternalLink" :size="13" /></a
      ><button
        v-if="business && proposal.milestone.status === 'submitted'"
        class="button primary full-width"
        :disabled="busy"
        @click="$emit('confirm', proposal.milestone!.id)"
      >
        <AppIcon name="ShieldCheck" :size="16" /> Подтвердить результат этапа
      </button>
    </div>
  </article>
</template>
