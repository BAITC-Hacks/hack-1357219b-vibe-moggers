<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useWorkspace } from '../stores/workspace'
import { transcribeAudio } from '../services/ai-api'
import AppIcon from './AppIcon.vue'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    context: string
    label?: string
    maxLength?: number
    disabled?: boolean
  }>(),
  { modelValue: '', label: 'Добавить голосом', maxLength: 3000 },
)
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const store = useWorkspace()
const state = ref<'idle' | 'recording' | 'transcribing'>('idle')
const elapsed = ref(0)
const error = ref('')
const success = ref('')
let recorder: MediaRecorder | null = null
let stream: MediaStream | null = null
let chunks: Blob[] = []
let interval: number | undefined
let timeout: number | undefined
let request: AbortController | undefined
let disposed = false
const requesting = ref(false)

const time = computed(
  () =>
    `${String(Math.floor(elapsed.value / 60)).padStart(2, '0')}:${String(elapsed.value % 60).padStart(2, '0')}`,
)

function clearTimers() {
  if (interval) window.clearInterval(interval)
  if (timeout) window.clearTimeout(timeout)
  interval = undefined
  timeout = undefined
}

function closeStream() {
  stream?.getTracks().forEach((track) => track.stop())
  stream = null
}

async function start() {
  if (state.value !== 'idle' || requesting.value || props.disabled) return
  error.value = ''
  success.value = ''
  if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') {
    error.value = 'Этот браузер не поддерживает запись. Введите ответ текстом.'
    return
  }
  try {
    requesting.value = true
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    if (disposed) {
      closeStream()
      return
    }
    const preferred = ['audio/webm;codecs=opus', 'audio/mp4', 'audio/webm'].find((type) =>
      MediaRecorder.isTypeSupported(type),
    )
    recorder = new MediaRecorder(stream, preferred ? { mimeType: preferred } : undefined)
    chunks = []
    recorder.ondataavailable = (event) => {
      if (event.data.size) chunks.push(event.data)
    }
    recorder.onstop = upload
    recorder.onerror = () => {
      clearTimers()
      if (recorder) recorder.onstop = null
      closeStream()
      state.value = 'idle'
      error.value = 'Запись прервалась. Попробуйте ещё раз или введите ответ текстом.'
    }
    recorder.start(250)
    elapsed.value = 0
    state.value = 'recording'
    interval = window.setInterval(() => elapsed.value++, 1000)
    timeout = window.setTimeout(stop, 60_000)
  } catch (err) {
    closeStream()
    error.value =
      err instanceof DOMException && ['NotAllowedError', 'PermissionDeniedError'].includes(err.name)
        ? 'Разрешите доступ к микрофону в браузере и попробуйте снова.'
        : 'Не удалось включить микрофон. Введите ответ текстом.'
  } finally {
    requesting.value = false
  }
}

function stop() {
  if (state.value !== 'recording' || recorder?.state !== 'recording') return
  state.value = 'transcribing'
  clearTimers()
  recorder?.stop()
  closeStream()
}

async function upload() {
  const blob = new Blob(chunks, { type: recorder?.mimeType || 'audio/webm' })
  recorder = null
  chunks = []
  if (blob.size < 800) {
    state.value = 'idle'
    error.value = 'Запись слишком короткая. Нажмите кнопку и произнесите полный ответ.'
    return
  }
  state.value = 'transcribing'
  request = new AbortController()
  try {
    const result = await transcribeAudio(store.actorId, blob, props.context, request.signal)
    const separator = props.modelValue.trim() ? ' ' : ''
    const combined = `${props.modelValue.trim()}${separator}${result.text.trim()}`
    emit('update:modelValue', combined.slice(0, props.maxLength))
    success.value =
      combined.length > props.maxLength
        ? 'Текст добавлен до ограничения поля. Проверьте окончание.'
        : 'Текст добавлен. Проверьте расшифровку перед продолжением.'
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Не удалось распознать запись.'
  } finally {
    state.value = 'idle'
    request = undefined
  }
}

onBeforeUnmount(() => {
  disposed = true
  clearTimers()
  request?.abort()
  if (recorder?.state === 'recording') {
    recorder.onstop = null
    recorder.stop()
  }
  closeStream()
})
</script>

<template>
  <div class="voice-recorder" :class="state">
    <button
      v-if="state === 'idle'"
      type="button"
      class="voice-button"
      :disabled="disabled || requesting"
      @click="start"
    >
      <span><AppIcon name="Mic" :size="18" /></span
      >{{ requesting ? 'Ожидаем разрешение…' : props.label }}
    </button>
    <button
      v-else-if="state === 'recording'"
      type="button"
      class="voice-button recording"
      @click="stop"
    >
      <span><AppIcon name="Square" :size="15" /></span>Остановить · {{ time }}
    </button>
    <div v-else class="voice-progress" role="status">
      <AppIcon name="LoaderCircle" class="spin" :size="18" />Расшифровываем запись…
    </div>
    <span v-if="state === 'recording'" class="voice-status" role="status"
      >Говорите. Запись остановится через минуту.</span
    >
    <span v-if="success" class="voice-success" role="status">{{ success }}</span>
    <span v-if="error" class="voice-error" role="alert">{{ error }}</span>
  </div>
</template>
