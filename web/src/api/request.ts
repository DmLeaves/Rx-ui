import axios from 'axios'
import type { AxiosInstance, AxiosResponse } from 'axios'
import router from '@/router'

// API 响应类型
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 创建 axios 实例
const request: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { code, message } = response.data
    
    if (code !== 0) {
      // 业务错误
      return Promise.reject(new Error(message || '请求失败'))
    }
    
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      // 未授权，跳转登录
      localStorage.removeItem('token')
      router.push({ name: 'Login' })
    }
    return Promise.reject(error)
  }
)

export default request
