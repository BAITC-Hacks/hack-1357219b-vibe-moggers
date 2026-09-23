<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useWorkspace } from '../stores/workspace'
import { useTask } from '../composables/useTask'
import { groups, fieldMeta, industryInfo, scoreFields, safeUrl } from '../domain/scoring'
import AppIcon from '../components/AppIcon.vue'
import ScorePanel from '../components/ScorePanel.vue'
import ScoreBadge from '../components/ScoreBadge.vue'
import ProposalForm from '../components/ProposalForm.vue'
import ProposalCard from '../components/ProposalCard.vue'
const store = useWorkspace()
const { task, proposals, loading, busy, error, load, action } = useTask()
const fields = computed(() => task.value?.confirmedSnapshot?.fields ?? task.value?.workingFields)
const score = computed(
  () => task.value?.confirmedSnapshot ?? (fields.value ? scoreFields(fields.value) : null),
)
const isOwner = computed(() => task.value?.ownerId === store.actorId)
const accepted = computed(() =>
  proposals.value.filter(
    (proposal) => proposal.teamId === store.actorId && proposal.status === 'accepted',
  ),
)
const milestones = reactive<Record<string, { resultText: string; evidenceUrl: string }>>({})
function milestone(id: string) {
  return (milestones[id] ||= { resultText: '', evidenceUrl: '' })
}
function jump(key: string) {
  document.getElementById(`detail-${key}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
function submit(id: string) {
  const data = milestone(id)
  if (!safeUrl(data.evidenceUrl)) {
    store.notify('Укажите http(s) ссылку на результат', 'error')
    return
  }
  return action(
    () => store.api.submitMilestone(id, data),
    'Результат этапа отправлен бизнесу на подтверждение.',
  )
}
function example(id: string) {
  milestones[id] = {
    resultText:
      'Подготовлены структура CSV и макет таблицы закупок. Синтетический пример передан для проверки первого этапа.',
    evidenceUrl: `${location.origin}/demo/prototype.html`,
  }
}
</script>
<template>
  <RouterLink to="/catalog" class="back-link"
    ><AppIcon name="ArrowLeft" :size="15" /> Все задачи</RouterLink
  >
  <div v-if="loading" class="empty-state">
    <AppIcon name="LoaderCircle" class="spin" :size="28" />
    <p>Открываем карточку…</p>
  </div>
  <div v-else-if="error || !task" class="empty-state">
    <AppIcon name="CircleAlert" :size="30" />
    <h2>Не удалось открыть задачу</h2>
    <p>{{ error }}</p>
    <button class="button secondary" @click="load">Повторить</button>
  </div>
  <template v-else-if="fields && score">
    <div class="detail-heading">
      <div class="detail-labels">
        <span class="topic-tag" :class="industryInfo(task.industry).color">{{
          industryInfo(task.industry).label
        }}</span
        ><ScoreBadge :score="score.total" /><span v-if="!task.publishedAt" class="private-label"
          >Не опубликована</span
        >
      </div>
      <h1>{{ task.confirmedSnapshot?.title || task.title }}</h1>
      <p>{{ fields.need.value || 'Бизнес ещё уточняет ожидаемые изменения.' }}</p>
      <div class="detail-meta">
        <span><AppIcon name="Building2" :size="15" /> Бизнес AI Sana</span
        ><span
          ><AppIcon name="Globe" :size="15" />{{
            task.publishedAt ? 'Открыта для всех команд' : 'Частный черновик'
          }}</span
        ><span v-if="fields.deadline.value"
          ><AppIcon name="Clock3" :size="15" />{{ fields.deadline.value }}</span
        >
      </div>
    </div>
    <div class="editor-layout detail-layout">
      <div class="editor-main">
        <div v-if="isOwner" class="owner-toolbar panel">
          <div><AppIcon name="ShieldCheck" :size="20" /><span>Это ваша задача</span></div>
          <div>
            <RouterLink :to="`/tasks/${task.id}/edit`" class="button secondary"
              ><AppIcon name="Pencil" :size="15" /> Редактировать</RouterLink
            ><RouterLink :to="`/business/tasks/${task.id}`" class="button primary"
              >Отклики <span class="button-count">{{ proposals.length }}</span
              ><AppIcon name="ArrowRight" :size="15"
            /></RouterLink>
          </div>
        </div>
        <div class="panel task-description">
          <section
            v-for="group in groups"
            :id="`detail-${group.key}`"
            :key="group.key"
            class="detail-group"
          >
            <h2>{{ group.label }}</h2>
            <dl>
              <div v-for="key in group.fields" :key="key">
                <dt>{{ fieldMeta[key].label }}</dt>
                <dd :class="{ unspecified: !fields[key].value }">
                  {{ fields[key].value || 'Пока не указано — можно уточнить у бизнеса' }}
                </dd>
              </div>
            </dl>
          </section>
        </div>
        <template v-if="!store.isBusiness && task.publishedAt">
          <div v-if="proposals.length" class="section-heading compact">
            <div>
              <h2>Ваши предложения</h2>
              <p>Решение о сотрудничестве принимает бизнес.</p>
            </div>
          </div>
          <ProposalCard v-for="proposal in proposals" :key="proposal.id" :proposal="proposal" />
          <form
            v-for="proposal in accepted.filter((item) => !item.milestone)"
            :key="`milestone-${proposal.id}`"
            class="panel milestone-form"
            @submit.prevent="submit(proposal.id)"
          >
            <div class="inline-title">
              <span class="soft-icon green"><AppIcon name="Flag" /></span>
              <div>
                <h2>Покажите первый результат</h2>
                <p class="muted">Баллы команды начислятся после подтверждения бизнеса.</p>
              </div>
            </div>
            <label class="field-label"
              >Что сделано<textarea
                v-model="milestone(proposal.id).resultText"
                required
                minlength="10"
                maxlength="3000"
                rows="3"
                :disabled="busy"
                placeholder="Какой этап завершён и что можно проверить?"
              /></label
            ><label class="field-label"
              >Ссылка на результат<input
                v-model="milestone(proposal.id).evidenceUrl"
                type="url"
                required
                :disabled="busy"
                placeholder="https://…"
            /></label>
            <div class="editor-actions">
              <button class="button primary" :disabled="busy">
                <AppIcon name="Send" :size="16" /> Передать результат</button
              ><button
                v-if="store.mode === 'demo'"
                class="text-link"
                type="button"
                @click="example(proposal.id)"
              >
                Заполнить пример
              </button>
            </div>
          </form>
          <div id="proposal"><ProposalForm :task-id="task.id" @submitted="load" /></div>
        </template>
      </div>
      <div class="detail-aside">
        <ScorePanel :score="score" @jump="jump" /><a
          v-if="!store.isBusiness && task.publishedAt"
          href="#proposal"
          class="button primary full-width"
          ><AppIcon name="Send" :size="16" /> Предложить решение</a
        >
        <div class="selection-note">
          <AppIcon name="Users" :size="19" />
          <p>Команды выбирают задачи.<br />Бизнес выбирает сотрудничество.</p>
        </div>
      </div>
    </div>
  </template>
</template>
