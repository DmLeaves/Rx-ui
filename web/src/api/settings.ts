import request from './request'
import type { ApiResponse } from './request'

export interface Settings {
  webListen: string
  webPort: number
  webBasePath: string
  webCertFile: string
  webKeyFile: string
  timeLocation: string
  frontendMode: string
  cdnProviders: string[]
}

export interface UpdateSettingsParams {
  webListen?: string
  webPort?: number
  webBasePath?: string
  webCertFile?: string
  webKeyFile?: string
  timeLocation?: string
  frontendMode?: string
  cdnProviders?: string[]
}

export const settingsApi = {
  getAll() {
    return request.get<ApiResponse<Settings>>('/settings')
  },

  update(params: UpdateSettingsParams) {
    return request.put<ApiResponse>('/settings', params)
  },

  reset() {
    return request.post<ApiResponse>('/settings/reset')
  }
}
