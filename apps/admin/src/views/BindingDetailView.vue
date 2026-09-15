<script setup lang="ts">
import { App, Button, Descriptions, DescriptionsItem, Space, Spin, Table, Tag } from 'antdv-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type BindingDetail } from '../api'

const { message, modal } = App.useApp()
const route = useRoute()
const router = useRouter()
const loading = ref(false)
const busy = ref(false)
const detail = ref<BindingDetail | null>(null)

const id = computed(() => String(route.params.id ?? ''))

function dash(v: unknown) {
  if (v === null || v === undefined || v === '') return '—'
  return String(v)
}

async function load() {
  if (!id.value) return
  loading.value = true
  try {
    detail.value = await api.binding(id.value)
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉详情失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(id, load)

async function run(fn: (id: string) => Promise<unknown>, ok: string) {
  busy.value = true
  try {
    await fn(id.value)
    message.success(ok)
  } catch (e) {
    message.error(e instanceof Error ? e.message : '操作失败')
  } finally {
    busy.value = false
    await load()
  }
}

function confirmDisable() {
  modal.confirm({
    title: '停用这辆车？',
    content: '只清本服务车主会话，不会向车辆下发命令。',
    onOk: () => run(api.disable, '已停用'),
  })
}

const jobColumns = [
  { title: '来源', dataIndex: 'kind' },
  { title: '状态', dataIndex: 'status' },
  { title: '开始', dataIndex: 'startedAt' },
  { title: '结束', dataIndex: 'finishedAt' },
  { title: '说明', dataIndex: 'errorPublic' },
  { title: '内部', dataIndex: 'errorInternal' },
]

const snap = computed(() => (detail.value?.snapshot ?? {}) as Record<string, unknown>)
const power = computed(() => (snap.value.power ?? {}) as Record<string, unknown>)
const extender = computed(() => (snap.value.extender ?? null) as Record<string, unknown> | null)
const tires = computed(() => (snap.value.tires ?? {}) as Record<string, Record<string, unknown>>)
const body = computed(() => (snap.value.body ?? {}) as Record<string, unknown>)
const doors = computed(() => (body.value.doors ?? {}) as Record<string, unknown>)
const windows = computed(() => (body.value.windows ?? {}) as Record<string, unknown>)
const climate = computed(() => (snap.value.climate ?? {}) as Record<string, unknown>)
const b12 = computed(() => (snap.value.battery12v ?? {}) as Record<string, unknown>)
const pack = computed(() => (snap.value.battery ?? {}) as Record<string, unknown>)

function tire(corner: string) {
  const t = tires.value[corner] ?? {}
  const bar = dash(t.bar)
  const temp = dash(t.tempC)
  return bar === '—' && temp === '—' ? '—' : `${bar} bar / ${temp} °C`
}

const energyColumns = [
  { title: '日', dataIndex: 'countTime' },
  { title: '总 kWh', dataIndex: 'totalKwh' },
  { title: '行驶', dataIndex: 'drivingKwh' },
  { title: '空调', dataIndex: 'acKwh' },
  { title: '回收', dataIndex: 'recoveryKwh' },
]
</script>

<template>
  <Spin :spinning="loading">
    <template v-if="detail">
      <Space style="margin-bottom: 16px" wrap>
        <Button @click="router.push('/bindings')">返回列表</Button>
        <Button :disabled="detail.disabled || busy" @click="run(api.sync, '已同步')">立即同步</Button>
        <Button :disabled="busy" @click="run(api.kick, '已踢会话')">踢车主会话</Button>
        <Button v-if="!detail.disabled" danger :disabled="busy" @click="confirmDisable">停用</Button>
        <Button v-else :disabled="busy" @click="run(api.enable, '已恢复')">恢复</Button>
      </Space>
      <Descriptions bordered :column="2" size="small">
        <DescriptionsItem label="VIN">{{ detail.vinMasked }}</DescriptionsItem>
        <DescriptionsItem label="车型">{{ detail.modelName }} {{ detail.modelCode }}</DescriptionsItem>
        <DescriptionsItem label="备注">{{ detail.nickname || '—' }}</DescriptionsItem>
        <DescriptionsItem label="配置">{{ detail.trim || '—' }}</DescriptionsItem>
        <DescriptionsItem label="增程">{{ detail.isExtender ? '是' : '否' }}</DescriptionsItem>
        <DescriptionsItem label="状态">
          <Tag v-if="detail.disabled">已停用</Tag>
          <Tag v-else-if="detail.stale" color="warning">陈旧</Tag>
          <Tag v-else :color="detail.syncStatus === 'ok' ? 'success' : 'warning'">{{ detail.syncStatus || '—' }}</Tag>
        </DescriptionsItem>
        <DescriptionsItem label="refresh">{{ detail.hasRefresh ? detail.refreshHint || '有' : '无' }}</DescriptionsItem>
        <DescriptionsItem label="access">{{ detail.hasAccess ? '有' : '无' }}</DescriptionsItem>
        <DescriptionsItem label="车主会话">{{ detail.sessionCount }}</DescriptionsItem>
        <DescriptionsItem label="最近同步">{{ detail.syncedAt || '—' }}</DescriptionsItem>
        <DescriptionsItem label="位置上报">{{ detail.locReportedAt || '—' }}</DescriptionsItem>
        <DescriptionsItem label="坐标">
          {{ detail.lng != null && detail.lat != null ? detail.lng + ', ' + detail.lat : '—' }}
        </DescriptionsItem>
      </Descriptions>

      <h3 style="margin-top: 24px">最近快照</h3>
      <Descriptions v-if="detail.snapshot" bordered :column="2" size="small">
        <DescriptionsItem label="拉取时刻">{{ dash(snap.fetchedAt) }}</DescriptionsItem>
        <DescriptionsItem label="车辆上报">{{ dash(snap.reportedAt) }}</DescriptionsItem>
        <DescriptionsItem label="电量 %">{{ dash(power.socPct) }}</DescriptionsItem>
        <DescriptionsItem label="纯电续航 km">{{ dash(power.evRangeKm) }}</DescriptionsItem>
        <DescriptionsItem label="综合续航 km">{{ dash(power.totalRangeKm) }}</DescriptionsItem>
        <DescriptionsItem label="充电">{{ dash(power.chargeStatus) }}</DescriptionsItem>
        <DescriptionsItem label="插枪">{{ dash(power.pluggedIn) }}</DescriptionsItem>
        <DescriptionsItem label="在线">{{ snap.online === null || snap.online === undefined ? '未知' : String(snap.online) }}</DescriptionsItem>
        <DescriptionsItem label="里程 km">{{ dash(snap.odometerKm) }}</DescriptionsItem>
        <DescriptionsItem label="座舱 / 外温">{{ dash(climate.cabinC) }} / {{ dash(climate.outsideC) }}</DescriptionsItem>
        <DescriptionsItem label="12V">{{ dash(b12.volts) }}</DescriptionsItem>
        <DescriptionsItem label="动力电池">{{ dash(pack.packVoltageV) }} V / {{ dash(pack.currentA) }} A</DescriptionsItem>
        <DescriptionsItem label="锁">{{ dash(body.locked) }}</DescriptionsItem>
        <DescriptionsItem label="门">FL {{ dash(doors.fl) }} · FR {{ dash(doors.fr) }} · RL {{ dash(doors.rl) }} · RR {{ dash(doors.rr) }} · 前盖 {{ dash(doors.hood) }} · 箱 {{ dash(doors.trunk) }}</DescriptionsItem>
        <DescriptionsItem label="窗">FL {{ dash(windows.fl) }} · FR {{ dash(windows.fr) }} · RL {{ dash(windows.rl) }} · RR {{ dash(windows.rr) }} · 天窗 {{ dash(windows.sunroof) }}</DescriptionsItem>
        <DescriptionsItem label="胎压胎温">FL {{ tire('fl') }} · FR {{ tire('fr') }} · RL {{ tire('rl') }} · RR {{ tire('rr') }}</DescriptionsItem>
        <DescriptionsItem v-if="extender" label="增程">油 {{ dash(extender.fuelPct) }}% · {{ dash(extender.fuelRangeKm) }} km · 开 {{ dash(extender.enabled) }} · 发电 {{ dash(extender.generating) }}</DescriptionsItem>
        <DescriptionsItem label="解码警告">{{ Array.isArray(snap.decodeWarnings) && (snap.decodeWarnings as string[]).length ? (snap.decodeWarnings as string[]).join(', ') : '无' }}</DescriptionsItem>
      </Descriptions>
      <p v-else>还没有快照</p>

      <h3 style="margin-top: 24px">能耗</h3>
      <p v-if="detail.energy">总计 {{ dash(detail.energy.totalKwh) }} kWh · 同步于 {{ dash(detail.energy.fetchedAt) }}</p>
      <Table
        v-if="detail.energy?.days?.length"
        row-key="countTime"
        size="small"
        :columns="energyColumns"
        :data-source="detail.energy.days"
        :pagination="false"
      />
      <p v-else>还没有能耗</p>

      <h3 style="margin-top: 24px">最近任务</h3>
      <Table row-key="id" :columns="jobColumns" :data-source="detail.jobs" :pagination="false" size="small" />
    </template>
  </Spin>
</template>
