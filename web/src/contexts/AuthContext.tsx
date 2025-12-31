import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { api } from '../api/client'

interface User {
  id: string
  email: string
  name: string
  groups: string[]
  isAdmin: boolean
}

interface AuthContextType {
  user: User | null
  loading: boolean
  login: (provider: string) => void
  logout: () => Promise<void>
  refreshSession: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    checkSession()
  }, [])

  async function checkSession() {
    try {
      const response = await api.get('/api/v1/auth/session')
      if (response.data.user) {
        setUser(response.data.user)
      }
    } catch (error) {
      // Not authenticated
      setUser(null)
    } finally {
      setLoading(false)
    }
  }

  function login(provider: string) {
    // Redirect to the appropriate login URL
    window.location.href = `/api/v1/auth/${provider}/login`
  }

  async function logout() {
    try {
      await api.post('/api/v1/auth/logout')
      setUser(null)
      window.location.href = '/login'
    } catch (error) {
      console.error('Logout failed:', error)
    }
  }

  async function refreshSession() {
    await checkSession()
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refreshSession }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
