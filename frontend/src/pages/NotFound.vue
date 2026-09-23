<script setup lang="ts">
import AppIcon from '../components/AppIcon.vue'
</script>
<template>
  <div class="empty-state not-found">
    <span class="eyebrow">404 · НЕ ТОТ ПОВОРОТ</span>
    <h1>Следующий шаг —<br />вернуться к задачам.</h1>
    <p>Эта страница не найдена. В каталоге всё на месте.</p>
    <RouterLink to="/catalog" class="button primary"
      >В каталог <AppIcon name="ArrowRight" :size="17"
    /></RouterLink>
  </div>
</template>
