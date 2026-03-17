import request from './request'
import type { ApiResponse } from './request'

export interface Certificate {
  id: number
  domain: string
  certFile: string
  keyFile: string
  certContent: string
  keyContent: string
  remark: string
  autoRenew: boolean
  expiresAt: string
  createdAt: string
  updatedAt: string
}

export interface CreateCertificateParams {
  domain: string
  certFile?: string
  keyFile?: string
  certContent?: string
  keyContent?: string
  remark?: string
  autoRenew?: boolean
  expiresAt?: string
}

export const certificateApi = {
  list() {
    return request.get<ApiResponse<Certificate[]>>('/certificates')
  },

  get(id: number) {
    return request.get<ApiResponse<Certificate>>(`/certificates/${id}`)
  },

  create(params: CreateCertificateParams) {
    return request.post<ApiResponse<Certificate>>('/certificates', params)
  },

  update(id: number, params: CreateCertificateParams) {
    return request.put<ApiResponse<Certificate>>(`/certificates/${id}`, params)
  },

  delete(id: number) {
    return request.delete<ApiResponse>(`/certificates/${id}`)
  },

  reload(id: number) {
    return request.post<ApiResponse<Certificate>>(`/certificates/${id}/reload`)
  },

  getExpiring(days: number = 30) {
    return request.get<ApiResponse<Certificate[]>>(`/certificates/expiring?days=${days}`)
  }
}
