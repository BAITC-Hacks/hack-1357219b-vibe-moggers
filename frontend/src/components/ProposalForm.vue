<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useWorkspace } from '../stores/workspace'
import { safeUrl } from '../domain/scoring'
import type { Proposal } from '../domain/types'
import AppIcon from './AppIcon.vue'
import { validateForm } from '../composables/formControls'
const props = defineProps<{ taskId: string }>()
const emit = defineEmits<{ submitted: [proposal: Proposal] }>()
const store = useWorkspace()
const busy = ref(false)
const error = ref('')
const form = reactive({ idea: '', plan: '', durationDays: 14, prototypeUrl: '' })
function fill() {
  form.idea =
    'Предлагаем веб-прототип с понятными правилами обработки данных и объяснением каждого результата.'
  form.plan =
    'Уточним структуру данных и критерии приёмки\nСоберём первый работающий прототип\nПроверим его с представителем бизнеса'
  form.durationDays = 14
  form.prototypeUrl = `${location.origin}/demo/prototype.html`
}
async function submit(event: Event) {
  if (busy.value) return
  error.value = ''
  error.value = validateForm(event, 'proposal-error')
  if (error.value) return
  if (!safeUrl(form.prototypeUrl)) {
    error.value = 'Добавьте ссылку, начинающуюся с http:// или https://'
    return
  }
  busy.value = true
  try {
    const proposal = await store.api.createProposal(props.taskId, {
      idea: form.idea.trim(),
      plan: form.plan
        .split('\n')
        .map((step) => step.trim())
        .filter(Boolean),
      durationDays: Number(form.durationDays),
      prototypeUrl: form.prototypeUrl.trim(),
    })
    form.idea = ''
    form.plan = ''
    form.prototypeUrl = ''
    emit('submitted', proposal)
    store.notify('Предложение отправлено. Решение примет представитель бизнеса.')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Не удалось отправить предложение'
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <form class="proposal-form panel" novalidate @submit.prevent="submit">
    <div class="inline-title">
      <span class="soft-icon blue"><AppIcon name="Send" /></span>
      <div>
        <h2>Предложите своё решение</h2>
        <p class="muted">
          От команды <b>{{ store.currentTeam?.name }}</b>
        </p>
      </div>
    </div>
    <div v-if="error" id="proposal-error" class="inline-alert error" role="alert">
      <AppIcon name="CircleAlert" :size="17" />{{ error }}
    </div>
    <label class="field-label"
      >Ваша идея<textarea
        v-autogrow
        class="resize-none"
        v-model="form.idea"
        required
        minlength="10"
        maxlength="3000"
        rows="3"
        placeholder="Как вы предлагаете решить задачу и почему этот подход сработает?"
        :disabled="busy"
      />
    </label>
    <label class="field-label"
      >План работы<textarea
        v-autogrow
        class="resize-none"
        v-model="form.plan"
        required
        maxlength="3000"
        rows="4"
        placeholder="Каждый шаг с новой строки"
        :disabled="busy"
      />
    </label>
    <div class="form-row">
      <label class="field-label"
        >Срок, дней<input
          v-model.number="form.durationDays"
          type="number"
          min="1"
          max="365"
          required
          :disabled="busy" /></label
      ><label class="field-label grow"
        >Ссылка на прототип<input
          v-model="form.prototypeUrl"
          type="url"
          required
          maxlength="2000"
          placeholder="https://…"
          :disabled="busy"
      /></label>
    </div>
    <div class="editor-actions">
      <button class="button primary" type="submit" :disabled="busy">
        <AppIcon :name="busy ? 'LoaderCircle' : 'Send'" :class="{ spin: busy }" :size="16" />{{
          busy ? 'Отправляем…' : 'Отправить предложение'
        }}</button
      ><button
        v-if="store.mode === 'demo'"
        class="text-link"
        type="button"
        :disabled="busy"
        @click="fill"
      >
        Заполнить пример
      </button>
    </div>
    <p class="small muted">
      Отклик доступен при любом рейтинге задачи. Бизнес самостоятельно выберет команды.
    </p>
  </form>
</template>
