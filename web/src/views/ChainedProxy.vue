<script setup lang="ts">
import { ref, onMounted, h, watch, computed } from 'vue'
import { NCard, NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, NSelect, NModal, NForm, NFormItem, NInput, NInputNumber, NSwitch, NAlert, useMessage } from 'naive-ui'
import type { DataTableColumns, SelectOption } from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, CreateOutline } from '@vicons/ionicons5'
import { proxyApi, type ChainedProxy, type ProxyUpsertParams } from '@/api/proxy'
import { inboundApi, type Inbound, type Client } from '@/api/inbound'

const message = useMessage()

// ===== 上半部：上游代理管理 =====
const proxies = ref<ChainedProxy[]>([])
const proxyLoading = ref(false)
const submitting = ref(false)

const showProxyModal = ref(false)
const editingProxy = ref<ChainedProxy | null>(null)
const proxyForm = ref<ProxyUpsertParams>({
  remark: '',
  protocol: 'socks',
  host: '',
  port: 0,
  username: '',
  password: '',
  enable: true,
  raw: ''
})

const protocolOptions: SelectOption[] = [
  { label: 'SOCKS5', value: 'socks' },
  { label: 'HTTP', value: 'http' }
]

const proxyColumns: DataTableColumns<ChainedProxy> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true }, render: (row) => row.remark || '-' },
  {
    title: '协议',
    key: 'protocol',
    width: 90,
    render: (row) => h(NTag, { size: 'small', type: 'info' }, { default: () => (row.protocol === 'http' ? 'HTTP' : 'SOCKS5') })
  },
  { title: '地址', key: 'host', ellipsis: { tooltip: true }, render: (row) => `${row.host}:${row.port}` },
  { title: '用户名', key: 'username', ellipsis: { tooltip: true }, render: (row) => row.username || '-' },
  {
    title: '状态',
    key: 'enable',
    width: 80,
    render: (row) => h(NTag, { type: row.enable ? 'success' : 'error', size: 'small' }, { default: () => (row.enable ? '启用' : '禁用') })
  },
  {
    title: '操作',
    key: 'actions',
    width: 110,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', quaternary: true, onClick: () => handleEditProxy(row) }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }),
        h(NPopconfirm, { onPositiveClick: () => handleDeleteProxy(row.id) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => '删除后，所有使用该代理的客户端将恢复直连。确定删除?'
        })
      ]
    })
  }
]

async function fetchProxies() {
  proxyLoading.value = true
  try {
    const res = await proxyApi.list()
    proxies.value = res.data.data || []
  } catch (error: any) {
    message.error(error.message || '获取代理列表失败')
  } finally {
    proxyLoading.value = false
  }
}

function handleAddProxy() {
  editingProxy.value = null
  proxyForm.value = { remark: '', protocol: 'socks', host: '', port: 0, username: '', password: '', enable: true, raw: '' }
  showProxyModal.value = true
}

function handleEditProxy(row: ChainedProxy) {
  editingProxy.value = row
  proxyForm.value = {
    remark: row.remark,
    protocol: row.protocol || 'socks',
    host: row.host,
    port: row.port,
    username: row.username,
    password: row.password,
    enable: row.enable,
    raw: ''
  }
  showProxyModal.value = true
}

