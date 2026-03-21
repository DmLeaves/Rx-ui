import request from './request'
import type { ApiResponse } from './request'

export interface ControlClient {
  clientId: string
  publicKey: string
  enabled: boolean
  remark: string
}

export interface GenerateClientResp {
  clientId: string
  publicKey: string
  privateKey: string
  skillUrl: string
  hint: string
}

export const controlApi = {
  listClients() {
    return request.get<ApiResponse<ControlClient[]>>('/control/clients')
  },
  deleteClient(id: string) {
    return request.delete<ApiResponse>(`/control/clients/${encodeURIComponent(id)}`)
  },
  generateClient(remark?: string) {
    return request.post<ApiResponse<GenerateClientResp>>('/control/clients/generate', { remark: remark || '' })
  },
  skillUrl() {
    return `${window.location.origin}${window.location.pathname.replace(/\/$/, '')}/api/v1/control/skill`
  }
}
