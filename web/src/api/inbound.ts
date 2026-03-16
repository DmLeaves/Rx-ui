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
  email: string
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
  email: string
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
    return request.post<ApiResponse>(`/inbounds/${id}/reset-traffic`)
  },

  // 客户端
  listClients(inboundId: number) {
    return request.get<ApiResponse<Client[]>>(`/inbounds/${inboundId}/clients`)
  },

  addClient(inboundId: number, params: CreateClientParams) {
    return request.post<ApiResponse<Client>>(`/inbounds/${inboundId}/clients`, params)
  },

  updateClient(inboundId: number, clientId: number, params: CreateClientParams) {
    return request.put<ApiResponse<Client>>(`/inbounds/${inboundId}/clients/${clientId}`, params)
  },

  deleteClient(inboundId: number, clientId: number) {
    return request.delete<ApiResponse>(`/inbounds/${inboundId}/clients/${clientId}`)
  }
}
