<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { NModal, NSpace, NButton, NInput, useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import { copyTextSmart } from '@/utils/clipboard'

const props = defineProps<{
  show: boolean
  title: string
  link: string
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
}>()

const message = useMessage()
const canvasRef = ref<HTMLCanvasElement>()

async function generateQR() {
  if (canvasRef.value && props.link) {
    try {
      await QRCode.toCanvas(canvasRef.value, props.link, {
        width: 256,
        margin: 2
      })
    } catch (error) {
      console.error('生成二维码失败:', error)
    }
  }
}

async function copyLink() {
  const text = props.link || ''
  if (!text) return

  const result = await copyTextSmart(text)
  if (result.ok) {
    message.success('已复制到剪贴板')
    return
  }

  if (result.method === 'manual') {
    message.warning('自动复制失败，已弹出手动复制框')
  } else {
    message.error(`复制失败: ${result.reason || 'unknown'}`)
  }
}

function handleClose() {
  emit('update:show', false)
}

watch(() => props.show, (show) => {
  if (show) {
    setTimeout(generateQR, 100)
  }
})

onMounted(() => {
  if (props.show) {
    generateQR()
  }
})
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="title"
    style="width: 350px;"
    @update:show="handleClose"
  >
    <n-space vertical align="center">
      <canvas ref="canvasRef"></canvas>
      <n-input :value="link" readonly style="width: 100%;">
        <template #suffix>
          <n-button text @click="copyLink">复制</n-button>
        </template>
      </n-input>
    </n-space>
  </n-modal>
</template>
