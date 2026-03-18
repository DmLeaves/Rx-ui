<script setup lang="ts">
import { ref, onMounted, onUnmounted, h } from 'vue'
import {
  NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, NTooltip,
  NModal, NInput, NUpload, useMessage
} from 'naive-ui'
import type { DataTableColumns, UploadFileInfo } from 'naive-ui'
import {
  AddOutline, RefreshOutline, TrashOutline, CreateOutline,
  QrCodeOutline, LinkOutline, RefreshCircleOutline,
  CloudDownloadOutline, CloudUploadOutline, ShareSocialOutline
} from '@vicons/ionicons5'
import { inboundApi, type Inbound, type CreateInboundParams } from '@/api/inbound'
import { systemApi } from '@/api/system'
import InboundModal from '@/components/InboundModal.vue'
import QRCodeModal from '@/components/QRCodeModal.vue'
import { formatBytes, formatExpiry } from '@/utils/format'
import { generateInboundLink, generateSubscription } from '@/utils/link'

const message = useMessage()
const loading = ref(false)
const inbounds = ref<Inbound[]>([])

// 弹窗状态
const showInboundModal = ref(false)
const editingInbound = ref<Inbound | null>(null)

const showQRModal = ref(false)
const qrTitle = ref('')
const qrLink = ref('')

// 订阅弹窗
const showSubscriptionModal = ref(false)
const subscriptionLink = ref('')

let trafficTimer: number | null = null

function getStatusType(row: Inbound): 'success' | 'error' | 'warning' {
  if (!row.enable) return 'error'
  if (row.expiryTime && row.expiryTime < Date.now()) return 'warning'
  if (row.total > 0 && (row.up + row.down) >= row.total) return 'warning'
  return 'success'
}

function getStatusText(row: Inbound): string {
  if (!row.enable) return '禁用'
  if (row.expiryTime && row.expiryTime < Date.now()) return '已过期'
  if (row.total > 0 && (row.up + row.down) >= row.total) return '流量耗尽'
  return '启用'
}

function showQR(row: Inbound) {
  const link = generateInboundLink(row)
  if (!link) {
    message.warning('该协议不支持生成二维码')
    return
  }
  qrTitle.value = row.remark || `入站 ${row.port}`
  qrLink.value = link
  showQRModal.value = true
}

async function copyLink(row: Inbound) {
  const link = generateInboundLink(row)
  if (!link) {
    message.warning('该协议不支持生成链接')
    return
  }

  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(link)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = link
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.focus()
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    message.success('链接已复制')
  } catch {
    message.error('复制失败，请手动复制')
  }
}

// 订阅功能
function openSubscription() {
  const base64 = generateSubscription(inbounds.value)
  if (!base64 || base64 === btoa('')) {
    message.warning('没有可用的订阅链接')
    return
  }
  subscriptionLink.value = base64
  showSubscriptionModal.value = true
}

async function copySubscription() {
  const text = subscriptionLink.value || ''
  if (!text) return

  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.focus()
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    message.success('订阅内容已复制')
  } catch {
    message.error('复制失败，请手动复制')
  }
}

function showSubscriptionQR() {
  qrTitle.value = '订阅二维码'
  qrLink.value = subscriptionLink.value
  showQRModal.value = true
}

// 导出功能
function handleExport() {
  const data = inbounds.value.map(item => ({
    remark: item.remark,
    enable: item.enable,
    listen: item.listen,
    port: item.port,
    protocol: item.protocol,
    settings: item.settings,
    streamSettings: item.streamSettings,
    sniffing: item.sniffing,
    tag: item.tag,
    total: item.total,
    expiryTime: item.expiryTime,
    certificateId: item.certificateId
  }))
  
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `inbounds-${new Date().toISOString().slice(0,10)}.json`
  a.click()
  URL.revokeObjectURL(url)
  message.success('导出成功')
}

// 导入功能
async function handleImport(options: { file: UploadFileInfo }) {
  const file = options.file.file
  if (!file) return
  
  try {
    const text = await file.text()
    const data = JSON.parse(text) as CreateInboundParams[]
    
    if (!Array.isArray(data)) {
      message.error('无效的JSON格式')
      return
    }
    
    let successCount = 0
    let errorCount = 0
    
    for (const item of data) {
      try {
        await inboundApi.create(item)
        successCount++
      } catch {
        errorCount++
      }
    }
    
    message.success(`导入完成: 成功 ${successCount} 个, 失败 ${errorCount} 个`)
    fetchInbounds()
  } catch {
    message.error('解析JSON失败')
  }
}

