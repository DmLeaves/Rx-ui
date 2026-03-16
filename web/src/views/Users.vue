<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import {
  NDataTable, NButton, NSpace, NIcon, NPopconfirm, NTag, NModal,
  NForm, NFormItem, NInput, NSwitch, useMessage
} from 'naive-ui'
import type { DataTableColumns, FormInst, FormRules } from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, KeyOutline } from '@vicons/ionicons5'
import { userApi, type User, type CreateUserParams, type UpdatePasswordParams } from '@/api/user'

const message = useMessage()
const loading = ref(false)
const users = ref<User[]>([])

// 添加用户弹窗
const showAddModal = ref(false)
const addFormRef = ref<FormInst | null>(null)
const addForm = ref<CreateUserParams>({
  username: '',
  password: '',
  enable: true
})

const addRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少6个字符', trigger: 'blur' }
  ]
}

// 修改密码弹窗
const showPasswordModal = ref(false)
const passwordFormRef = ref<FormInst | null>(null)
const editingUserId = ref<number | null>(null)
const passwordForm = ref<UpdatePasswordParams>({
  password: ''
})

const passwordRules: FormRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6个字符', trigger: 'blur' }
  ]
}

const columns: DataTableColumns<User> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '用户名', key: 'username' },
  {
    title: '状态',
    key: 'enable',
    width: 80,
    render: (row) => h(NTag, {
      type: row.enable ? 'success' : 'error',
      size: 'small'
    }, { default: () => row.enable ? '启用' : '禁用' })
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 180,
    render: (row) => new Date(row.createdAt).toLocaleString()
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => openPasswordModal(row.id)
        }, {
          icon: () => h(NIcon, null, { default: () => h(KeyOutline) }),
          default: () => '改密'
        }),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row.id)
        }, {
          trigger: () => h(NButton, {
            size: 'small',
            quaternary: true,
            type: 'error'
          }, {
            icon: () => h(NIcon, null, { default: () => h(TrashOutline) })
          }),
          default: () => '确定删除该用户?'
        })
      ]
    })
  }
]

async function fetchUsers() {
  loading.value = true
  try {
    const res = await userApi.list()
    users.value = res.data.data || []
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '获取列表失败'
    message.error(msg)
  } finally {
    loading.value = false
  }
}

function openAddModal() {
  addForm.value = { username: '', password: '', enable: true }
  showAddModal.value = true
}

async function handleAdd() {
  try {
    await addFormRef.value?.validate()
    await userApi.create(addForm.value)
    message.success('创建成功')
    showAddModal.value = false
    fetchUsers()
  } catch (error: unknown) {
    if (error instanceof Error) {
      message.error(error.message || '创建失败')
    }
  }
}

async function handleDelete(id: number) {
  try {
    await userApi.delete(id)
    message.success('删除成功')
    fetchUsers()
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : '删除失败'
    message.error(msg)
  }
}

function openPasswordModal(id: number) {
  editingUserId.value = id
  passwordForm.value = { password: '' }
  showPasswordModal.value = true
}

async function handleUpdatePassword() {
  try {
    await passwordFormRef.value?.validate()
    if (editingUserId.value) {
      await userApi.updatePassword(editingUserId.value, passwordForm.value)
      message.success('密码修改成功')
      showPasswordModal.value = false
    }
  } catch (error: unknown) {
    if (error instanceof Error) {
      message.error(error.message || '修改失败')
    }
  }
}

onMounted(() => {
  fetchUsers()
})
</script>

<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h2 style="margin: 0;">用户管理</h2>
      <n-space>
        <n-button @click="fetchUsers">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
        <n-button type="primary" @click="openAddModal">
          <template #icon><n-icon :component="AddOutline" /></template>
          添加用户
        </n-button>
      </n-space>
    </n-space>

    <n-data-table
      :columns="columns"
      :data="users"
      :loading="loading"
      :bordered="false"
      :row-key="(row: User) => row.id"
    />

    <!-- 添加用户弹窗 -->
    <n-modal
      v-model:show="showAddModal"
      preset="dialog"
      title="添加用户"
      positive-text="确定"
      negative-text="取消"
      @positive-click="handleAdd"
    >
      <n-form
        ref="addFormRef"
        :model="addForm"
        :rules="addRules"
        label-placement="left"
        label-width="80"
      >
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="addForm.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="addForm.password"
            type="password"
            show-password-on="click"
            placeholder="请输入密码"
          />
        </n-form-item>
        <n-form-item label="启用" path="enable">
          <n-switch v-model:value="addForm.enable" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 修改密码弹窗 -->
    <n-modal
      v-model:show="showPasswordModal"
      preset="dialog"
      title="修改密码"
      positive-text="确定"
      negative-text="取消"
      @positive-click="handleUpdatePassword"
    >
      <n-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-placement="left"
        label-width="80"
      >
        <n-form-item label="新密码" path="password">
          <n-input
            v-model:value="passwordForm.password"
            type="password"
            show-password-on="click"
            placeholder="请输入新密码"
          />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>