async function handleSubmitProxy() {
  if (submitting.value) return
  submitting.value = true
  try {
    // raw 优先：若填了粘贴串，仅传 raw + 协议/备注/启用
    const payload: ProxyUpsertParams = { ...proxyForm.value }
    if (payload.raw && payload.raw.trim()) {
      payload.host = undefined
      payload.port = undefined
      payload.username = undefined
      payload.password = undefined
    } else {
      payload.raw = ''
    }
    if (editingProxy.value) {
      await proxyApi.update(editingProxy.value.id, payload)
      message.success('更新成功')
    } else {
      await proxyApi.create(payload)
      message.success('添加成功')
    }
    showProxyModal.value = false
    await fetchProxies()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleDeleteProxy(id: number) {
  try {
    await proxyApi.delete(id)
    message.success('删除成功')
    await fetchProxies()
    await fetchClients() // 刷新映射区，被解绑的客户端会恢复直连
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// ===== 下半部：客户端 → 代理 映射 =====
const inbounds = ref<Inbound[]>([])
const inboundOptions = ref<SelectOption[]>([])
const selectedInboundId = ref<number | null>(null)
const clients = ref<Client[]>([])
const clientLoading = ref(false)

// 代理下拉选项：第一个永远是「不走代理（直连）」
const proxySelectOptions = computed<SelectOption[]>(() => [
  { label: '不走代理（直连）', value: 0 },
  ...proxies.value.map((p) => ({
    label: `${p.remark ? p.remark + ' · ' : ''}${p.host}:${p.port} [${p.protocol === 'http' ? 'HTTP' : 'SOCKS5'}]${p.enable ? '' : '（已禁用）'}`,
    value: p.id
  }))
])

const mappingColumns: DataTableColumns<Client> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '客户端备注', key: 'remark', ellipsis: { tooltip: true }, render: (row) => row.remark || '-' },
  { title: 'UUID/密码', key: 'uuid', ellipsis: { tooltip: true }, render: (row) => row.uuid || row.password || '-' },
  {
    title: '出口代理',
    key: 'proxyId',
    width: 320,
    render: (row) =>
      h(NSelect, {
        value: row.proxyId ?? 0,
        options: proxySelectOptions.value,
        size: 'small',
        style: 'width: 300px;',
        'onUpdate:value': (val: number) => handleAssignProxy(row, val)
      })
  }
]

async function fetchInbounds() {
  try {
    const res = await inboundApi.list()
    inbounds.value = res.data.data || []
    inboundOptions.value = inbounds.value.map((i) => ({
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
  clientLoading.value = true
  try {
    const res = await inboundApi.listClients(selectedInboundId.value)
    clients.value = res.data.data || []
  } catch (error: any) {
    message.error(error.message || '获取客户端列表失败')
  } finally {
    clientLoading.value = false
  }
}

async function handleAssignProxy(row: Client, val: number) {
  const proxyId = val && val > 0 ? val : null
  try {
    await proxyApi.setClientProxy(row.id, proxyId)
    row.proxyId = proxyId
    message.success(proxyId ? '已设置出口代理' : '已恢复直连')
  } catch (error: any) {
    message.error(error.message || '设置失败')
    await fetchClients()
  }
}

watch(selectedInboundId, () => {
  fetchClients()
})

onMounted(() => {
  fetchProxies()
  fetchInbounds()
})
</script>

<template>
  <div>
    <h2 style="margin: 0 0 16px;">链式代理</h2>

    <n-alert type="info" :bordered="false" style="margin-bottom: 16px;">
      在此配置上游代理后，可在下方「客户端代理映射」中为某个客户端指定出口代理；其流量将经该上游代理转发。未指定的客户端默认直连（不走代理）。
    </n-alert>

    <!-- 上游代理管理 -->
    <n-card title="上游代理" :bordered="false" style="margin-bottom: 16px;">
      <template #header-extra>
        <n-space>
          <n-button @click="fetchProxies">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            刷新
          </n-button>
          <n-button type="primary" @click="handleAddProxy">
            <template #icon><n-icon :component="AddOutline" /></template>
            添加代理
          </n-button>
        </n-space>
      </template>

      <n-data-table
        :columns="proxyColumns"
        :data="proxies"
        :loading="proxyLoading"
        :bordered="false"
        :row-key="(row: ChainedProxy) => row.id"
      />
    </n-card>

    <!-- 客户端 → 代理 映射 -->
    <n-card title="客户端代理映射" :bordered="false">
      <template #header-extra>
        <n-space align="center">
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
        </n-space>
      </template>

      <n-data-table
        :columns="mappingColumns"
        :data="clients"
        :loading="clientLoading"
        :bordered="false"
        :row-key="(row: Client) => row.id"
      />
    </n-card>

    <!-- 代理编辑弹窗 -->
    <n-modal
      v-model:show="showProxyModal"
      :title="editingProxy ? '编辑代理' : '添加代理'"
      preset="card"
      style="width: 520px;"
    >
      <n-form label-placement="left" label-width="92">
        <n-form-item v-if="!editingProxy" label="快速粘贴">
          <n-input
            v-model:value="proxyForm.raw"
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 2 }"
            placeholder="host:port:user:pass，例如 sg.arxlabs.io:443:用户名:密码（填写后将忽略下方字段）"
          />
        </n-form-item>

        <n-form-item label="备注">
          <n-input v-model:value="proxyForm.remark" placeholder="可选，便于识别" />
        </n-form-item>

        <n-form-item label="协议">
          <n-select v-model:value="proxyForm.protocol" :options="protocolOptions" />
        </n-form-item>

        <n-form-item label="主机地址">
          <n-input v-model:value="proxyForm.host" placeholder="如 sg.arxlabs.io" />
        </n-form-item>

        <n-form-item label="端口">
          <n-input-number v-model:value="proxyForm.port" :min="1" :max="65535" placeholder="如 443" style="width: 100%;" />
        </n-form-item>

        <n-form-item label="用户名">
          <n-input v-model:value="proxyForm.username" placeholder="可选" />
        </n-form-item>

        <n-form-item label="密码">
          <n-input v-model:value="proxyForm.password" placeholder="可选" />
        </n-form-item>

        <n-form-item label="启用">
          <n-switch v-model:value="proxyForm.enable" />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button :disabled="submitting" @click="showProxyModal = false">取消</n-button>
          <n-button type="primary" :loading="submitting" :disabled="submitting" @click="handleSubmitProxy">
            {{ editingProxy ? '保存' : '添加' }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>
