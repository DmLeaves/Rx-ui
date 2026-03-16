<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { NGrid, NGi, NCard, NStatistic, NProgress, NButton, NSpace, NIcon, useMessage } from 'naive-ui'
import { RefreshOutline, PlayOutline } from '@vicons/ionicons5'
import { systemApi, type SystemStatus } from '@/api/system'

const message = useMessage()
const loading = ref(false)
const status = ref<SystemStatus | null>(null)

let timer: number | null = null

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${mins}分钟`
  return `${mins}分钟`
}

async function fetchStatus() {
  try {
    const res = await systemApi.getStatus()
    status.value = res.data.data
  } catch (error: any) {
    console.error('获取系统状态失败:', error)
  }
}

async function restartXray() {
  loading.value = true
  try {
    await systemApi.restartXray()
    message.success('Xray 已重启')
    fetchStatus()
  } catch (error: any) {
    message.error(error.message || '重启失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchStatus()
  timer = window.setInterval(fetchStatus, 5000)
})

onUnmounted(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h2 style="margin: 0;">仪表盘</h2>
      <n-space>
        <n-button @click="fetchStatus">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
        <n-button type="primary" :loading="loading" @click="restartXray">
          <template #icon><n-icon :component="PlayOutline" /></template>
          重启 Xray
        </n-button>
      </n-space>
    </n-space>

    <n-grid :cols="4" :x-gap="16" :y-gap="16" v-if="status">
      <!-- Xray 状态 -->
      <n-gi>
        <n-card>
          <n-statistic label="Xray 状态">
            <template #default>
              <span :style="{ color: status.xrayRunning ? '#18a058' : '#d03050' }">
                {{ status.xrayRunning ? '运行中' : '已停止' }}
              </span>
            </template>
          </n-statistic>
        </n-card>
      </n-gi>

      <!-- 系统运行时间 -->
      <n-gi>
        <n-card>
          <n-statistic label="系统运行时间">
            {{ formatUptime(status.uptime) }}
          </n-statistic>
        </n-card>
      </n-gi>

      <!-- 面板运行时间 -->
      <n-gi>
        <n-card>
          <n-statistic label="面板运行时间">
            {{ formatUptime(status.panelUptime) }}
          </n-statistic>
        </n-card>
      </n-gi>

      <!-- 网络流量 -->
      <n-gi>
        <n-card>
          <n-statistic label="网络流量">
            ↑ {{ formatBytes(status.netUpload) }} / ↓ {{ formatBytes(status.netDownload) }}
          </n-statistic>
        </n-card>
      </n-gi>

      <!-- CPU -->
      <n-gi>
        <n-card title="CPU">
          <n-progress
            type="line"
            :percentage="Math.round(status.cpuPercent)"
            :indicator-placement="'inside'"
          />
          <p style="margin-top: 8px; color: #999;">{{ status.cpuCores }} 核心</p>
        </n-card>
      </n-gi>

      <!-- 内存 -->
      <n-gi>
        <n-card title="内存">
          <n-progress
            type="line"
            :percentage="Math.round(status.memPercent)"
            :indicator-placement="'inside'"
          />
          <p style="margin-top: 8px; color: #999;">
            {{ formatBytes(status.memUsed) }} / {{ formatBytes(status.memTotal) }}
          </p>
        </n-card>
      </n-gi>

      <!-- 磁盘 -->
      <n-gi>
        <n-card title="磁盘">
          <n-progress
            type="line"
            :percentage="Math.round(status.diskPercent)"
            :indicator-placement="'inside'"
          />
          <p style="margin-top: 8px; color: #999;">
            {{ formatBytes(status.diskUsed) }} / {{ formatBytes(status.diskTotal) }}
          </p>
        </n-card>
      </n-gi>

      <!-- 系统信息 -->
      <n-gi>
        <n-card title="系统信息">
          <p><strong>主机名:</strong> {{ status.hostname }}</p>
          <p><strong>系统:</strong> {{ status.os }} / {{ status.arch }}</p>
          <p><strong>Go:</strong> {{ status.goVersion }}</p>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>
