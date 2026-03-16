import request from './request'
import type { ApiResponse } from './request'

export interface User {
  id: number
  username: string
  enable: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateUserParams {
  username: string
  password: string
  enable?: boolean
}

export interface UpdatePasswordParams {
  password: string
}

export const userApi = {
  list() {
    return request.get<ApiResponse<User[]>>('/users')
  },

  create(params: CreateUserParams) {
    return request.post<ApiResponse<User>>('/users', params)
  },

  delete(id: number) {
    return request.delete<ApiResponse>(`/users/${id}`)
  },

  updatePassword(id: number, params: UpdatePasswordParams) {
    return request.put<ApiResponse>(`/users/${id}/password`, params)
  }
}
