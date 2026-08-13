<template>
  <div class="mx-auto max-w-5xl px-4 py-8">
    <div class="flex items-center justify-between">
      <div>
        <router-link to="/" class="text-sm text-indigo-600 hover:underline">← все устройства</router-link>
        <h1 class="mt-1 text-2xl font-semibold">{{ device?.name || '...' }}</h1>
        <p class="text-sm text-slate-500">{{ device?.room || 'без комнаты' }}</p>
      </div>
      <span
        :class="device?.status === 'online' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'"
        class="rounded-full px-3 py-1 text-sm font-medium"
      >
        {{ device?.status || '—' }}
      </span>
    </div>

    <!-- управление -->
    <div class="mt-6 rounded-2xl bg-white p-6 shadow">
      <h2 class="font-medium text-slate-700">Управление</h2>

      <div v-if="device?.type === 'light'" class="mt-4 flex items-center gap-4">
        <button
          :class="state.on ? 'bg-amber-400 text-white' : 'bg-slate-100 text-slate-600'"
          class="rounded-lg px-4 py-2 text-sm"
          @click="send(state.on ? 'off' : 'on')"
        >
          {{ state.on ? 'Выключить' : 'Включить' }}
        </button>
        <label class="flex items-center gap-2 text-sm text-slate-600">
          Яркость
          <input v-model.number="brightness" type="range" min="0" max="100" class="w-40"
            @change="send('set_brightness', brightness)" />
          <span class="text-slate-400">{{ brightness }}%</span>
        </label>
      </div>

      <div v-else-if="device?.type === 'plug'" class="mt-4">
        <button
          :class="state.on ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-600'"
          class="rounded-lg px-4 py-2 text-sm"
          @click="send(state.on ? 'off' : 'on')"
        >
          {{ state.on ? 'Выключить' : 'Включить' }}
        </button>
      </div>

      <div v-else-if="device?.type === 'thermostat'" class="mt-4 flex items-center gap-3">
        <span class="text-sm text-slate-600">Целевая температура</span>
        <input v-model.number="target" type="number" class="w-20 rounded-lg border px-2 py-1.5 text-sm" />
        <button class="rounded-lg bg-indigo-600 px-4 py-2 text-sm text-white" @click="send('set_target', target)">
          Применить
        </button>
      </div>

      <p v-else class="mt-2 text-sm text-slate-500">Датчик — управлять нечем, просто смотрим телеметрию.</p>
    </div>

    <!-- график -->
    <div class="mt-6 rounded-2xl bg-white p-6 shadow">
      <div class="flex items-center justify-between">
        <h2 class="font-medium text-slate-700">Телеметрия</h2>
        <div class="flex gap-4 text-sm">
          <span class="text-slate-500">батарея {{ last?.battery ?? '—' }}%</span>
          <span class="text-slate-500">{{ last?.temp ?? '—' }}°C</span>
        </div>
      </div>
      <LiveChart class="mt-4" :points="points" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api'
import LiveChart from '../components/LiveChart.vue'

const route = useRoute()
const devices = ref([])
const points = ref([])
let ws = null

const device = computed(() => devices.value.find((d) => d.id === route.params.id))
const state = computed(() => device.value?.state || {})
const last = computed(() => points.value[points.value.length - 1])

const brightness = ref(100)
const target = ref(22)

async function loadDevice() {
  const { data } = await api.get('/api/v1/devices')
  devices.value = data
  if (device.value) {
    brightness.value = device.value.state?.brightness ?? 100
    target.value = device.value.state?.target_temp ?? 22
  }
}

async function loadHistory() {
  const { data } = await api.get(`/api/v1/devices/${route.params.id}/telemetry`)
  points.value = data.map((t) => ({ ...t.payload, ts: new Date(t.ts).getTime() }))
}

function openWs() {
  const base = import.meta.env.VITE_WS_URL || 'ws://localhost:4000'
  ws = new WebSocket(base + '/ws/dashboard')
  ws.onopen = () => ws.send(JSON.stringify({ type: 'subscribe', device_id: route.params.id }))
  ws.onmessage = (e) => {
    const p = JSON.parse(e.data)
    points.value.push(p)
  }
}

async function send(action, value) {
  try {
    await api.post(`/api/v1/devices/${route.params.id}/command`, { action, value })
    await loadDevice()
  } catch (e) {
    alert(e.response?.data?.error || 'не отправилось')
  }
}

onMounted(async () => {
  await loadDevice()
  await loadHistory()
  openWs()
})

onBeforeUnmount(() => ws?.close())
</script>
