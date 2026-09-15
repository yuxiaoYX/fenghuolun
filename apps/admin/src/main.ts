import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import 'antdv-next/dist/reset.css'
import './style.css'

createApp(App).use(router).mount('#app')
