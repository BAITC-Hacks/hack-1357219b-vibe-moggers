<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { authenticate } from '../services/session'
import { useWorkspace } from '../stores/workspace'
import { validateForm } from '../composables/formControls'
const route = useRoute(),
  router = useRouter(),
  store = useWorkspace()
const register = computed(() => route.path === '/register')
const role = ref<'business' | 'team'>(route.query.role === 'team' ? 'team' : 'business')
const email = ref(''),
  password = ref(''),
  name = ref(''),
  error = ref(''),
  busy = ref(false),
  visible = ref(false)
async function submit(event: Event) {
  if (busy.value) return
  error.value = validateForm(event, 'auth-error')
  if (error.value) return
  busy.value = true
  try {
    await authenticate(register.value ? 'register' : 'login', {
      email: email.value.trim(),
      password: password.value,
      ...(register.value ? { name: name.value.trim(), role: role.value } : {}),
    })
    password.value = ''
    const next = typeof route.query.next === 'string' ? route.query.next : ''
    await router.replace(
      /^\/(catalog|applications|business|tasks\/[^/?#]+)(?:#proposal)?$/.test(next)
        ? next
        : '/profile',
    )
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Не удалось войти.'
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <section class="onboarding-page auth-page">
    <RouterLink :to="{ path: '/welcome', query: route.query }" class="back-link"
      >← Выбор роли</RouterLink
    >
    <span class="eyebrow">ВАШ АККАУНТ QADAM</span>
    <h1>{{ register ? 'Начните свой первый проект' : 'С возвращением' }}</h1>
    <p class="guide-intro">
      {{
        register
          ? 'Сохраняйте задачи и отклики в одном аккаунте на разных устройствах.'
          : 'Войдите, чтобы вернуться к задачам и командам.'
      }}
    </p>
    <form class="panel profile-form" novalidate @submit.prevent="submit">
      <template v-if="register">
        <label class="field-label"
          >Ваша роль<select v-model="role" :disabled="busy">
            <option value="business">Заказчик</option>
            <option value="team">Исполнитель</option>
          </select></label
        >
        <label class="field-label"
          >{{ role === 'business' ? 'Имя или организация' : 'Название команды'
          }}<input
            v-model="name"
            required
            minlength="2"
            maxlength="80"
            autocomplete="organization"
            :disabled="busy"
        /></label>
      </template>
      <label class="field-label"
        >Email<input
          v-model="email"
          type="email"
          required
          maxlength="254"
          autocomplete="username"
          :disabled="busy"
      /></label>
      <label class="field-label"
        >Пароль<input
          v-model="password"
          :type="visible ? 'text' : 'password'"
          required
          :minlength="register ? 12 : 1"
          maxlength="128"
          :autocomplete="register ? 'new-password' : 'current-password'"
          :disabled="busy"
          aria-describedby="password-note"
      /></label>
      <button class="text-link" type="button" :aria-pressed="visible" @click="visible = !visible">
        {{ visible ? 'Скрыть пароль' : 'Показать пароль' }}
      </button>
      <p id="password-note" class="muted small">
        {{
          register
            ? 'От 12 символов. Можно использовать длинную фразу с пробелами.'
            : 'Введите пароль от своего аккаунта.'
        }}
      </p>
      <p v-if="error" id="auth-error" class="profile-error" role="alert">{{ error }}</p>
      <button class="button primary" :disabled="busy" :aria-busy="busy">
        {{ busy ? 'Подождите…' : register ? 'Создать аккаунт' : 'Войти' }}
      </button>
      <RouterLink
        :to="{ path: register ? '/login' : '/register', query: route.query }"
        class="text-link"
        >{{ register ? 'Уже есть аккаунт? Войти' : 'Нет аккаунта? Зарегистрироваться' }}</RouterLink
      >
    </form>
    <p v-if="store.mode === 'demo'" class="local-profile-note">
      Хотите проверить возможности до подключения сервиса входа?
      <RouterLink :to="{ path: '/demo-profile', query: { ...route.query, role } }"
        >Открыть локальное демо</RouterLink
      >. Это отдельный пробный профиль без регистрации.
    </p>
  </section>
</template>
