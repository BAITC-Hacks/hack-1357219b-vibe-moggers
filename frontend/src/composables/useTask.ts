import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspace } from '../stores/workspace'
import type { Proposal, Task } from '../domain/types'

export function useTask() {
  const route = useRoute()
  const store = useWorkspace()
  const task = ref<Task | null>(null)
  const proposals = ref<Proposal[]>([])
  const loading = ref(true)
  const busy = ref(false)
  const error = ref('')
  let request = 0
  async function load() {
    const token = ++request
    loading.value = true
    error.value = ''
    try {
      const id = String(route.params.id)
      const nextTask = await store.api.getTask(id)
      const nextProposals = store.profile ? await store.api.listProposals(id) : []
      if (token === request) {
        task.value = nextTask
        proposals.value = nextProposals
      }
    } catch (err) {
      if (token === request) {
        error.value = err instanceof Error ? err.message : 'Не удалось открыть задачу'
        task.value = null
      }
    } finally {
      if (token === request) loading.value = false
    }
  }
  async function action(callback: () => Promise<unknown>, success: string) {
    if (busy.value) return
    busy.value = true
    try {
      await callback()
      await load()
      await store.refresh()
      store.notify(success)
    } catch (err) {
      store.notify(err instanceof Error ? err.message : 'Не удалось выполнить действие', 'error')
    } finally {
      busy.value = false
    }
  }
  watch(() => [route.params.id, store.actorId], load, { immediate: true })
  return { task, proposals, loading, busy, error, load, action }
}
