<script setup lang="ts">
import {
  DatabaseOutlined,
  DashboardOutlined,
  SettingOutlined,
  SyncOutlined,
  UserOutlined,
  CarOutlined,
} from '@antdv-next/icons'
import { Layout, LayoutContent, LayoutHeader, LayoutSider, Menu, theme } from 'antdv-next'
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'

const { token } = theme.useToken()
const route = useRoute()
const router = useRouter()

const items = [
  { key: '/', icon: DashboardOutlined, label: '总览' },
  { key: '/bindings', icon: CarOutlined, label: '绑定' },
  { key: '/jobs', icon: SyncOutlined, label: '任务' },
  { key: '/data', icon: DatabaseOutlined, label: '数据' },
  { key: '/settings', icon: SettingOutlined, label: '设置' },
  { key: '/account', icon: UserOutlined, label: '账号' },
]

const selected = computed(() => {
  const p = route.path
  if (p.startsWith('/bindings')) return ['/bindings']
  if (items.some((it) => it.key === p)) return [p]
  return ['/']
})

const title = computed(() => (typeof route.meta.title === 'string' ? route.meta.title : '总览'))

function go(info: { key: string | number }) {
  const key = String(info.key)
  if (key !== route.path) {
    router.push(key)
  }
}
</script>

<template>
  <Layout class="admin-layout">
    <LayoutSider collapsible>
      <div class="admin-sider-title">风火轮</div>
      <Menu
        theme="dark"
        mode="inline"
        :selected-keys="selected"
        :items="items"
        @click="go"
      />
    </LayoutSider>
    <Layout>
      <LayoutHeader class="admin-header" :style="{ background: token.colorBgContainer }">
        <strong>{{ title }}</strong>
        <span style="opacity: 0.65">只读运维 · 不下发车控</span>
      </LayoutHeader>
      <LayoutContent class="admin-content">
        <div class="admin-content-inner" :style="{ background: token.colorBgContainer, borderRadius: token.borderRadiusLG + 'px' }">
          <RouterView />
        </div>
      </LayoutContent>
    </Layout>
  </Layout>
</template>
