<script setup lang="ts">
import AppIcon from '../components/AppIcon.vue'
import { groups, fieldMeta } from '../domain/scoring'
const paths = [
  {
    title: 'Мне нужно решить задачу',
    icon: 'Building2' as const,
    role: 'business',
    action: 'Разместить задачу',
    steps: [
      [
        'Опишите, что нужно сделать',
        'Например: «Хочу собирать заявки клиентов в одной таблице». Не нужно заранее писать техническое задание.',
      ],
      [
        'Уточните детали и опубликуйте',
        'Ответьте на вопросы о данных, сроках и результате. Проверьте карточку — в каталог попадёт только то, что вы подтвердите.',
      ],
      [
        'Выберите команду',
        'Сравните предложения и договоритесь о первом результате. Вы сами решаете, с кем работать.',
      ],
    ],
  },
  {
    title: 'Я хочу работать над проектом',
    icon: 'Users' as const,
    role: 'team',
    action: 'Найти задачу',
    steps: [
      [
        'Найдите подходящую задачу',
        'Изучите описание, доступные данные и сроки. Смотреть каталог можно без профиля.',
      ],
      [
        'Предложите решение',
        'Создайте профиль команды, опишите подход, план и срок. Добавьте ссылку на прототип.',
      ],
      [
        'Передайте первый результат',
        'Дождитесь выбора заказчика, выполните этап и отправьте результат на проверку. Подтверждённый этап добавит баллы команде.',
      ],
    ],
  },
]
</script>
<template>
  <section class="guide-page">
    <span class="eyebrow">ЗНАКОМСТВО С QADAM</span>
    <h1>
      У вас задача. У команды — навыки.<br /><span class="text-blue"
        >Здесь вы находите друг друга.</span
      >
    </h1>
    <p class="guide-intro">
      Qadam помогает заказчикам подготовить понятное описание работы, а студенческим командам —
      предложить решение и получить опыт на проекте.
    </p>
    <div class="journey-grid">
      <article v-for="path in paths" :key="path.role" class="journey panel">
        <span class="soft-icon blue"><AppIcon :name="path.icon" /></span>
        <h2>{{ path.title }}</h2>
        <ol>
          <li v-for="(step, i) in path.steps" :key="i">
            <span class="journey-number">{{ i + 1 }}</span>
            <div>
              <h3>{{ step[0] }}</h3>
              <p>{{ step[1] }}</p>
            </div>
          </li>
        </ol>
        <RouterLink
          :to="path.role === 'business' ? '/tasks/new' : '/catalog'"
          class="button"
          :class="path.role === 'business' ? 'primary' : 'secondary'"
          >{{ path.action }} <AppIcon name="ArrowRight" :size="17"
        /></RouterLink>
      </article>
    </div>
    <section class="readiness-explainer panel">
      <div>
        <span class="eyebrow">ЧТО ОЗНАЧАЮТ БАЛЛЫ</span>
        <h2>90 из 100 — насколько подробно описана задача</h2>
        <p>
          Это не оценка компании или исполнителя. Чем больше заказчик уточнил и подтвердил, тем
          проще команде оценить работу. Откликнуться можно при любом балле.
        </p>
      </div>
      <details>
        <summary>Из чего складывается оценка</summary>
        <div class="help-score-row" v-for="group in groups" :key="group.key">
          <span>{{ group.label }}</span
          ><b>{{ group.fields.reduce((sum, key) => sum + fieldMeta[key].weight, 0) }} баллов</b>
        </div>
      </details>
    </section>
  </section>
</template>
