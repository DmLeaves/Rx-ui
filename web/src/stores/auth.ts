import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref('')

  const isLoggedIn = computed(() => !!token.value)

  async function login(usernameInput: string, password: string) {
    const res = await authApi.login({ username: usernameInput, password })
    token.value = res.data.data.token
    username.value = usernameInput
    localStorage.setItem('token', token.value)
    router.push({ name: 'Dashboard' })
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      token.value = ''
      username.value = ''
      localStorage.removeItem('token')
      router.push({ name: 'Login' })
    }
  }

  return {
    token,
    username,
    isLoggedIn,
    login,
    logout
  }
})
