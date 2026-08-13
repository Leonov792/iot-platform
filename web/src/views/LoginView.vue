<template>
  <div class="flex min-h-screen items-center justify-center bg-slate-100 px-4">
    <div class="w-full max-w-sm rounded-2xl bg-white p-8 shadow-lg">
      <h1 class="text-2xl font-semibold text-slate-800">Умный дом</h1>
      <p class="mt-1 text-sm text-slate-500">{{ isLogin ? 'Вход' : 'Регистрация' }}</p>

      <form class="mt-6 space-y-4" @submit.prevent="submit">
        <div>
          <label class="text-sm text-slate-600">Почта</label>
          <input
            v-model="email"
            type="email"
            required
            class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
          />
        </div>
        <div>
          <label class="text-sm text-slate-600">Пароль</label>
          <input
            v-model="password"
            type="password"
            required
            minlength="6"
            class="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
          />
        </div>

        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded-lg bg-indigo-600 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {{ isLogin ? 'Войти' : 'Создать аккаунт' }}
        </button>
      </form>

      <button
        class="mt-4 w-full text-center text-sm text-indigo-600 hover:underline"
        @click="isLogin = !isLogin"
      >
        {{ isLogin ? 'Нет аккаунта? Зарегистрируйся' : 'Уже есть? Войти' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const email = ref('')
const password = ref('')
const isLogin = ref(true)
const loading = ref(false)
const error = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  try {
    if (isLogin.value) await auth.login(email.value, password.value)
    else await auth.register(email.value, password.value)
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.error || 'что-то пошло не так'
  } finally {
    loading.value = false
  }
}
</script>
