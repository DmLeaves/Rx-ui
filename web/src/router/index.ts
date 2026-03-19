import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

import Login from '@/views/Login.vue'
import Layout from '@/views/Layout.vue'
import Dashboard from '@/views/Dashboard.vue'
import Inbounds from '@/views/Inbounds.vue'
import Clients from '@/views/Clients.vue'
import Certificates from '@/views/Certificates.vue'
import Settings from '@/views/Settings.vue'
import Users from '@/views/Users.vue'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: Layout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: Dashboard
      },
      {
        path: 'inbounds',
        name: 'Inbounds',
        component: Inbounds
      },
      {
        path: 'clients',
        name: 'Clients',
        component: Clients
      },
      {
        path: 'certificates',
        name: 'Certificates',
        component: Certificates
      },
      {
        path: 'settings',
        name: 'Settings',
        component: Settings
      },
      {
        path: 'users',
        name: 'Users',
        component: Users
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth && !token) {
    next({ name: 'Login' })
  } else if (to.name === 'Login' && token) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})

export default router
