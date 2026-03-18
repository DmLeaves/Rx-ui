<script setup lang="ts">
import { ref, onMounted, computed, h } from 'vue'
import { NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, NModal, NForm, NFormItem, NInput, NSwitch, NAlert, useMessage, NTabs, NTabPane } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, CreateOutline, WarningOutline, SyncOutline } from '@vicons/ionicons5'
import { certificateApi, type Certificate, type CreateCertificateParams } from '@/api/certificate'

const message = useMessage()
const loading = ref(false)
const certificates = ref<Certificate[]>([])
const expiringCerts = ref<Certificate[]>([])
const onlyExpiring = ref(false)

// 弹窗
const showModal = ref(false)
const editingCert = ref<Certificate | null>(null)
const activeTab = ref<'file' | 'content'>('file')
const certForm = ref<CreateCertificateParams>({
  domain: '',
  certFile: '',
  keyFile: '',
  certContent: '',
  keyContent: '',
  remark: '',
  autoRenew: false
})

function isValidExpiryDate(dateStr: string): boolean {
  if (!dateStr) return false
  const d = new Date(dateStr)
  return !Number.isNaN(d.getTime()) && d.getUTCFullYear() > 2000
}

function formatDate(dateStr: string): string {
  if (!isValidExpiryDate(dateStr)) return '-'
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

function getDaysUntilExpiry(dateStr: string): number {
  if (!isValidExpiryDate(dateStr)) return Infinity
  const expiry = new Date(dateStr)
  const now = new Date()
  return Math.ceil((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
}

function getExpiryStatus(dateStr: string): 'success' | 'warning' | 'error' {
  const days = getDaysUntilExpiry(dateStr)
  if (days < 0) return 'error'
  if (days < 30) return 'warning'
  return 'success'
}

function getExpiryText(dateStr: string): string {
  const days = getDaysUntilExpiry(dateStr)
  if (days < 0) return `已过期 ${-days} 天`
  if (days === 0) return '今天过期'
  if (days < 30) return `${days} 天后过期`
  return formatDate(dateStr)
}

const displayCertificates = computed(() => {
  if (!onlyExpiring.value) return certificates.value
  const ids = new Set(expiringCerts.value.map(c => c.id))
  return certificates.value.filter(c => ids.has(c.id))
})

const columns: DataTableColumns<Certificate> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '域名', key: 'domain', ellipsis: { tooltip: true } },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true }, render: (row) => row.remark || '-' },
  {
    title: '过期时间',
    key: 'expiresAt',
    width: 150,
    render: (row) => h(NTag, { type: getExpiryStatus(row.expiresAt), size: 'small' }, { default: () => getExpiryText(row.expiresAt) })
  },
  {
    title: '自动续期',
    key: 'autoRenew',
    width: 90,
    render: (row) => h(NTag, { type: row.autoRenew ? 'success' : 'default', size: 'small' }, { default: () => row.autoRenew ? '是' : '否' })
  },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', quaternary: true, onClick: () => handleEdit(row) }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }),
        h(NButton, { size: 'small', quaternary: true, onClick: () => handleReload(row.id) }, { icon: () => h(NIcon, null, { default: () => h(SyncOutline) }) }),
        h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => '确定删除该证书?'
        })
      ]
    })
  }
]

