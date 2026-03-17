import request from './request'
import type { ApiResponse } from './request'

export interface Settings {
  webPort: string
  webBasePath: string
  webCertFile: string
  webKeyFile: string
  xrayBinPath: string
  timeZone: string
}

export const settingsApi = {
  getAll() {
    return request.get<ApiResponse<Settings>>('/settings')
  },

  update(params: Partial<Settings>) {
    return request.put<ApiResponse>('/settings', params)
  }
}
