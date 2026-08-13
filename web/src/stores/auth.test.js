import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'

vi.mock('../api', () => ({
  default: { post: vi.fn() }
}))

import api from '../api'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('логин сохраняет токен в store и localStorage', async () => {
    api.post.mockResolvedValue({ data: { token: 'tok-123' } })

    const auth = useAuthStore()
    await auth.login('a@b.c', 'secret')

    expect(auth.token).toBe('tok-123')
    expect(localStorage.getItem('token')).toBe('tok-123')
    expect(api.post).toHaveBeenCalledWith('/api/v1/auth/login', { email: 'a@b.c', password: 'secret' })
  })

  it('logout чистит токен', () => {
    localStorage.setItem('token', 'tok-123')
    const auth = useAuthStore()

    auth.logout()

    expect(auth.token).toBe('')
    expect(localStorage.getItem('token')).toBeNull()
  })
})
