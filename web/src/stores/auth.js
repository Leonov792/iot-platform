import { defineStore } from 'pinia'
import api from '../api'

function decodeRole(token) {
  if (!token) return ''
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.role || ''
  } catch {
    return ''
  }
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || ''
  }),
  getters: {
    isAuthed: (s) => !!s.token,
    role: (s) => decodeRole(s.token)
  },
  actions: {
    async login(email, password) {
      const { data } = await api.post('/api/v1/auth/login', { email, password })
      this.token = data.token
      localStorage.setItem('token', data.token)
    },
    async register(email, password) {
      const { data } = await api.post('/api/v1/auth/register', { email, password })
      this.token = data.token
      localStorage.setItem('token', data.token)
    },
    logout() {
      this.token = ''
      localStorage.removeItem('token')
    }
  }
})
