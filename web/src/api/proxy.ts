import request from './request'
import type { ApiResponse } from './request'

// 连锁（上游）代理
export interface ChainedProxy {
  id: number
  remark: string
  protocol: string // socks | http
  host: string
  port: number
  username: string
  password: string
  enable: boolean
  createdAt: string
  updatedAt: string
}

export interface ProxyUpsertParams {
  remark?: string
  protocol?: string
  host?: string
  port?: number
  username?: string
  password?: string
  enable?: boolean
  // 直接粘贴 "host:port:user:pass"，后端解析后覆盖结构化字段
  raw?: string
}

export const proxyApi = {
  list() {
    return request.get<ApiResponse<ChainedProxy[]>>('/proxies')
  },

  create(params: ProxyUpsertParams) {
    return request.post<ApiResponse<ChainedProxy>>('/proxies', params)
  },

  update(id: number, params: ProxyUpsertParams) {
    return request.put<ApiResponse<ChainedProxy>>(`/proxies/${id}`, params)
  },

  delete(id: number) {
    return request.delete<ApiResponse>(`/proxies/${id}`)
  },

  // 设置/清除某客户端的连锁代理；proxyId 传 null 表示恢复直连（不走代理）
  setClientProxy(clientId: number, proxyId: number | null) {
    return request.put<ApiResponse>(`/clients/${clientId}/proxy`, { proxyId })
  }
}
