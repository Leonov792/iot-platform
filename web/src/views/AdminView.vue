<template>
  <div class="mx-auto max-w-6xl px-4 py-8">
    <header class="flex items-center justify-between">
      <div>
        <router-link to="/" class="text-sm text-indigo-600 hover:underline">← устройства</router-link>
        <h1 class="mt-1 text-2xl font-semibold">Администрирование</h1>
      </div>
    </header>

    <!-- Локальный ИИ -->
    <section class="mt-8 rounded-2xl bg-white p-6 shadow">
      <h2 class="font-medium text-slate-700">Локальный ИИ (Ollama)</h2>

      <div v-if="loadingAI" class="mt-3 text-sm text-slate-400">грузим...</div>
      <div v-else class="mt-3 grid gap-4 text-sm sm:grid-cols-3">
        <div class="rounded-lg bg-slate-50 p-3">
          <p class="text-slate-500">Статус</p>
          <p :class="ai.online ? 'text-emerald-600' : 'text-red-600'" class="font-medium">
            {{ ai.online ? 'online' : 'offline' }}
          </p>
        </div>
        <div class="rounded-lg bg-slate-50 p-3">
          <p class="text-slate-500">Модель</p>
          <p class="font-medium">{{ ai.model || '—' }}</p>
        </div>
        <div class="rounded-lg bg-slate-50 p-3">
          <p class="text-slate-500">Загружено в память</p>
          <p class="font-medium">{{ ai.running?.length ? ai.running.join(', ') : '—' }}</p>
        </div>
      </div>

      <div v-if="ai.models?.length" class="mt-4">
        <p class="text-xs text-slate-500">Доступные модели: {{ ai.models.join(', ') }}</p>
      </div>

      <div v-if="recommendations.length" class="mt-4">
        <p class="mb-2 text-sm font-medium text-slate-600">Предиктивные рекомендации</p>
        <ul class="space-y-2">
          <li v-for="r in recommendations" :key="r.device_id + r.action + r.weekday"
            class="rounded-lg border border-slate-200 p-3 text-sm">
            {{ r.text }}
          </li>
        </ul>
      </div>
    </section>

    <!-- Матрица прав -->
    <section class="mt-6 rounded-2xl bg-white p-6 shadow">
      <h2 class="font-medium text-slate-700">Пользователи и права</h2>

      <p v-if="!users.length" class="mt-3 text-sm text-slate-400">нет членов дома</p>
      <div v-else class="mt-4 overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b text-left text-slate-500">
              <th class="py-2 pr-4">Почта</th>
              <th class="py-2 pr-4">Роль</th>
              <th class="py-2 pr-4">Расписание (зона · дни · часы)</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.id" class="border-b">
              <td class="py-2 pr-4">{{ u.email }}</td>
              <td class="py-2 pr-4">
                <select
                  :value="u.role"
                  :disabled="u.role === 'owner'"
                  class="rounded-lg border px-2 py-1 disabled:opacity-50"
                  @change="setRole(u, $event.target.value)"
                >
                  <option value="owner">owner</option>
                  <option value="family">family</option>
                  <option value="staff">staff</option>
                </select>
              </td>
              <td class="py-2">
                <div v-for="(s, i) in u.schedule" :key="i" class="flex items-center gap-2 py-1">
                  <span class="text-slate-500">{{ s.zone }}</span>
                  <input v-model="s.start" class="w-20 rounded border px-1 py-0.5 text-xs" />
                  <span>—</span>
                  <input v-model="s.end" class="w-20 rounded border px-1 py-0.5 text-xs" />
                  <button class="text-xs text-indigo-600 hover:underline" @click="saveSchedule(u)">сохранить</button>
                </div>
                <span v-if="!u.schedule?.length" class="text-slate-400">—</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Логи Modbus -->
    <section class="mt-6 rounded-2xl bg-white p-6 shadow">
      <div class="flex items-center justify-between">
        <h2 class="font-medium text-slate-700">Логи Modbus</h2>
        <button class="text-sm text-indigo-600 hover:underline" @click="loadLogs">обновить</button>
      </div>

      <div v-if="!logs.length" class="mt-3 text-sm text-slate-400">операций пока не было</div>
      <div v-else class="mt-3 max-h-72 overflow-y-auto rounded-lg bg-slate-50 p-2 font-mono text-xs">
        <div v-for="(l, i) in logs" :key="i" class="flex gap-3 py-0.5">
          <span class="text-slate-400">{{ new Date(l.ts).toLocaleTimeString() }}</span>
          <span :class="l.error ? 'text-red-600' : 'text-slate-600'">
            {{ l.kind }} {{ l.device }}/{{ l.target }} {{ l.value ?? '' }} {{ l.error ? '— ' + l.error : '' }}
          </span>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const AI_URL = import.meta.env.VITE_AI_URL || 'http://localhost:8095'
const MODBUS_URL = import.meta.env.VITE_MODBUS_URL || 'http://localhost:8090'

const ai = ref({ online: false, model: '', models: [], running: [] })
const recommendations = ref([])
const users = ref([])
const logs = ref([])
const loadingAI = ref(true)

async function loadAI() {
  loadingAI.value = true
  try {
    const { data } = await api.get(AI_URL + '/v1/status')
    ai.value = data
  } catch (e) {
    ai.value = { online: false, model: '', models: [], running: [] }
  }
  loadingAI.value = false

  try {
    const { data } = await api.get(AI_URL + '/v1/recommendations')
    recommendations.value = data
  } catch (e) {
    recommendations.value = []
  }
}

async function loadUsers() {
  try {
    const { data } = await api.get('/api/v1/users')
    users.value = data.map((u) => ({ ...u, schedule: u.schedule || [] }))
  } catch (e) {
    users.value = []
  }
}

async function loadLogs() {
  try {
    const { data } = await api.get(MODBUS_URL + '/logs')
    logs.value = data
  } catch (e) {
    logs.value = []
  }
}

async function setRole(u, role) {
  try {
    await api.put(`/api/v1/users/${u.id}/role`, { role })
    u.role = role
  } catch (e) {
    alert(e.response?.data?.error || 'не сохранилось')
  }
}

async function saveSchedule(u) {
  try {
    await api.put(`/api/v1/users/${u.id}/schedule`, { schedule: u.schedule })
  } catch (e) {
    alert(e.response?.data?.error || 'не сохранилось')
  }
}

onMounted(() => {
  loadAI()
  loadUsers()
  loadLogs()
})
</script>
