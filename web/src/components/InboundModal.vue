<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NSwitch, NButton, NSpace, NDatePicker, NTooltip, NIcon, NCard, NGrid, NGridItem, NAlert, useMessage } from 'naive-ui'
import { HelpCircleOutline } from '@vicons/ionicons5'
import type { CreateInboundParams } from '@/api/inbound'
import { certificateApi, type Certificate } from '@/api/certificate'
import { v4 as uuidv4 } from 'uuid'

const props = defineProps<{
  show: boolean
  editData?: any
}>()

const message = useMessage()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'submit', data: CreateInboundParams): void
}>()

const isEdit = computed(() => !!props.editData?.id)
const title = computed(() => isEdit.value ? '编辑入站规则' : '添加入站规则')

const formData = ref<CreateInboundParams>({
  remark: '',
  enable: true,
  listen: '',
  port: 0,
  protocol: 'vmess',
  settings: '',
  streamSettings: '',
  sniffing: '',
  tag: '',
  total: 0,
  expiryTime: 0
})

// 流量限制（GB）
const totalGB = computed({
  get: () => formData.value.total ? formData.value.total / (1024 * 1024 * 1024) : 0,
  set: (v: number) => { formData.value.total = v * 1024 * 1024 * 1024 }
})

// 到期时间
const expiryDate = computed({
  get: () => formData.value.expiryTime ? formData.value.expiryTime : null,
  set: (v: number | null) => { formData.value.expiryTime = v || 0 }
})

// 协议特定设置
const protocolSettings = ref({
  // VMess/VLESS
  uuid: '',
  alterId: 0,
  flow: '',
  // Shadowsocks
  method: 'aes-256-gcm',
  password: '',
  // Trojan
  trojanPassword: '',
  // SOCKS/HTTP
  auth: 'noauth',
  user: '',
  pass: '',
  // Dokodemo-door
  address: '',
  followRedirect: false,
  network: 'tcp,udp'
})

// 传输设置
const streamSettings = ref({
  network: 'tcp',
  security: 'none',
  // TLS
  serverName: '',
  // TCP
  tcpHeaderType: 'none',
  // WS
  wsPath: '/',
  wsHost: '',
  // gRPC
  grpcServiceName: '',
  // KCP
  kcpType: 'none',
  kcpSeed: ''
})

// 嗅探设置
const sniffingSettings = ref({
  enabled: true,
  destOverride: ['http', 'tls']
})

const certificates = ref<Certificate[]>([])
const selectedCertificateId = ref<number | null>(null)
const tlsCertificateFile = ref('')
const tlsKeyFile = ref('')

const protocolOptions = [
  { label: 'vmess', value: 'vmess' },
  { label: 'vless', value: 'vless' },
  { label: 'trojan', value: 'trojan' },
  { label: 'shadowsocks', value: 'shadowsocks' },
  { label: 'dokodemo-door', value: 'dokodemo-door' },
  { label: 'socks', value: 'socks' },
  { label: 'http', value: 'http' }
]

const networkOptions = [
  { label: 'tcp', value: 'tcp' },
  { label: 'kcp', value: 'kcp' },
  { label: 'ws', value: 'ws' },
  { label: 'http', value: 'http' },
  { label: 'quic', value: 'quic' },
  { label: 'grpc', value: 'grpc' }
]

const securityOptions = [
  { label: 'none', value: 'none' },
  { label: 'tls', value: 'tls' }
]

const ssMethodOptions = [
  { label: 'aes-256-gcm', value: 'aes-256-gcm' },
  { label: 'aes-128-gcm', value: 'aes-128-gcm' },
  { label: 'chacha20-poly1305', value: 'chacha20-poly1305' }
]

const headerTypeOptions = [
  { label: 'none', value: 'none' },
  { label: 'http', value: 'http' }
]

const flowOptions = [
  { label: '无', value: '' },
  { label: 'xtls-rprx-vision', value: 'xtls-rprx-vision' },
  { label: 'xtls-rprx-direct', value: 'xtls-rprx-direct' }
]

const certificateOptions = computed(() => certificates.value.map(c => ({
  label: `${c.domain || '未命名'}${c.remark ? ` (${c.remark})` : ''}`,
  value: c.id
})))

