<script setup lang="ts">
import { NCard, NEmpty } from 'naive-ui'
</script>

<template>
  <div>
    <h2 style="margin: 0 0 16px 0;">客户端管理</h2>
    <n-card>
      <n-empty description="客户端管理页面开发中...">
        <template #extra>
          <p>提示：客户端用于管理同一入站规则下的多个用户凭证，支持独立的流量统计和有效期。</p>
        </template>
      </n-empty>
    </n-card>
  </div>
</template>
