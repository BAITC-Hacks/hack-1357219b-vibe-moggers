import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { account, logout } from '../services/session'
import { DemoApi } from '../services/demo-api'
import { HttpApi } from '../services/http-api'
import type { LocalProfile, Task, Team } from '../domain/types'

export const useWorkspace = defineStore('workspace', () => {
  const actorId = ref('guest')
  const demoApi = new DemoApi(() => actorId.value, localStorage)
  const liveApi = new HttpApi(() => actorId.value)
  const localProfile = ref<LocalProfile | null>(null)
  const profile = computed<LocalProfile | null>(() =>
    account.value ? { ...account.value, id: account.value.actorId } : localProfile.value,
  )
  const api = computed(() =>
    account.value || import.meta.env.VITE_DATA_MODE === 'api' ? liveApi : demoApi,
  )
  const mode = computed(() => api.value.mode)
  try {
    const saved = localStorage.getItem('qadam:profile')
    if (saved && import.meta.env.VITE_DATA_MODE !== 'api')
      localProfile.value = demoApi.getProfile(saved)
  } catch {
    /* Optional demo persistence. */
  }
  actorId.value = profile.value?.id ?? 'guest'
  const catalog = ref<Task[]>([])
  const mine = ref<Task[]>([])
  const teams = ref<Team[]>([])
  const loading = ref(false)
  const error = ref('')
  const loaded = ref(false)
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
  async function createProfile(role: LocalProfile['role'], name: string) {
    if (!(api.value instanceof DemoApi))
      throw new Error('Регистрация пока не подключена. Просмотр каталога доступен без профиля.')
    const next = demoApi.registerProfile(role, name)
    localProfile.value = next
    actorId.value = next.id
    mine.value = []
    try {
      localStorage.setItem('qadam:profile', next.id)
    } catch {
      notify('Профиль создан, но браузер не сохранил вход. Не закрывайте вкладку.', 'info')
    }
    await refresh()
  }
  function savedProfile(role: LocalProfile['role']) {
    return api.value instanceof DemoApi ? demoApi.savedProfile(role) : null
  }
  async function resumeProfile(role: LocalProfile['role']) {
    const saved = savedProfile(role)
    if (!saved) throw new Error('Локальный профиль не найден. Создайте новый.')
    localProfile.value = saved
    actorId.value = saved.id
    try {
      localStorage.setItem('qadam:profile', saved.id)
    } catch {
      /* Session remains usable. */
    }
    await refresh()
  }
  async function leaveProfile() {
    if (account.value) await logout()
    localProfile.value = null
    actorId.value = 'guest'
    mine.value = []
    try {
      localStorage.removeItem('qadam:profile')
    } catch {
      /* Memory state still resets. */
    }
    await refresh()
  }
  watch(account, () => {
    localProfile.value = null
    try {
      localStorage.removeItem('qadam:profile')
    } catch {
      /* Optional demo persistence. */
    }
    actorId.value = profile.value?.id ?? 'guest'
    mine.value = []
    catalog.value = []
    teams.value = []
    void refresh()
  })
  return {
    account,
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
    createProfile,
    savedProfile,
    resumeProfile,
    leaveProfile,
  }
})