// 是否可以设置传输
const canEnableStream = computed(() => 
  ['vmess', 'vless', 'trojan', 'shadowsocks'].includes(formData.value.protocol)
)

// 是否可以设置 TLS
const canEnableTls = computed(() => 
  ['vmess', 'vless', 'trojan', 'shadowsocks'].includes(formData.value.protocol) &&
  ['tcp', 'ws', 'http', 'grpc'].includes(streamSettings.value.network)
)

// 是否可以设置嗅探
const canSniffing = computed(() => 
  ['vmess', 'vless', 'trojan', 'shadowsocks'].includes(formData.value.protocol)
)

function generateUUID() {
  protocolSettings.value.uuid = uuidv4()
}

function generatePassword() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < 16; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

function generateSsPassword() {
  protocolSettings.value.password = generatePassword()
}

function generateTrojanPassword() {
  protocolSettings.value.trojanPassword = generatePassword()
}

async function loadCertificates() {
  try {
    const res = await certificateApi.list()
    certificates.value = res.data.data || []
  } catch {
    certificates.value = []
  }
}

function handleCertificateChange(id: number | null) {
  selectedCertificateId.value = id
  const cert = certificates.value.find(c => c.id === id)
  if (!cert) return
  tlsCertificateFile.value = cert.certFile || ''
  tlsKeyFile.value = cert.keyFile || ''
}

function openCertificateManager() {
  window.open('#/certificates', '_blank')
}

function buildSettings(): string {
  const protocol = formData.value.protocol
  let settings: any = {}

  switch (protocol) {
    case 'vmess':
      settings = {
        clients: [{
          id: protocolSettings.value.uuid || uuidv4(),
          alterId: protocolSettings.value.alterId || 0
        }]
      }
      break
    case 'vless':
      settings = {
        clients: [{
          id: protocolSettings.value.uuid || uuidv4(),
          flow: protocolSettings.value.flow || ''
        }],
        decryption: 'none'
      }
      break
    case 'trojan':
      settings = {
        clients: [{
          password: protocolSettings.value.trojanPassword || generatePassword()
        }]
      }
      break
    case 'shadowsocks':
      settings = {
        method: protocolSettings.value.method,
        password: protocolSettings.value.password || generatePassword(),
        network: 'tcp,udp'
      }
      break
    case 'dokodemo-door':
      settings = {
        address: protocolSettings.value.address,
        port: formData.value.port,
        network: protocolSettings.value.network,
        followRedirect: protocolSettings.value.followRedirect
      }
      break
    case 'socks':
    case 'http':
      if (protocolSettings.value.auth === 'password') {
        settings = {
          auth: 'password',
          accounts: [{ user: protocolSettings.value.user, pass: protocolSettings.value.pass }]
        }
      }
      break
  }

  return JSON.stringify(settings)
}

function buildStreamSettings(): string {
  const ss = streamSettings.value
  const result: any = {
    network: ss.network,
    security: ss.security
  }

  switch (ss.network) {
    case 'tcp':
      result.tcpSettings = {
        header: { type: ss.tcpHeaderType }
      }
      break
    case 'kcp':
      result.kcpSettings = {
        header: { type: ss.kcpType },
        seed: ss.kcpSeed
      }
      break
    case 'ws':
      result.wsSettings = {
        path: ss.wsPath,
        headers: ss.wsHost ? { Host: ss.wsHost } : {}
      }
      break
    case 'grpc':
      result.grpcSettings = {
        serviceName: ss.grpcServiceName
      }
      break
  }

  if (ss.security === 'tls') {
    const certificates: any[] = []
    if (tlsCertificateFile.value.trim() && tlsKeyFile.value.trim()) {
      certificates.push({
        certificateFile: tlsCertificateFile.value.trim(),
        keyFile: tlsKeyFile.value.trim()
      })
    }

    result.tlsSettings = {
      serverName: ss.serverName,
      ...(certificates.length ? { certificates } : {})
    }
  }

  return JSON.stringify(result)
}

function buildSniffing(): string {
  return JSON.stringify(sniffingSettings.value)
}

