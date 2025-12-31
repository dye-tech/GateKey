import { useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { Link } from 'react-router-dom'
import {
  getGateways,
  getUserProxyApps,
  getAdminGateways,
  getNetworks,
  getUsers,
  getAccessRules,
  getProxyApps,
  Gateway,
  UserProxyApplication,
  AdminGateway,
  Network,
  SSOUser,
  AccessRule,
  ProxyApplication,
} from '../api/client'

interface DashboardStats {
  gateways: { total: number; online: number; offline: number }
  networks: number
  users: number
  accessRules: number
  proxyApps: number
}

export default function Dashboard() {
  const { user } = useAuth()
  const [userGateways, setUserGateways] = useState<Gateway[]>([])
  const [userProxyApps, setUserProxyApps] = useState<UserProxyApplication[]>([])
  const [adminStats, setAdminStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadDashboardData()
  }, [user])

  async function loadDashboardData() {
    try {
      setLoading(true)

      // Load user data
      const [gateways, proxyApps] = await Promise.all([
        getGateways().catch(() => []),
        getUserProxyApps().catch(() => []),
      ])
      setUserGateways(gateways)
      setUserProxyApps(proxyApps)

      // Load admin stats if admin
      if (user?.isAdmin) {
        const [adminGateways, networks, users, accessRules, allProxyApps] = await Promise.all([
          getAdminGateways().catch(() => [] as AdminGateway[]),
          getNetworks().catch(() => [] as Network[]),
          getUsers().catch(() => [] as SSOUser[]),
          getAccessRules().catch(() => [] as AccessRule[]),
          getProxyApps().catch(() => [] as ProxyApplication[]),
        ])

        // Gateway is online if heartbeat is within last 2 minutes
        const isOnline = (heartbeat: string | null) => {
          if (!heartbeat) return false
          const lastBeat = new Date(heartbeat).getTime()
          const now = Date.now()
          return (now - lastBeat) < 2 * 60 * 1000
        }

        setAdminStats({
          gateways: {
            total: adminGateways.length,
            online: adminGateways.filter(g => g.isActive && isOnline(g.lastHeartbeat)).length,
            offline: adminGateways.filter(g => !g.isActive || !isOnline(g.lastHeartbeat)).length,
          },
          networks: networks.length,
          users: users.length,
          accessRules: accessRules.length,
          proxyApps: allProxyApps.length,
        })
      }
    } catch (err) {
      console.error('Failed to load dashboard data:', err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Welcome section */}
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">
          Welcome back, {user?.name || user?.email}
        </h1>
        <p className="text-gray-500 mt-1">
          {user?.isAdmin
            ? 'Manage your VPN infrastructure and monitor system status.'
            : 'Access your VPN configurations and internal applications.'}
        </p>
      </div>

      {/* Admin Stats */}
      {user?.isAdmin && adminStats && (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
          <Link to="/admin/gateways" className="card hover:shadow-lg transition-shadow p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">Gateways</p>
                <p className="text-2xl font-bold text-gray-900">{adminStats.gateways.total}</p>
              </div>
              <div className="p-2 bg-green-100 rounded-lg">
                <svg className="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                </svg>
              </div>
            </div>
            <div className="mt-2 flex items-center text-xs">
              <span className="text-green-600">{adminStats.gateways.online} online</span>
              {adminStats.gateways.offline > 0 && (
                <span className="text-red-600 ml-2">{adminStats.gateways.offline} offline</span>
              )}
            </div>
          </Link>

          <Link to="/admin/networks" className="card hover:shadow-lg transition-shadow p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">Networks</p>
                <p className="text-2xl font-bold text-gray-900">{adminStats.networks}</p>
              </div>
              <div className="p-2 bg-blue-100 rounded-lg">
                <svg className="h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                </svg>
              </div>
            </div>
            <p className="mt-2 text-xs text-gray-500">CIDR blocks defined</p>
          </Link>

          <Link to="/admin/users" className="card hover:shadow-lg transition-shadow p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">Users</p>
                <p className="text-2xl font-bold text-gray-900">{adminStats.users}</p>
              </div>
              <div className="p-2 bg-purple-100 rounded-lg">
                <svg className="h-6 w-6 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
              </div>
            </div>
            <p className="mt-2 text-xs text-gray-500">SSO users registered</p>
          </Link>

          <Link to="/admin/access-rules" className="card hover:shadow-lg transition-shadow p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">Access Rules</p>
                <p className="text-2xl font-bold text-gray-900">{adminStats.accessRules}</p>
              </div>
              <div className="p-2 bg-orange-100 rounded-lg">
                <svg className="h-6 w-6 text-orange-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
            </div>
            <p className="mt-2 text-xs text-gray-500">IP/hostname rules</p>
          </Link>

          <Link to="/admin/proxy-apps" className="card hover:shadow-lg transition-shadow p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">Proxy Apps</p>
                <p className="text-2xl font-bold text-gray-900">{adminStats.proxyApps}</p>
              </div>
              <div className="p-2 bg-cyan-100 rounded-lg">
                <svg className="h-6 w-6 text-cyan-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                </svg>
              </div>
            </div>
            <p className="mt-2 text-xs text-gray-500">Web applications</p>
          </Link>
        </div>
      )}

      {/* Quick actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Connect to VPN */}
        <Link to="/connect" className="card hover:shadow-lg transition-shadow cursor-pointer">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-primary-100 rounded-lg">
              <svg className="h-6 w-6 text-primary-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-gray-900">Connect to VPN</h3>
              <p className="text-sm text-gray-500">
                {loading ? 'Loading...' : `${userGateways.length} gateway${userGateways.length !== 1 ? 's' : ''} available`}
              </p>
            </div>
            <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </div>
        </Link>

        {/* Web Access */}
        <Link to="/web-access" className="card hover:shadow-lg transition-shadow cursor-pointer">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-green-100 rounded-lg">
              <svg className="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
              </svg>
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-gray-900">Web Access</h3>
              <p className="text-sm text-gray-500">
                {loading ? 'Loading...' : `${userProxyApps.length} app${userProxyApps.length !== 1 ? 's' : ''} available`}
              </p>
            </div>
            <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </div>
        </Link>

        {/* Help */}
        <Link to="/help" className="card hover:shadow-lg transition-shadow cursor-pointer">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-blue-100 rounded-lg">
              <svg className="h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-gray-900">Help & Documentation</h3>
              <p className="text-sm text-gray-500">Getting started guides</p>
            </div>
            <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </div>
        </Link>
      </div>

      {/* Available Resources */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Available Gateways */}
        <div className="card">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Your Gateways</h2>
            <Link to="/connect" className="text-sm text-primary-600 hover:text-primary-700">
              View all
            </Link>
          </div>
          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-600"></div>
            </div>
          ) : userGateways.length > 0 ? (
            <div className="space-y-3">
              {userGateways.slice(0, 4).map((gateway) => {
                const isOnline = gateway.isActive && gateway.lastHeartbeat &&
                  (Date.now() - new Date(gateway.lastHeartbeat).getTime()) < 2 * 60 * 1000
                return (
                  <div key={gateway.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className={`w-2 h-2 rounded-full ${isOnline ? 'bg-green-500' : 'bg-gray-400'}`} />
                      <div>
                        <p className="font-medium text-gray-900">{gateway.name}</p>
                        <p className="text-xs text-gray-500">{gateway.hostname}</p>
                      </div>
                    </div>
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      isOnline
                        ? 'bg-green-100 text-green-700'
                        : 'bg-gray-100 text-gray-600'
                    }`}>
                      {isOnline ? 'Online' : 'Offline'}
                    </span>
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              <svg className="mx-auto h-10 w-10 text-gray-400 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
              </svg>
              <p>No gateways available</p>
              <p className="text-xs mt-1">Contact your administrator for access</p>
            </div>
          )}
        </div>

        {/* Available Web Apps */}
        <div className="card">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Your Web Apps</h2>
            <Link to="/web-access" className="text-sm text-primary-600 hover:text-primary-700">
              View all
            </Link>
          </div>
          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-600"></div>
            </div>
          ) : userProxyApps.length > 0 ? (
            <div className="space-y-3">
              {userProxyApps.slice(0, 4).map((app) => (
                <a
                  key={app.id}
                  href={`/proxy/${app.slug}/`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center space-x-3">
                    {app.iconUrl ? (
                      <img src={app.iconUrl} alt="" className="w-8 h-8 rounded" />
                    ) : (
                      <div className="w-8 h-8 bg-primary-100 rounded flex items-center justify-center">
                        <span className="text-primary-600 font-semibold text-sm">
                          {app.name.charAt(0).toUpperCase()}
                        </span>
                      </div>
                    )}
                    <div>
                      <p className="font-medium text-gray-900">{app.name}</p>
                      <p className="text-xs text-gray-500">{app.slug}</p>
                    </div>
                  </div>
                  <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </a>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500">
              <svg className="mx-auto h-10 w-10 text-gray-400 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
              </svg>
              <p>No web apps available</p>
              <p className="text-xs mt-1">Contact your administrator for access</p>
            </div>
          )}
        </div>
      </div>

      {/* Getting Started */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Getting Started with GateKey CLI</h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="p-4 bg-gray-50 rounded-lg">
            <div className="flex items-center space-x-3 mb-2">
              <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">1</span>
              <span className="font-medium text-gray-900">Install</span>
            </div>
            <p className="text-sm text-gray-600">Download and install the GateKey CLI for your platform</p>
          </div>
          <div className="p-4 bg-gray-50 rounded-lg">
            <div className="flex items-center space-x-3 mb-2">
              <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">2</span>
              <span className="font-medium text-gray-900">Configure</span>
            </div>
            <p className="text-sm text-gray-600">Run <code className="bg-gray-200 px-1 rounded text-xs">gatekey config init</code></p>
          </div>
          <div className="p-4 bg-gray-50 rounded-lg">
            <div className="flex items-center space-x-3 mb-2">
              <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">3</span>
              <span className="font-medium text-gray-900">Authenticate</span>
            </div>
            <p className="text-sm text-gray-600">Run <code className="bg-gray-200 px-1 rounded text-xs">gatekey login</code></p>
          </div>
          <div className="p-4 bg-gray-50 rounded-lg">
            <div className="flex items-center space-x-3 mb-2">
              <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">4</span>
              <span className="font-medium text-gray-900">Connect</span>
            </div>
            <p className="text-sm text-gray-600">Run <code className="bg-gray-200 px-1 rounded text-xs">gatekey connect</code></p>
          </div>
        </div>
      </div>
    </div>
  )
}
