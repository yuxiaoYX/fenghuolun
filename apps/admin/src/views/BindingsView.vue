<script setup lang="ts">
import { App, Button, Select, Space, Table, Tag } from 'antdv-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type BindingRow } from '../api'

const { message, modal } = App.useApp()
const router = useRouter()
const route = useRoute()
const rows = ref<BindingRow[]>([])
const loading = ref(false)
const busy = ref('')
const filter = ref((route.query.filter as string) || '')

async function load() {
  loading.value = true
  try {
    const data = await api.bindings()
    rows.value = data.items ?? []
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉绑定失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(
  () => route.query.filter,
  (v) => {
    filter.value = typeof v === 'string' ? v : ''
  },
)

const shown = computed(() => {
  switch (filter.value) {
    case 'disabled':
      return rows.value.filter((r) => r.disabled)
    case 'active':
      return rows.value.filter((r) => !r.disabled)
    case 'auth_failed':
      return rows.value.filter((r) => r.syncStatus === 'auth_failed')
    case 'fail':
      return rows.value.filter((r) => r.syncStatus && r.syncStatus !== 'ok' && r.syncStatus !== 'running')
    default:
      return rows.value
  }
})

function onFilter(v: unknown) {
  const next = typeof v === 'string' ? v : ''
  filter.value = next
  router.replace(next ? { path: '/bindings', query: { filter: next } } : { path: '/bindings' })
}

async function run(id: string, fn: (id: string) => Promise<unknown>, ok: string) {
  busy.value = id
  try {
    await fn(id)
    message.success(ok)
  } catch (e) {
    message.error(e instanceof Error ? e.message : '操作失败')
  } finally {
    busy.value = ''
    await load()
  }
}

function confirmDisable(id: string) {
  modal.confirm({
    title: '停用这辆车？',
    content: '只清本服务车主会话，不会向车辆下发命令。',
    onOk: () => run(id, api.disable, '已停用'),
  })
}

const columns = [
  { title: '车型', key: 'model', customRender: ({ record }: { record: BindingRow }) => `${record.modelName} ${record.modelCode}` },
  { title: 'VIN', dataIndex: 'vinMasked' },
  { title: '状态', key: 'status' },
  { title: '最近同步', dataIndex: 'syncedAt', customRender: ({ text }: { text: string | null }) => text || '—' },
  { title: '说明', dataIndex: 'syncError' },
  { title: '', key: 'actions' },
]
</script>

<template>
  <Space style="margin-bottom: 16px">
    <Select
      :value="filter || undefined"
      allow-clear
      placeholder="全部状态"
      style="width: 180px"
      :options="[
        { value: 'active', label: '启用' },
        { value: 'disabled', label: '已停用' },
        { value: 'auth_failed', label: '凭证失效' },
        { value: 'fail', label: '同步失败' },
      ]"
      @update:value="onFilter"
    />
  </Space>
  <Table
    row-key="id"
    :columns="columns"
    :data-source="shown"
    :loading="loading"
    :pagination="false"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'status'">
        <Tag v-if="record.disabled" color="default">已停用</Tag>
        <Tag v-else-if="record.syncStatus === 'ok'" color="success">ok</Tag>
        <Tag v-else-if="record.syncStatus === 'running'" color="processing">同步中</Tag>
        <Tag v-else-if="record.syncStatus === 'auth_failed'" color="error">{{ record.syncStatus }}</Tag>
        <Tag v-else-if="record.syncStatus">{{ record.syncStatus }}</Tag>
        <span v-else>—</span>
      </template>
      <template v-else-if="column.key === 'actions'">
        <Space>
          <Button size="small" @click="router.push('/bindings/' + record.id)">详情</Button>
          <Button size="small" :disabled="record.disabled || busy === record.id" @click="run(record.id, api.sync, '已同步')">
            立即同步
          </Button>
          <Button v-if="!record.disabled" size="small" danger :disabled="busy === record.id" @click="confirmDisable(record.id)">
            停用
          </Button>
          <Button v-else size="small" :disabled="busy === record.id" @click="run(record.id, api.enable, '已恢复')">
            恢复
          </Button>
        </Space>
      </template>
    </template>
  </Table>
  <p style="margin-top: 12px; opacity: 0.65">立即同步走和定时任务同一条只读路径，不会远程控车。失败也会刷新列表。</p>
</template>
