<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NForm, NFormItem, NInput, NInputNumber, NButton, NSpace, NSelect, useMessage } from 'naive-ui'
import { settingsApi, type Settings } from '@/api/settings'

const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const settings = ref<Settings>({
  webListen: '',
  webPort: 54321,
  webBasePath: '/',
  webCertFile: '',
  webKeyFile: '',
  timeLocation: 'Asia/Shanghai',
  frontendMode: 'embedded',
  cdnProviders: []
})

const frontendModeOptions = [
  { label: '嵌入式（推荐）', value: 'embedded' },
  { label: 'CDN', value: 'cdn' },
  { label: '本地文件', value: 'local' }
]

const timezoneOptions = [
  { label: 'Asia/Shanghai (中国)', value: 'Asia/Shanghai' },
  { label: 'Asia/Tokyo (日本)', value: 'Asia/Tokyo' },
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

async function resetSettings() {
  try {
    await settingsApi.reset()
    message.success('设置已重置')
    fetchSettings()
  } catch (error: any) {
    message.error(error.message || '重置失败')
  }
}

onMounted(() => {
  fetchSettings()
})
</script>

<template>
  <div>
    <h2 style="margin: 0 0 16px 0;">系统设置</h2>
    
    <n-card title="Web 服务">
      <n-form label-placement="left" label-width="120">
        <n-form-item label="监听地址">
          <n-input v-model:value="settings.webListen" placeholder="留空表示监听所有地址" />
        </n-form-item>
        <n-form-item label="端口">
          <n-input-number v-model:value="settings.webPort" :min="1" :max="65535" />
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

    <n-card title="其他设置" style="margin-top: 16px;">
      <n-form label-placement="left" label-width="120">
        <n-form-item label="时区">
          <n-select v-model:value="settings.timeLocation" :options="timezoneOptions" />
        </n-form-item>
        <n-form-item label="前端资源">
          <n-select v-model:value="settings.frontendMode" :options="frontendModeOptions" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-space style="margin-top: 24px;">
      <n-button type="primary" :loading="saving" @click="saveSettings">
        保存设置
      </n-button>
      <n-button @click="resetSettings">
        重置为默认
      </n-button>
    </n-space>
  </div>
</template>
