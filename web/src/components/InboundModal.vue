<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NSwitch, NTabs, NTabPane, NButton, NSpace, NDivider } from 'naive-ui'
import type { CreateInboundParams } from '@/api/inbound'
import { v4 as uuidv4 } from 'uuid'

const props = defineProps<{
  show: boolean
  editData?: any
}>()

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

// 协议特定设置
const protocolSettings = ref({
  // VMess/VLESS
  clients: [{ id: '', email: '', alterId: 0, flow: '' }],
  // Shadowsocks
  method: 'aes-256-gcm',
  password: '',
  // Trojan
  trojanPassword: '',
  // SOCKS/HTTP
  auth: 'noauth',
  accounts: [{ user: '', pass: '' }],
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
  allowInsecure: false,
  // TCP
  tcpHeaderType: 'none',
  // WS
  wsPath: '/',
  wsHost: '',
  // gRPC
  grpcServiceName: '',
  // HTTP/2
  h2Path: '/',
  h2Host: ''
})

// 嗅探设置
const sniffingSettings = ref({
  enabled: true,
  destOverride: ['http', 'tls']
})

const protocolOptions = [
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Shadowsocks', value: 'shadowsocks' },
  { label: 'Dokodemo-door', value: 'dokodemo-door' },
  { label: 'SOCKS', value: 'socks' },
  { label: 'HTTP', value: 'http' }
]

const networkOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'WebSocket', value: 'ws' },
  { label: 'gRPC', value: 'grpc' },
  { label: 'HTTP/2', value: 'http' },
  { label: 'KCP', value: 'kcp' },
  { label: 'QUIC', value: 'quic' }
]

const securityOptions = [
  { label: '无', value: 'none' },
  { label: 'TLS', value: 'tls' },
  { label: 'Reality', value: 'reality' }
]

const ssMethodOptions = [
  { label: 'aes-256-gcm', value: 'aes-256-gcm' },
  { label: 'aes-128-gcm', value: 'aes-128-gcm' },
  { label: 'chacha20-poly1305', value: 'chacha20-poly1305' },
  { label: 'xchacha20-poly1305', value: 'xchacha20-poly1305' }
]

const flowOptions = [
  { label: '无', value: '' },
  { label: 'xtls-rprx-vision', value: 'xtls-rprx-vision' }
]

function generateUUID() {
  protocolSettings.value.clients[0].id = uuidv4()
}

function generatePassword() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < 16; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  protocolSettings.value.password = result
  protocolSettings.value.trojanPassword = result
}

