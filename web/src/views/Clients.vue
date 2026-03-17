<script setup lang="ts">
import { ref, onMounted, h, watch } from 'vue'
import { NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, NSelect, NModal, NForm, NFormItem, NInput, NInputNumber, NSwitch, useMessage } from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, CreateOutline, QrCodeOutline } from '@vicons/ionicons5'
import { inboundApi, type Inbound, type Client, type CreateClientParams } from '@/api/inbound'
import QRCodeModal from '@/components/QRCodeModal.vue'
import { v4 as uuidv4 } from 'uuid'

const message = useMessage()
const loading = ref(false)

// 入站规则列表
const inbounds = ref<Inbound[]>([])
const inboundOptions = ref<SelectOption[]>([])
const selectedInboundId = ref<number | null>(null)

// 客户端列表
const clients = ref<Client[]>([])

// 弹窗
const showClientModal = ref(false)
const editingClient = ref<Client | null>(null)
const clientForm = ref<CreateClientParams>({
  remark: '',
  uuid: '',
  password: '',
  flow: '',
  enable: true,
  total: 0,
  expiryTime: 0
})

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

const columns: DataTableColumns<Client> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true } },
  { 
    title: '状态', 
    key: 'enable',
    width: 80,
    render: (row) => h(NTag, { type: row.enable ? 'success' : 'error', size: 'small' }, { default: () => row.enable ? '启用' : '禁用' })
  },
  {
    title: 'UUID/密码',
    key: 'uuid',
    width: 150,
    ellipsis: { tooltip: true },
    render: (row) => row.uuid || row.password || '-'
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
    width: 150,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', quaternary: true, onClick: () => handleShowQR(row) }, { icon: () => h(NIcon, null, { default: () => h(QrCodeOutline) }) }),
        h(NButton, { size: 'small', quaternary: true, onClick: () => handleEdit(row) }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }),
        h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => '确定删除该客户端?'
        })
      ]
    })
  }
]

async function fetchInbounds() {
  try {
    const res = await inboundApi.list()
    inbounds.value = res.data.data || []
    inboundOptions.value = inbounds.value.map(i => ({
      label: `${i.remark || i.tag || '入站'} (${i.protocol}:${i.port})`,
      value: i.id
    }))
    if (inbounds.value.length > 0 && !selectedInboundId.value) {
      selectedInboundId.value = inbounds.value[0].id
    }
  } catch (error: any) {
    message.error(error.message || '获取入站列表失败')
  }
}

async function fetchClients() {
  if (!selectedInboundId.value) {
    clients.value = []
    return
  }
  loading.value = true
  try {
    const res = await inboundApi.listClients(selectedInboundId.value)
    clients.value = res.data.data || []
  } catch (error: any) {
    message.error(error.message || '获取客户端列表失败')
  } finally {
    loading.value = false
  }
}

function getCurrentInbound(): Inbound | undefined {
  return inbounds.value.find(i => i.id === selectedInboundId.value)
}

function handleAdd() {
  editingClient.value = null
  const inbound = getCurrentInbound()
  clientForm.value = {
    remark: '',
    uuid: ['vmess', 'vless'].includes(inbound?.protocol || '') ? uuidv4() : '',
    password: '',
    flow: '',
    enable: true,
    total: 0,
    expiryTime: 0
  }
  showClientModal.value = true
}

function handleEdit(row: Client) {
  editingClient.value = row
  clientForm.value = {
    remark: row.remark || '',
    uuid: row.uuid,
    password: row.password,
    flow: row.flow,
    enable: row.enable,
    total: row.total,
    expiryTime: row.expiryTime
  }
  showClientModal.value = true
}

async function handleSubmit() {
  if (!selectedInboundId.value) return

  try {
    if (editingClient.value) {
      await inboundApi.updateClient(selectedInboundId.value, editingClient.value.id, clientForm.value)
      message.success('更新成功')
    } else {
      await inboundApi.addClient(selectedInboundId.value, clientForm.value)
      message.success('添加成功')
    }
    showClientModal.value = false
    fetchClients()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

async function handleDelete(clientId: number) {
  if (!selectedInboundId.value) return
  try {
    await inboundApi.deleteClient(selectedInboundId.value, clientId)
    message.success('删除成功')
    fetchClients()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

async function handleShowQR(row: Client) {
  const inbound = getCurrentInbound()
  if (!inbound) return

  // 使用 link.ts 生成链接
  const { generateClientLink } = await import('@/utils/link')
  const link = generateClientLink(row, inbound)
  
  if (!link) {
    message.warning('该协议不支持生成二维码')
    return
  }

  qrTitle.value = row.remark || 'client'
  qrLink.value = link
  showQRModal.value = true
}

function generateUUID() {
  clientForm.value.uuid = uuidv4()
}

function generatePassword() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < 16; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  clientForm.value.password = result
}

watch(selectedInboundId, () => {
  fetchClients()
})

onMounted(() => {
  fetchInbounds()
})
</script>

<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h2 style="margin: 0;">客户端管理</h2>
      <n-space>
        <n-select
          v-model:value="selectedInboundId"
          :options="inboundOptions"
          placeholder="选择入站规则"
          style="width: 280px;"
        />
        <n-button @click="fetchClients">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
        <n-button type="primary" :disabled="!selectedInboundId" @click="handleAdd">
          <template #icon><n-icon :component="AddOutline" /></template>
          添加客户端
        </n-button>
      </n-space>
    </n-space>

    <n-data-table
      :columns="columns"
      :data="clients"
      :loading="loading"
      :bordered="false"
      :row-key="(row: Client) => row.id"
    />

    <!-- 客户端编辑弹窗 -->
    <n-modal
      v-model:show="showClientModal"
      :title="editingClient ? '编辑客户端' : '添加客户端'"
      preset="card"
      style="width: 500px;"
    >
      <n-form label-placement="left" label-width="100">
        <n-form-item label="备注">
          <n-input v-model:value="clientForm.remark" placeholder="可选，用于标识客户端" />
        </n-form-item>

        <n-form-item v-if="['vmess', 'vless'].includes(getCurrentInbound()?.protocol || '')" label="UUID">
          <n-space>
            <n-input v-model:value="clientForm.uuid" style="width: 280px;" />
            <n-button @click="generateUUID">生成</n-button>
          </n-space>
        </n-form-item>

        <n-form-item v-if="getCurrentInbound()?.protocol === 'vless'" label="Flow">
          <n-select
            v-model:value="clientForm.flow"
            :options="[{ label: '无', value: '' }, { label: 'xtls-rprx-vision', value: 'xtls-rprx-vision' }]"
          />
        </n-form-item>

        <n-form-item v-if="['trojan', 'shadowsocks'].includes(getCurrentInbound()?.protocol || '')" label="密码">
          <n-space>
            <n-input v-model:value="clientForm.password" style="width: 280px;" />
            <n-button @click="generatePassword">生成</n-button>
          </n-space>
        </n-form-item>

        <n-form-item label="启用">
          <n-switch v-model:value="clientForm.enable" />
        </n-form-item>

        <n-form-item label="流量限制 (GB)">
          <n-input-number
            :value="clientForm.total ? clientForm.total / (1024 * 1024 * 1024) : 0"
            @update:value="(v: number | null) => clientForm.total = (v || 0) * 1024 * 1024 * 1024"
            :min="0"
            :precision="2"
            placeholder="0 表示不限制"
            style="width: 100%;"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showClientModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit">{{ editingClient ? '保存' : '添加' }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <QRCodeModal
      v-model:show="showQRModal"
      :title="qrTitle"
      :link="qrLink"
    />
  </div>
</template>
