<script setup lang="ts">
import { App, Button, Card, Form, FormItem, Input, InputPassword } from 'antdv-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError, api, getToken, setToken } from '../api'

const router = useRouter()
const { message } = App.useApp()
const username = ref('')
const password = ref('')
const loading = ref(false)

onMounted(() => {
  if (getToken()) {
    router.replace('/')
  }
})

async function submit() {
  loading.value = true
  try {
    const data = await api.login(username.value, password.value)
    setToken(data.session)
    await router.push('/')
  } catch (e) {
    message.error(e instanceof ApiError ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <Card class="login-card" title="风火轮管理">
      <p style="margin-top: 0; opacity: 0.7">仅管理员。第一位账号用 FENGHUOLUN_ADMIN_BOOTSTRAP_* 创建。</p>
      <Form layout="vertical" @submit.prevent="submit">
        <FormItem label="用户名">
          <Input v-model:value="username" autocomplete="username" />
        </FormItem>
        <FormItem label="密码">
          <InputPassword v-model:value="password" autocomplete="current-password" />
        </FormItem>
        <Button type="primary" html-type="submit" block :loading="loading">进入</Button>
      </Form>
    </Card>
  </div>
</template>
