<script setup lang="ts">
import { NCard, NEmpty } from 'naive-ui'
</script>

<template>
  <div>
    <h2 style="margin: 0 0 16px 0;">证书管理</h2>
    <n-card>
      <n-empty description="证书管理页面开发中...">
        <template #extra>
          <p>提示：证书管理用于管理 TLS 证书，支持多域名、自动续期提醒。</p>
        </template>
      </n-empty>
    </n-card>
  </div>
</template>
