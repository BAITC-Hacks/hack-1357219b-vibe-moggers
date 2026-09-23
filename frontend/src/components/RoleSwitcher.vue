<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useWorkspace } from '../stores/workspace'
import AppIcon from './AppIcon.vue'

const store = useWorkspace()
const router = useRouter()
const busy = ref(false)

async function change(event: Event) {
  const select = event.target as HTMLSelectElement
  const id = select.value
  if (busy.value || id === store.actorId) return
  busy.value = true
  try {
    if (router.currentRoute.value.path !== '/catalog') {
      const failure = await router.push('/catalog')
      if (failure) {
        select.value = store.actorId
        return
      }
    }
    await store.selectActor(id)
    if (id === 'demo-business-1') await router.push('/business')
    store.notify(
      id === 'guest' ? 'Роль сброшена.' : `Профиль переключён: ${store.profile?.name}.`,
      'info',
    )
  } catch (err) {
    select.value = store.actorId
    store.notify(err instanceof Error ? err.message : 'Не удалось переключить профиль.', 'error')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <label class="role-switcher">
    <span class="actor-avatar">
      <AppIcon
        :name="store.isBusiness ? 'Building2' : store.isTeam ? 'Users' : 'CircleHelp'"
        :size="18"
      />
    </span>
    <span class="role-switcher-copy">
      <small>Демо-профиль</small>
      <select
        :value="store.actorId"
        :disabled="busy"
        aria-label="Демонстрационный профиль"
        @change="change"
      >
        <option value="guest">Без роли</option>
        <optgroup label="Бизнес">
          <option value="demo-business-1">Бизнес Qadam</option>
        </optgroup>
        <optgroup label="Студенческие команды">
          <option v-for="team in store.teams" :key="team.id" :value="team.id">
            {{ team.name }}
          </option>
        </optgroup>
      </select>
    </span>
  </label>
</template>