const columns: DataTableColumns<Inbound> = [
  { title: 'ID', key: 'id', width: 60 },
  { 
    title: '备注', 
    key: 'remark', 
    ellipsis: { tooltip: true },
    render: (row) => row.remark || '-'
  },
  { 
    title: '状态', 
    key: 'enable',
    width: 90,
    render: (row) => h(NTag, { type: getStatusType(row), size: 'small' }, { default: () => getStatusText(row) })
  },
  { title: '端口', key: 'port', width: 80 },
  { 
    title: '协议', 
    key: 'protocol', 
    width: 100,
    render: (row) => h(NTag, { type: 'info', size: 'small' }, { default: () => row.protocol })
  },
  { 
    title: '流量', 
    key: 'traffic',
    width: 180,
    render: (row) => {
      const used = row.up + row.down
      const total = row.total
      if (total > 0) {
        const percent = Math.min(100, (used / total) * 100).toFixed(1)
        return `${formatBytes(used)} / ${formatBytes(total)} (${percent}%)`
      }
      return `↑${formatBytes(row.up)} ↓${formatBytes(row.down)}`
    }
  },
  {
    title: '到期',
    key: 'expiryTime',
    width: 100,
    render: (row) => formatExpiry(row.expiryTime)
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NTooltip, null, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, onClick: () => showQR(row) }, { icon: () => h(NIcon, null, { default: () => h(QrCodeOutline) }) }),
          default: () => '二维码'
        }),
        h(NTooltip, null, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, onClick: () => copyLink(row) }, { icon: () => h(NIcon, null, { default: () => h(LinkOutline) }) }),
          default: () => '复制链接'
        }),
        h(NTooltip, null, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, onClick: () => handleEdit(row) }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }),
          default: () => '编辑'
        }),
        h(NPopconfirm, { onPositiveClick: () => handleResetTraffic(row.id) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true }, { icon: () => h(NIcon, null, { default: () => h(RefreshCircleOutline) }) }),
          default: () => '确定重置流量?'
        }),
        h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => '确定删除该入站规则?'
        })
      ]
    })
  }
]

async function fetchInbounds() {
  loading.value = true
  try {
    const res = await inboundApi.list()
    inbounds.value = res.data.data || []
    await syncTraffic()
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '获取列表失败'
    message.error(msg)
  } finally {
    loading.value = false
  }
}

async function syncTraffic() {
  try {
    const res = await systemApi.getTraffic()
    const stats = res.data.data || []
    const byTag = new Map(stats.map(item => [item.tag, item]))
    inbounds.value = inbounds.value.map(i => {
      const s = byTag.get(i.tag)
      if (!s) return i
      return { ...i, up: s.uplink, down: s.downlink }
    })
  } catch {
    // Xray 未运行时静默
  }
}

function handleAdd() {
  editingInbound.value = null
  showInboundModal.value = true
}

function handleEdit(row: Inbound) {
  editingInbound.value = row
  showInboundModal.value = true
}

async function handleSubmit(data: CreateInboundParams) {
  // 前置端口冲突校验
  const conflict = inbounds.value.find(i => i.port === data.port && (!editingInbound.value || i.id !== editingInbound.value.id))
  if (conflict) {
    message.error(`端口 ${data.port} 已被入站「${conflict.remark || conflict.id}」占用`)
    return
  }

  try {
    if (editingInbound.value) {
      await inboundApi.update(editingInbound.value.id, data)
      message.success('更新成功')
    } else {
      await inboundApi.create(data)
      message.success('添加成功')
    }
    showInboundModal.value = false
    fetchInbounds()
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '操作失败'
    message.error(msg)
  }
}

async function handleDelete(id: number) {
  try {
    await inboundApi.delete(id)
    message.success('删除成功')
    fetchInbounds()
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '删除失败'
    message.error(msg)
  }
}

async function handleResetTraffic(id: number) {
  try {
    await inboundApi.resetTraffic(id)
    message.success('流量已重置')
    fetchInbounds()
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '重置失败'
    message.error(msg)
  }
}

onMounted(() => {
  fetchInbounds()
  trafficTimer = window.setInterval(syncTraffic, 5000)
})

onUnmounted(() => {
  if (trafficTimer) clearInterval(trafficTimer)
})
</script>

<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h2 style="margin: 0;">入站规则</h2>
      <n-space>
        <n-button @click="openSubscription">
          <template #icon><n-icon :component="ShareSocialOutline" /></template>
          订阅链接
        </n-button>
        <n-button @click="handleExport">
          <template #icon><n-icon :component="CloudDownloadOutline" /></template>
          导出
        </n-button>
        <n-upload
          :show-file-list="false"
          accept=".json"
          :custom-request="({ file }) => handleImport({ file })"
        >
          <n-button>
            <template #icon><n-icon :component="CloudUploadOutline" /></template>
            导入
          </n-button>
        </n-upload>
        <n-button @click="fetchInbounds">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
        <n-button type="primary" @click="handleAdd">
          <template #icon><n-icon :component="AddOutline" /></template>
          添加
        </n-button>
      </n-space>
    </n-space>

    <n-data-table
      :columns="columns"
      :data="inbounds"
      :loading="loading"
      :bordered="false"
      :row-key="(row: Inbound) => row.id"
    />

    <InboundModal
      v-model:show="showInboundModal"
      :edit-data="editingInbound"
      @submit="handleSubmit"
    />

    <QRCodeModal
      v-model:show="showQRModal"
      :title="qrTitle"
      :link="qrLink"
    />

    <!-- 订阅弹窗 -->
    <n-modal
      v-model:show="showSubscriptionModal"
      preset="card"
      title="订阅链接"
      style="width: 600px; max-width: 90vw;"
    >
      <n-space vertical>
        <p style="color: var(--n-text-color-3); margin: 0;">
          订阅内容包含所有启用的入站规则链接（Base64编码）
        </p>
        <n-input
          :value="subscriptionLink"
          type="textarea"
          :autosize="{ minRows: 3, maxRows: 6 }"
          readonly
        />
        <n-space>
          <n-button type="primary" @click="copySubscription">复制内容</n-button>
          <n-button @click="showSubscriptionQR">显示二维码</n-button>
        </n-space>
      </n-space>
    </n-modal>
  </div>
</template>
