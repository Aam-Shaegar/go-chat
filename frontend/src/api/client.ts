import axios from 'axios'

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:5050/api/v1'

export const client = axios.create({
  baseURL: BASE_URL,
  withCredentials: true, // нужно для refresh_token cookie
})

// Автоматически добавляем access token в каждый запрос
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// При 401 — пробуем обновить токен, потом повторяем запрос
client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config

    if (error.response?.status === 401 && !original._retry) {
      original._retry = true
      try {
        const { data } = await axios.post(
          `${BASE_URL}/jwt/refresh`,
          {},
          { withCredentials: true }
        )
        localStorage.setItem('access_token', data.access_token)
        original.headers.Authorization = `Bearer ${data.access_token}`
        return client(original)
      } catch {
        localStorage.removeItem('access_token')
        window.location.href = '/login'
      }
    }

    return Promise.reject(error)
  }
)