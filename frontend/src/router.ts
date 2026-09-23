import { createRouter, createWebHistory } from 'vue-router'
import { restoreSession } from './services/session'
import { useWorkspace } from './stores/workspace'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/catalog' },
    {
      path: '/welcome',
      component: () => import('./pages/Welcome.vue'),
      meta: { title: 'Добро пожаловать' },
    },
    { path: '/login', component: () => import('./pages/Auth.vue'), meta: { title: 'Вход' } },
    {
      path: '/register',
      component: () => import('./pages/Auth.vue'),
      meta: { title: 'Регистрация' },
    },
    {
      path: '/demo-profile',
      component: () => import('./pages/Profile.vue'),
      meta: { title: 'Пробный профиль' },
    },
    {
      path: '/start',
      redirect: (to) => ({ path: '/register', query: to.query }),
      meta: { title: 'Начать работу' },
    },
    {
      path: '/profile',
      component: () => import('./pages/Profile.vue'),
      meta: { title: 'Мой профиль' },
    },
    {
      path: '/how-it-works',
      component: () => import('./pages/HowItWorks.vue'),
      meta: { title: 'Как работает Qadam' },
    },
    {
      path: '/catalog',
      component: () => import('./pages/Catalog.vue'),
      meta: { title: 'Каталог задач' },
    },
    {
      path: '/tasks/new',
      component: () => import('./pages/TaskEditor.vue'),
      meta: { title: 'Новая задача' },
    },
    {
      path: '/tasks/:id/edit',
      component: () => import('./pages/TaskEditor.vue'),
      meta: { title: 'Конструктор задачи' },
    },
    {
      path: '/tasks/:id',
      component: () => import('./pages/TaskDetails.vue'),
      meta: { title: 'Карточка задачи' },
    },
    {
      path: '/business',
      component: () => import('./pages/Workspace.vue'),
      meta: { title: 'Мои задачи' },
    },
    {
      path: '/applications',
      component: () => import('./pages/Workspace.vue'),
      meta: { title: 'Мои отклики' },
    },
    {
      path: '/business/tasks/:id',
      component: () => import('./pages/BusinessTask.vue'),
      meta: { title: 'Отклики и результаты' },
    },
    {
      path: '/:pathMatch(.*)*',
      component: () => import('./pages/NotFound.vue'),
      meta: { title: 'Страница не найдена' },
    },
  ],
  scrollBehavior: (to, _from, saved) => {
    if (saved) return saved
    const target = to.hash ? document.getElementById(to.hash.slice(1)) : null
    return target ? { el: target } : { top: 0 }
  },
})
router.beforeEach(async (to) => {
  await restoreSession()
  const store = useWorkspace()
  let intent = ''
  try {
    intent = sessionStorage.getItem('qadam:intent') || ''
  } catch {
    /* Optional preference. */
  }
  if (
    !store.profile &&
    !intent &&
    !['/welcome', '/login', '/register', '/demo-profile', '/how-it-works'].includes(to.path) &&
    !['business', 'team'].includes(String(to.query.role))
  )
    return { path: '/welcome', query: { next: to.fullPath } }
  if (to.path === '/profile' && !store.profile) return '/login'
  const business =
    to.path === '/tasks/new' || to.path.endsWith('/edit') || to.path.startsWith('/business')
  const team = to.path === '/applications'
  if ((business && !store.isBusiness) || (team && !store.isTeam)) {
    return {
      path: store.profile ? '/profile' : '/register',
      query: { role: business ? 'business' : 'team', next: to.fullPath },
    }
  }
})
router.afterEach((to, _from, failure) => {
  if (failure) return
  document.title = `${String(to.meta.title || 'Рабочее пространство')} — Qadam`
})
