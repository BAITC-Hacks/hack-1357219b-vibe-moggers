<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWorkspace } from './stores/workspace'
import AppIcon from './components/AppIcon.vue'
import { groups, fieldMeta } from './domain/scoring'
const store = useWorkspace()
const route = useRoute()
const router = useRouter()
const helpDialog = ref<HTMLDialogElement | null>(null)
async function changeActor(event: Event) {
  const select = event.target as HTMLSelectElement
  const next = select.value
  const failure = await router.push('/catalog')
  if (failure && route.path !== '/catalog') {
    select.value = store.actorId
    return
  }
  await store.setActor(next)
}
onMounted(() => store.refresh())
</script>
<template>
  <a class="skip-link" href="#main">Перейти к содержимому</a>
  <div class="app-shell">
    <aside class="sidebar">
      <RouterLink to="/catalog" class="brand" aria-label="Qadam — главная"
        ><span class="brand-mark"><AppIcon name="ArrowUpRight" :size="27" /></span>qadam<span
          class="brand-dot"
          >.</span
        ></RouterLink
      >
      <div class="workspace-tag">
        <span class="workspace-emblem"><AppIcon name="Building2" :size="17" /></span>
        <div><b>AI Sana</b><span>Рабочее пространство</span></div>
        <span class="workspace-dot" />
      </div>
      <div class="nav-label">ПЛАТФОРМА</div>
      <nav aria-label="Главная навигация">
        <RouterLink to="/catalog" class="nav-item"
          ><AppIcon name="LayoutGrid" :size="19" />Каталог задач<span
            class="nav-count"
            v-if="store.loaded"
            >{{ store.catalog.length }}</span
          ></RouterLink
        >
        <RouterLink v-if="store.isBusiness" to="/business" class="nav-item"
          ><AppIcon name="FolderOpen" :size="19" />Мои задачи</RouterLink
        >
        <RouterLink v-else to="/applications" class="nav-item"
          ><AppIcon name="MessageSquare" :size="19" />Мои отклики</RouterLink
        >
        <RouterLink v-if="store.isBusiness" to="/tasks/new" class="nav-item"
          ><AppIcon name="Plus" :size="19" />Создать задачу</RouterLink
        >
      </nav>
      <div class="sidebar-guide">
        <span class="guide-icon"><AppIcon name="Sparkles" :size="20" /></span
        ><b>От идеи к первому<br />результату</b>
        <p>Уточните задачу. Найдите команду. Сделайте первый шаг.</p>
        <button type="button" @click="helpDialog?.showModal()">
          Как это работает <AppIcon name="ArrowUpRight" :size="15" />
        </button>
      </div>
      <div class="sidebar-bottom">
        <button class="nav-item help-link" type="button" @click="helpDialog?.showModal()">
          <AppIcon name="CircleHelp" :size="19" />О платформе
        </button>
        <div class="event-signature">
          <span class="tiny-star">✳</span>
          <div>Создано для <b>HackAlem AI</b></div>
          <span>2026</span>
        </div>
      </div>
    </aside>
    <div class="app-content">
      <header class="topbar">
        <div class="breadcrumbs">
          <span>Рабочее пространство</span><AppIcon name="ChevronRight" :size="13" /><b>{{
            route.meta.title
          }}</b>
        </div>
        <div class="topbar-actions">
          <span class="mode-pill" :class="store.mode"
            ><span />{{ store.mode === 'demo' ? 'Демо' : store.loaded ? 'Go API' : 'API' }}</span
          >
          <div class="actor-control">
            <span class="actor-avatar"
              ><AppIcon :name="store.isBusiness ? 'Building2' : 'GraduationCap'" :size="18"
            /></span>
            <div>
              <label for="actor-select">{{
                store.isBusiness ? 'Режим бизнеса' : 'Режим команды'
              }}</label
              ><select id="actor-select" :value="store.actorId" @change="changeActor">
                <option value="demo-business-1">Бизнес AI Sana</option>
                <option v-for="team in store.teams" :key="team.id" :value="team.id">
                  {{ team.name }}
                </option>
              </select>
            </div>
          </div>
        </div>
      </header>
      <div v-if="store.mode === 'demo'" class="demo-strip">
        <AppIcon name="Globe" :size="13" /><span
          >Демонстрационное пространство · синтетические данные сохраняются в этом браузере</span
        ><span class="demo-strip-end">Все действия можно попробовать</span>
      </div>
      <main id="main" class="main-content">
        <RouterView v-slot="{ Component }"
          ><component :is="Component" :key="route.path"
        /></RouterView>
      </main>
      <footer class="main-footer">
        <span>Qadam · Каждый результат начинается с понятной задачи.</span
        ><span>Бизнес и студенты — на одной стороне.</span>
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
      /><span>{{ toast.message }}</span
      ><button type="button" aria-label="Закрыть уведомление" @click="store.dismiss(toast.id)">
        <AppIcon name="X" :size="16" />
      </button>
    </div>
  </div>
  <dialog
    ref="helpDialog"
    class="help-dialog"
    aria-labelledby="help-title"
    @click="
      (event) => {
        if (event.target === helpDialog) helpDialog?.close()
      }
    "
  >
    <button
      class="dialog-close icon-button"
      type="button"
      aria-label="Закрыть справку"
      @click="helpDialog?.close()"
    >
      <AppIcon name="X" />
    </button>
    <span class="eyebrow"><AppIcon name="Sparkles" :size="15" /> ПОНЯТНАЯ РАБОТА ВМЕСТЕ</span>
    <h2 id="help-title">Один шаг навстречу<br />большому результату.</h2>
    <p>
      Бизнес уточняет задачу с помощью AI и подтверждает сведения. Студенты выбирают задачу и
      предлагают решение. Бизнес принимает решение о сотрудничестве.
    </p>
    <h3>Как начисляется рейтинг</h3>
    <div class="help-score-row" v-for="group in groups" :key="group.key">
      <span>{{ group.label }}</span
      ><b>{{ group.fields.reduce((sum, key) => sum + fieldMeta[key].weight, 0) }} баллов</b>
    </div>
    <p class="muted small">
      Низкая готовность не скрывает задачу и не запрещает отклик. AI помогает с описанием, а команду
      всегда выбирает бизнес. В демо используются подготовленные вопросы.
    </p>
    <button type="button" class="button primary full-width" @click="helpDialog?.close()">
      Всё понятно <AppIcon name="Check" :size="17" />
    </button>
  </dialog>
</template>
