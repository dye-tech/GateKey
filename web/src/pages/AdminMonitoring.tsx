import { useState, useEffect, useRef, useCallback } from 'react'
import {
  getLoginLogs,
  getLoginLogStats,
  purgeLoginLogs,
  getLoginLogRetention,
  setLoginLogRetention,
  getActiveSessions,
  getAdminGateways,
  getMeshHubs,
  LoginLog,
  LoginLogStats,
  ActiveSession,
  AdminGateway,
  MeshHub,
} from '../api/client'

type TabType = 'realtime' | 'logs' | 'stats' | 'settings'

// Convert country code to flag emoji
function countryCodeToFlag(countryCode: string): string {
  if (!countryCode || countryCode.length !== 2) return ''
  const codePoints = countryCode
    .toUpperCase()
    .split('')
    .map(char => 127397 + char.charCodeAt(0))
  return String.fromCodePoint(...codePoints)
}

// Format bytes to human readable
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Format duration from connectedAt timestamp
function formatDuration(connectedAt: string): string {
  const start = new Date(connectedAt).getTime()
  const now = Date.now()
  const diff = Math.floor((now - start) / 1000) // seconds

  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ${diff % 60}s`
  const hours = Math.floor(diff / 3600)
  const mins = Math.floor((diff % 3600) / 60)
  return `${hours}h ${mins}m`
}

// Format last seen time and check if stale (>15 seconds old)
function formatLastSeen(lastSeenAt: string | undefined): { text: string; isStale: boolean } {
  if (!lastSeenAt) {
    return { text: 'Unknown', isStale: true }
  }
  const lastSeen = new Date(lastSeenAt).getTime()
  const now = Date.now()
  const diffSeconds = Math.floor((now - lastSeen) / 1000)

  // Consider stale if not seen in 15+ seconds
  const isStale = diffSeconds > 15

  if (diffSeconds < 5) return { text: 'Just now', isStale: false }
  if (diffSeconds < 60) return { text: `${diffSeconds}s ago`, isStale }
  if (diffSeconds < 3600) return { text: `${Math.floor(diffSeconds / 60)}m ago`, isStale }
  return { text: `${Math.floor(diffSeconds / 3600)}h ago`, isStale: true }
}

export default function AdminMonitoring() {
  const [activeTab, setActiveTab] = useState<TabType>('realtime')
  const [logs, setLogs] = useState<LoginLog[]>([])
  const [stats, setStats] = useState<LoginLogStats | null>(null)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // Real-time monitoring state
  const [sessions, setSessions] = useState<ActiveSession[]>([])
  const [gateways, setGateways] = useState<AdminGateway[]>([])
  const [meshHubs, setMeshHubs] = useState<MeshHub[]>([])
  const [refreshInterval, setRefreshInterval] = useState<number>(10)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())
  const [autoRefresh, setAutoRefresh] = useState(true)
  const refreshTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Filters
  const [filterEmail, setFilterEmail] = useState('')
  const [filterIP, setFilterIP] = useState('')
  const [filterProvider, setFilterProvider] = useState('')
  const [filterSuccess, setFilterSuccess] = useState<string>('')
  const [page, setPage] = useState(0)
  const pageSize = 25

  // Settings
  const [retentionDays, setRetentionDays] = useState(30)
  const [purgeDays, setPurgeDays] = useState(30)

  // Load real-time data
  const loadRealtimeData = useCallback(async () => {
    try {
      const [sessionsData, gatewaysData, hubsData] = await Promise.all([
        getActiveSessions(),
        getAdminGateways(),
        getMeshHubs(),
      ])
      setSessions(sessionsData.sessions || [])
      setGateways(gatewaysData || [])
      setMeshHubs(hubsData || [])
      setLastRefresh(new Date())
    } catch (err) {
      console.error('Failed to load real-time data:', err)
    }
  }, [])

  // Auto-refresh for real-time tab
  useEffect(() => {
    if (activeTab === 'realtime' && autoRefresh && refreshInterval > 0) {
      refreshTimerRef.current = setInterval(() => {
        loadRealtimeData()
      }, refreshInterval * 1000)

      return () => {
        if (refreshTimerRef.current) {
          clearInterval(refreshTimerRef.current)
        }
      }
    }
  }, [activeTab, autoRefresh, refreshInterval, loadRealtimeData])

  useEffect(() => {
    loadData()
  }, [activeTab, page, filterEmail, filterIP, filterProvider, filterSuccess])

  async function loadData() {
    try {
      setLoading(true)
      setError(null)

      if (activeTab === 'realtime') {
        await loadRealtimeData()
      } else if (activeTab === 'logs') {
        const filter: Record<string, unknown> = {
          limit: pageSize,
          offset: page * pageSize,
        }
        if (filterEmail) filter.userEmail = filterEmail
        if (filterIP) filter.ipAddress = filterIP
        if (filterProvider) filter.provider = filterProvider
        if (filterSuccess === 'true') filter.success = true
        if (filterSuccess === 'false') filter.success = false

        const result = await getLoginLogs(filter)
        setLogs(result.logs)
        setTotal(result.total)
      } else if (activeTab === 'stats') {
        const statsData = await getLoginLogStats(30)
        setStats(statsData)
      } else if (activeTab === 'settings') {
        const retention = await getLoginLogRetention()
        setRetentionDays(retention.days)
      }
    } catch (err) {
      setError('Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  async function handleSaveRetention() {
    try {
      await setLoginLogRetention(retentionDays)
      setSuccess('Retention setting saved successfully')
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to save retention setting')
    }
  }

  async function handlePurgeLogs() {
    if (!confirm(`Are you sure you want to delete logs older than ${purgeDays} days? This action cannot be undone.`)) {
      return
    }
    try {
      const result = await purgeLoginLogs(purgeDays)
      setSuccess(`Successfully deleted ${result.deletedCount} log entries`)
      setTimeout(() => setSuccess(null), 3000)
      loadData()
    } catch (err) {
      setError('Failed to purge logs')
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function clearFilters() {
    setFilterEmail('')
    setFilterIP('')
    setFilterProvider('')
    setFilterSuccess('')
    setPage(0)
  }

  // Calculate totals for real-time stats
  const totalBandwidthIn = sessions.reduce((sum, s) => sum + s.bytesRecv, 0)
  const totalBandwidthOut = sessions.reduce((sum, s) => sum + s.bytesSent, 0)
  const onlineGateways = gateways.filter(g => g.isActive).length
  const onlineMeshHubs = meshHubs.filter(h => h.status === 'online').length
  const uniqueUsers = new Set(sessions.map(s => s.userId)).size

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">Monitoring</h1>
        <p className="text-theme-tertiary mt-1">
          Real-time VPN sessions, login activity, statistics, and log management.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-theme">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('realtime')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'realtime'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Real-time
          </button>
          <button
            onClick={() => setActiveTab('logs')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'logs'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Login Logs
          </button>
          <button
            onClick={() => setActiveTab('stats')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'stats'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Statistics
          </button>
          <button
            onClick={() => setActiveTab('settings')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'settings'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Settings
          </button>
        </nav>
      </div>

      {/* Messages */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
          {error}
        </div>
      )}
      {success && (
        <div className="p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg text-green-700 dark:text-green-400">
          {success}
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <>
          {/* Real-time Tab */}
          {activeTab === 'realtime' && (
            <div className="space-y-6">
              {/* Refresh Controls */}
              <div className="card">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div className="flex items-center gap-4">
                    <label className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={autoRefresh}
                        onChange={(e) => setAutoRefresh(e.target.checked)}
                        className="rounded border-theme text-primary-600 focus:ring-primary-500"
                      />
                      <span className="text-sm text-theme-secondary">Auto-refresh</span>
                    </label>
                    <select
                      value={refreshInterval}
                      onChange={(e) => setRefreshInterval(Number(e.target.value))}
                      disabled={!autoRefresh}
                      className="px-3 py-1.5 border border-theme rounded-lg text-sm bg-theme-primary text-theme-primary focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
                    >
                      <option value={5}>5 seconds</option>
                      <option value={10}>10 seconds</option>
                      <option value={30}>30 seconds</option>
                      <option value={60}>60 seconds</option>
                    </select>
                    <button
                      onClick={loadRealtimeData}
                      className="px-3 py-1.5 text-sm bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
                    >
                      Refresh Now
                    </button>
                  </div>
                  <div className="text-sm text-theme-tertiary">
                    Last updated: {lastRefresh.toLocaleTimeString()}
                  </div>
                </div>
              </div>

              {/* Stats Cards */}
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-4">
                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-theme-tertiary uppercase tracking-wider">Active Sessions</div>
                      <div className="text-2xl font-bold text-theme-primary">{sessions.length}</div>
                    </div>
                    <div className="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg">
                      <svg className="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                      </svg>
                    </div>
                  </div>
                </div>

                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-theme-tertiary uppercase tracking-wider">Unique Users</div>
                      <div className="text-2xl font-bold text-theme-primary">{uniqueUsers}</div>
                    </div>
                    <div className="p-2 bg-indigo-100 dark:bg-blue-900/30 rounded-lg">
                      <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                    </div>
                  </div>
                </div>

                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-theme-tertiary uppercase tracking-wider">Gateways Online</div>
                      <div className="text-2xl font-bold text-theme-primary">{onlineGateways}/{gateways.length}</div>
                    </div>
                    <div className="p-2 bg-blue-100 dark:bg-green-900/30 rounded-lg">
                      <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
                      </svg>
                    </div>
                  </div>
                </div>

                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-theme-tertiary uppercase tracking-wider">Mesh Hubs Online</div>
                      <div className="text-2xl font-bold text-theme-primary">{onlineMeshHubs}/{meshHubs.length}</div>
                    </div>
                    <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                      <svg className="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                      </svg>
                    </div>
                  </div>
                </div>

                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-theme-tertiary uppercase tracking-wider">Bandwidth In</div>
                      <div className="text-2xl font-bold text-green-600">{formatBytes(totalBandwidthIn)}</div>
                    </div>
                    <div className="p-2 bg-blue-100 dark:bg-green-900/30 rounded-lg">
                      <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                      </svg>
                    </div>
                  </div>
                </div>

                <div className="card">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-theme-tertiary uppercase tracking-wider">Bandwidth Out</div>
                      <div className="text-2xl font-bold text-blue-600">{formatBytes(totalBandwidthOut)}</div>
                    </div>
                    <div className="p-2 bg-indigo-100 dark:bg-blue-900/30 rounded-lg">
                      <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
                      </svg>
                    </div>
                  </div>
                </div>
              </div>

              {/* Active Sessions Table */}
              <div className="card overflow-hidden">
                <h3 className="text-lg font-medium text-theme-primary mb-4">Active VPN Sessions</h3>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-theme">
                    <thead className="bg-theme-secondary">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">User</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Gateway/Hub</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Client IP</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">VPN IP</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Duration</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Last Seen</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Traffic</th>
                      </tr>
                    </thead>
                    <tbody className="bg-theme-card divide-y divide-theme">
                      {sessions.length === 0 ? (
                        <tr>
                          <td colSpan={7} className="px-4 py-8 text-center text-theme-tertiary">
                            No active VPN sessions
                          </td>
                        </tr>
                      ) : (
                        sessions.map((session) => (
                          <tr key={session.id} className="hover:bg-theme-secondary">
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-medium text-theme-primary">{session.userEmail}</div>
                              {session.userName && (
                                <div className="text-xs text-theme-tertiary">{session.userName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="flex items-center gap-2">
                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                  session.nodeType === 'gateway'
                                    ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
                                    : 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400'
                                }`}>
                                  {session.nodeType}
                                </span>
                                <span className="text-sm text-theme-primary">{session.gatewayName}</span>
                              </div>
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-theme-primary">
                              {session.clientIp}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-theme-primary">
                              {session.vpnAddress}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                              {formatDuration(session.connectedAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm">
                              {(() => {
                                const { text, isStale } = formatLastSeen(session.lastSeenAt)
                                return (
                                  <div className="flex items-center gap-1.5">
                                    <span className={`w-2 h-2 rounded-full ${isStale ? 'bg-yellow-500' : 'bg-green-500'}`} />
                                    <span className={isStale ? 'text-yellow-600 dark:text-yellow-400' : 'text-theme-tertiary'}>
                                      {text}
                                    </span>
                                  </div>
                                )
                              })()}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm">
                              <div className="flex items-center gap-3">
                                <span className="text-green-600" title="Downloaded">
                                  <svg className="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                                  </svg>
                                  {formatBytes(session.bytesRecv)}
                                </span>
                                <span className="text-blue-600" title="Uploaded">
                                  <svg className="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
                                  </svg>
                                  {formatBytes(session.bytesSent)}
                                </span>
                              </div>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Gateway & Mesh Hub Status */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Gateway Status */}
                <div className="card">
                  <h3 className="text-lg font-medium text-theme-primary mb-4">Gateway Status</h3>
                  {gateways.length === 0 ? (
                    <div className="text-center text-theme-tertiary py-4">No gateways configured</div>
                  ) : (
                    <div className="space-y-3">
                      {gateways.map((gateway) => (
                        <div key={gateway.id} className="flex items-center justify-between p-3 bg-theme-secondary rounded-lg">
                          <div className="flex items-center gap-3">
                            <div className={`w-3 h-3 rounded-full ${gateway.isActive ? 'bg-green-500' : 'bg-red-500'}`} />
                            <div>
                              <div className="font-medium text-theme-primary">{gateway.name}</div>
                              <div className="text-xs text-theme-tertiary">{gateway.hostname || gateway.publicIp}</div>
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="text-sm text-theme-secondary">
                              {sessions.filter(s => s.gatewayId === gateway.id).length} sessions
                            </div>
                            <div className="text-xs text-theme-tertiary">
                              {gateway.vpnProtocol.toUpperCase()}:{gateway.vpnPort}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Mesh Hub Status */}
                <div className="card">
                  <h3 className="text-lg font-medium text-theme-primary mb-4">Mesh Hub Status</h3>
                  {meshHubs.length === 0 ? (
                    <div className="text-center text-theme-tertiary py-4">No mesh hubs configured</div>
                  ) : (
                    <div className="space-y-3">
                      {meshHubs.map((hub) => (
                        <div key={hub.id} className="flex items-center justify-between p-3 bg-theme-secondary rounded-lg">
                          <div className="flex items-center gap-3">
                            <div className={`w-3 h-3 rounded-full ${
                              hub.status === 'online' ? 'bg-green-500' :
                              hub.status === 'pending' ? 'bg-yellow-500' : 'bg-red-500'
                            }`} />
                            <div>
                              <div className="font-medium text-theme-primary">{hub.name}</div>
                              <div className="text-xs text-theme-tertiary">{hub.publicEndpoint}</div>
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="text-sm text-theme-secondary">
                              {hub.connectedSpokes} spokes / {hub.connectedClients} clients
                            </div>
                            <div className="text-xs text-theme-tertiary">
                              {hub.vpnProtocol?.toUpperCase() || 'UDP'}:{hub.vpnPort}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              {/* Sessions by Gateway Chart */}
              {sessions.length > 0 && (
                <div className="card">
                  <h3 className="text-lg font-medium text-theme-primary mb-4">Sessions by Gateway</h3>
                  <div className="space-y-3">
                    {Array.from(new Set(sessions.map(s => s.gatewayId))).map(gatewayId => {
                      const gatewayName = sessions.find(s => s.gatewayId === gatewayId)?.gatewayName || 'Unknown'
                      const count = sessions.filter(s => s.gatewayId === gatewayId).length
                      const percentage = (count / sessions.length) * 100
                      return (
                        <div key={gatewayId} className="flex items-center">
                          <div className="w-32 text-sm font-medium text-theme-secondary truncate">{gatewayName}</div>
                          <div className="flex-1 mx-3">
                            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary-500 rounded-full transition-all duration-300"
                                style={{ width: `${percentage}%` }}
                              />
                            </div>
                          </div>
                          <div className="w-16 text-sm text-theme-tertiary text-right">{count} ({percentage.toFixed(0)}%)</div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Login Logs Tab */}
          {activeTab === 'logs' && (
            <div className="space-y-4">
              {/* Filters */}
              <div className="card">
                <h3 className="text-sm font-medium text-theme-secondary mb-3">Filters</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div>
                    <label className="block text-xs text-theme-tertiary mb-1">Email</label>
                    <input
                      type="text"
                      value={filterEmail}
                      onChange={(e) => { setFilterEmail(e.target.value); setPage(0) }}
                      placeholder="Search by email..."
                      className="w-full px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-theme-tertiary mb-1">IP Address</label>
                    <input
                      type="text"
                      value={filterIP}
                      onChange={(e) => { setFilterIP(e.target.value); setPage(0) }}
                      placeholder="Search by IP..."
                      className="w-full px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-theme-tertiary mb-1">Provider</label>
                    <select
                      value={filterProvider}
                      onChange={(e) => { setFilterProvider(e.target.value); setPage(0) }}
                      className="w-full px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    >
                      <option value="">All Providers</option>
                      <option value="oidc">OIDC</option>
                      <option value="saml">SAML</option>
                      <option value="local">Local</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs text-theme-tertiary mb-1">Status</label>
                    <select
                      value={filterSuccess}
                      onChange={(e) => { setFilterSuccess(e.target.value); setPage(0) }}
                      className="w-full px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    >
                      <option value="">All</option>
                      <option value="true">Success</option>
                      <option value="false">Failed</option>
                    </select>
                  </div>
                </div>
                {(filterEmail || filterIP || filterProvider || filterSuccess) && (
                  <button
                    onClick={clearFilters}
                    className="mt-3 text-sm text-primary-600 hover:text-primary-800"
                  >
                    Clear filters
                  </button>
                )}
              </div>

              {/* Logs Table */}
              <div className="card overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-theme">
                    <thead className="bg-theme-tertiary">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Time</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">User</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Provider</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">IP Address</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Location</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Status</th>
                      </tr>
                    </thead>
                    <tbody className="bg-theme-card divide-y divide-theme">
                      {logs.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="px-4 py-8 text-center text-theme-tertiary">
                            No login logs found
                          </td>
                        </tr>
                      ) : (
                        logs.map((log) => (
                          <tr key={log.id} className="hover:bg-theme-tertiary">
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                              {formatDate(log.createdAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-medium text-theme-primary">{log.userEmail}</div>
                              {log.userName && (
                                <div className="text-xs text-theme-tertiary">{log.userName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                log.provider === 'oidc' ? 'bg-indigo-600 text-white' :
                                log.provider === 'saml' ? 'bg-violet-600 text-white' :
                                'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300'
                              }`}>
                                {log.provider.toUpperCase()}
                              </span>
                              {log.providerName && (
                                <div className="text-xs text-theme-tertiary mt-0.5">{log.providerName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-primary font-mono">
                              {log.ipAddress}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                              {log.countryCode && (
                                <span className="mr-1">{countryCodeToFlag(log.countryCode)}</span>
                              )}
                              {log.city && log.country ? `${log.city}, ${log.country}` : log.country || '-'}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              {log.success ? (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-600 text-white">
                                  <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                  </svg>
                                  Success
                                </span>
                              ) : (
                                <div>
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-600 text-white">
                                    <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                      <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                                    </svg>
                                    Failed
                                  </span>
                                  {log.failureReason && (
                                    <div className="text-xs text-red-600 mt-0.5">{log.failureReason}</div>
                                  )}
                                </div>
                              )}
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                {total > pageSize && (
                  <div className="px-4 py-3 border-t border-theme flex items-center justify-between">
                    <div className="text-sm text-theme-tertiary">
                      Showing {page * pageSize + 1} to {Math.min((page + 1) * pageSize, total)} of {total} logs
                    </div>
                    <div className="flex space-x-2">
                      <button
                        onClick={() => setPage(p => Math.max(0, p - 1))}
                        disabled={page === 0}
                        className="px-3 py-1 text-sm border rounded hover:bg-theme-tertiary disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Previous
                      </button>
                      <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={(page + 1) * pageSize >= total}
                        className="px-3 py-1 text-sm border rounded hover:bg-theme-tertiary disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Next
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Statistics Tab */}
          {activeTab === 'stats' && stats && (
            <div className="space-y-6">
              {/* Summary Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
                <div className="card">
                  <div className="text-sm text-theme-tertiary">Total Logins</div>
                  <div className="text-2xl font-bold text-theme-primary">{stats.totalLogins}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-theme-tertiary">Successful</div>
                  <div className="text-2xl font-bold text-green-600">{stats.successfulLogins}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-theme-tertiary">Failed</div>
                  <div className="text-2xl font-bold text-red-600">{stats.failedLogins}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-theme-tertiary">Unique Users</div>
                  <div className="text-2xl font-bold text-theme-primary">{stats.uniqueUsers}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-theme-tertiary">Unique IPs</div>
                  <div className="text-2xl font-bold text-theme-primary">{stats.uniqueIps}</div>
                </div>
              </div>

              {/* Charts Row */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Logins by Provider */}
                <div className="card">
                  <h3 className="text-lg font-medium text-theme-primary mb-4">Logins by Provider</h3>
                  <div className="space-y-3">
                    {Object.entries(stats.loginsByProvider).length === 0 ? (
                      <div className="text-theme-tertiary text-sm">No data available</div>
                    ) : (
                      Object.entries(stats.loginsByProvider).map(([provider, count]) => (
                        <div key={provider} className="flex items-center">
                          <div className="w-20 text-sm font-medium text-theme-secondary uppercase">{provider}</div>
                          <div className="flex-1 mx-3">
                            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                              <div
                                className={`h-full rounded-full ${
                                  provider === 'oidc' ? 'bg-blue-500' :
                                  provider === 'saml' ? 'bg-purple-500' :
                                  'bg-theme-tertiary0'
                                }`}
                                style={{ width: `${(count / stats.totalLogins) * 100}%` }}
                              />
                            </div>
                          </div>
                          <div className="w-16 text-sm text-theme-tertiary text-right">{count}</div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Logins by Country */}
                <div className="card">
                  <h3 className="text-lg font-medium text-theme-primary mb-4">Logins by Country</h3>
                  <div className="space-y-3">
                    {Object.entries(stats.loginsByCountry).length === 0 ? (
                      <div className="text-theme-tertiary text-sm">No data available</div>
                    ) : (
                      Object.entries(stats.loginsByCountry).slice(0, 10).map(([country, count]) => (
                        <div key={country} className="flex items-center">
                          <div className="w-24 text-sm font-medium text-theme-secondary truncate">{country}</div>
                          <div className="flex-1 mx-3">
                            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary-500 rounded-full"
                                style={{ width: `${(count / stats.totalLogins) * 100}%` }}
                              />
                            </div>
                          </div>
                          <div className="w-16 text-sm text-theme-tertiary text-right">{count}</div>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>

              {/* Recent Failures */}
              {stats.recentFailures.length > 0 && (
                <div className="card">
                  <h3 className="text-lg font-medium text-theme-primary mb-4">Recent Failed Logins</h3>
                  <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-theme">
                      <thead className="bg-theme-tertiary">
                        <tr>
                          <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Time</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">User</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">IP Address</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Reason</th>
                        </tr>
                      </thead>
                      <tbody className="bg-theme-card divide-y divide-theme">
                        {stats.recentFailures.map((log) => (
                          <tr key={log.id}>
                            <td className="px-4 py-3 text-sm text-theme-tertiary">{formatDate(log.createdAt)}</td>
                            <td className="px-4 py-3 text-sm text-theme-primary">{log.userEmail}</td>
                            <td className="px-4 py-3 text-sm text-theme-primary font-mono">{log.ipAddress}</td>
                            <td className="px-4 py-3 text-sm text-red-600">{log.failureReason || '-'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Settings Tab */}
          {activeTab === 'settings' && (
            <div className="space-y-6">
              {/* Retention Setting */}
              <div className="card">
                <h3 className="text-lg font-medium text-theme-primary mb-2">Log Retention</h3>
                <p className="text-sm text-theme-tertiary mb-4">
                  Configure how long login logs are kept. Logs older than this will be automatically deleted.
                  Set to 0 to keep logs forever.
                </p>
                <div className="flex items-center space-x-4">
                  <div className="w-32">
                    <input
                      type="number"
                      min="0"
                      value={retentionDays}
                      onChange={(e) => setRetentionDays(parseInt(e.target.value) || 0)}
                      className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <span className="text-theme-tertiary">days</span>
                  <button
                    onClick={handleSaveRetention}
                    className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
                  >
                    Save
                  </button>
                </div>
              </div>

              {/* Manual Purge */}
              <div className="card">
                <h3 className="text-lg font-medium text-theme-primary mb-2">Manual Purge</h3>
                <p className="text-sm text-theme-tertiary mb-4">
                  Manually delete login logs older than a specified number of days.
                  This action cannot be undone.
                </p>
                <div className="flex items-center space-x-4">
                  <span className="text-theme-tertiary">Delete logs older than</span>
                  <div className="w-24">
                    <input
                      type="number"
                      min="1"
                      value={purgeDays}
                      onChange={(e) => setPurgeDays(parseInt(e.target.value) || 1)}
                      className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <span className="text-theme-tertiary">days</span>
                  <button
                    onClick={handlePurgeLogs}
                    className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
                  >
                    Purge Logs
                  </button>
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
