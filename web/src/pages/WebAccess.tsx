import { useState, useEffect } from 'react'
import { getUserProxyApps, UserProxyApplication } from '../api/client'

export default function WebAccess() {
  const [apps, setApps] = useState<UserProxyApplication[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadApps()
  }, [])

  async function loadApps() {
    try {
      setLoading(true)
      const data = await getUserProxyApps()
      setApps(data)
      setError(null)
    } catch {
      setError('Failed to load applications')
    } finally {
      setLoading(false)
    }
  }

  function openApp(app: UserProxyApplication) {
    window.open(app.proxyUrl, '_blank', 'noopener,noreferrer')
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">Web Access</h1>
        <p className="text-theme-tertiary mt-1">
          Access internal web applications directly from your browser without VPN client
        </p>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : apps.length === 0 ? (
        <div className="card text-center py-12">
          <svg className="mx-auto h-12 w-12 text-theme-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-theme-primary">No applications available</h3>
          <p className="mt-2 text-theme-tertiary">
            You don't have access to any web applications yet.
            Contact your administrator if you need access.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {apps.map((app) => (
            <div
              key={app.id}
              className="card hover:shadow-lg transition-shadow cursor-pointer group"
              onClick={() => openApp(app)}
            >
              <div className="flex items-start space-x-4">
                {/* Icon */}
                <div className="flex-shrink-0">
                  {app.iconUrl ? (
                    <img
                      src={app.iconUrl}
                      alt=""
                      className="h-12 w-12 rounded-lg"
                      onError={(e) => {
                        // Fallback to default icon if image fails to load
                        (e.target as HTMLImageElement).style.display = 'none'
                      }}
                    />
                  ) : (
                    <div className="h-12 w-12 rounded-lg bg-primary-100 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 flex items-center justify-center">
                      <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                      </svg>
                    </div>
                  )}
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <h3 className="text-lg font-semibold text-theme-primary group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                    {app.name}
                  </h3>
                  <p className="text-sm text-theme-tertiary mt-1 line-clamp-2">
                    {app.description || 'No description'}
                  </p>
                </div>

                {/* Launch indicator */}
                <div className="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                  <svg className="h-5 w-5 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </div>
              </div>

              {/* Footer */}
              <div className="mt-4 pt-4 border-t border-theme flex items-center justify-between">
                <span className="text-xs text-theme-muted font-mono">
                  {app.slug}
                </span>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    openApp(app)
                  }}
                  className="text-sm font-medium text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
                >
                  Open
                  <svg className="inline-block ml-1 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Info section */}
      {apps.length > 0 && (
        <div className="info-box">
          <div className="flex items-start space-x-3">
            <svg className="h-5 w-5 info-box-icon mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <h4 className="info-box-title">Secure Web Access</h4>
              <p className="info-box-text mt-1">
                These applications are accessed through a secure reverse proxy. Your session is authenticated
                and all access is logged. No VPN client installation required.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
