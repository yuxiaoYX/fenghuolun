<script setup lang="ts">
import { Alert, App, Button, Card, Form, FormItem, Input, InputNumber, TextArea } from 'antdv-next'
import { onMounted, ref } from 'vue'
import { api, type Settings, type SystemInfo } from '../api'

const { message, modal } = App.useApp()
const loading = ref(false)
const saving = ref(false)
const checking = ref(false)
const applying = ref(false)
const sys = ref<SystemInfo | null>(null)
const form = ref<Settings>({
  cronSync: '',
  corsOrigins: '',
  snapshotKeep: 0,
  staleAfterSec: 7200,
})

onMounted(async () => {
  loading.value = true
  try {
    form.value = await api.settings()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉设置失败')
  }
  try {
    sys.value = await api.system()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉版本失败')
  } finally {
    loading.value = false
  }
})

async function save() {
  saving.value = true
  try {
    form.value = await api.saveSettings(form.value)
    message.success('已保存，定时同步已热替换')
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function check() {
  checking.value = true
  try {
    sys.value = await api.system(true)
    if (sys.value.updateAvailable) {
      message.success('有新版本 ' + sys.value.latest)
    } else if (sys.value.latest) {
      message.success('已是最新版')
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : '检查更新失败')
  } finally {
    checking.value = false
  }
}

function confirmUpdate() {
  const target = sys.value?.latest
  if (!target) return
  modal.confirm({
    title: '更新到 ' + target + '？',
    content: '会备份数据库、拉取镜像并重建容器。页面会短暂不可用，登录状态保留。不要关闭浏览器。',
    okText: '开始更新',
    cancelText: '取消',
    onOk: () => applyUpdate(),
  })
}

async function applyUpdate() {
  applying.value = true
  try {
    const r = await api.systemUpdate()
    message.success('正在更新到 ' + r.target + '，请稍候')
    await waitUntilVersion(r.target)
    message.success('已更新到 ' + r.target)
    location.reload()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '更新失败')
    applying.value = false
  }
}

async function waitUntilVersion(target: string) {
  const deadline = Date.now() + 180000
  let delay = 2000
  while (Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, delay))
    delay = Math.min(delay + 500, 5000)
    try {
      const z = await api.healthz()
      if (z.version && z.version === target) {
        return
      }
    } catch {
      // 重建期间会断一下
    }
  }
  throw new Error('等待新版本超时。看 docker logs fenghuolun-updater 或 docker logs fenghuolun')
}
</script>

<template>
  <Form layout="vertical" style="max-width: 640px" :disabled="loading">
    <FormItem label="定时同步" extra="15m、@every 10m，或 off 关闭。改完立即热替换，不必重启。">
      <Input v-model:value="form.cronSync" placeholder="15m" />
    </FormItem>
    <FormItem label="CORS 来源" extra="逗号分隔。本机 127.0.0.1 / localhost 端口始终放行。">
      <TextArea v-model:value="form.corsOrigins" :rows="3" />
    </FormItem>
    <FormItem label="每车保留快照条数" extra="0 表示不裁。写入新快照时裁掉更旧的。">
      <InputNumber v-model:value="form.snapshotKeep" :min="0" style="width: 160px" />
    </FormItem>
    <FormItem label="陈旧阈值（秒）" extra="车辆上报时刻距现在超过此时长，快照标 stale。最少 60。">
      <InputNumber v-model:value="form.staleAfterSec" :min="60" style="width: 160px" />
    </FormItem>
    <Button type="primary" :loading="saving" @click="save">保存</Button>
  </Form>
  <p style="margin-top: 16px; opacity: 0.65">监听地址、SQLite 路径、TOKEN_KEK 仍只在环境变量里，这里改不了。</p>

  <Card title="系统更新" style="max-width: 640px; margin-top: 24px" :loading="loading && !sys">
    <p>当前版本：{{ sys?.version || '—' }}</p>
    <p>
      最新 Release：
      <a v-if="sys?.latestUrl" :href="sys.latestUrl" target="_blank" rel="noreferrer">{{ sys.latest || '—' }}</a>
      <span v-else>{{ sys?.latest || '—' }}</span>
    </p>
    <Alert
      v-if="sys?.hint"
      :type="sys.canApply ? 'info' : sys.updateAvailable ? 'warning' : 'info'"
      :message="sys.hint"
      show-icon
      style="margin-bottom: 12px"
    />
    <Button :loading="checking" @click="check">检查更新</Button>
    <Button type="primary" style="margin-left: 8px" :disabled="!sys?.canApply" :loading="applying" @click="confirmUpdate">
      更新到最新版
    </Button>
  </Card>
</template>
