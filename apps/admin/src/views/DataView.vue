<script setup lang="ts">
import { App, Alert, Select, Space, Table } from 'antdv-next'
import { computed, onMounted, ref, watch } from 'vue'
import { api, type BindingRow, type TableMeta } from '../api'

const { message } = App.useApp()
const tables = ref<TableMeta[]>([])
const bindings = ref<BindingRow[]>([])
const name = ref('binding')
const bindingId = ref<string>()
const loading = ref(false)
const items = ref<Record<string, unknown>[]>([])
const total = ref(0)
const note = ref('')
const page = ref(1)

const needsBinding = computed(() => ['snapshot', 'energy', 'sync_job', 'owner_session', 'fill_event'].includes(name.value))

const columns = computed(() => {
  const first = items.value[0]
  if (!first) return []
  return Object.keys(first).map((k) => ({
    title: k,
    dataIndex: k,
    ellipsis: true,
    customRender: ({ text }: { text: unknown }) => {
      if (text === null || text === undefined || text === '') return '—'
      if (typeof text === 'object') return JSON.stringify(text)
      return String(text)
    },
  }))
})

async function loadMeta() {
  try {
    const data = await api.tables()
    tables.value = data.items ?? []
    if (!tables.value.some((t) => t.name === name.value) && tables.value[0]) {
      name.value = tables.value[0].name
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉表清单失败')
  }
  try {
    bindings.value = (await api.bindings()).items ?? []
  } catch {
    // ignore
  }
}

async function load() {
  loading.value = true
  try {
    const data = await api.tableRows(name.value, {
      bindingId: needsBinding.value ? bindingId.value : undefined,
      page: page.value,
      pageSize: 20,
    })
    items.value = data.items ?? []
    total.value = data.total
    note.value = data.note ?? ''
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉表数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadMeta()
  await load()
})

watch(name, () => {
  page.value = 1
  if (!needsBinding.value) bindingId.value = undefined
  load()
})
watch(bindingId, () => {
  page.value = 1
  load()
})
</script>

<template>
  <Space style="margin-bottom: 16px" wrap>
    <Select
      v-model:value="name"
      style="width: 220px"
      :options="tables.map((t) => ({ value: t.name, label: t.label + ' · ' + t.name }))"
    />
    <Select
      v-if="needsBinding"
      v-model:value="bindingId"
      allow-clear
      placeholder="全部车辆"
      style="width: 220px"
      :options="bindings.map((b) => ({ value: b.id, label: `${b.vinMasked} ${b.modelCode}` }))"
    />
  </Space>
  <Alert
    v-if="note"
    :message="note"
    type="info"
    show-icon
    style="margin-bottom: 12px"
  />
  <Table
    :columns="columns"
    :data-source="items"
    :loading="loading"
    :pagination="{ current: page, pageSize: 20, total, showSizeChanger: false }"
    :scroll="{ x: true }"
    @change="(p: { current?: number }) => { page = p.current ?? 1; load() }"
  />
  <p style="margin-top: 12px; opacity: 0.65">只读脱敏。时间为北京时间墙钟。无 SQL、无导出、无完整 VIN 与密文。</p>
</template>