function validateForm(): string | null {
  if (!formData.value.port || formData.value.port < 1 || formData.value.port > 65535) {
    return '端口必须在 1-65535 之间'
  }

  const protocol = formData.value.protocol
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

  if (protocol === 'vmess' || protocol === 'vless') {
    if (!protocolSettings.value.uuid || !uuidRegex.test(protocolSettings.value.uuid.trim())) {
      return 'UUID 格式不正确'
    }
  }

  if (protocol === 'trojan' && !protocolSettings.value.trojanPassword?.trim()) {
    return 'Trojan 密码不能为空'
  }

  if (protocol === 'shadowsocks') {
    if (!protocolSettings.value.method) return 'Shadowsocks 加密方式不能为空'
    if (!protocolSettings.value.password?.trim()) return 'Shadowsocks 密码不能为空'
  }

  if (protocol === 'dokodemo-door' && !protocolSettings.value.address?.trim()) {
    return 'Dokodemo-door 目标地址不能为空'
  }

  if ((protocol === 'socks' || protocol === 'http') && protocolSettings.value.auth === 'password') {
    if (!protocolSettings.value.user?.trim() || !protocolSettings.value.pass?.trim()) {
      return '账号认证模式下，用户名和密码不能为空'
    }
  }

  if (canEnableStream.value) {
    if (streamSettings.value.network === 'ws' && !streamSettings.value.wsPath?.trim()) {
      return 'WS 模式下 Path 不能为空'
    }
    if (streamSettings.value.network === 'grpc' && !streamSettings.value.grpcServiceName?.trim()) {
      return 'gRPC 模式下 ServiceName 不能为空'
    }
    if (streamSettings.value.security === 'tls' && !streamSettings.value.serverName?.trim()) {
      return 'TLS 模式下 SNI/ServerName 不能为空'
    }
    if (streamSettings.value.security === 'tls') {
      const hasCertPair = !!tlsCertificateFile.value.trim() && !!tlsKeyFile.value.trim()
      if (!hasCertPair) {
        return 'TLS 模式下请先选择证书，或手动填写证书/私钥文件路径'
      }
    }
  }

  return null
}

function handleSubmit() {
  const err = validateForm()
  if (err) {
    message.error(err)
    return
  }

  const data: CreateInboundParams = {
    ...formData.value,
    settings: buildSettings(),
    streamSettings: buildStreamSettings(),
    sniffing: buildSniffing(),
    tag: formData.value.tag || `inbound-${formData.value.port}`
  }
  emit('submit', data)
}

function handleClose() {
  emit('update:show', false)
}

function parseJSON<T = any>(text?: string, fallback: T = {} as T): T {
  if (!text) return fallback
  try {
    return JSON.parse(text) as T
  } catch {
    return fallback
  }
}

function resetForCreate() {
  formData.value = {
    remark: '',
    enable: true,
    listen: '',
    port: Math.floor(Math.random() * 40000) + 10000,
    protocol: 'vmess',
    settings: '',
    streamSettings: '',
    sniffing: '',
    tag: '',
    total: 0,
    expiryTime: 0
  }
  protocolSettings.value = {
    uuid: uuidv4(),
    alterId: 0,
    flow: '',
    method: 'aes-256-gcm',
    password: generatePassword(),
    trojanPassword: generatePassword(),
    auth: 'noauth',
    user: '',
    pass: '',
    address: '',
    followRedirect: false,
    network: 'tcp,udp'
  }
  streamSettings.value = {
    network: 'tcp',
    security: 'none',
    serverName: '',
    tcpHeaderType: 'none',
    wsPath: '/',
    wsHost: '',
    grpcServiceName: '',
    kcpType: 'none',
    kcpSeed: ''
  }
  sniffingSettings.value = {
    enabled: true,
    destOverride: ['http', 'tls']
  }
  selectedCertificateId.value = null
  tlsCertificateFile.value = ''
  tlsKeyFile.value = ''
}

