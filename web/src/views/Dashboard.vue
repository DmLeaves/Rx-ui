<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { NGrid, NGi, NCard, NStatistic, NProgress, NButton, NSpace, NIcon, useMessage } from 'naive-ui'
import { RefreshOutline, PlayOutline, StopOutline, ReloadOutline } from '@vicons/ionicons5'
import { systemApi, type SystemStatus } from '@/api/system'

const message = useMessage()
const loading = ref(false)
const status = ref<SystemStatus | null>(null)

let timer: number | null = null

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatUptime(seconds: number): string {
  if (!seconds) return '0分钟'
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

async function startXray() {
  loading.value = true
  try {
    await systemApi.startXray()
    message.success('Xray 已启动')
    fetchStatus()
  } catch (error: any) {
    message.error(error.message || '启动失败')
  } finally {
    loading.value = false
  }
}

async function stopXray() {
  loading.value = true
  try {
    await systemApi.stopXray()
    message.success('Xray 已停止')
    fetchStatus()
  } catch (error: any) {
    message.error(error.message || '停止失败')
  } finally {
    loading.value = false
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
        <n-button 
          v-if="status && !status.xray.running" 
          type="success" 
          :loading="loading" 
          @click="startXray"
        >
          <template #icon><n-icon :component="PlayOutline" /></template>
          启动 Xray
        </n-button>
        <n-button 
          v-if="status && status.xray.running" 
          type="warning" 
          :loading="loading" 
          @click="stopXray"
        >
          <template #icon><n-icon :component="StopOutline" /></template>
          停止 Xray
        </n-button>
        <n-button type="primary" :loading="loading" @click="restartXray">
          <template #icon><n-icon :component="ReloadOutline" /></template>
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
              <span :style="{ color: status.xray.running ? '#18a058' : '#d03050' }">
                {{ status.xray.running ? '运行中' : '已停止' }}
              </span>
            </template>
          </n-statistic>
          <p style="margin-top: 8px; color: #999; font-size: 12px;">{{ status.xray.version }}</p>
        </n-card>
      </n-gi>

      <!-- 入站规则数 -->
      <n-gi>
        <n-card>
          <n-statistic label="入站规则">
            {{ status.inboundCount }}
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

      <!-- 流量统计 -->
      <n-gi>
        <n-card>
          <n-statistic label="总流量">
            ↑ {{ formatBytes(status.traffic.up) }} / ↓ {{ formatBytes(status.traffic.down) }}
          </n-statistic>
        </n-card>
      </n-gi>

      <!-- CPU -->
      <n-gi>
        <n-card title="CPU">
          <n-progress
            type="line"
            :percentage="Math.round(status.cpu.percent)"
            :indicator-placement="'inside'"
          />
          <p style="margin-top: 8px; color: #999;">{{ status.cpu.cores }} 核心</p>
        </n-card>
      </n-gi>

      <!-- 内存 -->
      <n-gi>
        <n-card title="内存">
          <n-progress
            type="line"
            :percentage="status.memory.total ? Math.round(status.memory.used / status.memory.total * 100) : 0"
            :indicator-placement="'inside'"
          />
          <p style="margin-top: 8px; color: #999;">
            {{ formatBytes(status.memory.used) }} / {{ formatBytes(status.memory.total) }}
          </p>
        </n-card>
      </n-gi>

      <!-- 系统负载 -->
      <n-gi>
        <n-card title="系统负载">
          <p><strong>1分钟:</strong> {{ status.load[0]?.toFixed(2) || 0 }}</p>
          <p><strong>5分钟:</strong> {{ status.load[1]?.toFixed(2) || 0 }}</p>
          <p><strong>15分钟:</strong> {{ status.load[2]?.toFixed(2) || 0 }}</p>
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>
