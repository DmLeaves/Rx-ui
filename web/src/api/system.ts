import request from './request'
import type { ApiResponse } from './request'

export interface SystemStatus {
  cpu: {
    cores: number
    percent: number
  }
  memory: {
    total: number
    used: number
  }
  load: number[]
  uptime: number
  traffic: {
    up: number
    down: number
  }
  xray: {
    running: boolean
    version: string
  }
  panelUptime: number
  inboundCount: number
}

export interface XrayStatus {
  running: boolean
  version: string
}

export const systemApi = {
  getStatus() {
    return request.get<ApiResponse<SystemStatus>>('/system/status')
  },

  getXrayStatus() {
    return request.get<ApiResponse<XrayStatus>>('/xray/status')
  },

  startXray() {
    return request.post<ApiResponse>('/xray/start')
  },

  stopXray() {
    return request.post<ApiResponse>('/xray/stop')
  },

  restartXray() {
    return request.post<ApiResponse>('/xray/restart')
  }
}