function fillForEdit() {
  if (!props.editData) return

  formData.value = {
    remark: props.editData.remark || '',
    enable: !!props.editData.enable,
    listen: props.editData.listen || '',
    port: props.editData.port || 0,
    protocol: props.editData.protocol || 'vmess',
    settings: props.editData.settings || '',
    streamSettings: props.editData.streamSettings || '',
    sniffing: props.editData.sniffing || '',
    tag: props.editData.tag || '',
    total: props.editData.total || 0,
    expiryTime: props.editData.expiryTime || 0
  }

  const ps = parseJSON<any>(props.editData.settings, {})
  const ss = parseJSON<any>(props.editData.streamSettings, {})
  const sn = parseJSON<any>(props.editData.sniffing, { enabled: true, destOverride: ['http', 'tls'] })

  const firstClient = Array.isArray(ps.clients) && ps.clients.length > 0 ? ps.clients[0] : {}
  protocolSettings.value.uuid = firstClient.id || protocolSettings.value.uuid || uuidv4()
  protocolSettings.value.alterId = firstClient.alterId || 0
  protocolSettings.value.flow = firstClient.flow || ''
  protocolSettings.value.trojanPassword = firstClient.password || ''
  protocolSettings.value.method = ps.method || 'aes-256-gcm'
  protocolSettings.value.password = ps.password || ''
  protocolSettings.value.address = ps.address || ''
  protocolSettings.value.network = ps.network || 'tcp,udp'
  protocolSettings.value.followRedirect = !!ps.followRedirect
  protocolSettings.value.auth = ps.auth || 'noauth'
  if (Array.isArray(ps.accounts) && ps.accounts.length > 0) {
    protocolSettings.value.user = ps.accounts[0]?.user || ''
    protocolSettings.value.pass = ps.accounts[0]?.pass || ''
  } else {
    protocolSettings.value.user = ''
    protocolSettings.value.pass = ''
  }

  streamSettings.value.network = ss.network || 'tcp'
  streamSettings.value.security = ss.security || 'none'
  streamSettings.value.serverName = ss.tlsSettings?.serverName || ''
  streamSettings.value.tcpHeaderType = ss.tcpSettings?.header?.type || 'none'
  streamSettings.value.wsPath = ss.wsSettings?.path || '/'
  streamSettings.value.wsHost = ss.wsSettings?.headers?.Host || ''
  streamSettings.value.grpcServiceName = ss.grpcSettings?.serviceName || ''
  streamSettings.value.kcpType = ss.kcpSettings?.header?.type || 'none'
  streamSettings.value.kcpSeed = ss.kcpSettings?.seed || ''

  const certPair = Array.isArray(ss.tlsSettings?.certificates) && ss.tlsSettings.certificates.length > 0
    ? ss.tlsSettings.certificates[0]
    : null
  tlsCertificateFile.value = certPair?.certificateFile || ''
  tlsKeyFile.value = certPair?.keyFile || ''

  const matched = certificates.value.find(c => c.certFile === tlsCertificateFile.value && c.keyFile === tlsKeyFile.value)
  selectedCertificateId.value = matched?.id ?? null

  sniffingSettings.value = {
    enabled: !!sn.enabled,
    destOverride: Array.isArray(sn.destOverride) ? sn.destOverride : ['http', 'tls']
  }
}

// 监听显示状态
watch(() => props.show, async (show) => {
  if (!show) return

  await loadCertificates()

  if (props.editData) {
    fillForEdit()
  } else {
    resetForCreate()
  }
})
</script>

