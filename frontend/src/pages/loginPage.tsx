import { useState } from 'react'
import type { FormEvent } from 'react'
import { useAuthStore } from '../store/authStore'
import { authApi } from '../api/auth'

export function LoginPage() {
  const [isRegister, setIsRegister] = useState(false)
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { setAuth } = useAuthStore()

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      const { data } = isRegister
        ? await authApi.register(username.trim(), email.trim(), password)
        : await authApi.login(email.trim(), password)

      setAuth(data.user, data.access_token)
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { message?: string } } })
        ?.response?.data?.message
      setError(msg || 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="grid min-h-screen place-items-center bg-[#eef3f8] px-4 py-8 text-slate-950">
      <section className="w-full max-w-sm">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-4 grid h-16 w-16 place-items-center rounded-full bg-[#229ed9] text-white shadow-lg shadow-sky-200">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="h-8 w-8" aria-hidden="true">
              <path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4v8z" />
            </svg>
          </div>
          <h1 className="text-2xl font-semibold tracking-normal text-slate-950">GoChat</h1>
          <p className="mt-1 text-sm text-slate-500">
            {isRegister ? 'Create account' : 'Sign in to continue'}
          </p>
        </div>

        <div className="rounded-2xl bg-white p-5 shadow-xl shadow-slate-200/70">
          <form onSubmit={handleSubmit} className="space-y-4">
            {isRegister && (
              <Field
                label="Username"
                type="text"
                value={username}
                onChange={setUsername}
                placeholder="username"
                minLength={3}
                required
              />
            )}

            <Field
              label="Email"
              type="email"
              value={email}
              onChange={setEmail}
              placeholder="you@example.com"
              required
            />

            <Field
              label="Password"
              type="password"
              value={password}
              onChange={setPassword}
              placeholder="password"
              minLength={8}
              required
            />

            {error && (
              <p className="rounded-xl bg-red-50 px-3 py-2 text-sm font-medium text-red-600">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="h-11 w-full rounded-full bg-[#229ed9] text-sm font-semibold text-white transition hover:bg-[#168ac0] disabled:cursor-not-allowed disabled:bg-slate-300"
            >
              {loading ? 'Loading...' : isRegister ? 'Create account' : 'Sign in'}
            </button>
          </form>

          <div className="mt-5 text-center text-sm text-slate-500">
            <span>{isRegister ? 'Already have an account?' : 'Need an account?'}</span>
            <button
              type="button"
              onClick={() => {
                setIsRegister(!isRegister)
                setError('')
              }}
              className="ml-1 font-semibold text-[#229ed9] transition hover:text-[#168ac0]"
            >
              {isRegister ? 'Sign in' : 'Create one'}
            </button>
          </div>
        </div>
      </section>
    </main>
  )
}

function Field({ label, type, value, onChange, placeholder, required = false, minLength }: {
  label: string
  type: string
  value: string
  onChange: (value: string) => void
  placeholder: string
  required?: boolean
  minLength?: number
}) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium text-slate-700">{label}</span>
      <input
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        required={required}
        minLength={minLength}
        className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm text-slate-950 outline-none transition placeholder:text-slate-400 focus:border-[#229ed9]"
      />
    </label>
  )
}
