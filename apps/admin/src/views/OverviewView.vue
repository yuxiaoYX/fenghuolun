<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

type Row = {
  id: string
  vinMasked: string
  modelCode: string
  modelName: string
  syncStatus: string
  syncError: string
  syncedAt: string | null
  disabled: boolean
}

const router = useRouter()
const health = ref('检查中…')
const rows = ref<Row[]>([])
const note = ref('')
const busyId = ref('')

onMounted(async () => {
  const token = localStorage.getItem('fhl_admin') ?? ''
  if (!token) {
    router.replace('/login')
    return
  }
  try {
    const res = await fetch('/healthz')
    const body = (await res.json()) as { ok?: boolean; data?: { status?: string; phase?: string } }
    health.value = res.ok && body.ok
      ? `后端 ${body.data?.status ?? 'ok'} · 分期 ${body.data?.phase ?? '?'}`
      : `后端 HTTP ${res.status}`
  } catch {
    health.value = '后端未启动（默认 http://127.0.0.1:8080）'
  }
  try {
    const res = await fetch('/api/v1/admin/bindings', {
      headers: { Authorization: 'Bearer ' + token },
    })
    const body = (await res.json()) as { ok?: boolean; data?: { items?: Row[] } }
    if (!body.ok) {
      localStorage.removeItem('fhl_admin')
      router.replace('/login')
      return
    }
    rows.value = body.data?.items ?? []
    note.value = rows.value.length === 0 ? '还没有绑定' : ''
  } catch {
    note.value = '拉绑定列表失败'
  }
})

async function disableRow(id: string) {
  const token = localStorage.getItem('fhl_admin') ?? ''
  if (!token || !id) {
    return
  }
  busyId.value = id
  note.value = ''
  try {
    const res = await fetch('/api/v1/admin/bindings/' + encodeURIComponent(id) + '/disable', {
      method: 'POST',
      headers: { Authorization: 'Bearer ' + token, 'Content-Type': 'application/json' },
    })
    const body = (await res.json()) as { ok?: boolean; error?: { message?: string } }
    if (!body.ok) {
      note.value = body.error?.message ?? '停用失败'
      return
    }
    rows.value = rows.value.map((r) => (r.id === id ? { ...r, disabled: true } : r))
  } catch {
    note.value = '停用失败'
  } finally {
    busyId.value = ''
  }
}
</script>

<template>
  <main class="page">
    <section class="card wide">
      <h1>总览</h1>
      <p class="status">{{ health }}</p>
      <p v-if="note" class="hint">{{ note }}</p>
      <table v-if="rows.length > 0">
        <thead>
          <tr>
            <th>车型</th>
            <th>VIN</th>
            <th>同步</th>
            <th>同步时间</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.id">
            <td>{{ r.modelName }} {{ r.modelCode }}</td>
            <td>{{ r.vinMasked }}</td>
            <td>{{ r.syncStatus }} {{ r.syncError }}{{ r.disabled ? ' · 已停用' : '' }}</td>
            <td>{{ r.syncedAt || '—' }}</td>
            <td>
              <button
                class="row-btn"
                type="button"
                :disabled="r.disabled || busyId === r.id"
                @click="disableRow(r.id)"
              >
                {{ r.disabled ? '已停用' : '停用' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p class="hint">不展示凭证明文。停用只清本服务会话，不会向车辆下发命令。</p>
      <RouterLink to="/login">返回登录</RouterLink>
    </section>
  </main>
</template>