async function fetchCertificates() {
  loading.value = true
  try {
    const res = await certificateApi.list()
    certificates.value = res.data.data || []
  } catch (error: any) {
    message.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchExpiring() {
  try {
    const res = await certificateApi.getExpiring(30)
    expiringCerts.value = res.data.data || []
  } catch (error) {
    // 静默失败
  }
}

function handleAdd() {
  editingCert.value = null
  activeTab.value = 'file'
  certForm.value = {
    domain: '',
    certFile: '',
    keyFile: '',
    certContent: '',
    keyContent: '',
    remark: '',
    autoRenew: false
  }
  showModal.value = true
}

function handleEdit(row: Certificate) {
  editingCert.value = row
  // 证书/私钥原文属于敏感信息，后端不回显；编辑时默认进入“文件路径”页
  activeTab.value = 'file'
  certForm.value = {
    domain: row.domain,
    certFile: row.certFile,
    keyFile: row.keyFile,
    certContent: row.certContent,
    keyContent: row.keyContent,
    remark: row.remark,
    autoRenew: row.autoRenew,
    expiresAt: row.expiresAt
  }
  showModal.value = true
}

async function handleSubmit() {
  if (!certForm.value.domain) {
    message.warning('请输入域名')
    return
  }

  try {
    let saved: Certificate | null = null
    if (editingCert.value) {
      const res = await certificateApi.update(editingCert.value.id, certForm.value)
      saved = res.data.data || null
      message.success('更新成功')
    } else {
      const res = await certificateApi.create(certForm.value)
      saved = res.data.data || null
      message.success('添加成功')
    }

    if (saved?.certFile && saved?.keyFile) {
      message.info(`证书已落盘: ${saved.certFile}`)
    }

    showModal.value = false
    fetchCertificates()
    fetchExpiring()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

async function handleReload(id: number) {
  try {
    await certificateApi.reload(id)
    message.success('证书信息已刷新')
    fetchCertificates()
    fetchExpiring()
  } catch (error: any) {
    message.error(error.message || '刷新失败')
  }
}

async function handleDelete(id: number) {
  try {
    await certificateApi.delete(id)
    message.success('删除成功')
    fetchCertificates()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

onMounted(() => {
  fetchCertificates()
  fetchExpiring()
})
</script>

<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h2 style="margin: 0;">证书管理</h2>
      <n-space>
        <n-button @click="fetchCertificates">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
        <n-button @click="onlyExpiring = !onlyExpiring">
          {{ onlyExpiring ? '显示全部' : '仅看30天内过期' }}
        </n-button>
        <n-button type="primary" @click="handleAdd">
          <template #icon><n-icon :component="AddOutline" /></template>
          添加证书
        </n-button>
      </n-space>
    </n-space>

    <!-- 即将过期警告 -->
    <n-alert v-if="expiringCerts.length > 0" type="warning" style="margin-bottom: 16px;">
      <template #icon>
        <n-icon :component="WarningOutline" />
      </template>
      有 {{ expiringCerts.length }} 个证书将在 30 天内过期：
      {{ expiringCerts.map(c => c.domain).join(', ') }}
    </n-alert>

    <n-data-table
      :columns="columns"
      :data="displayCertificates"
      :loading="loading"
      :bordered="false"
      :row-key="(row: Certificate) => row.id"
    />

    <!-- 证书编辑弹窗 -->
    <n-modal
      v-model:show="showModal"
      :title="editingCert ? '编辑证书' : '添加证书'"
      preset="card"
      style="width: 600px;"
    >
      <n-tabs type="line" v-model:value="activeTab">
        <n-tab-pane name="file" tab="文件路径">
          <n-form label-placement="left" label-width="100">
            <n-form-item label="域名">
              <n-input v-model:value="certForm.domain" placeholder="例如: example.com" />
            </n-form-item>
            <n-form-item label="证书文件">
              <n-input v-model:value="certForm.certFile" placeholder="证书文件路径 (.crt/.pem)" />
            </n-form-item>
            <n-form-item label="私钥文件">
              <n-input v-model:value="certForm.keyFile" placeholder="私钥文件路径 (.key)" />
            </n-form-item>
            <n-form-item label="备注">
              <n-input v-model:value="certForm.remark" placeholder="可选" />
            </n-form-item>
            <n-form-item label="自动续期">
              <n-switch v-model:value="certForm.autoRenew" />
            </n-form-item>
          </n-form>
        </n-tab-pane>

        <n-tab-pane name="content" tab="直接输入">
          <n-alert v-if="editingCert" type="info" style="margin-bottom: 12px;">
            已保存的证书/私钥原文不会回显（安全考虑）。如需修改，请重新粘贴新的证书与私钥内容后保存。
          </n-alert>
          <n-form label-placement="left" label-width="100">
            <n-form-item label="域名">
              <n-input v-model:value="certForm.domain" placeholder="例如: example.com" />
            </n-form-item>
            <n-form-item label="证书内容">
              <n-input
                v-model:value="certForm.certContent"
                type="textarea"
                :rows="6"
                placeholder="粘贴证书内容 (-----BEGIN CERTIFICATE-----...)"
              />
            </n-form-item>
            <n-form-item label="私钥内容">
              <n-input
                v-model:value="certForm.keyContent"
                type="textarea"
                :rows="6"
                placeholder="粘贴私钥内容 (-----BEGIN PRIVATE KEY-----...)"
              />
            </n-form-item>
            <n-form-item label="证书路径">
              <n-input :value="certForm.certFile || ''" readonly placeholder="保存后自动生成（如 data/certs/*.crt）" />
            </n-form-item>
            <n-form-item label="私钥路径">
              <n-input :value="certForm.keyFile || ''" readonly placeholder="保存后自动生成（如 data/certs/*.key）" />
            </n-form-item>
            <n-form-item label="备注">
              <n-input v-model:value="certForm.remark" placeholder="可选" />
            </n-form-item>
            <n-form-item label="自动续期">
              <n-switch v-model:value="certForm.autoRenew" />
            </n-form-item>
          </n-form>
        </n-tab-pane>
      </n-tabs>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit">{{ editingCert ? '保存' : '添加' }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>
