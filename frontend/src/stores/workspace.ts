import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { DemoApi } from '../services/demo-api'
import { HttpApi } from '../services/http-api'
import type { Task, Team } from '../domain/types'

export const useWorkspace = defineStore('workspace', () => {
  const actorId = ref(localStorage.getItem('qadam:actor') || 'demo-business-1')
  const api =
    import.meta.env.VITE_DATA_MODE === 'api'
      ? new HttpApi(() => actorId.value)
      : new DemoApi(() => actorId.value, localStorage)
  const mode = api.mode
  const catalog = ref<Task[]>([])
  const mine = ref<Task[]>([])
  const teams = ref<Team[]>([])
  const loading = ref(false)
  const error = ref('')
  const loaded = ref(false)
  const isBusiness = computed(() => actorId.value === 'demo-business-1')
  const currentTeam = computed(() => teams.value.find((team) => team.id === actorId.value))
  const toasts = ref<{ id: number; message: string; type: 'success' | 'error' | 'info' }[]>([])
  let toastId = 0
  let requestId = 0
  function notify(message: string, type: 'success' | 'error' | 'info' = 'success') {
    const id = ++toastId
    toasts.value.push({ id, message, type })
    setTimeout(() => dismiss(id), type === 'error' ? 9000 : 5000)
  }
  function dismiss(id: number) {
    toasts.value = toasts.value.filter((toast) => toast.id !== id)
  }
  async function refresh() {
    const request = ++requestId
    loading.value = true
    error.value = ''
    try {
      const [tasks, profiles, own] = await Promise.all([
        api.listTasks(),
        api.listTeams(),
        isBusiness.value ? api.listOwnTasks() : Promise.resolve([]),
      ])
      if (request !== requestId) return
      catalog.value = tasks
      teams.value = profiles
      mine.value = own
      loaded.value = true
    } catch (err) {
      if (request === requestId) {
        error.value = err instanceof Error ? err.message : 'Не удалось загрузить данные'
        loaded.value = false
      }
    } finally {
      if (request === requestId) loading.value = false
    }
  }
  async function setActor(id: string) {
    actorId.value = id
    mine.value = []
    try {
      localStorage.setItem('qadam:actor', id)
    } catch {
      /* A role switch can still work without persistence. */
    }
    await refresh()
  }
  return {
    actorId,
    mode,
    api,
    catalog,
    mine,
    teams,
    loading,
    error,
    loaded,
    isBusiness,
    currentTeam,
    toasts,
    notify,
    dismiss,
    refresh,
    setActor,
  }
})
