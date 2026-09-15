import { createRouter, createWebHistory } from 'vue-router'
import LoginView from './views/LoginView.vue'
import OverviewView from './views/OverviewView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView },
    { path: '/', component: OverviewView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})
