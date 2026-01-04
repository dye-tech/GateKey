import { ReactNode, useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import Sidebar from './Sidebar'

interface LayoutProps {
  children: ReactNode
}

function ThemeToggle() {
  const { setTheme, resolvedTheme } = useTheme()

  return (
    <div className="relative">
      <button
        onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
        className="p-2 rounded-lg text-theme-secondary hover:bg-theme-tertiary transition-colors"
        title={`Switch to ${resolvedTheme === 'dark' ? 'light' : 'dark'} mode`}
      >
        {resolvedTheme === 'dark' ? (
          // Sun icon for dark mode (click to switch to light)
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
        ) : (
          // Moon icon for light mode (click to switch to dark)
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
        )}
      </button>
    </div>
  )
}

export default function Layout({ children }: LayoutProps) {
  const { user } = useAuth()
  const { resolvedTheme } = useTheme()
  const [sidebarOpen, setSidebarOpen] = useState(() => {
    // On desktop, check localStorage for preference; default to open
    if (typeof window !== 'undefined' && window.innerWidth >= 1024) {
      const stored = localStorage.getItem('sidebarOpen')
      return stored !== null ? stored === 'true' : true
    }
    // On mobile, default to closed
    return false
  })

  // Save sidebar state to localStorage on desktop
  useEffect(() => {
    if (window.innerWidth >= 1024) {
      localStorage.setItem('sidebarOpen', String(sidebarOpen))
    }
  }, [sidebarOpen])

  // Close sidebar on mobile when navigating
  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 1024) {
        setSidebarOpen(false)
      }
    }
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  return (
    <div className="min-h-screen bg-theme-secondary flex">
      {/* Sidebar */}
      <Sidebar isOpen={sidebarOpen} onToggle={() => setSidebarOpen(!sidebarOpen)} />

      {/* Main content area */}
      <div className="flex-1 flex flex-col min-h-screen">
        {/* Top header bar */}
        <header className="bg-theme-header shadow-sm sticky top-0 z-30 border-b border-theme">
          <div className="px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center h-16">
              {/* Left side - hamburger menu for mobile */}
              <div className="flex items-center">
                <button
                  onClick={() => setSidebarOpen(!sidebarOpen)}
                  className="lg:hidden p-2 rounded-md text-theme-secondary hover:text-theme-primary hover:bg-theme-tertiary"
                >
                  <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                  </svg>
                </button>
                {/* Show logo on mobile when sidebar is closed */}
                <img
                  src={resolvedTheme === 'dark' ? '/logo-transparent.png' : '/logo.png'}
                  alt="GateKey"
                  className="h-16 ml-2 lg:hidden"
                />
              </div>

              {/* Right side - theme toggle and user info */}
              <div className="flex items-center gap-4">
                <ThemeToggle />
                <div className="hidden sm:flex items-center text-sm">
                  <span className="text-theme-tertiary">Signed in as</span>
                  <span className="ml-1 font-medium text-theme-primary">{user?.email}</span>
                </div>
              </div>
            </div>
          </div>
        </header>

        {/* Main content */}
        <main className="flex-1 px-4 sm:px-6 lg:px-8 py-6">
          <div className="max-w-7xl mx-auto">
            {children}
          </div>
        </main>

        {/* Footer */}
        <footer className="bg-theme-card border-t border-theme mt-auto">
          <div className="px-4 sm:px-6 lg:px-8 py-4">
            <div className="max-w-7xl mx-auto">
              <p className="text-sm text-theme-muted text-center sm:text-left">
                GateKey - Zero Trust VPN with Software Defined Perimeter
              </p>
            </div>
          </div>
        </footer>
      </div>
    </div>
  )
}