<template>
  <n-modal
    :show="show"
    :title="title"
    preset="card"
    style="width: 750px; max-height: 90vh;"
    :mask-closable="false"
    @update:show="handleClose"
  >
    <div style="max-height: 70vh; overflow-y: auto; padding-right: 10px;">
      <!-- 基础设置 -->
      <n-card size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-grid :cols="2" :x-gap="24">
            <n-grid-item>
              <n-form-item label="备注">
                <n-input v-model:value="formData.remark" placeholder="" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item>
              <n-form-item label="启用">
                <n-switch v-model:value="formData.enable" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item>
              <n-form-item label="协议">
                <n-select v-model:value="formData.protocol" :options="protocolOptions" style="width: 100%;" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item>
              <n-form-item>
                <template #label>
                  监听 IP
                  <n-tooltip trigger="hover">
                    <template #trigger>
                      <n-icon :component="HelpCircleOutline" style="vertical-align: middle; margin-left: 4px;" />
                    </template>
                    默认留空即可
                  </n-tooltip>
                </template>
                <n-input v-model:value="formData.listen" placeholder="" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item>
              <n-form-item label="端口">
                <n-input-number v-model:value="formData.port" :min="1" :max="65535" style="width: 100%;" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item>
              <n-form-item>
                <template #label>
                  总流量(GB)
                  <n-tooltip trigger="hover">
                    <template #trigger>
                      <n-icon :component="HelpCircleOutline" style="vertical-align: middle; margin-left: 4px;" />
                    </template>
                    0 表示不限制
                  </n-tooltip>
                </template>
                <n-input-number v-model:value="totalGB" :min="0" :precision="2" style="width: 100%;" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item :span="2">
              <n-form-item>
                <template #label>
                  到期时间
                  <n-tooltip trigger="hover">
                    <template #trigger>
                      <n-icon :component="HelpCircleOutline" style="vertical-align: middle; margin-left: 4px;" />
                    </template>
                    留空则永不到期
                  </n-tooltip>
                </template>
                <n-date-picker v-model:value="expiryDate" type="datetime" clearable style="width: 100%;" />
              </n-form-item>
            </n-grid-item>
          </n-grid>
        </n-form>
      </n-card>

      <!-- VMess 设置 -->
      <n-card v-if="formData.protocol === 'vmess'" title="VMess 设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="UUID">
            <n-space>
              <n-input v-model:value="protocolSettings.uuid" style="width: 340px;" />
              <n-button @click="generateUUID">生成</n-button>
            </n-space>
          </n-form-item>
          <n-form-item label="alterId">
            <n-input-number v-model:value="protocolSettings.alterId" :min="0" style="width: 150px;" />
          </n-form-item>
        </n-form>
      </n-card>

      <!-- VLESS 设置 -->
      <n-card v-if="formData.protocol === 'vless'" title="VLESS 设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="UUID">
            <n-space>
              <n-input v-model:value="protocolSettings.uuid" style="width: 340px;" />
              <n-button @click="generateUUID">生成</n-button>
            </n-space>
          </n-form-item>
          <n-form-item label="flow">
            <n-select v-model:value="protocolSettings.flow" :options="flowOptions" style="width: 200px;" />
          </n-form-item>
        </n-form>
      </n-card>

      <!-- Trojan 设置 -->
      <n-card v-if="formData.protocol === 'trojan'" title="Trojan 设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="密码">
            <n-space>
              <n-input v-model:value="protocolSettings.trojanPassword" style="width: 340px;" />
              <n-button @click="generateTrojanPassword">生成</n-button>
            </n-space>
          </n-form-item>
        </n-form>
      </n-card>

      <!-- Shadowsocks 设置 -->
      <n-card v-if="formData.protocol === 'shadowsocks'" title="Shadowsocks 设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="加密方式">
            <n-select v-model:value="protocolSettings.method" :options="ssMethodOptions" style="width: 200px;" />
          </n-form-item>
          <n-form-item label="密码">
            <n-space>
              <n-input v-model:value="protocolSettings.password" style="width: 340px;" />
              <n-button @click="generateSsPassword">生成</n-button>
            </n-space>
          </n-form-item>
        </n-form>
      </n-card>

      <!-- Dokodemo-door 设置 -->
      <n-card v-if="formData.protocol === 'dokodemo-door'" title="Dokodemo-door 设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="目标地址">
            <n-input v-model:value="protocolSettings.address" placeholder="转发目标地址" />
          </n-form-item>
          <n-form-item label="网络类型">
            <n-input v-model:value="protocolSettings.network" placeholder="tcp,udp" />
          </n-form-item>
          <n-form-item label="透明代理">
            <n-switch v-model:value="protocolSettings.followRedirect" />
          </n-form-item>
        </n-form>
      </n-card>

      <!-- SOCKS/HTTP 设置 -->
      <n-card v-if="formData.protocol === 'socks' || formData.protocol === 'http'" :title="formData.protocol.toUpperCase() + ' 设置'" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="认证方式">
            <n-select v-model:value="protocolSettings.auth" :options="[{ label: '无认证', value: 'noauth' }, { label: '密码认证', value: 'password' }]" style="width: 150px;" />
          </n-form-item>
          <template v-if="protocolSettings.auth === 'password'">
            <n-form-item label="用户名">
              <n-input v-model:value="protocolSettings.user" style="width: 200px;" />
            </n-form-item>
            <n-form-item label="密码">
              <n-input v-model:value="protocolSettings.pass" style="width: 200px;" />
            </n-form-item>
          </template>
        </n-form>
      </n-card>

      <!-- 传输设置 -->
      <n-card v-if="canEnableStream" title="传输设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="80" label-placement="left">
          <n-grid :cols="2" :x-gap="24">
            <n-grid-item>
              <n-form-item label="传输协议">
                <n-select v-model:value="streamSettings.network" :options="networkOptions" style="width: 100%;" />
              </n-form-item>
            </n-grid-item>
            <n-grid-item v-if="canEnableTls">
              <n-form-item label="安全性">
                <n-select v-model:value="streamSettings.security" :options="securityOptions" style="width: 100%;" />
              </n-form-item>
            </n-grid-item>
          </n-grid>

          <!-- TCP 设置 -->
          <template v-if="streamSettings.network === 'tcp'">
            <n-form-item label="伪装类型">
              <n-select v-model:value="streamSettings.tcpHeaderType" :options="headerTypeOptions" style="width: 150px;" />
            </n-form-item>
          </template>

          <!-- KCP 设置 -->
          <template v-if="streamSettings.network === 'kcp'">
            <n-form-item label="伪装类型">
              <n-select v-model:value="streamSettings.kcpType" :options="headerTypeOptions" style="width: 150px;" />
            </n-form-item>
            <n-form-item label="seed">
              <n-input v-model:value="streamSettings.kcpSeed" placeholder="可选" style="width: 200px;" />
            </n-form-item>
          </template>

          <!-- WebSocket 设置 -->
          <template v-if="streamSettings.network === 'ws'">
            <n-form-item label="路径">
              <n-input v-model:value="streamSettings.wsPath" placeholder="/" />
            </n-form-item>
            <n-form-item label="Host">
              <n-input v-model:value="streamSettings.wsHost" placeholder="可选" />
            </n-form-item>
          </template>

          <!-- gRPC 设置 -->
          <template v-if="streamSettings.network === 'grpc'">
            <n-form-item label="serviceName">
              <n-input v-model:value="streamSettings.grpcServiceName" />
            </n-form-item>
          </template>
        </n-form>
      </n-card>

      <!-- TLS 设置 -->
      <n-card v-if="canEnableTls && streamSettings.security === 'tls'" title="TLS 设置" size="small" style="margin-bottom: 16px;">
        <n-form :label-width="110" label-placement="left">
          <n-form-item label="服务器名称">
            <n-input v-model:value="streamSettings.serverName" placeholder="SNI，建议填写域名" />
          </n-form-item>

          <n-alert v-if="certificates.length === 0" type="warning" style="margin-bottom: 12px;">
            当前未发现可用证书，请先到“证书管理”添加证书，或手动填写证书/私钥文件路径。
            <n-button text type="primary" style="margin-left: 8px;" @click="openCertificateManager">去证书管理</n-button>
          </n-alert>

          <n-form-item label="选择证书">
            <n-select
              :value="selectedCertificateId"
              :options="certificateOptions"
              clearable
              placeholder="从证书管理中选择"
              @update:value="handleCertificateChange"
            />
          </n-form-item>

          <n-form-item label="证书文件">
            <n-input v-model:value="tlsCertificateFile" placeholder="如: /etc/ssl/certs/fullchain.pem" />
          </n-form-item>

          <n-form-item label="私钥文件">
            <n-input v-model:value="tlsKeyFile" placeholder="如: /etc/ssl/private/privkey.pem" />
          </n-form-item>
        </n-form>
      </n-card>

      <!-- 嗅探设置 -->
      <n-card v-if="canSniffing" title="嗅探设置" size="small">
        <n-form :label-width="80" label-placement="left">
          <n-form-item label="启用嗅探">
            <n-switch v-model:value="sniffingSettings.enabled" />
          </n-form-item>
          <n-form-item v-if="sniffingSettings.enabled" label="目标覆盖">
            <n-select
              v-model:value="sniffingSettings.destOverride"
              multiple
              :options="[
                { label: 'http', value: 'http' },
                { label: 'tls', value: 'tls' },
                { label: 'quic', value: 'quic' },
                { label: 'fakedns', value: 'fakedns' }
              ]"
              style="width: 300px;"
            />
          </n-form-item>
        </n-form>
      </n-card>
    </div>

    <template #footer>
      <n-space justify="end">
        <n-button @click="handleClose">取消</n-button>
        <n-button type="primary" @click="handleSubmit">{{ isEdit ? '保存' : '添加' }}</n-button>
      </n-space>
    </template>
  </n-modal>
</template>
