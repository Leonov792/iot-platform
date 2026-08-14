import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import DevicesView from '../views/DevicesView.vue'
import DeviceView from '../views/DeviceView.vue'
import AdminView from '../views/AdminView.vue'

function decodeRole(token) {
  if (!token) return ''
  try {
    return JSON.parse(atob(token.split('.')[1])).role || ''
  } catch {
    return ''
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: LoginView },
    { path: '/', component: DevicesView, meta: { requiresAuth: true } },
    { path: '/devices/:id', component: DeviceView, meta: { requiresAuth: true } },
    { path: '/admin', component: AdminView, meta: { requiresAuth: true, role: 'owner' } }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) return '/login'
  if (to.path === '/login' && token) return '/'
  if (to.meta.role && decodeRole(token) !== to.meta.role) return '/'
})

export default router
