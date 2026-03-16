import request from './request'
import type { ApiResponse } from './request'

export interface LoginParams {
  username: string
  password: string
}

export interface LoginResult {
  token: string
}

export const authApi = {
  login(params: LoginParams) {
    return request.post<ApiResponse<LoginResult>>('/auth/login', params)
  },

  logout() {
    return request.post<ApiResponse>('/auth/logout')
  },

  getCurrentUser() {
    return request.get<ApiResponse>('/auth/me')
  },

  changePassword(oldPassword: string, newPassword: string) {
    return request.put<ApiResponse>('/auth/password', { oldPassword, newPassword })
  }
}
