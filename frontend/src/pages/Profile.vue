<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkspace } from '../stores/workspace'
import AppIcon from '../components/AppIcon.vue'

const route = useRoute()
const router = useRouter()
const store = useWorkspace()
const role = ref<'business' | 'team'>(
  route.query.role === 'team' || store.isTeam ? 'team' : 'business',
)
const teamId = ref(store.isTeam ? store.actorId : '')
const error = ref('')
const busy = ref(false)

const next = computed(() => {
  const value = typeof route.query.next === 'string' ? route.query.next : ''
  if (
    role.value === 'business' &&
    /^\/(tasks\/new|business(?:\/.*)?|tasks\/[^/]+\/edit)$/.test(value)
  )
    return value
  if (role.value === 'team' && /^\/(applications|tasks\/[^/?#]+(?:#proposal)?)$/.test(value))
    return value
  return role.value === 'business' ? '/business' : '/catalog'
})

watch(
  () => store.teams,
  (teams) => {
    if (!teamId.value && teams.length) teamId.value = teams[0]!.id
  },
  { immediate: true },
)

onMounted(async () => {
  if (!store.loaded) await store.refresh()
})

async function applyProfile() {
  if (busy.value) return
  error.value = ''
  const id = role.value === 'business' ? 'demo-business-1' : teamId.value
  if (!id) {
    error.value = 'Выберите студенческую команду.'
    return
  }
  busy.value = true
  try {
    await store.selectActor(id)
    await router.replace(next.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Не удалось переключить профиль.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="onboarding-page role-page">
    <RouterLink to="/catalog" class="back-link">
      <AppIcon name="ArrowLeft" :size="16" />Вернуться в каталог
    </RouterLink>
    <span class="eyebrow">ДЕМОНСТРАЦИОННЫЙ РЕЖИМ</span>
    <h1>Кем вы хотите продолжить?</h1>
    <p class="guide-intro">
      Для пятичасового MVP регистрация не нужна. Выберите сторону проекта и покажите полный путь от
      задачи до результата.
    </p>

    <div class="role-options" role="group" aria-label="Роль на платформе">
      <button
        type="button"
        :class="{ selected: role === 'business' }"
        :aria-pressed="role === 'business'"
        @click="role = 'business'"
      >
        <AppIcon name="Building2" :size="27" />
        <b>Бизнес</b>
        <span>Создать задачу, уточнить её с AI и выбрать команду.</span>
        <AppIcon v-if="role === 'business'" name="CheckCircle2" class="role-check" />
      </button>
      <button
        type="button"
        :class="{ selected: role === 'team' }"
        :aria-pressed="role === 'team'"
        @click="role = 'team'"
      >
        <AppIcon name="GraduationCap" :size="27" />
        <b>Студенческая команда</b>
        <span>Выбрать задачу, отправить предложение и показать результат.</span>
        <AppIcon v-if="role === 'team'" name="CheckCircle2" class="role-check" />
      </button>
    </div>

    <div v-if="role === 'team'" class="panel preset-teams">
      <div>
        <h2>Выберите готовую команду</h2>
        <p>У каждой команды уже есть профиль, навыки и история предложений.</p>
      </div>
      <label v-for="team in store.teams" :key="team.id" class="preset-team">
        <input v-model="teamId" type="radio" name="team" :value="team.id" />
        <span class="actor-avatar"><AppIcon name="Users" :size="18" /></span>
        <span
          ><b>{{ team.name }}</b
          ><small>{{ team.technologies.join(' · ') }}</small></span
        >
        <strong>{{ team.points }} баллов</strong>
      </label>
      <p v-if="store.loading" class="muted">Загружаем профили команд…</p>
      <p v-else-if="!store.teams.length" class="profile-error" role="alert">
        Профили команд пока недоступны. Обновите страницу или выберите роль бизнеса.
      </p>
    </div>

    <p v-if="store.profile" class="current-demo-role">
      Сейчас выбрано: <b>{{ store.profile.name }}</b>
    </p>
    <p v-if="error" class="profile-error" role="alert">{{ error }}</p>
    <div class="auth-actions">
      <button
        class="button primary"
        :disabled="busy || (role === 'team' && !teamId)"
        @click="applyProfile"
      >
        <AppIcon :name="busy ? 'LoaderCircle' : 'ArrowRight'" :class="{ spin: busy }" :size="17" />
        {{ busy ? 'Переключаем…' : `Продолжить как ${role === 'business' ? 'бизнес' : 'команда'}` }}
      </button>
      <RouterLink to="/catalog" class="button ghost">Смотреть каталог без роли</RouterLink>
    </div>
    <p class="local-profile-note">
      Это демонстрационные профили хакатона. Email, номер телефона и пароль не требуются.
    </p>
  </section>
</template>
