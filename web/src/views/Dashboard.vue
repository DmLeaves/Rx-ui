<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { NGrid, NGi, NCard, NStatistic, NButton, NSpace, NIcon, useMessage } from 'naive-ui'
import { RefreshOutline, PlayOutline, StopOutline, ReloadOutline } from '@vicons/ionicons5'
import { systemApi, type SystemStatus, type XrayEvent } from '@/api/system'

const message = useMessage()
const loading = ref(false)
const status = ref<SystemStatus | null>(null)
const xrayEvents = ref<XrayEvent[]>([])
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

async function fetchXrayEvents() {
  try {
    const res = await systemApi.getXrayEvents(80)
    xrayEvents.value = res.data.data || []
  } catch (error: any) {
    console.error('获取 Xray 事件失败:', error)
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
  fetchXrayEvents()
  timer = window.setInterval(() => {
    fetchStatus()
    fetchXrayEvents()
  }, 10000)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
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
        <n-button v-if="status && !status.xray.running" type="success" :loading="loading" @click="startXray">
          <template #icon><n-icon :component="PlayOutline" /></template>
          启动 Xray
        </n-button>
        <n-button v-if="status && status.xray.running" type="warning" :loading="loading" @click="stopXray">
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
      <n-gi><n-card><n-statistic label="Xray 状态"><template #default><span :style="{ color: status.xray.running ? '#18a058' : '#d03050' }">{{ status.xray.running ? '运行中' : '已停止' }}</span></template></n-statistic><p style="margin-top: 8px; color: #999; font-size: 12px;">{{ status.xray.version }}</p><p style="margin-top: 4px; color: #999; font-size: 12px;">守护目标：{{ status.xray.desired === false ? '手动停止' : '保持运行' }}</p></n-card></n-gi>
      <n-gi><n-card><n-statistic label="入站规则">{{ status.inboundCount }}</n-statistic></n-card></n-gi>
      <n-gi><n-card><n-statistic label="系统运行时间">{{ formatUptime(status.uptime) }}</n-statistic></n-card></n-gi>
      <n-gi><n-card><n-statistic label="面板运行时间">{{ formatUptime(status.panelUptime) }}</n-statistic></n-card></n-gi>
      <n-gi><n-card><n-statistic label="总流量">↑ {{ formatBytes(status.traffic.up) }} / ↓ {{ formatBytes(status.traffic.down) }}</n-statistic></n-card></n-gi>
    </n-grid>

    <n-card title="Xray 守护事件日志" style="margin-top: 16px;" v-if="xrayEvents.length">
      <div style="max-height: 280px; overflow: auto; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 1.5; white-space: pre-wrap;">
        <div v-for="(e, idx) in xrayEvents" :key="idx" :style="{ color: e.level === 'ERROR' ? '#d03050' : e.level === 'WARN' ? '#f0a020' : '#333' }">
          [{{ e.time }}] [{{ e.level }}] [{{ e.type }}] {{ e.message }}
        </div>
      </div>
    </n-card>
  </div>
</template>
