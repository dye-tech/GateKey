import { ReactNode, useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import Sidebar from './Sidebar'

interface LayoutProps {
  children: ReactNode
}

export default function Layout({ children }: LayoutProps) {
  const { user } = useAuth()
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
    <div className="min-h-screen bg-gray-50 flex">
      {/* Sidebar */}
      <Sidebar isOpen={sidebarOpen} onToggle={() => setSidebarOpen(!sidebarOpen)} />

      {/* Main content area */}
      <div className="flex-1 flex flex-col min-h-screen">
        {/* Top header bar */}
        <header className="bg-white shadow-sm sticky top-0 z-30">
          <div className="px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between items-center h-16">
              {/* Left side - hamburger menu for mobile */}
              <div className="flex items-center">
                <button
                  onClick={() => setSidebarOpen(!sidebarOpen)}
                  className="lg:hidden p-2 rounded-md text-gray-500 hover:text-gray-600 hover:bg-gray-100"
                >
                  <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                  </svg>
                </button>
                {/* Show logo on mobile when sidebar is closed */}
                <img
                  src="/logo.png"
                  alt="GateKey"
                  className="h-16 ml-2 lg:hidden"
                />
              </div>

              {/* Right side - user info */}
              <div className="flex items-center">
                <div className="hidden sm:flex items-center text-sm">
                  <span className="text-gray-500">Signed in as</span>
                  <span className="ml-1 font-medium text-gray-900">{user?.email}</span>
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
        <footer className="bg-white border-t border-gray-200 mt-auto">
          <div className="px-4 sm:px-6 lg:px-8 py-4">
            <div className="max-w-7xl mx-auto">
              <p className="text-sm text-gray-500 text-center sm:text-left">
                GateKey - Zero Trust VPN with Software Defined Perimeter
              </p>
            </div>
          </div>
        </footer>
      </div>
    </div>
  )
}
