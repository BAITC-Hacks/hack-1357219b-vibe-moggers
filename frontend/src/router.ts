import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/catalog' },
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
  scrollBehavior: (_to, _from, saved) => saved || { top: 0 },
})
router.afterEach((to) => {
  document.title = `${String(to.meta.title || 'Рабочее пространство')} — Qadam`
})