function buildSettings(): string {
  const protocol = formData.value.protocol
  let settings: any = {}

  switch (protocol) {
    case 'vmess':
      settings = {
        clients: protocolSettings.value.clients.map(c => ({
          id: c.id || uuidv4(),
          alterId: c.alterId || 0,
          email: c.email || ''
        }))
      }
      break
    case 'vless':
      settings = {
        clients: protocolSettings.value.clients.map(c => ({
          id: c.id || uuidv4(),
          flow: c.flow || '',
          email: c.email || ''
        })),
        decryption: 'none'
      }
      break
    case 'trojan':
      settings = {
        clients: [{
          password: protocolSettings.value.trojanPassword,
          email: protocolSettings.value.clients[0].email
        }]
      }
      break
    case 'shadowsocks':
      settings = {
        method: protocolSettings.value.method,
        password: protocolSettings.value.password,
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
          accounts: protocolSettings.value.accounts
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

  // 网络设置
  switch (ss.network) {
    case 'tcp':
      result.tcpSettings = {
        header: { type: ss.tcpHeaderType }
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
    case 'http':
      result.httpSettings = {
        path: ss.h2Path,
        host: ss.h2Host ? [ss.h2Host] : []
      }
      break
  }

  // TLS 设置
  if (ss.security === 'tls') {
    result.tlsSettings = {
      serverName: ss.serverName,
      allowInsecure: ss.allowInsecure
    }
  }

  return JSON.stringify(result)
}

function buildSniffing(): string {
  return JSON.stringify(sniffingSettings.value)
}

function handleSubmit() {
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

// 监听编辑数据
watch(() => props.editData, (data) => {
  if (data) {
    formData.value = { ...data }
    // TODO: 解析 settings/streamSettings/sniffing
  }
}, { immediate: true })

// 监听显示状态，重置表单
watch(() => props.show, (show) => {
  if (show && !props.editData) {
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
    protocolSettings.value.clients[0].id = uuidv4()
  }
})
</script>

<template>
  <n-modal
    :show="show"
    :title="title"
    preset="card"
    style="width: 700px;"
    :mask-closable="false"
    @update:show="handleClose"
  >
    <n-tabs type="line">
      <n-tab-pane name="basic" tab="基础设置">
        <n-form label-placement="left" label-width="100">
          <n-form-item label="备注">
            <n-input v-model:value="formData.remark" placeholder="可选" />
          </n-form-item>
          <n-form-item label="启用">
            <n-switch v-model:value="formData.enable" />
          </n-form-item>
          <n-form-item label="监听 IP">
            <n-input v-model:value="formData.listen" placeholder="留空监听所有" />
          </n-form-item>
          <n-form-item label="端口">
            <n-input-number v-model:value="formData.port" :min="1" :max="65535" style="width: 100%;" />
          </n-form-item>
          <n-form-item label="协议">
            <n-select v-model:value="formData.protocol" :options="protocolOptions" />
          </n-form-item>
        </n-form>
      </n-tab-pane>

      <n-tab-pane name="protocol" tab="协议设置">
        <!-- VMess / VLESS -->
        <n-form v-if="formData.protocol === 'vmess' || formData.protocol === 'vless'" label-placement="left" label-width="100">
          <n-form-item label="UUID">
            <n-space>
              <n-input v-model:value="protocolSettings.clients[0].id" style="width: 320px;" />
              <n-button @click="generateUUID">生成</n-button>
            </n-space>
          </n-form-item>
          <n-form-item v-if="formData.protocol === 'vmess'" label="AlterID">
            <n-input-number v-model:value="protocolSettings.clients[0].alterId" :min="0" />
          </n-form-item>
          <n-form-item v-if="formData.protocol === 'vless'" label="Flow">
            <n-select v-model:value="protocolSettings.clients[0].flow" :options="flowOptions" />
          </n-form-item>
          <n-form-item label="Email">
            <n-input v-model:value="protocolSettings.clients[0].email" placeholder="可选，用于标识" />
          </n-form-item>
        </n-form>

        <!-- Trojan -->
        <n-form v-else-if="formData.protocol === 'trojan'" label-placement="left" label-width="100">
          <n-form-item label="密码">
            <n-space>
              <n-input v-model:value="protocolSettings.trojanPassword" style="width: 320px;" />
              <n-button @click="generatePassword">生成</n-button>
            </n-space>
          </n-form-item>
        </n-form>

        <!-- Shadowsocks -->
        <n-form v-else-if="formData.protocol === 'shadowsocks'" label-placement="left" label-width="100">
          <n-form-item label="加密方式">
            <n-select v-model:value="protocolSettings.method" :options="ssMethodOptions" />
          </n-form-item>
          <n-form-item label="密码">
            <n-space>
              <n-input v-model:value="protocolSettings.password" style="width: 320px;" />
              <n-button @click="generatePassword">生成</n-button>
            </n-space>
          </n-form-item>
        </n-form>

        <!-- Dokodemo-door -->
        <n-form v-else-if="formData.protocol === 'dokodemo-door'" label-placement="left" label-width="100">
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

        <!-- SOCKS / HTTP -->
        <n-form v-else label-placement="left" label-width="100">
          <n-form-item label="认证方式">
            <n-select v-model:value="protocolSettings.auth" :options="[{ label: '无认证', value: 'noauth' }, { label: '密码认证', value: 'password' }]" />
          </n-form-item>
          <template v-if="protocolSettings.auth === 'password'">
            <n-form-item label="用户名">
              <n-input v-model:value="protocolSettings.accounts[0].user" />
            </n-form-item>
            <n-form-item label="密码">
              <n-input v-model:value="protocolSettings.accounts[0].pass" />
            </n-form-item>
          </template>
        </n-form>
      </n-tab-pane>

      <n-tab-pane name="stream" tab="传输设置">
        <n-form label-placement="left" label-width="100">
          <n-form-item label="传输协议">
            <n-select v-model:value="streamSettings.network" :options="networkOptions" />
          </n-form-item>
          <n-form-item label="安全性">
            <n-select v-model:value="streamSettings.security" :options="securityOptions" />
          </n-form-item>

          <n-divider />

          <!-- TCP -->
          <template v-if="streamSettings.network === 'tcp'">
            <n-form-item label="伪装类型">
              <n-select v-model:value="streamSettings.tcpHeaderType" :options="[{ label: '无', value: 'none' }, { label: 'HTTP', value: 'http' }]" />
            </n-form-item>
          </template>

          <!-- WebSocket -->
          <template v-if="streamSettings.network === 'ws'">
            <n-form-item label="路径">
              <n-input v-model:value="streamSettings.wsPath" placeholder="/" />
            </n-form-item>
            <n-form-item label="Host">
              <n-input v-model:value="streamSettings.wsHost" placeholder="可选" />
            </n-form-item>
          </template>

          <!-- gRPC -->
          <template v-if="streamSettings.network === 'grpc'">
            <n-form-item label="ServiceName">
              <n-input v-model:value="streamSettings.grpcServiceName" />
            </n-form-item>
          </template>

          <!-- TLS -->
          <template v-if="streamSettings.security === 'tls'">
            <n-divider>TLS 设置</n-divider>
            <n-form-item label="服务器名称">
              <n-input v-model:value="streamSettings.serverName" placeholder="SNI" />
            </n-form-item>
            <n-form-item label="允许不安全">
              <n-switch v-model:value="streamSettings.allowInsecure" />
            </n-form-item>
          </template>
        </n-form>
      </n-tab-pane>

      <n-tab-pane name="sniffing" tab="嗅探设置">
        <n-form label-placement="left" label-width="100">
          <n-form-item label="启用嗅探">
            <n-switch v-model:value="sniffingSettings.enabled" />
          </n-form-item>
          <n-form-item v-if="sniffingSettings.enabled" label="目标覆盖">
            <n-select
              v-model:value="sniffingSettings.destOverride"
              multiple
              :options="[
                { label: 'HTTP', value: 'http' },
                { label: 'TLS', value: 'tls' },
                { label: 'QUIC', value: 'quic' },
                { label: 'FAKEDNS', value: 'fakedns' }
              ]"
            />
          </n-form-item>
        </n-form>
      </n-tab-pane>
    </n-tabs>

    <template #footer>
      <n-space justify="end">
        <n-button @click="handleClose">取消</n-button>
        <n-button type="primary" @click="handleSubmit">{{ isEdit ? '保存' : '添加' }}</n-button>
      </n-space>
    </template>
  </n-modal>
</template>
