<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NSelect, NSwitch, NAlert, NDataTable, useMessage } from 'naive-ui'
import { settingsApi, type Settings } from '@/api/settings'
import { controlApi, type ControlClient, type GenerateClientResp } from '@/api/control'
import { copyTextSmart } from '@/utils/clipboard'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const settings = ref<Settings>({
  webPort: '54321',
  webBasePath: '/',
  webCertFile: '',
  webKeyFile: '',
  xrayBinPath: '/usr/local/bin/xray',
  timeZone: 'Asia/Shanghai',
  acmeEmail: '',
  acmeDnsProvider: 'cloudflare',
  acmeDnsApiToken: '',
  acmeEnabled: 'false'
})

const clients = ref<ControlClient[]>([])
const generating = ref(false)
const generated = ref<GenerateClientResp | null>(null)
const aiRemark = ref('')

const skillLink = computed(() => controlApi.skillUrl())
const discoveryLink = computed(() => `${window.location.origin}/api/v1/control/discovery`)

const timezoneOptions = [
  { label: 'Asia/Shanghai (中国)', value: 'Asia/Shanghai' },
  { label: 'Asia/Tokyo (日本)', value: 'Asia/Tokyo' },
  { label: 'Asia/Hong_Kong (香港)', value: 'Asia/Hong_Kong' },
  { label: 'Asia/Singapore (新加坡)', value: 'Asia/Singapore' },
  { label: 'America/New_York (美东)', value: 'America/New_York' },
  { label: 'America/Los_Angeles (美西)', value: 'America/Los_Angeles' },
  { label: 'Europe/London (伦敦)', value: 'Europe/London' },
  { label: 'UTC', value: 'UTC' }
]

