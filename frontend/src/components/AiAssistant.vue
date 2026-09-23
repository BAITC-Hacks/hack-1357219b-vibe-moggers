<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { request } from '../services/session'
import AppIcon from './AppIcon.vue'
const route = useRoute()
const dialog = ref<HTMLDialogElement>(),
  input = ref<HTMLTextAreaElement>(),
  log = ref<HTMLElement>()
const chat = ref(false),
  draft = ref(''),
  error = ref(''),
  busy = ref(false)
const messages = ref<{ role: 'user' | 'assistant'; content: string }[]>([])
let controller: AbortController | undefined
function open() {
  dialog.value?.showModal()
}
function close() {
  controller?.abort()
  dialog.value?.close()
}
async function startChat() {
  chat.value = true
  await nextTick()
  input.value?.focus()
}
async function send() {
  if (busy.value || !draft.value.trim()) return
  const question = draft.value.trim()
  error.value = ''
  busy.value = true
  controller = new AbortController()
  try {
    const result = await request<{ reply: string }>(
      '/ai/chat',
      'POST',
      {
        messages: [...messages.value.slice(-18), { role: 'user', content: question }],
        context: {
          page: route.path,
          taskId: typeof route.params.id === 'string' ? route.params.id : undefined,
        },
      },
      controller.signal,
    )
    if (
      !result ||
      typeof result.reply !== 'string' ||
      !result.reply.trim() ||
      result.reply.length > 12000
    )
      throw new Error('AI вернул неверный ответ. Попробуйте ещё раз.')
    messages.value.push(
      { role: 'user', content: question },
      { role: 'assistant', content: result.reply },
    )
    draft.value = ''
    await nextTick()
    log.value?.scrollTo({ top: log.value.scrollHeight, behavior: 'smooth' })
  } catch (err) {
    if (!controller.signal.aborted)
      error.value = err instanceof Error ? err.message : 'AI недоступен.'
  } finally {
    busy.value = false
  }
}
watch(
  () => route.path,
  () => {
    close()
    messages.value = []
    error.value = ''
  },
)
onBeforeUnmount(() => controller?.abort())
</script>
<template>
  <button
    class="assistant-emblem"
    aria-label="Qadam AI — помощь с проектом"
    aria-haspopup="dialog"
    @click="open"
  >
    <span><AppIcon name="Sparkles" :size="26" /></span><b>Qadam AI</b
    ><small>Помочь с проектом</small>
  </button>
  <Teleport to="body">
    <dialog
      ref="dialog"
      class="assistant-dialog"
      aria-labelledby="assistant-title"
      @cancel.prevent="close"
      @click="
        (event) => {
          if (event.target === dialog) close()
        }
      "
    >
      <div class="assistant-shell">
        <header>
          <span class="soft-icon blue"><AppIcon name="Sparkles" /></span>
          <div>
            <h2 id="assistant-title">Ваш помощник Qadam</h2>
            <p>От идеи к понятной задаче</p>
          </div>
          <button class="button ghost" aria-label="Закрыть помощника" @click="close">
            <AppIcon name="X" />
          </button>
        </header>
        <div v-if="!chat" class="assistant-intro">
          <span class="eyebrow">ОДИН ПРОЕКТ. СЛЕДУЮЩИЙ ШАГ.</span>
          <h3>Разберёмся вместе</h3>
          <p>
            Помогу разобраться в платформе, сформулировать задачу или подготовить предложение
            команде.
          </p>
          <ol>
            <li><b>Опишите цель</b><span>Что нужно изменить и для кого?</span></li>
            <li>
              <b>Уточните детали с AI</b><span>Данные, ограничения и ожидаемый результат.</span>
            </li>
            <li>
              <b>Начните сотрудничество</b
              ><span>Выберите команду и договоритесь о первом результате.</span>
            </li>
          </ol>
          <button class="button primary" @click="startChat">
            Задать вопрос <AppIcon name="MessageSquare" :size="18" />
          </button>
        </div>
        <template v-else>
          <div
            ref="log"
            class="assistant-messages"
            aria-label="Переписка с AI"
            role="log"
            aria-live="polite"
          >
            <p v-if="!messages.length" class="muted">
              Например: «Как описать результат задачи, чтобы команда поняла, что нужно сделать?»
            </p>
            <article v-for="(message, index) in messages" :key="index" :class="message.role">
              <b>{{ message.role === 'user' ? 'Вы' : 'Qadam AI' }}</b>
              <p>{{ message.content }}</p>
            </article>
          </div>
          <form class="assistant-composer" novalidate @submit.prevent="send">
            <p class="muted small">
              AI учитывает текущую страницу. Не отправляйте пароли и секреты. Ответы AI стоит
              проверять.
            </p>
            <label for="assistant-question" class="field-label">Ваш вопрос</label
            ><textarea
              id="assistant-question"
              ref="input"
              v-model="draft"
              class="resize-none"
              rows="3"
              maxlength="2000"
              required
              :disabled="busy"
              placeholder="Чем помочь с проектом?"
            />
            <p v-if="error" class="profile-error" role="alert">{{ error }}</p>
            <button class="button primary" :disabled="busy || !draft.trim()" :aria-busy="busy">
              {{ busy ? 'AI готовит ответ…' : error ? 'Повторить отправку' : 'Отправить'
              }}<AppIcon name="Send" :size="17" />
            </button>
          </form>
        </template>
      </div>
    </dialog>
  </Teleport>
</template>
