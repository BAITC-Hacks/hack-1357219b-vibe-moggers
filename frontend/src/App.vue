<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspace } from './stores/workspace'
import AiAssistant from './components/AiAssistant.vue'
import AppIcon from './components/AppIcon.vue'
import RoleSwitcher from './components/RoleSwitcher.vue'
const store = useWorkspace()
const route = useRoute()
onMounted(() => store.refresh())
</script>
<template>
  <a class="skip-link" href="#main">Перейти к содержимому</a>
  <div class="app-shell">
    <aside class="sidebar">
      <RouterLink to="/catalog" class="brand" aria-label="Qadam — главная">
        <span class="brand-mark"><AppIcon name="ArrowUpRight" :size="27" /></span>qadam<span
          class="brand-dot"
          >.</span
        >
      </RouterLink>
      <p class="sidebar-caption">Задачи бизнеса.<br />Решения от команд.</p>
      <div class="nav-label">ПЛАТФОРМА</div>
      <nav aria-label="Главная навигация">
        <RouterLink to="/catalog" class="nav-item"
          ><AppIcon name="LayoutGrid" :size="19" />Каталог задач</RouterLink
        >
        <RouterLink v-if="store.isBusiness" to="/business" class="nav-item"
          ><AppIcon name="FolderOpen" :size="19" />Мои задачи</RouterLink
        >
        <RouterLink v-if="store.isTeam" to="/applications" class="nav-item"
          ><AppIcon name="MessageSquare" :size="19" />Мои отклики</RouterLink
        >
        <RouterLink v-if="!store.isTeam" to="/tasks/new" class="nav-item"
          ><AppIcon name="Plus" :size="19" />Разместить задачу</RouterLink
        >
        <RouterLink to="/how-it-works" class="nav-item"
          ><AppIcon name="CircleHelp" :size="19" />Как это работает</RouterLink
        >
      </nav>
      <AiAssistant />
      <div class="sidebar-bottom">
        <span class="sidebar-caption">От задачи — к совместной работе.</span>
      </div>
    </aside>
    <div class="app-content">
      <header class="topbar">
        <span class="page-location">{{ route.meta.title }}</span>
        <div class="topbar-actions">
          <RoleSwitcher />
          <RouterLink to="/start" class="text-link">Сменить роль</RouterLink>
        </div>
      </header>
      <div v-if="store.mode === 'demo'" class="demo-strip">
        <AppIcon name="Globe" :size="14" /><span
          >Демо-режим: выбранная роль и изменения сохраняются только в этом браузере.</span
        >
      </div>
      <main id="main" class="main-content" tabindex="-1">
        <RouterView v-slot="{ Component }"
          ><component :is="Component" :key="route.path"
        /></RouterView>
      </main>
      <footer class="main-footer">
        <span>Qadam · Платформа задач и команд</span>
        <RouterLink to="/how-it-works">Как начать работу</RouterLink>
      </footer>
    </div>
  </div>
  <div class="toast-stack" aria-live="polite" aria-atomic="false">
    <div v-for="toast in store.toasts" :key="toast.id" class="toast" :class="toast.type">
      <AppIcon
        :name="
          toast.type === 'error'
            ? 'CircleAlert'
            : toast.type === 'success'
              ? 'CheckCircle2'
              : 'CircleHelp'
        "
        :size="19"
      />
      <span>{{ toast.message }}</span
      ><button type="button" aria-label="Закрыть уведомление" @click="store.dismiss(toast.id)">
        <AppIcon name="X" :size="16" />
      </button>
    </div>
  </div>
</template>
