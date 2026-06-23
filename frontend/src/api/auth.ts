import { client } from './client'
import type { AuthResponse } from '../types'

export const authApi = {
  register: (username: string, email: string, password: string) =>
    client.post<AuthResponse>('/auth/register', { username, email, password }),

  login: (email: string, password: string) =>
    client.post<AuthResponse>('/auth/login', { email, password }),

  refresh: () =>
    client.post<{ access_token: string }>('/jwt/refresh'),
}