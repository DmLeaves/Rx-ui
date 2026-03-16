import request from './request'
import type { ApiResponse } from './request'

export interface SystemStatus {
  hostname: string
  os: string
  arch: string
  uptime: number
  cpuPercent: number
  cpuCores: number
  memTotal: number
  memUsed: number
  memPercent: number
  diskTotal: number
  diskUsed: number
  diskPercent: number
  netUpload: number
  netDownload: number
  xrayRunning: boolean
  xrayVersion: string
  panelUptime: number
  goVersion: string
}

export interface XrayVersion {
  version: string
  running: boolean
}

export const systemApi = {
  getStatus() {
    return request.get<ApiResponse<SystemStatus>>('/system/status')
  },

  restartXray() {
    return request.post<ApiResponse>('/system/xray/restart')
  },

  getXrayVersion() {
    return request.get<ApiResponse<XrayVersion>>('/system/xray/version')
  }
}
