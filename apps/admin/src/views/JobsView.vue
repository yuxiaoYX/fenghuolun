<script setup lang="ts">
import { App, Select, Space, Table } from 'antdv-next'
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type BindingRow, type JobRow } from '../api'

const { message } = App.useApp()
const route = useRoute()
const router = useRouter()
const loading = ref(false)
const rows = ref<JobRow[]>([])
const bindings = ref<BindingRow[]>([])
const total = ref(0)
const page = ref(1)
const status = ref<string | undefined>(typeof route.query.status === 'string' ? route.query.status : undefined)
const bindingId = ref<string | undefined>(typeof route.query.bindingId === 'string' ? route.query.bindingId : undefined)

async function load() {
  loading.value = true
  try {
    const data = await api.jobs({ status: status.value, bindingId: bindingId.value, page: page.value, pageSize: 20 })
    rows.value = data.items ?? []
    total.value = data.total
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉任务失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    bindings.value = (await api.bindings()).items ?? []
  } catch {
    // 筛选没有绑定列表也能看任务
  }
  await load()
})

watch([status, bindingId], () => {
  page.value = 1
  const q: Record<string, string> = {}
  if (status.value) q.status = status.value
  if (bindingId.value) q.bindingId = bindingId.value
  router.replace({ path: '/jobs', query: q })
  load()
})

const columns = [
  { title: 'VIN', dataIndex: 'vinMasked' },
  { title: '来源', dataIndex: 'kind' },
  { title: '状态', dataIndex: 'status' },
  { title: '开始', dataIndex: 'startedAt' },
  { title: '结束', dataIndex: 'finishedAt' },
  { title: '说明', dataIndex: 'errorPublic' },
  { title: '内部', dataIndex: 'errorInternal' },
]
</script>

<template>
  <Space style="margin-bottom: 16px" wrap>
    <Select
      v-model:value="bindingId"
      allow-clear
      placeholder="全部车辆"
      style="width: 220px"
      :options="bindings.map((b) => ({ value: b.id, label: `${b.vinMasked} ${b.modelCode}` }))"
    />
    <Select
      v-model:value="status"
      allow-clear
      placeholder="全部状态"
      style="width: 180px"
      :options="[
        { value: 'running', label: 'running' },
        { value: 'ok', label: 'ok' },
        { value: 'auth_failed', label: 'auth_failed' },
        { value: 'upstream', label: 'upstream' },
        { value: 'decode', label: 'decode' },
      ]"
    />
  </Space>
  <Table
    row-key="id"
    :columns="columns"
    :data-source="rows"
    :loading="loading"
    :pagination="{ current: page, pageSize: 20, total, showSizeChanger: false }"
    @change="(p: { current?: number }) => { page = p.current ?? 1; load() }"
  />
</template>
