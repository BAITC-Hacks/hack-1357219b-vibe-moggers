import { createApp } from 'vue'
import { createPinia } from 'pinia'
import '@fontsource-variable/manrope'
import './styles.css'
import App from './App.vue'
import { router } from './router'
import { autoGrow } from './composables/formControls'

createApp(App).directive('autogrow', autoGrow).use(createPinia()).use(router).mount('#app')
