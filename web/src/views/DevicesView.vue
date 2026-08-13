<template>
  <div class="mx-auto max-w-6xl px-4 py-8">
    <header class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold">Устройства</h1>
      <div class="flex items-center gap-3">
        <button
          class="rounded-lg bg-indigo-600 px-4 py-2 text-sm text-white hover:bg-indigo-700"
          @click="showForm = !showForm"
        >
          + Добавить
        </button>
        <button
          class="rounded-lg border border-slate-300 px-4 py-2 text-sm hover:bg-slate-100"
          @click="logout"
        >
          Выйти
        </button>
      </div>
    </header>

    <form
      v-if="showForm"
      class="mt-6 grid grid-cols-2 gap-3 rounded-2xl bg-white p-6 shadow"
      @submit.prevent="create"
    >
      <input v-model="form.name" placeholder="Название" required class="rounded-lg border px-3 py-2 text-sm" />
      <select v-model="form.type" class="rounded-lg border px-3 py-2 text-sm">
        <option value="light">Лампа</option>
        <option value="plug">Розетка</option>
        <option value="thermostat">Термостат</option>
        <option value="sensor">Датчик</option>
      </select>
      <input v-model="form.room" placeholder="Комната" class="rounded-lg border px-3 py-2 text-sm" />
      <input v-model="form.id" placeholder="ID (напр. lamp-1)" required class="rounded-lg border px-3 py-2 text-sm" />
      <div class="col-span-2 flex gap-2">
        <button type="submit" class="rounded-lg bg-indigo-600 px-4 py-2 text-sm text-white">Создать</button>
        <button type="button" class="rounded-lg border px-4 py-2 text-sm" @click="showForm = false">Отмена</button>
      </div>
    </form>

    <p v-if="loading" class="mt-10 text-center text-slate-500">грузим...</p>
    <p v-else-if="!devices.length" class="mt-10 text-center text-slate-500">
      пока пусто, добавь первое устройство
    </p>

    <div v-for="(roomDevices, room) in grouped" :key="room" class="mt-8">
      <h2 class="text-lg font-medium text-slate-600">{{ room || 'Без комнаты' }}</h2>
      <div class="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <DeviceCard v-for="d in roomDevices" :key="d.id" :device="d" @removed="load" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import api from '../api'
import DeviceCard from '../components/DeviceCard.vue'

const router = useRouter()
const auth = useAuthStore()

const devices = ref([])
const loading = ref(true)
const showForm = ref(false)
const form = ref({ name: '', type: 'light', room: '', id: '' })

const grouped = computed(() => {
  const g = {}
  for (const d of devices.value) {
    ;(g[d.room] ||= []).push(d)
  }
  return g
})

async function load() {
  loading.value = true
  const { data } = await api.get('/api/v1/devices')
  devices.value = data
  loading.value = false
}

async function create() {
  await api.post('/api/v1/devices', {
    id: form.value.id,
    name: form.value.name,
    type: form.value.type,
    room: form.value.room
  })
  form.value = { name: '', type: 'light', room: '', id: '' }
  showForm.value = false
  await load()
}

function logout() {
  auth.logout()
  router.push('/login')
}

onMounted(load)
</script>
