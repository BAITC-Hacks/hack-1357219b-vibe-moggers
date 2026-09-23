<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useWorkspace } from '../stores/workspace'
import {
  clone,
  emptyFields,
  fieldKeys,
  fieldMeta,
  fieldValid,
  groups,
  industries,
  scoreFields,
} from '../domain/scoring'
import type { AnalyzeResult, FieldKey, Industry, Source, Task } from '../domain/types'
import { demo40, demo95, demoDraft, demoTitle } from '../data/demo'
import AppIcon from '../components/AppIcon.vue'
import LeaveDialog from '../components/LeaveDialog.vue'
import { validateForm } from '../composables/formControls'
import ScorePanel from '../components/ScorePanel.vue'

const store = useWorkspace()
const route = useRoute()
const router = useRouter()
const task = ref<Task | null>(null)
const step = ref(route.params.id ? 3 : 1)
const busy = ref(false)
const loading = ref(!!route.params.id)
const error = ref('')
const dirty = ref(false)
const reviewed = ref(false)
const canLeave = ref(false)
const fields = ref(emptyFields())
const form = reactive({ title: '', industry: 'retail' as Industry, draft: '' })
const analysis = ref<AnalyzeResult | null>(null)
const answers = ref<Record<string, string>>({})
const openGroups = ref<string[]>(['context'])
const preview = computed(() => scoreFields(fields.value, true))
const confirmed = computed(() => task.value?.confirmedSnapshot?.total ?? 0)
const savedAndConfirmed = computed(
  () =>
    !dirty.value &&
    task.value?.confirmedRevision === task.value?.revision &&
    !!task.value?.confirmedSnapshot,
)
const canEdit = computed(
  () => store.isBusiness && (!task.value || task.value.ownerId === store.actorId),
)
watch(
  [form, fields, answers],
  () => {
    dirty.value = true
    reviewed.value = false
  },
  { deep: true, flush: 'sync' },
)
function applyTask(value: Task) {
  task.value = value
  form.title = value.title
  form.industry = value.industry
  form.draft = value.draft
  fields.value = clone(value.workingFields)
  dirty.value = false
}
onMounted(async () => {
  if (!route.params.id) return
  try {
    const value = await store.api.getTask(String(route.params.id))
    applyTask(value)
    answers.value = Object.fromEntries(value.answers.map((answer) => [answer.id, answer.text]))
    dirty.value = false
  } catch (err) {
    error.value = message(err)
  } finally {
    loading.value = false
  }
})
function message(err: unknown) {
  return err instanceof Error ? err.message : 'Не удалось выполнить действие'
}
async function perform(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await action()
  } catch (err) {
    error.value = message(err)
  } finally {
    busy.value = false
  }
}
function fillIdea() {
  form.title = demoTitle
  form.draft = demoDraft
  form.industry = 'retail'
}
function fillExample(complete: boolean) {
  form.title = demoTitle
  form.industry = 'retail'
  fields.value = emptyFields()
  for (const [key, value] of Object.entries(complete ? demo95 : demo40))
    fields.value[key as FieldKey].value = value
  openGroups.value = complete ? ['success', 'contact'] : ['context', 'users']
  store.notify('Пример заполнен. Проверьте сведения и подтвердите их.', 'info')
}
function sources(): Source[] {
  return [
    { id: 'draft', text: form.draft },
    ...Object.entries(answers.value)
      .filter(([, text]) => text.trim())
      .map(([id, text]) => ({ id, text })),
  ]
}
async function ensureTask() {
  if (!form.title.trim() || form.draft.trim().length < 10)
    throw new Error('Добавьте название и описание хотя бы из 10 символов.')
  if (!task.value) task.value = await store.api.createTask({ ...form })
}
function clarify(event?: Event) {
  if (event) {
    error.value = validateForm(event, 'editor-error')
    if (error.value) return
  }
  return perform(async () => {
    await ensureTask()
    analysis.value = await store.liveApi.analyze({
      stage: 'clarify',
      sources: sources(),
      currentFields: clone(fields.value),
    })
    step.value = 2
    window.scrollTo({ top: 0, behavior: 'smooth' })
  })
}
function assemble() {
  return perform(async () => {
    await ensureTask()
    const result = await store.liveApi.analyze({
      stage: 'assemble',
      sources: sources(),
      currentFields: clone(fields.value),
    })
    analysis.value = result
    for (const suggestion of result.fieldSuggestions) {
      fields.value[suggestion.field] = {
        value: suggestion.value,
        confirmed: false,
        source: { id: suggestion.sourceId, quote: suggestion.value },
      }
    }
    step.value = 3
    openGroups.value = ['context']
    window.scrollTo({ top: 0, behavior: 'smooth' })
  })
}
function manualCard() {
  return perform(async () => {
    await ensureTask()
    step.value = 3
  })
}
async function save() {
  await ensureTask()
  const updated = await store.api.saveCard(task.value!.id, {
    revision: task.value!.revision,
    title: form.title,
    industry: form.industry,
    answers: sources().filter((source) => source.id !== 'draft'),
    fields: clone(fields.value),
  })
  applyTask(updated)
}
function saveDraft() {
  return perform(async () => {
    await save()
    await store.refresh()
    store.notify('Черновик сохранён. Публичная карточка не изменилась.')
  })
}
function confirm() {
  return perform(async () => {
    if (!reviewed.value) throw new Error('Сначала подтвердите, что проверили сведения.')
    await save()
    applyTask(
      await store.api.confirmTask(
        task.value!.id,
        task.value!.revision,
        fieldKeys.filter((key) => fieldValid(key, fields.value[key].value)),
      ),
    )
    reviewed.value = true
    await store.refresh()
    store.notify(
      `Сведения подтверждены. Готовность — ${task.value!.confirmedSnapshot?.total ?? 0} из 100.`,
    )
  })
}
function publish() {
  return perform(async () => {
    if (!task.value || !savedAndConfirmed.value)
      throw new Error('Подтвердите текущую версию карточки.')
    applyTask(await store.api.publishTask(task.value.id, task.value.revision))
    await store.refresh()
    store.notify('Задача опубликована. Теперь команды могут откликнуться.')
    canLeave.value = true
    await router.push(`/tasks/${task.value!.id}`)
  })
}
async function jump(key: string) {
  if (step.value !== 3) {
    if (!task.value) return
    step.value = 3
  }
  if (!openGroups.value.includes(key)) openGroups.value.push(key)
  await nextTick()
  document
    .getElementById(`field-group-${key}`)
    ?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}
