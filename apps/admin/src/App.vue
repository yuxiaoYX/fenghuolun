<script setup lang="ts">
import { App as AntdApp, ConfigProvider } from 'antdv-next'
import zhCN from 'antdv-next/locale/zh_CN'
import { RouterView } from 'vue-router'
</script>

<template>
  <ConfigProvider :locale="zhCN">
    <AntdApp>
      <RouterView />
    </AntdApp>
  </ConfigProvider>
</template>
