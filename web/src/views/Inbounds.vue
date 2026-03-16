<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, CreateOutline } from '@vicons/ionicons5'
import { inboundApi, type Inbound } from '@/api/inbound'

const message = useMessage()
const loading = ref(false)
const inbounds = ref<Inbound[]>([])

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const columns: DataTableColumns<Inbound> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true } },
  { 
    title: '状态', 
    key: 'enable',
    width: 80,
    render: (row) => h(NTag, { type: row.enable ? 'success' : 'error' }, { default: () => row.enable ? '启用' : '禁用' })
  },
  { title: '端口', key: 'port', width: 80 },
  { title: '协议', key: 'protocol', width: 100 },
  { 
    title: '流量', 
    key: 'traffic',
    width: 150,
    render: (row) => `↑${formatBytes(row.up)} ↓${formatBytes(row.down)}`
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row) => h(NSpace, null, {
      default: () => [
        h(NButton, { size: 'small', onClick: () => handleEdit(row) }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) }),
        h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', type: 'error' }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => '确定删除?'
        })
      ]
    })
  }
]

async function fetchInbounds() {
  loading.value = true
  try {
    const res = await inboundApi.list()
    inbounds.value = res.data.data
  } catch (error: any) {
    message.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

function handleAdd() {
  // TODO: 打开添加弹窗
  message.info('添加功能开发中...')
}

function handleEdit(row: Inbound) {
  // TODO: 打开编辑弹窗
  message.info(`编辑 ${row.remark}`)
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
    />
  </div>
</template>
