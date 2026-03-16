<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, NTooltip, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, CreateOutline, QrCodeOutline, LinkOutline, RefreshCircleOutline } from '@vicons/ionicons5'
import { inboundApi, type Inbound, type CreateInboundParams } from '@/api/inbound'
import InboundModal from '@/components/InboundModal.vue'
import QRCodeModal from '@/components/QRCodeModal.vue'

const message = useMessage()
const loading = ref(false)
const inbounds = ref<Inbound[]>([])

// 弹窗状态
const showInboundModal = ref(false)
const editingInbound = ref<Inbound | null>(null)

const showQRModal = ref(false)
const qrTitle = ref('')
const qrLink = ref('')

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatExpiry(timestamp: number): string {
  if (!timestamp) return '永久'
  const date = new Date(timestamp)
  const now = new Date()
  if (date < now) return '已过期'
  return date.toLocaleDateString('zh-CN')
}

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

function generateLink(row: Inbound): string {
  try {
    const settings = JSON.parse(row.settings || '{}')
    const streamSettings = JSON.parse(row.streamSettings || '{}')
    const protocol = row.protocol

    switch (protocol) {
      case 'vmess': {
        const client = settings.clients?.[0]
        if (!client) return ''
        const config = {
          v: '2',
          ps: row.remark || `${row.port}`,
          add: window.location.hostname,
          port: row.port,
          id: client.id,
          aid: client.alterId || 0,
          net: streamSettings.network || 'tcp',
          type: streamSettings.tcpSettings?.header?.type || 'none',
          host: streamSettings.wsSettings?.headers?.Host || '',
          path: streamSettings.wsSettings?.path || '',
          tls: streamSettings.security === 'tls' ? 'tls' : ''
        }
        return 'vmess://' + btoa(JSON.stringify(config))
      }
      case 'vless': {
        const client = settings.clients?.[0]
        if (!client) return ''
        const params = new URLSearchParams()
        params.set('type', streamSettings.network || 'tcp')
        if (streamSettings.security) params.set('security', streamSettings.security)
        if (client.flow) params.set('flow', client.flow)
        if (streamSettings.wsSettings?.path) params.set('path', streamSettings.wsSettings.path)
        if (streamSettings.grpcSettings?.serviceName) params.set('serviceName', streamSettings.grpcSettings.serviceName)
        return `vless://${client.id}@${window.location.hostname}:${row.port}?${params.toString()}#${encodeURIComponent(row.remark || '')}`
      }
      case 'trojan': {
        const client = settings.clients?.[0]
        if (!client) return ''
        return `trojan://${client.password}@${window.location.hostname}:${row.port}#${encodeURIComponent(row.remark || '')}`
      }
      case 'shadowsocks': {
        const method = settings.method || 'aes-256-gcm'
        const password = settings.password || ''
        const userinfo = btoa(`${method}:${password}`)
        return `ss://${userinfo}@${window.location.hostname}:${row.port}#${encodeURIComponent(row.remark || '')}`
      }
      default:
        return ''
    }
  } catch (e) {
    console.error('生成链接失败:', e)
    return ''
  }
}

function showQR(row: Inbound) {
  const link = generateLink(row)
  if (!link) {
    message.warning('该协议不支持生成二维码')
    return
  }
  qrTitle.value = row.remark || `入站 ${row.port}`
  qrLink.value = link
  showQRModal.value = true
}

function copyLink(row: Inbound) {
  const link = generateLink(row)
  if (!link) {
    message.warning('该协议不支持生成链接')
    return
  }
  navigator.clipboard.writeText(link)
  message.success('链接已复制')
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
  } catch (error: any) {
    message.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
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
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

async function handleDelete(id: number) {
  try {
    await inboundApi.delete(id)
    message.success('删除成功')
    fetchInbounds()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

async function handleResetTraffic(id: number) {
  try {
    await inboundApi.resetTraffic(id)
    message.success('流量已重置')
    fetchInbounds()
  } catch (error: any) {
    message.error(error.message || '重置失败')
  }
}

onMounted(() => {
  fetchInbounds()
})
</script>

<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h2 style="margin: 0;">入站规则</h2>
      <n-space>
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
  </div>
</template>
