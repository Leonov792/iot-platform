import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DeviceCard from './DeviceCard.vue'

vi.mock('../api', () => ({
  default: { post: vi.fn(), delete: vi.fn() }
}))

import api from '../api'

const device = {
  id: 'lamp-1',
  name: 'Лампа',
  type: 'light',
  room: 'Зал',
  status: 'online',
  state: { on: true }
}

const mountCard = () =>
  mount(DeviceCard, {
    props: { device },
    global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } }
  })

describe('DeviceCard', () => {
  beforeEach(() => vi.clearAllMocks())

  it('включённая лампа шлёт команду off', async () => {
    api.post.mockResolvedValue({ data: {} })

    const wrapper = mountCard()
    await wrapper.find('button').trigger('click')

    expect(api.post).toHaveBeenCalledWith('/api/v1/devices/lamp-1/command', { action: 'off' })
  })

  it('удаление шлёт delete', async () => {
    api.delete.mockResolvedValue({})

    const wrapper = mountCard()
    const del = wrapper.findAll('button').find((b) => b.text().includes('удалить'))
    await del.trigger('click')

    expect(api.delete).toHaveBeenCalledWith('/api/v1/devices/lamp-1')
  })
})
