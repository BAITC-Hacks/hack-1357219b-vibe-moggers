<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkspace } from '../stores/workspace'
import AppIcon from '../components/AppIcon.vue'
import type { LocalProfile } from '../domain/types'
const route = useRoute()
const router = useRouter()
const store = useWorkspace()
const role = ref<LocalProfile['role'] | null>(
  route.query.role === 'business' ? 'business' : route.query.role === 'team' ? 'team' : null,
)
const name = ref('')
const nameInput = ref<HTMLInputElement | null>(null)
const error = ref('')
const busy = ref(false)
const saved = computed(() => (role.value ? store.savedProfile(role.value) : null))
function chooseRole(value: LocalProfile['role']) {
  role.value = value
  error.value = ''
}
const next = computed(() => {
  const value = typeof route.query.next === 'string' ? route.query.next : ''
  if (
    role.value === 'business' &&
    /^\/(tasks\/new|business(?:\/.*)?|tasks\/[^/]+\/edit)$/.test(value)
  )
    return value
  if (role.value === 'team' && /^\/(applications|tasks\/[^/?#]+(?:#proposal)?)$/.test(value))
    return value
  return role.value === 'business' ? '/tasks/new' : '/catalog'
})
async function submit() {
  if (busy.value || !role.value) return
  error.value = ''
  if (name.value.trim().length < 2 || name.value.trim().length > 80) {
    error.value = 'Укажите название от 2 до 80 символов.'
    nameInput.value?.focus()
    return
  }
  busy.value = true
  try {
    await store.createProfile(role.value, name.value)
    await router.replace(next.value)
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : 'Не удалось создать профиль. Попробуйте ещё раз.'
  } finally {
    busy.value = false
  }
}
async function leave() {
  if (busy.value) return
  busy.value = true
  try {
    await store.leaveProfile()
    await router.replace('/login')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Не удалось выйти.'
  } finally {
    busy.value = false
  }
}
async function resume() {
  if (!role.value || busy.value) return
  busy.value = true
  try {
    await store.resumeProfile(role.value)
    await router.replace(next.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Не удалось открыть профиль.'
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <section class="onboarding-page">
    <RouterLink to="/catalog" class="back-link"
      ><AppIcon name="ArrowLeft" :size="16" />Вернуться в каталог</RouterLink
    >
    <template v-if="store.profile">
      <span class="eyebrow">МОЙ ПРОФИЛЬ</span>
      <h1>{{ store.profile.name }}</h1>
      <div class="panel profile-summary">
        <span class="soft-icon blue"
          ><AppIcon :name="store.isBusiness ? 'Building2' : 'Users'"
        /></span>
        <div>
          <h2>{{ store.isBusiness ? 'Заказчик' : 'Исполнитель' }}</h2>
          <p>
            {{
              store.isBusiness
                ? 'Ваши задачи, предложения команд и результаты — в личном разделе.'
                : 'Ваши предложения и результаты работы — в разделе откликов.'
            }}
          </p>
        </div>
      </div>
      <p v-if="route.query.role && route.query.role !== store.profile.role" class="inline-alert">
        Для этого действия нужен профиль
        {{ route.query.role === 'business' ? 'заказчика' : 'исполнителя' }}. Сейчас вы работаете как
        {{ store.isBusiness ? 'заказчик' : 'исполнитель' }}.
      </p>
      <RouterLink :to="store.isBusiness ? '/business' : '/applications'" class="button primary"
        >{{ store.isBusiness ? 'Мои задачи' : 'Мои отклики' }}
        <AppIcon name="ArrowRight" :size="17"
      /></RouterLink>
      <div v-if="store.account" class="panel profile-form">
        <h2>Аккаунт и безопасность</h2>
        <p>{{ store.account.email }}</p>
        <p>
          Вход подтверждён сервером. Пароль и токен сессии не сохраняются в хранилище браузера
          приложением.
        </p>
        <p v-if="error" class="profile-error" role="alert">{{ error }}</p>
        <button class="button secondary" :disabled="busy" @click="leave">
          {{ busy ? 'Выходим…' : 'Выйти из аккаунта' }}
        </button>
      </div>
      <details v-else class="local-profile-note">
        <summary>О локальном профиле</summary>
        <p>
          Этот профиль доступен только в текущем браузере. Настоящий вход пока не подключён. При
          выходе задачи сохранятся: можно вернуться через «Начать работу», выбрав ту же роль.
        </p>
        <button class="button secondary" :disabled="busy" @click="leave">
          Выйти из локального профиля
        </button>
      </details>
    </template>
    <template v-else>
      <span class="eyebrow">ПЕРВЫЙ ШАГ</span>
      <h1>Что вы хотите сделать?</h1>
      <p class="guide-intro">
        Выберите свою сторону проекта. Профиль понадобится для задач и откликов — каталог открыт
        всем.
      </p>
      <div class="role-options" role="group" aria-label="Ваша роль">
        <button
          type="button"
          :class="{ selected: role === 'business' }"
          :aria-pressed="role === 'business'"
          @click="chooseRole('business')"
        >
          <AppIcon name="Building2" :size="26" /><b>Я заказчик</b
          ><span>Разместить задачу и найти команду</span
          ><AppIcon v-if="role === 'business'" name="CheckCircle2" class="role-check" />
        </button>
        <button
          type="button"
          :class="{ selected: role === 'team' }"
          :aria-pressed="role === 'team'"
          @click="chooseRole('team')"
        >
          <AppIcon name="Users" :size="26" /><b>Я исполнитель</b
          ><span>Найти проект и предложить решение</span
          ><AppIcon v-if="role === 'team'" name="CheckCircle2" class="role-check" />
        </button>
      </div>
      <div v-if="saved" class="panel profile-form">
        <h2>Продолжить как {{ saved.name }}?</h2>
        <p>В этом браузере уже есть ваш локальный профиль. Задачи и отклики сохранены.</p>
        <p v-if="error" class="profile-error" role="alert">{{ error }}</p>
        <button class="button primary" :disabled="busy" @click="resume">
          Продолжить работу <AppIcon name="ArrowRight" :size="17" />
        </button>
      </div>
      <form
        v-else-if="role && store.mode === 'demo'"
        class="panel profile-form"
        novalidate
        @submit.prevent="submit"
      >
        <h2>
          {{
            role === 'business' ? 'Как представить вас командам?' : 'Как называется ваша команда?'
          }}
        </h2>
        <p>
          Название будет видно
          {{ role === 'business' ? 'рядом с вашими задачами' : 'в ваших откликах' }}.
        </p>
        <label for="profile-name" class="field-label">{{
          role === 'business' ? 'Имя или название организации' : 'Название команды'
        }}</label>
        <input
          id="profile-name"
          ref="nameInput"
          v-model="name"
          class="profile-name"
          maxlength="80"
          :disabled="busy"
          :aria-invalid="!!error"
          :aria-describedby="error ? 'profile-error' : 'profile-note'"
          :placeholder="role === 'business' ? 'Например, магазин Arman' : 'Например, Digital Team'"
        />
        <p v-if="error" id="profile-error" class="profile-error" role="alert">{{ error }}</p>
        <p id="profile-note" class="local-profile-note">
          Пробная версия: создаём локальный профиль в этом браузере. Email и пароль пока не нужны.
        </p>
        <button class="button primary" :disabled="busy" :aria-busy="busy">
          <AppIcon
            :name="busy ? 'LoaderCircle' : 'ArrowRight'"
            :class="{ spin: busy }"
            :size="17"
          />{{ busy ? 'Создаём профиль…' : 'Создать профиль и продолжить' }}
        </button>
      </form>
      <div v-else-if="role" class="panel profile-form">
        <h2>Регистрация скоро появится</h2>
        <p>Сервис входа ещё не подключён. Пока вы можете смотреть задачи в открытом каталоге.</p>
        <RouterLink to="/catalog" class="button primary">Смотреть задачи</RouterLink>
      </div>
    </template>
  </section>
</template>
