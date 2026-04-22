let isLoginMode = true

function initAuth() {
  document.getElementById('tab-login').onclick    = () => switchTab(true)
  document.getElementById('tab-register').onclick = () => switchTab(false)

  document.getElementById('auth-password').onkeydown = e => {
    if (e.key === 'Enter') document.getElementById('auth-submit').click()
  }

  document.getElementById('auth-submit').onclick = submitAuth
}

function switchTab(login) {
  isLoginMode = login
  document.getElementById('tab-login').classList.toggle('active', login)
  document.getElementById('tab-register').classList.toggle('active', !login)
  document.getElementById('field-username').style.display = login ? 'none' : 'block'
  document.getElementById('auth-submit').textContent = login ? 'Войти' : 'Зарегистрироваться'
  document.getElementById('auth-error').textContent = ''
}

async function submitAuth() {
  const email    = document.getElementById('auth-email').value.trim()
  const password = document.getElementById('auth-password').value
  const username = document.getElementById('auth-username').value.trim()

  if (!email || !password) { setAuthErr('Заполните все поля'); return }
  if (!isLoginMode && !username) { setAuthErr('Введите имя пользователя'); return }

  const btn = document.getElementById('auth-submit')
  btn.textContent = '...'; btn.disabled = true

  try {
    const path = isLoginMode ? '/api/auth/login' : '/api/auth/register'
    const body = isLoginMode ? { email, password } : { email, password, username }
    const data = await apiPost(path, body)
    if (data.error) { setAuthErr(data.error); return }

    token = data.access_token
    currentUser = data.user
    localStorage.setItem('gochat_token', token)
    localStorage.setItem('gochat_user', JSON.stringify(currentUser))
    showApp()
  } finally {
    btn.textContent = isLoginMode ? 'Войти' : 'Зарегистрироваться'
    btn.disabled = false
  }
}

function setAuthErr(msg) {
  document.getElementById('auth-error').textContent = msg
}