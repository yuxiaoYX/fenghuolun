<script setup lang="ts">
import { App, Button, Form, FormItem, InputPassword } from 'antdv-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, clearToken } from '../api'

const { message } = App.useApp()
const router = useRouter()
const username = ref('')
const oldPassword = ref('')
const newPassword = ref('')
const saving = ref(false)

onMounted(async () => {
  try {
    const a = await api.account()
    username.value = a.username
  } catch (e) {
    message.error(e instanceof Error ? e.message : '拉账号失败')
  }
})

async function changePassword() {
  saving.value = true
  try {
    await api.password(oldPassword.value, newPassword.value)
    message.success('密码已改')
    oldPassword.value = ''
    newPassword.value = ''
  } catch (e) {
    message.error(e instanceof Error ? e.message : '改密失败')
  } finally {
    saving.value = false
  }
}

async function logout() {
  try {
    await api.logout()
  } catch {
    // 本地清会话即可
  }
  clearToken()
  await router.replace('/login')
}
</script>

<template>
  <p>当前用户：{{ username || '—' }}</p>
  <Form layout="vertical" style="max-width: 400px">
    <FormItem label="当前密码">
      <InputPassword v-model:value="oldPassword" autocomplete="current-password" />
    </FormItem>
    <FormItem label="新密码" extra="至少 8 位">
      <InputPassword v-model:value="newPassword" autocomplete="new-password" />
    </FormItem>
    <Button type="primary" :loading="saving" @click="changePassword">改密码</Button>
    <Button style="margin-left: 8px" @click="logout">退出登录</Button>
  </Form>
</template>
