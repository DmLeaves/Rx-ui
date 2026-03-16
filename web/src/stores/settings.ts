import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSettingsStore = defineStore('settings', () => {
  const darkMode = ref(localStorage.getItem('darkMode') === 'true')
  const collapsed = ref(false)

  function toggleDarkMode() {
    darkMode.value = !darkMode.value
    localStorage.setItem('darkMode', String(darkMode.value))
  }

  function toggleCollapsed() {
    collapsed.value = !collapsed.value
  }

  return {
    darkMode,
    collapsed,
    toggleDarkMode,
    toggleCollapsed
  }
})
