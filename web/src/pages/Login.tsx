import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { getProviders, localLogin, AuthProvider } from '../api/client'

export default function Login() {
  const [providers, setProviders] = useState<AuthProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showLocalForm, setShowLocalForm] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loginLoading, setLoginLoading] = useState(false)
  const { user, refreshSession } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  // Check if this is a CLI login flow
  const cliState = searchParams.get('state')
  const isCLILogin = searchParams.get('cli') === 'true'
  const urlError = searchParams.get('error')
  const [cliCompleteAttempted, setCLICompleteAttempted] = useState(false)

  useEffect(() => {
    if (user) {
      // If this is a CLI login and user is already logged in,
      // complete the CLI callback with existing session
      // Don't retry if we already attempted or if there's an error from a previous attempt
      if (isCLILogin && cliState && !cliCompleteAttempted && !urlError) {
        setCLICompleteAttempted(true)
        // Redirect to backend to complete CLI flow with existing session
        window.location.href = `/api/v1/auth/cli/complete?state=${encodeURIComponent(cliState)}`
        return
      }
      // If there was an error completing CLI, let user try SSO login again
      if (!isCLILogin) {
        navigate('/')
      }
    }
  }, [user, navigate, isCLILogin, cliState, cliCompleteAttempted, urlError])

  useEffect(() => {
    loadProviders()
  }, [])

  async function loadProviders() {
    try {
      const data = await getProviders()
      setProviders(data)
      // If no SSO providers are configured, default to local login form
      const ssoProviders = data.filter(p => p.type !== 'local')
      if (ssoProviders.length === 0) {
        setShowLocalForm(true)
      }
    } catch (err) {
      setError('Failed to load authentication providers')
    } finally {
      setLoading(false)
    }
  }

  function handleLogin(provider: AuthProvider) {
    if (provider.type === 'local') {
      setShowLocalForm(true)
      setError(null)
    } else {
      // Append CLI state to login URL if this is a CLI login flow
      let loginUrl = provider.loginUrl
      if (isCLILogin && cliState) {
        loginUrl += (loginUrl.includes('?') ? '&' : '?') + 'cli_state=' + encodeURIComponent(cliState)
      }
      window.location.href = loginUrl
    }
  }

  async function handleLocalLogin(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoginLoading(true)

    try {
      await localLogin(username, password)
      await refreshSession()
      navigate('/')
    } catch (err) {
      setError('Invalid username or password')
    } finally {
      setLoginLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary-50 to-primary-100">
      <div className="max-w-md w-full mx-4">
        <div className="card">
          {/* Logo and title */}
          <div className="text-center mb-8">
            <div className="flex justify-center mb-4">
              <img src="/logo.png" alt="GateKey" className="h-32" />
            </div>
            <p className="text-gray-500 mt-2">Zero Trust VPN Access</p>
          </div>

          {/* Error message */}
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {error}
            </div>
          )}

          {/* Loading state */}
          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : showLocalForm ? (
            /* Local login form */
            <form onSubmit={handleLocalLogin} className="space-y-4">
              <div>
                <label htmlFor="username" className="block text-sm font-medium text-gray-700 mb-1">
                  Username
                </label>
                <input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  placeholder="admin"
                  required
                  autoFocus
                />
              </div>
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
                  Password
                </label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  placeholder="Enter password"
                  required
                />
              </div>
              <button
                type="submit"
                disabled={loginLoading}
                className="w-full btn btn-primary py-3 disabled:opacity-50"
              >
                {loginLoading ? (
                  <span className="flex items-center justify-center">
                    <svg className="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    Signing in...
                  </span>
                ) : (
                  'Sign In'
                )}
              </button>
              <button
                type="button"
                onClick={() => setShowLocalForm(false)}
                className="w-full text-sm text-gray-500 hover:text-gray-700"
              >
                Back to provider selection
              </button>
            </form>
          ) : (
            <>
              {/* Provider buttons */}
              <div className="space-y-3">
                {providers.filter(p => p.type !== 'local').map((provider) => (
                  <button
                    key={`${provider.type}:${provider.name}`}
                    onClick={() => handleLogin(provider)}
                    className="w-full flex items-center justify-center space-x-3 btn btn-primary py-3"
                  >
                    {provider.type === 'oidc' && (
                      <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                      </svg>
                    )}
                    {provider.type === 'saml' && (
                      <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                        <path d="M10 2a5 5 0 00-5 5v2a2 2 0 00-2 2v5a2 2 0 002 2h10a2 2 0 002-2v-5a2 2 0 00-2-2H7V7a3 3 0 015.905-.75 1 1 0 001.937-.5A5.002 5.002 0 0010 2z" />
                      </svg>
                    )}
                    <span>{provider.displayName}</span>
                  </button>
                ))}
              </div>

              {providers.filter(p => p.type !== 'local').length === 0 && (
                <p className="text-center text-gray-500 py-4">
                  No SSO providers configured yet.
                </p>
              )}

              {/* Local user login link */}
              {providers.some(p => p.type === 'local') && (
                <div className="mt-6 pt-4 border-t border-gray-200">
                  <button
                    onClick={() => setShowLocalForm(true)}
                    className="w-full flex items-center justify-center space-x-2 text-sm text-gray-600 hover:text-gray-900"
                  >
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                    <span>Sign in as Local User</span>
                  </button>
                </div>
              )}
            </>
          )}

          {/* Info text */}
          <div className="mt-8 pt-6 border-t border-gray-200">
            <p className="text-xs text-gray-500 text-center">
              {showLocalForm
                ? 'Use your local account credentials to sign in.'
                : 'Sign in with your organization\'s identity provider to access VPN configurations.'}
            </p>
          </div>
        </div>

        {/* Footer */}
        <p className="text-center text-sm text-gray-500 mt-4">
          Secure access powered by GateKey SDP
        </p>
      </div>
    </div>
  )
}
