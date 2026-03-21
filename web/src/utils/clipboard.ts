export interface CopyResult {
  ok: boolean
  method?: 'clipboard' | 'execCommand' | 'manual'
  reason?: string
}

function copyByExecCommand(text: string): boolean {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '0'
  textarea.style.left = '-9999px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)

  try {
    textarea.focus()
    textarea.select()
    textarea.setSelectionRange(0, textarea.value.length)
    return document.execCommand('copy')
  } finally {
    document.body.removeChild(textarea)
  }
}

export async function copyTextSmart(text: string): Promise<CopyResult> {
  const value = String(text ?? '')
  if (!value) return { ok: false, reason: 'empty_text' }

  // 首选 Clipboard API
  try {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function' && window.isSecureContext) {
      await navigator.clipboard.writeText(value)
      return { ok: true, method: 'clipboard' }
    }
  } catch (err: any) {
    // 继续 fallback
    console.warn('[copy] clipboard api failed:', err?.name || err?.message || err)
  }

  // 退化到 execCommand
  try {
    const ok = copyByExecCommand(value)
    if (ok) return { ok: true, method: 'execCommand' }
  } catch (err: any) {
    console.warn('[copy] execCommand failed:', err?.name || err?.message || err)
  }

  // 最后兜底：prompt 手动复制
  try {
    window.prompt('自动复制失败，请手动复制下面内容：', value)
    return { ok: false, method: 'manual', reason: 'manual_prompt' }
  } catch {
    return { ok: false, reason: 'all_methods_failed' }
  }
}
