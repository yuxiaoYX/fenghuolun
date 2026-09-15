<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const username = ref('')
const password = ref('')
const err = ref('')

async function submit() {
  err.value = ''
  try {
    const res = await fetch('/api/v1/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    const body = (await res.json()) as { ok?: boolean; data?: { session?: string }; error?: { message?: string } }
    if (!body.ok || !body.data?.session) {
      err.value = body.error?.message ?? '登录失败'
      return
    }
    localStorage.setItem('fhl_admin', body.data.session)
    router.push('/')
  } catch {
    err.value = '后端未启动（默认 :8080）'
  }
}
</script>

<template>
  <main class="page">
    <section class="card">
      <h1>风火轮管理</h1>
      <p>仅管理员。用 FENGHUOLUN_ADMIN_BOOTSTRAP_* 创建第一位账号。</p>
      <form @submit.prevent="submit">
        <label for="user">用户名</label>
        <input id="user" v-model="username" autocomplete="username" />
        <label for="pass">密码</label>
        <input id="pass" v-model="password" type="password" autocomplete="current-password" />
        <button type="submit">进入</button>
      </form>
      <p v-if="err" class="status">{{ err }}</p>
    </section>
  </main>
</template>
