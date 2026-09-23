<script setup lang="ts">
import { computed, ref } from 'vue'
import { useWorkspace } from '../stores/workspace'
import { industries, readinessLabels } from '../domain/scoring'
import AppIcon from '../components/AppIcon.vue'
import TaskCard from '../components/TaskCard.vue'
const store = useWorkspace()
const search = ref('')
const industry = ref('')
const readiness = ref('')
const readyOnly = ref(false)
const sort = ref('score')
const readyCount = computed(
  () => store.catalog.filter((task) => (task.confirmedSnapshot?.total ?? 0) >= 70).length,
)
const filtered = computed(() =>
  store.catalog
    .filter((task) => {
      const text = [
        task.title,
        task.confirmedSnapshot?.fields.need.value,
        task.confirmedSnapshot?.fields.context.value,
      ]
        .join(' ')
        .toLowerCase()
      return (
        text.includes(search.value.toLowerCase().trim()) &&
        (!industry.value || task.industry === industry.value) &&
        (!readiness.value || task.confirmedSnapshot?.readiness === readiness.value) &&
        (!readyOnly.value || (task.confirmedSnapshot?.total ?? 0) >= 70)
      )
    })
    .sort((a, b) =>
      sort.value === 'recent'
        ? (b.publishedAt || '').localeCompare(a.publishedAt || '')
        : (b.confirmedSnapshot?.total ?? 0) - (a.confirmedSnapshot?.total ?? 0) ||
          (a.publishedAt || '').localeCompare(b.publishedAt || '') ||
          a.id.localeCompare(b.id),
    ),
)
const hasFilters = computed(
  () => !!(search.value || industry.value || readiness.value || readyOnly.value),
)
function reset() {
  search.value = ''
  industry.value = ''
  readiness.value = ''
  readyOnly.value = false
}
</script>
<template>
  <section class="catalog-hero">
    <div class="hero-copy">
      <span class="eyebrow"><span class="eyebrow-dot" /> БИЗНЕС × СТУДЕНТЫ</span>
      <h1>Хорошие задачи.<br /><span>Настоящий результат.</span></h1>
      <p>
        Превращайте идеи в понятные задачи.<br class="desktop-only" />
        Объединяйтесь с командами, чтобы воплотить их в жизнь.
      </p>
      <div class="hero-actions">
        <RouterLink v-if="store.isBusiness" to="/tasks/new" class="button primary"
          ><AppIcon name="Plus" :size="18" /> Создать задачу</RouterLink
        ><a v-else href="#catalog" class="button primary"
          >Найти свою задачу <AppIcon name="ArrowRight" :size="18" /></a
        ><span class="hero-note"
          ><AppIcon name="Sparkles" :size="16" /> AI поможет с первым шагом</span
        >
      </div>
    </div>
    <div class="hero-art" aria-hidden="true">
      <div class="art-grid" />
      <div class="art-orbit orbit-one" />
      <div class="art-orbit orbit-two" />
      <div class="floating-note note-idea">
        <span class="art-icon"><AppIcon name="Lightbulb" :size="20" /></span>
        <div><small>ВСЁ НАЧИНАЕТСЯ С ИДЕИ</small><b>«А что, если…»</b></div>
      </div>
      <div class="art-connector"><span /><AppIcon name="Sparkles" :size="19" /><span /></div>
      <div class="floating-task">
        <div class="floating-task-head">
          <span class="art-icon blue"><AppIcon name="ClipboardList" :size="20" /></span
          ><span class="floating-ready"><span />Готова к работе</span>
        </div>
        <b>Из идеи — в понятную задачу</b>
        <div class="art-line" />
        <div class="art-line short" />
        <div class="floating-score">
          <span>Проверено бизнесом</span><strong>95<span>/100</span></strong>
        </div>
        <div class="art-score-bar"><span /></div>
      </div>
      <div class="floating-team">
        <div class="avatar-stack"><span>А</span><span>М</span><span>Д</span></div>
        <div><b>Ваш следующий шаг</b><small>Начать работу вместе</small></div>
        <span class="team-check"><AppIcon name="Check" :size="14" /></span>
      </div>
      <span class="art-spark spark-a">✳</span><span class="art-spark spark-b">+</span>
    </div>
  </section>
  <div class="stat-strip">
    <div>
      <span class="stat-icon blue"><AppIcon name="ClipboardList" /></span
      ><strong>{{ store.loaded ? store.catalog.length : '—' }}</strong
      ><span>открытых задач</span>
    </div>
    <div>
      <span class="stat-icon green"><AppIcon name="CheckCheck" /></span
      ><strong>{{ store.loaded ? readyCount : '—' }}</strong
      ><span>готовы к работе</span>
    </div>
    <div>
      <span class="stat-icon purple"><AppIcon name="Users" /></span
      ><strong>{{ store.loaded ? store.teams.length : '—' }}</strong
      ><span>студенческих команд</span>
    </div>
    <p><AppIcon name="Globe" :size="16" /> Открытый выбор.<br />Возможности для каждого.</p>
  </div>
  <section id="catalog" class="catalog-section">
    <div class="section-heading">
      <div>
        <h2>
          Открытый каталог <span class="count-pill">{{ store.catalog.length }}</span>
        </h2>
        <p>Найдите задачу, в которой ваши навыки станут результатом.</p>
      </div>
      <span class="quiet-label"><span class="green-dot" /> Все задачи доступны командам</span>
    </div>
    <div class="catalog-tabs" role="group" aria-label="Быстрый фильтр готовности">
      <button
        type="button"
        :class="{ active: !readyOnly }"
        :aria-pressed="!readyOnly"
        @click="readyOnly = false"
      >
        Все задачи <span>{{ store.catalog.length }}</span></button
      ><button
        type="button"
        :class="{ active: readyOnly }"
        :aria-pressed="readyOnly"
        @click="readyOnly = true"
      >
        <AppIcon name="Zap" :size="15" /> Готовы к работе <span>{{ readyCount }}</span></button
      ><span class="tabs-hint">Выше готовность — проще начать</span>
    </div>
    <div class="catalog-filters">
      <label class="search-field"
        ><AppIcon name="Search" :size="18" /><input
          v-model="search"
          type="search"
          placeholder="Поиск по задачам и ключевым словам"
          aria-label="Поиск задач" /></label
      ><label class="select-field"
        ><span class="sr-only">Тема задачи</span
        ><select v-model="industry">
          <option value="">Все направления</option>
          <option v-for="item in industries" :key="item.value" :value="item.value">
            {{ item.short }}
          </option>
        </select></label
      ><label class="select-field"
        ><span class="sr-only">Уровень готовности</span
        ><select v-model="readiness">
          <option value="">Любая готовность</option>
          <option v-for="(label, value) in readinessLabels" :key="value" :value="value">
            {{ label }}
          </option>
        </select></label
      >
    </div>
    <div class="results-toolbar">
      <span
        >{{ filtered.length }}
        {{
          filtered.length === 1
            ? 'задача'
            : filtered.length >= 2 && filtered.length <= 4
              ? 'задачи'
              : 'задач'
        }}<button v-if="hasFilters" class="text-link reset-filter" @click="reset">
          Сбросить фильтры <AppIcon name="X" :size="12" /></button></span
      ><label
        ><AppIcon name="SlidersHorizontal" :size="14" /><span class="sr-only">Порядок задач</span
        ><select v-model="sort">
          <option value="score">Сначала готовые</option>
          <option value="recent">Сначала новые</option>
        </select></label
      >
    </div>
    <div v-if="store.error" class="empty-state error-state">
      <AppIcon name="CircleAlert" :size="30" />
      <h3>Не удалось загрузить каталог</h3>
      <p>{{ store.error }}</p>
      <button class="button secondary" @click="store.refresh">Попробовать ещё раз</button>
    </div>
    <div v-else-if="store.loading && !store.loaded" class="task-grid">
      <div v-for="n in 6" :key="n" class="skeleton-card">
        <div class="skeleton-icon skeleton" />
        <div class="skeleton" />
        <div class="skeleton" />
        <div class="skeleton short" />
      </div>
    </div>
    <div v-else-if="filtered.length" class="task-grid">
      <TaskCard v-for="(task, index) in filtered" :key="task.id" :task="task" :index="index" />
    </div>
    <div v-else class="empty-state">
      <span class="empty-icon"><AppIcon name="Search" :size="28" /></span>
      <h3>{{ hasFilters ? 'Пока нет совпадений' : 'Здесь начнётся совместная работа' }}</h3>
      <p>
        {{
          hasFilters
            ? 'Попробуйте другое слово или расширьте фильтры.'
            : 'Опубликуйте первую задачу — студенты смогут предложить решение.'
        }}
      </p>
      <button v-if="hasFilters" class="button secondary" @click="reset">Показать все задачи</button
      ><RouterLink v-else-if="store.isBusiness" to="/tasks/new" class="button primary"
        >Создать задачу</RouterLink
      >
    </div>
    <div class="catalog-bottom-note">
      <AppIcon name="ShieldCheck" :size="18" />
      <p>
        <b>Готовность — ориентир, а выбор за вами.</b> Можно откликнуться на любую задачу, даже если
        её детали ещё уточняются.
      </p>
    </div>
  </section>
</template>
