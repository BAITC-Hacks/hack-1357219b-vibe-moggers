import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { DemoApi } from '../services/demo-api'
import { HttpApi } from '../services/http-api'
import type { LocalProfile, Task, Team } from '../domain/types'

export const useWorkspace = defineStore('workspace', () => {
  const actorId = ref(readActor())
  const demoApi = new DemoApi(() => actorId.value, localStorage)
  const liveApi = new HttpApi(() => actorId.value)
  const api = computed(() => (import.meta.env.VITE_DATA_MODE === 'demo' ? demoApi : liveApi))
  const mode = computed(() => api.value.mode)
  const catalog = ref<Task[]>([])
  const mine = ref<Task[]>([])
  const teams = ref<Team[]>([])
  const loading = ref(false)
  const error = ref('')
  const loaded = ref(false)
  const profile = computed<LocalProfile | null>(() => {
    if (actorId.value === 'demo-business-1')
      return { id: actorId.value, role: 'business', name: 'Бизнес Qadam' }
    const team = teams.value.find((item) => item.id === actorId.value)
    const fallbackName = presetTeamNames[actorId.value]
    return team
      ? { id: team.id, role: 'team', name: team.name }
      : fallbackName
        ? { id: actorId.value, role: 'team', name: fallbackName }
        : null
  })
  const isBusiness = computed(() => profile.value?.role === 'business')
  const isTeam = computed(() => profile.value?.role === 'team')
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
        api.value.listTasks(),
        api.value.listTeams(),
        isBusiness.value ? api.value.listOwnTasks() : Promise.resolve([]),
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
  async function selectActor(id: string) {
    const allowed =
      id === 'guest' || id === 'demo-business-1' || teams.value.some((t) => t.id === id)
    if (!allowed) throw new Error('Выбранный демонстрационный профиль не найден.')
    actorId.value = id
    try {
      if (id === 'guest') localStorage.removeItem('qadam:actor')
      else localStorage.setItem('qadam:actor', id)
    } catch {
      notify('Роль выбрана, но браузер не сможет запомнить её после закрытия вкладки.', 'info')
    }
    mine.value = []
    await refresh()
  }
  async function leaveProfile() {
    await selectActor('guest')
  }
  return {
    liveApi,
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
    isTeam,
    profile,
    currentTeam,
    toasts,
    notify,
    dismiss,
    refresh,
    selectActor,
    leaveProfile,
  }
})

function readActor() {
  try {
    const value = localStorage.getItem('qadam:actor')
    return value === 'demo-business-1' || /^team-[1-5]$/.test(value || '') ? value! : 'guest'
  } catch {
    return 'guest'
  }
}

const presetTeamNames: Record<string, string> = {
  'team-1': 'Команда веб-разработки',
  'team-2': 'Steppe Flow',
  'team-3': 'Campus Builders',
  'team-4': 'Route Lab',
  'team-5': 'Agro Makers',
}
