<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NSelect, NSwitch, NAlert, useMessage } from 'naive-ui'
import { settingsApi, type Settings } from '@/api/settings'
import { telegramApi, type TelegramStatus } from '@/api/telegram'

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

const telegramToken = ref('')
const telegramStatus = ref<TelegramStatus | null>(null)
const tgLoading = ref(false)
const authSecret = ref('')

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

async function fetchTelegramStatus() {
  try {
    const res = await telegramApi.status()
    telegramStatus.value = res.data.data
  } catch {
    telegramStatus.value = null
  }
}

async function setupTelegramBot() {
  if (!telegramToken.value.trim()) {
    message.warning('请输入 Bot Token')
    return
  }
  tgLoading.value = true
  try {
    const res = await telegramApi.setup(telegramToken.value.trim())
    authSecret.value = res.data.data?.authSecret || ''
    message.success('Bot 添加成功')
    await fetchTelegramStatus()
  } catch (error: any) {
    message.error(error.message || 'Bot 添加失败')
  } finally {
    tgLoading.value = false
  }
}

async function toggleTelegram(enabled: boolean) {
  tgLoading.value = true
  try {
    await telegramApi.toggle(enabled)
    await fetchTelegramStatus()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    tgLoading.value = false
  }
}

async function resetAuthSecret() {
  tgLoading.value = true
  try {
    const res = await telegramApi.resetSecret()
    authSecret.value = res.data.data?.authSecret || ''
    message.success('密钥已重置')
    await fetchTelegramStatus()
  } catch (error: any) {
    message.error(error.message || '重置失败')
  } finally {
    tgLoading.value = false
  }
}

onMounted(() => {
  fetchSettings()
  fetchTelegramStatus()
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

    <n-card title="Telegram Bot（AI 控制入口）" style="margin-top: 16px;">
      <n-alert type="info" style="margin-bottom: 12px;">
        首次添加后会生成授权密钥，用户在 Telegram 私聊 Bot 发送 <code>/auth &lt;密钥&gt;</code> 完成授权。随后可用 <code>/capabilities</code>、<code>/query</code>、<code>/exec</code> 进行机器可读控制。
      </n-alert>
      <n-form label-placement="left" label-width="140">
        <n-form-item label="Bot Token">
          <n-input v-model:value="telegramToken" type="password" show-password-on="click" placeholder="123456:ABC..." />
        </n-form-item>
        <n-form-item>
          <n-space>
            <n-button type="primary" :loading="tgLoading" @click="setupTelegramBot">添加 / 验证 Bot</n-button>
            <n-button :loading="tgLoading" @click="toggleTelegram(true)">启用</n-button>
            <n-button :loading="tgLoading" @click="toggleTelegram(false)">停用</n-button>
            <n-button :loading="tgLoading" @click="resetAuthSecret">重置授权密钥</n-button>
          </n-space>
        </n-form-item>
      </n-form>

      <n-alert v-if="telegramStatus" type="success" style="margin-top: 10px;">
        状态：{{ telegramStatus.enabled ? '已启用' : '未启用' }} ｜ Worker：{{ telegramStatus.workerRunning ? '运行中' : '未运行' }} ｜ 已授权：{{ telegramStatus.authorized }}
        <br />
        Token：{{ telegramStatus.tokenMasked || '未配置' }}
      </n-alert>

      <n-alert v-if="authSecret" type="warning" style="margin-top: 10px;">
        首次授权密钥：<code>{{ authSecret }}</code>（请妥善保存）
      </n-alert>
    </n-card>

    <n-card title="其他设置" style="margin-top: 16px;">
      <n-form label-placement="left" label-width="120">
        <n-form-item label="时区">
          <n-select v-model:value="settings.timeZone" :options="timezoneOptions" style="width: 300px;" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-space style="margin-top: 24px;">
      <n-button type="primary" :loading="saving" @click="saveSettings">
        保存设置
      </n-button>
    </n-space>
  </div>
</template>
