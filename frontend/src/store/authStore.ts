import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '../types'
import { useChatStore } from './chatStore'

interface AuthState {
  user: User | null
  accessToken: string | null
  isAuthenticated: boolean
  setAuth: (user: User, token: string) => void
  setAccessToken: (token: string) => void
  clearAuth: () => void
}

type PersistedAuthState = Pick<AuthState, 'user' | 'accessToken'>

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,

      setAuth: (user, token) => {
        localStorage.setItem('access_token', token)
        set({ user, accessToken: token, isAuthenticated: true })
      },

      setAccessToken: (token) => {
        localStorage.setItem('access_token', token)
        set({ accessToken: token, isAuthenticated: true })
      },

      clearAuth: () => {
        localStorage.removeItem('access_token')
        useChatStore.getState().resetChat()
        set({ user: null, accessToken: null, isAuthenticated: false })
      },
    }),
    {
      name: 'auth',
      partialize: (state) => ({ user: state.user, accessToken: state.accessToken }),
      merge: (persisted, current) => {
        const auth = persisted as Partial<PersistedAuthState> | undefined
        const accessToken = auth?.accessToken ?? null
        if (accessToken) {
          localStorage.setItem('access_token', accessToken)
        } else {
          localStorage.removeItem('access_token')
        }

        return {
          ...current,
          user: auth?.user ?? null,
          accessToken,
          isAuthenticated: Boolean(auth?.user && accessToken),
        }
      },
    }
  )
)