async function fetchSettings() {
  loading.value = true
  try {
    const res = await settingsApi.getAll()
    settings.value = res.data.data
  } catch (error: any) {
    message.error(error.message || '获取设置失败')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    await settingsApi.update(settings.value)
    message.success('设置已保存')
  } catch (error: any) {
    message.error(error.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function fetchClients() {
  try {
    const res = await controlApi.listClients()
    clients.value = res.data.data || []
  } catch (error: any) {
    message.error(error.message || '获取 AI 接入客户端失败')
  }
}

async function generateAccessInfo() {
  generating.value = true
  try {
    const res = await controlApi.generateClient(aiRemark.value)
    generated.value = res.data.data
    message.success('已生成接入信息（私钥仅显示一次）')
    await fetchClients()
  } catch (error: any) {
    message.error(error.message || '生成失败')
  } finally {
    generating.value = false
  }
}

async function removeClient(id: string) {
  try {
    await controlApi.deleteClient(id)
    message.success('已删除客户端')
    await fetchClients()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

async function copyText(text: string) {
  const result = await copyTextSmart(text)
  if (result.ok) {
    message.success('已复制')
    return
  }
  if (result.method === 'manual') {
    message.warning('自动复制失败，已弹出手动复制框')
  } else {
    message.error(`复制失败: ${result.reason || 'unknown'}`)
  }
}

const columns = [
  { title: 'Client ID', key: 'clientId' },
  { title: '备注', key: 'remark' },
  { title: '启用', key: 'enabled', render: (row: ControlClient) => (row.enabled ? '是' : '否') },
  { title: '公钥', key: 'publicKey', render: (row: ControlClient) => row.publicKey?.slice(0, 20) + '...' },
  {
    title: '操作',
    key: 'actions',
    render: (row: ControlClient) => {
      return h(NSpace, { size: 8 }, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => copyText(row.clientId) }, { default: () => '复制ID' }),
          h(NButton, { size: 'small', type: 'error', onClick: () => removeClient(row.clientId) }, { default: () => '删除' })
        ]
      })
    }
  }
]

onMounted(() => {
  fetchSettings()
  fetchClients()
})
</script>

<template>
  <div>
    <h2 style="margin: 0 0 16px 0;">系统设置</h2>
    
    <n-card title="Web 服务">
      <n-form label-placement="left" label-width="120">
        <n-form-item label="端口">
          <n-input v-model:value="settings.webPort" placeholder="54321" style="width: 150px;" />
        </n-form-item>
        <n-form-item label="基础路径">
          <n-input v-model:value="settings.webBasePath" placeholder="/" />
        </n-form-item>
        <n-form-item label="证书文件">
          <n-input v-model:value="settings.webCertFile" placeholder="HTTPS 证书路径（可选）" />
        </n-form-item>
        <n-form-item label="私钥文件">
          <n-input v-model:value="settings.webKeyFile" placeholder="HTTPS 私钥路径（可选）" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="Xray 设置" style="margin-top: 16px;">
      <n-form label-placement="left" label-width="120">
        <n-form-item label="Xray 路径">
          <n-input v-model:value="settings.xrayBinPath" placeholder="/usr/local/bin/xray" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="ACME 自动续签（Lego）" style="margin-top: 16px;">
      <n-alert type="info" style="margin-bottom: 12px;">
        配置后可在证书页启用 autoRenew，并支持手动“立即续签”。当前先支持 Cloudflare DNS。
      </n-alert>
      <n-form label-placement="left" label-width="120">
        <n-form-item label="启用 ACME">
          <n-switch v-model:value="settings.acmeEnabled" checked-value="true" unchecked-value="false" />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input v-model:value="settings.acmeEmail" placeholder="acme 账户邮箱" />
        </n-form-item>
        <n-form-item label="DNS Provider">
          <n-select
            v-model:value="settings.acmeDnsProvider"
            :options="[{ label: 'Cloudflare', value: 'cloudflare' }]"
            style="width: 260px;"
          />
        </n-form-item>
        <n-form-item label="DNS API Token">
          <n-input v-model:value="settings.acmeDnsApiToken" type="password" show-password-on="click" placeholder="Cloudflare API Token" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="其他设置" style="margin-top: 16px;">
      <n-form label-placement="left" label-width="120">
        <n-form-item label="时区">
          <n-select v-model:value="settings.timeZone" :options="timezoneOptions" style="width: 300px;" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="AI 接入信息（给人复制给 AI）" style="margin-top: 16px;">
      <n-alert type="info" style="margin-bottom: 12px;">
        一键生成后会返回 clientId + 私钥（仅显示一次），AI 通过 discovery/manifest/errors 自探索能力，避免版本漂移。
      </n-alert>
      <n-space vertical>
        <n-form label-placement="left" label-width="100">
          <n-form-item label="备注">
            <n-input v-model:value="aiRemark" placeholder="例如：openclaw-main-agent" style="max-width: 360px;" />
          </n-form-item>
        </n-form>
        <n-space>
          <n-button type="primary" :loading="generating" @click="generateAccessInfo">一键生成接入信息</n-button>
          <n-button @click="copyText(skillLink)">复制 Skill 链接</n-button>
          <n-button @click="copyText(discoveryLink)">复制 Discovery 链接</n-button>
        </n-space>

        <n-alert v-if="generated" type="warning">
          <div><b>Client ID:</b> {{ generated.clientId }}</div>
          <div><b>Private Key(Base64):</b> {{ generated.privateKey }}</div>
          <div><b>Skill URL:</b> {{ generated.skillUrl }}</div>
          <div style="margin-top: 8px;">⚠️ 私钥只会显示这一次，请马上保存。</div>
          <n-space style="margin-top: 8px;">
            <n-button size="small" @click="copyText(generated.clientId)">复制 Client ID</n-button>
            <n-button size="small" @click="copyText(generated.privateKey)">复制 Private Key</n-button>
            <n-button size="small" @click="copyText(generated.skillUrl)">复制 Skill URL</n-button>
          </n-space>
        </n-alert>

        <n-data-table :columns="columns" :data="clients" :pagination="false" />
      </n-space>
    </n-card>

    <n-space style="margin-top: 24px;">
      <n-button type="primary" :loading="saving" @click="saveSettings">
        保存设置
      </n-button>
    </n-space>
  </div>
</template>
