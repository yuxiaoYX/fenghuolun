import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from './api'
import LoginView from './views/LoginView.vue'
import AdminLayout from './layout/AdminLayout.vue'
import OverviewView from './views/OverviewView.vue'
import BindingsView from './views/BindingsView.vue'
import BindingDetailView from './views/BindingDetailView.vue'
import JobsView from './views/JobsView.vue'
import DataView from './views/DataView.vue'
import SettingsView from './views/SettingsView.vue'
import AccountView from './views/AccountView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView, meta: { public: true } },
    {
      path: '/',
      component: AdminLayout,
      children: [
        { path: '', component: OverviewView, meta: { title: '总览' } },
        { path: 'bindings', component: BindingsView, meta: { title: '绑定' } },
        { path: 'bindings/:id', component: BindingDetailView, meta: { title: '绑定详情' } },
        { path: 'jobs', component: JobsView, meta: { title: '任务' } },
        { path: 'data', component: DataView, meta: { title: '数据' } },
        { path: 'settings', component: SettingsView, meta: { title: '设置' } },
        { path: 'account', component: AccountView, meta: { title: '账号' } },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) {
    return true
  }
  if (!getToken()) {
    return '/login'
  }
  return true
})