function toggleGroup(key: string) {
  openGroups.value = openGroups.value.includes(key)
    ? openGroups.value.filter((item) => item !== key)
    : [...openGroups.value, key]
}
function beforeUnload(event: BeforeUnloadEvent) {
  if (dirty.value && !canLeave.value) {
    event.preventDefault()
    event.returnValue = ''
  }
}
window.addEventListener('beforeunload', beforeUnload)
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
const leaveDialog = ref<InstanceType<typeof LeaveDialog> | null>(null)
onBeforeRouteLeave(() => !dirty.value || canLeave.value || (leaveDialog.value?.ask() ?? false))
</script>
<template>
  <LeaveDialog ref="leaveDialog" />
  <div class="page-title-row">
    <div>
      <RouterLink to="/catalog" class="back-link"
        ><AppIcon name="ArrowLeft" :size="15" /> В каталог</RouterLink
      >
      <h1>
        {{ task?.publishedAt ? 'Сделаем задачу ещё понятнее' : 'От идеи — к понятной задаче' }}
      </h1>
      <p>Расскажите о потребности. Мы поможем подготовить её к работе с командой.</p>
    </div>
    <span class="private-label"
      ><AppIcon :name="task?.publishedAt ? 'Globe' : 'ShieldCheck'" :size="15" />{{
        task?.publishedAt ? 'Задача опубликована' : 'Черновик виден только вам'
      }}</span
    >
  </div>
  <div v-if="loading" class="empty-state">
    <AppIcon class="spin" name="LoaderCircle" :size="28" />
    <p>Загружаем вашу задачу…</p>
  </div>
  <div v-else-if="!canEdit || (route.params.id && !task)" class="empty-state">
    <AppIcon name="ShieldCheck" :size="30" />
    <h2>{{ !canEdit ? 'Конструктор доступен владельцу задачи' : 'Не удалось открыть задачу' }}</h2>
    <p>{{ error || 'Для размещения задачи нужен профиль заказчика.' }}</p>
    <RouterLink to="/catalog" class="button secondary">Вернуться в каталог</RouterLink>
  </div>
  <div v-else class="editor-layout">
    <div class="editor-main">
      <div class="stepper">
        <button
          v-for="(label, index) in ['Ваша идея', 'Уточнение', 'Карточка задачи']"
          :key="label"
          type="button"
          :class="{ current: step === index + 1, complete: step > index + 1 }"
          :disabled="busy || index + 1 > step"
          @click="step = index + 1"
        >
          <span
            ><AppIcon v-if="step > index + 1" name="Check" :size="14" /><template v-else>{{
              String(index + 1).padStart(2, '0')
            }}</template></span
          >{{ label }}
        </button>
      </div>
      <div v-if="error" id="editor-error" class="inline-alert error" role="alert">
        <AppIcon name="CircleAlert" :size="19" /><span>{{ error }}</span
        ><button
          v-if="step === 1 && task"
          type="button"
          class="text-link"
          :disabled="busy"
          @click="manualCard"
        >
          Заполнить карточку вручную
        </button>
      </div>
      <form v-if="step === 1" class="panel editor-panel" novalidate @submit.prevent="clarify">
        <span class="soft-icon blue large"><AppIcon name="Lightbulb" :size="24" /></span>
        <h2>Какая задача у вашего бизнеса?</h2>
        <p class="muted">Опишите своими словами — специальное техническое задание пока не нужно.</p>
        <div class="form-row">
          <label class="field-label"
            >Название задачи<input
              v-model="form.title"
              required
              maxlength="160"
              placeholder="Например: помощник планирования закупок"
              :disabled="busy" /></label
          ><label class="field-label industry-field"
            >Направление<select v-model="form.industry" :disabled="busy">
              <option v-for="item in industries" :key="item.value" :value="item.value">
                {{ item.short }}
              </option>
            </select></label
          >
        </div>
        <label class="field-label"
          >Что хотите улучшить?<textarea
            v-autogrow
            class="resize-none"
            v-model="form.draft"
            required
            minlength="10"
            maxlength="6000"
            rows="7"
            placeholder="Расскажите, что происходит сейчас, в чём сложность и какого результата вы хотите достичь…"
            :disabled="busy || !!task"
          />
        </label>
        <div class="field-caption">
          <span>Без персональных и конфиденциальных данных</span
          ><span>{{ form.draft.length }}/6000</span>
        </div>
        <button
          v-if="store.mode === 'demo' && !task"
          type="button"
          class="example-button"
          @click="fillIdea"
        >
          <AppIcon name="Sparkles" :size="16" /> Попробовать на примере магазина
          <AppIcon name="ArrowUpRight" :size="14" />
        </button>
        <div class="editor-actions">
          <button class="button primary" type="submit" :disabled="busy">
            <AppIcon
              :name="busy ? 'LoaderCircle' : 'Sparkles'"
              :class="{ spin: busy }"
              :size="17"
            />{{
              busy
                ? 'Готовим вопросы…'
                : store.mode === 'demo'
                  ? 'Получить вопросы'
                  : 'Уточнить с AI'
            }}</button
          ><button
            class="button ghost"
            type="button"
            :disabled="busy || !form.title || form.draft.length < 10"
            @click="manualCard"
          >
            Заполнить вручную <AppIcon name="ArrowRight" :size="15" />
          </button>
        </div>
      </form>
      <div v-else-if="step === 2" class="panel editor-panel">
        <div class="inline-title">
          <span class="soft-icon blue"><AppIcon name="Sparkles" /></span>
          <div>
            <h2>Добавим немного конкретики</h2>
            <p class="muted">Ответьте на вопросы, которые помогут команде понять задачу.</p>
          </div>
        </div>
        <div
          v-for="(question, index) in analysis?.questions"
          :key="question.id"
          class="question-block"
        >
          <label :for="question.id"
            ><span>{{ String(index + 1).padStart(2, '0') }}</span
            >{{ question.text }}</label
          ><textarea
            v-autogrow
            class="resize-none"
            :id="question.id"
            v-model="answers[question.id]"
            rows="3"
            maxlength="3000"
            placeholder="Что известно сейчас? Если информации нет, можно пропустить."
            :disabled="busy"
          />
        </div>
        <div class="editor-actions">
          <button class="button primary" :disabled="busy" @click="assemble()">
            <AppIcon
              :name="busy ? 'LoaderCircle' : 'ArrowRight'"
              :class="{ spin: busy }"
              :size="17"
            />{{ busy ? 'Готовим карточку…' : 'Перейти к карточке' }}</button
          ><button class="button ghost" :disabled="busy" @click="manualCard">
            Заполнить вручную
          </button>
        </div>
      </div>
      <div v-else>
        <div class="card-editor-heading">
          <div>
            <h2>Проверьте карточку задачи</h2>
            <p>Неизвестные поля можно оставить пустыми и дополнить позже.</p>
          </div>
          <button
            class="text-link"
            type="button"
            @click="
              openGroups =
                openGroups.length === groups.length ? ['context'] : groups.map((group) => group.key)
            "
          >
            {{ openGroups.length === groups.length ? 'Свернуть' : 'Развернуть всё' }}
          </button>
        </div>
        <div v-if="analysis?.warnings.length" class="inline-alert info">
          <AppIcon name="CircleHelp" :size="18" /><span>{{
            analysis.mode === 'fallback'
              ? 'В резервном режиме внесите сведения в поля вручную. Ваши ответы доступны ниже.'
              : analysis.warnings.join(' ')
          }}</span>
        </div>
        <details v-if="Object.values(answers).some(Boolean)" class="panel answer-reference">
          <summary>Ваши ответы на уточняющие вопросы</summary>
          <p v-for="(answer, key) in answers" :key="key">{{ answer }}</p>
        </details>
        <div v-if="store.mode === 'demo'" class="demo-fill">
          <span><AppIcon name="Sparkles" :size="16" /> Синтетический пример</span
          ><button type="button" :disabled="busy" @click="fillExample(false)">
            Заполнить на 40</button
          ><button type="button" :disabled="busy" @click="fillExample(true)">
            Дополнить до 95 <AppIcon name="ArrowUpRight" :size="13" />
          </button>
        </div>
        <div class="panel card-title-fields">
          <label class="field-label"
            >Название<input v-model="form.title" maxlength="160" :disabled="busy" /></label
          ><label class="field-label"
            >Направление<select v-model="form.industry" :disabled="busy">
              <option v-for="item in industries" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select></label
          >
        </div>
        <section
          v-for="(group, index) in groups"
          :id="`field-group-${group.key}`"
          :key="group.key"
          class="field-group panel"
        >
          <button
            class="field-group-heading"
            type="button"
            :aria-expanded="openGroups.includes(group.key)"
            @click="toggleGroup(group.key)"
          >
            <span class="group-number">{{ String(index + 1).padStart(2, '0') }}</span>
            <h3>{{ group.label }}</h3>
            <span
              class="group-points"
              :class="{ full: preview.breakdown[index]!.earned === preview.breakdown[index]!.max }"
              >{{ preview.breakdown[index]!.earned }}/{{ preview.breakdown[index]!.max }}</span
            ><AppIcon
              name="ChevronDown"
              :size="17"
              :class="{ rotated: openGroups.includes(group.key) }"
            />
          </button>
          <div v-show="openGroups.includes(group.key)" class="field-group-body">
            <label v-for="key in group.fields" :key="key" class="field-label"
              ><span
                >{{ fieldMeta[key].label }}<small>+{{ fieldMeta[key].weight }}</small></span
              ><textarea
                v-autogrow
                class="resize-none"
                v-model="fields[key].value"
                :rows="key === 'contact' || key === 'deadline' ? 2 : 3"
                :placeholder="fieldMeta[key].placeholder"
                maxlength="3000"
                :disabled="busy"
              /><span
                v-if="fields[key].source && fields[key].source!.quote === fields[key].value"
                class="field-source"
                ><AppIcon name="CopyCheck" :size="12" /> Из вашего ответа:
                {{ fields[key].source!.id }}</span
              ><span
                v-if="fields[key].value && !fieldValid(key, fields[key].value)"
                class="field-warning"
                >Уточните значение: {{ fieldMeta[key].placeholder.toLowerCase() }}</span
              ></label
            >
          </div>
        </section>
        <div class="panel confirmation-panel">
          <label class="checkbox-label"
            ><input v-model="reviewed" type="checkbox" :disabled="busy" /><span
              ><b>Я проверил(а) указанные сведения</b
              ><small
                >Они соответствуют задаче моего бизнеса. Пустые поля остаются неизвестными.</small
              ></span
            ></label
          >
          <div class="confirmation-actions">
            <button class="button secondary" :disabled="busy" @click="saveDraft">
              <AppIcon name="Save" :size="16" /> Сохранить черновик</button
            ><button
              v-if="!savedAndConfirmed"
              class="button primary"
              :disabled="busy || !reviewed || !form.title.trim()"
              @click="confirm"
            >
              <AppIcon
                :name="busy ? 'LoaderCircle' : 'ShieldCheck'"
                :class="{ spin: busy }"
                :size="17"
              />
              Подтвердить сведения</button
            ><button
              v-else-if="!task?.publishedAt"
              class="button primary"
              :disabled="busy"
              @click="publish"
            >
              <AppIcon name="Globe" :size="17" /> Опубликовать задачу</button
            ><RouterLink v-else :to="`/tasks/${task.id}`" class="button primary"
              >Открыть задачу <AppIcon name="ArrowUpRight" :size="17"
            /></RouterLink>
          </div>
        </div>
      </div>
    </div>
    <ScorePanel
      :score="savedAndConfirmed && task?.confirmedSnapshot ? task.confirmedSnapshot : preview"
      :confirmed="confirmed"
      :preview="!savedAndConfirmed"
      @jump="jump"
    />
  </div>
</template>
