import request from './request'
import type { ApiResponse } from './request'

export interface Inbound {
  id: number
  userId: number
  remark: string
  enable: boolean
  listen: string
  port: number
  protocol: string
  settings: string
  streamSettings: string
  sniffing: string
  tag: string
  up: number
  down: number
  total: number
  expiryTime: number
  certificateId?: number
  createdAt: string
  updatedAt: string
}

export interface Client {
  id: number
  inboundId: number
  remark?: string
  uuid: string
  password: string
  flow: string
  enable: boolean
  up: number
  down: number
  total: number
  expiryTime: number
  certificateId?: number
  createdAt: string
  updatedAt: string
}

export interface CreateInboundParams {
  remark: string
  enable: boolean
  listen?: string
  port: number
  protocol: string
  settings: string
  streamSettings?: string
  sniffing?: string
  tag?: string
  total?: number
  expiryTime?: number
  certificateId?: number
}

export interface CreateClientParams {
  remark?: string
  uuid?: string
  password?: string
  flow?: string
  enable: boolean
  total?: number
  expiryTime?: number
  certificateId?: number
}

export const inboundApi = {
  // 入站规则
  list() {
    return request.get<ApiResponse<Inbound[]>>('/inbounds')
  },

  get(id: number) {
    return request.get<ApiResponse<Inbound>>(`/inbounds/${id}`)
  },

  create(params: CreateInboundParams) {
    return request.post<ApiResponse<Inbound>>('/inbounds', params)
  },

  update(id: number, params: CreateInboundParams) {
    return request.put<ApiResponse<Inbound>>(`/inbounds/${id}`, params)
  },

  delete(id: number) {
    return request.delete<ApiResponse>(`/inbounds/${id}`)
  },

  resetTraffic(id: number) {
    return request.post<ApiResponse>(`/inbounds/${id}/resetTraffic`)
  },

  // 客户端
  listClients(inboundId: number) {
    return request.get<ApiResponse<Client[]>>(`/clients?inboundId=${inboundId}`)
  },

  addClient(inboundId: number, params: CreateClientParams) {
    return request.post<ApiResponse<Client>>(`/clients`, { ...params, inboundId })
  },

  updateClient(_inboundId: number, clientId: number, params: CreateClientParams) {
    return request.put<ApiResponse<Client>>(`/clients/${clientId}`, params)
  },

  deleteClient(_inboundId: number, clientId: number) {
    return request.delete<ApiResponse>(`/clients/${clientId}`)
  }
}
