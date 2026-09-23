<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
const route = useRoute(),
  router = useRouter()
const role = ref<'business' | 'team'>(route.query.role === 'team' ? 'team' : 'business')
function choose(value: 'business' | 'team') {
  role.value = value
}
function proceed(path: string) {
  try {
    sessionStorage.setItem('qadam:intent', role.value)
  } catch {
    /* Route query also carries intent. */
  }
  router.push({
    path,
    query: {
      role: role.value,
      next: typeof route.query.next === 'string' ? route.query.next : '/catalog',
    },
  })
}
</script>
<template>
  <section class="onboarding-page">
    <span class="eyebrow">ДОБРО ПОЖАЛОВАТЬ В QADAM</span>
    <h1>С чего начнём?</h1>
    <p class="guide-intro">
      Здесь бизнес находит команду, а команды — реальные проекты. Выберите, что вам нужно.
    </p>
    <div class="role-options" role="group" aria-label="Ваша роль">
      <button
        :aria-pressed="role === 'business'"
        :class="{ selected: role === 'business' }"
        @click="choose('business')"
      >
        <AppIcon name="Building2" :size="28" /><b>Я заказчик</b
        ><span>Описать задачу, уточнить её с AI и найти исполнителей.</span>
      </button>
      <button
        :aria-pressed="role === 'team'"
        :class="{ selected: role === 'team' }"
        @click="choose('team')"
      >
        <AppIcon name="Users" :size="28" /><b>Я исполнитель</b
        ><span>Выбрать проект, предложить решение и показать результат.</span>
      </button>
    </div>
    <div class="auth-actions">
      <button class="button primary" @click="proceed('/register')">
        Создать аккаунт <AppIcon name="ArrowRight" :size="18" /></button
      ><button class="button secondary" @click="proceed('/login')">У меня есть аккаунт</button
      ><button class="button ghost" @click="proceed('/catalog')">Сначала посмотреть задачи</button>
    </div>
    <p class="muted">
      Выбор помогает настроить первый шаг. Публикация задач и отклики доступны после входа.
    </p>
  </section>
</template>
