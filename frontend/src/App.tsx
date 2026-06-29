import { useAuthStore } from './store/authStore'
import { LoginPage } from './pages/loginPage'
import { ChatPage } from './pages/chatPage'

function App() {
  const { isAuthenticated } = useAuthStore()

  return isAuthenticated ? <ChatPage /> : <LoginPage />
}

export default App