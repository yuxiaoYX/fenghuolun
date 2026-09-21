<script setup lang="ts">
import { App, Alert, Card, Col, Row, Spin } from 'antdv-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, type AdminHealth } from '../api'

const { message } = App.useApp()
const router = useRouter()
const loading = ref(true)
const phase = ref('')
const health = ref<AdminHealth | null>(null)

onMounted(async () => {
  try {
    const z = await api.healthz()
    phase.value = `后端 ${z.status} · 分期 ${z.phase}` + (z.version ? ` · ${z.version}` : '')
  } catch {
    phase.value = '后端未启动（默认 http://127.0.0.1:8088）'
  }
  try {
    health.value = await api.health()
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉总览失败')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <Spin :spinning="loading">
    <Alert v-if="phase" :message="phase" type="info" show-icon style="margin-bottom: 16px" />
    <Row v-if="health" :gutter="16">
      <Col :xs="24" :sm="12" :md="6">
        <Card title="绑定" hoverable style="cursor: pointer" @click="router.push('/bindings')">
          {{ health.bindingsActive }} 启用 / {{ health.bindingsDisabled }} 停用
        </Card>
      </Col>
      <Col :xs="24" :sm="12" :md="6">
        <Card title="今日同步" hoverable style="cursor: pointer" @click="router.push('/jobs')">
          {{ health.syncOkToday }} 成功 / {{ health.syncFailToday }} 失败
        </Card>
      </Col>
      <Col :xs="24" :sm="12" :md="6">
        <Card title="凭证失效" hoverable style="cursor: pointer" @click="router.push('/bindings?filter=auth_failed')">
          {{ health.tokenInvalid }}
        </Card>
      </Col>
      <Col :xs="24" :sm="12" :md="6">
        <Card title="上游连续失败" hoverable style="cursor: pointer" @click="router.push('/jobs?status=upstream')">
          {{ health.upstreamStreak }}
        </Card>
      </Col>
    </Row>
    <p v-if="health" style="margin-top: 16px; opacity: 0.7">
      定时同步：{{ health.cronSync || '关闭' }}。点卡片进对应列表。
    </p>
  </Spin>
</template>
