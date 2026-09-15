<script setup lang="ts">
import { App, Button, Form, FormItem, Input, InputNumber, TextArea } from 'antdv-next'
import { onMounted, ref } from 'vue'
import { api, type Settings } from '../api'

const { message } = App.useApp()
const loading = ref(false)
const saving = ref(false)
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
</template>
