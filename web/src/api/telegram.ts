import request from './request'
import type { ApiResponse } from './request'

export interface TelegramStatus {
  enabled: boolean
  configured: boolean
  tokenMasked: string
  authSecretSet: boolean
  workerRunning: boolean
  authorized: number
}

export const telegramApi = {
  status() {
    return request.get<ApiResponse<TelegramStatus>>('/telegram/status')
  },
  setup(token: string) {
    return request.post<ApiResponse<{ bot: string, authSecret: string }>>('/telegram/setup', { token })
  },
  toggle(enabled: boolean) {
    return request.post<ApiResponse>('/telegram/toggle', { enabled })
  },
  resetSecret() {
    return request.post<ApiResponse<{ authSecret: string }>>('/telegram/reset-secret')
  }
}
