import axios from 'axios'

// в проде ходим на тот же origin (nginx проксирует /api), в дев — vite proxy
const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || ''
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// если токен протух — выкидываем на логин. туповато, но работает
api.interceptors.response.use(
  (r) => r,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api
