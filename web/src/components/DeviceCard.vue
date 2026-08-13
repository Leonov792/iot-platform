<template>
  <div class="rounded-2xl bg-white p-5 shadow transition hover:shadow-md">
    <div class="flex items-start justify-between">
      <div>
        <h3 class="font-medium">{{ device.name }}</h3>
        <p class="text-xs text-slate-400">{{ typeLabel }} · {{ device.room || 'без комнаты' }}</p>
      </div>
      <span
        :class="device.status === 'online' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'"
        class="rounded-full px-2 py-0.5 text-xs font-medium"
      >
        {{ device.status }}
      </span>
    </div>

    <div class="mt-4 flex items-center gap-2">
      <template v-if="device.type === 'light'">
        <button
          :class="state.on ? 'bg-amber-400 text-white' : 'bg-slate-100 text-slate-600'"
          class="rounded-lg px-3 py-1.5 text-sm"
          @click="send(state.on ? 'off' : 'on')"
        >
          {{ state.on ? 'Выкл' : 'Вкл' }}
        </button>
      </template>

      <template v-else-if="device.type === 'plug'">
        <button
          :class="state.on ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-600'"
          class="rounded-lg px-3 py-1.5 text-sm"
          @click="send(state.on ? 'off' : 'on')"
        >
          {{ state.on ? 'Выключить' : 'Включить' }}
        </button>
      </template>

      <template v-else-if="device.type === 'thermostat'">
        <input
          v-model.number="target"
          type="number"
          class="w-20 rounded-lg border px-2 py-1.5 text-sm"
        />
        <button
          class="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm text-white"
          @click="send('set_target', target)"
        >
          ОК
        </button>
      </template>

      <template v-else>
        <span class="text-sm text-slate-500">датчик</span>
      </template>
    </div>

    <div class="mt-4 flex items-center justify-between">
      <button class="text-sm text-red-500 hover:underline" @click="remove">удалить</button>
      <router-link :to="`/devices/${device.id}`" class="text-sm text-indigo-600 hover:underline">
        открыть →
      </router-link>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import api from '../api'

const props = defineProps({ device: { type: Object, required: true } })
const emit = defineEmits(['removed'])

const labels = { light: 'Лампа', plug: 'Розетка', thermostat: 'Термостат', sensor: 'Датчик' }
const typeLabel = computed(() => labels[props.device.type] || props.device.type)
const state = computed(() => props.device.state || {})

const target = ref(state.value.target_temp || 22)

async function send(action, value) {
  try {
    await api.post(`/api/v1/devices/${props.device.id}/command`, { action, value })
  } catch (e) {
    alert(e.response?.data?.error || 'не отправилось')
  }
}

async function remove() {
  await api.delete(`/api/v1/devices/${props.device.id}`)
  emit('removed')
}
</script>
