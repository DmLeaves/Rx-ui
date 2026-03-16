<script setup lang="ts">
import { h, computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NButton, NIcon, NSpace, NSwitch } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { HomeOutline, ServerOutline, PeopleOutline, ShieldCheckmarkOutline, SettingsOutline, LogOutOutline, SunnyOutline, MoonOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'

const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const activeKey = computed(() => route.name as string)

const menuOptions: MenuOption[] = [
  {
    label: () => h(RouterLink, { to: '/' }, { default: () => '仪表盘' }),
    key: 'Dashboard',
    icon: () => h(NIcon, null, { default: () => h(HomeOutline) })
  },
  {
    label: () => h(RouterLink, { to: '/inbounds' }, { default: () => '入站规则' }),
    key: 'Inbounds',
    icon: () => h(NIcon, null, { default: () => h(ServerOutline) })
  },
  {
    label: () => h(RouterLink, { to: '/clients' }, { default: () => '客户端' }),
    key: 'Clients',
    icon: () => h(NIcon, null, { default: () => h(PeopleOutline) })
  },
  {
    label: () => h(RouterLink, { to: '/certificates' }, { default: () => '证书管理' }),
    key: 'Certificates',
    icon: () => h(NIcon, null, { default: () => h(ShieldCheckmarkOutline) })
  },
  {
    label: () => h(RouterLink, { to: '/settings' }, { default: () => '系统设置' }),
    key: 'Settings',
    icon: () => h(NIcon, null, { default: () => h(SettingsOutline) })
  }
]
</script>

<template>
  <n-layout has-sider style="height: 100vh;">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="220"
      :collapsed="settingsStore.collapsed"
      show-trigger
      @collapse="settingsStore.toggleCollapsed"
      @expand="settingsStore.toggleCollapsed"
    >
      <div class="logo">
        <h2 v-if="!settingsStore.collapsed">Rx-ui</h2>
        <h2 v-else>R</h2>
      </div>
      <n-menu
        :value="activeKey"
        :options="menuOptions"
        :collapsed="settingsStore.collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
      />
    </n-layout-sider>
    
    <n-layout>
      <n-layout-header bordered style="padding: 12px 24px;">
        <n-space justify="end" align="center">
          <n-switch
            :value="settingsStore.darkMode"
            @update:value="settingsStore.toggleDarkMode"
          >
            <template #checked-icon>
              <n-icon :component="MoonOutline" />
            </template>
            <template #unchecked-icon>
              <n-icon :component="SunnyOutline" />
            </template>
          </n-switch>
          <n-button quaternary circle @click="authStore.logout">
            <template #icon>
              <n-icon :component="LogOutOutline" />
            </template>
          </n-button>
        </n-space>
      </n-layout-header>
      
      <n-layout-content content-style="padding: 24px;">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<style scoped>
.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 60px;
  font-weight: bold;
}
.logo h2 {
  margin: 0;
  color: var(--n-text-color);
}
</style>
