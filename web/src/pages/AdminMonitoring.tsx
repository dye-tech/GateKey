import { useState, useEffect } from 'react'
import {
  getLoginLogs,
  getLoginLogStats,
  purgeLoginLogs,
  getLoginLogRetention,
  setLoginLogRetention,
  LoginLog,
  LoginLogStats,
} from '../api/client'

type TabType = 'logs' | 'stats' | 'settings'

// Convert country code to flag emoji
function countryCodeToFlag(countryCode: string): string {
  if (!countryCode || countryCode.length !== 2) return ''
  const codePoints = countryCode
    .toUpperCase()
    .split('')
    .map(char => 127397 + char.charCodeAt(0))
  return String.fromCodePoint(...codePoints)
}

export default function AdminMonitoring() {
  const [activeTab, setActiveTab] = useState<TabType>('logs')
  const [logs, setLogs] = useState<LoginLog[]>([])
  const [stats, setStats] = useState<LoginLogStats | null>(null)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

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

  useEffect(() => {
    loadData()
  }, [activeTab, page, filterEmail, filterIP, filterProvider, filterSuccess])

  async function loadData() {
    try {
      setLoading(true)
      setError(null)

      if (activeTab === 'logs') {
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">Login Monitoring</h1>
        <p className="text-gray-500 mt-1">
          Monitor user login activity, view statistics, and manage log retention.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('logs')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'logs'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Login Logs
          </button>
          <button
            onClick={() => setActiveTab('stats')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'stats'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Statistics
          </button>
          <button
            onClick={() => setActiveTab('settings')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'settings'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Settings
          </button>
        </nav>
      </div>

      {/* Messages */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}
      {success && (
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
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
          {/* Login Logs Tab */}
          {activeTab === 'logs' && (
            <div className="space-y-4">
              {/* Filters */}
              <div className="card">
                <h3 className="text-sm font-medium text-gray-700 mb-3">Filters</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Email</label>
                    <input
                      type="text"
                      value={filterEmail}
                      onChange={(e) => { setFilterEmail(e.target.value); setPage(0) }}
                      placeholder="Search by email..."
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">IP Address</label>
                    <input
                      type="text"
                      value={filterIP}
                      onChange={(e) => { setFilterIP(e.target.value); setPage(0) }}
                      placeholder="Search by IP..."
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Provider</label>
                    <select
                      value={filterProvider}
                      onChange={(e) => { setFilterProvider(e.target.value); setPage(0) }}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    >
                      <option value="">All Providers</option>
                      <option value="oidc">OIDC</option>
                      <option value="saml">SAML</option>
                      <option value="local">Local</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Status</label>
                    <select
                      value={filterSuccess}
                      onChange={(e) => { setFilterSuccess(e.target.value); setPage(0) }}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
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
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Time</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">User</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Provider</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">IP Address</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Location</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {logs.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                            No login logs found
                          </td>
                        </tr>
                      ) : (
                        logs.map((log) => (
                          <tr key={log.id} className="hover:bg-gray-50">
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {formatDate(log.createdAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-medium text-gray-900">{log.userEmail}</div>
                              {log.userName && (
                                <div className="text-xs text-gray-500">{log.userName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                log.provider === 'oidc' ? 'bg-blue-100 text-blue-800' :
                                log.provider === 'saml' ? 'bg-purple-100 text-purple-800' :
                                'bg-gray-100 text-gray-800'
                              }`}>
                                {log.provider.toUpperCase()}
                              </span>
                              {log.providerName && (
                                <div className="text-xs text-gray-500 mt-0.5">{log.providerName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 font-mono">
                              {log.ipAddress}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {log.countryCode && (
                                <span className="mr-1">{countryCodeToFlag(log.countryCode)}</span>
                              )}
                              {log.city && log.country ? `${log.city}, ${log.country}` : log.country || '-'}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              {log.success ? (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                                  <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                  </svg>
                                  Success
                                </span>
                              ) : (
                                <div>
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">
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
                  <div className="px-4 py-3 border-t border-gray-200 flex items-center justify-between">
                    <div className="text-sm text-gray-500">
                      Showing {page * pageSize + 1} to {Math.min((page + 1) * pageSize, total)} of {total} logs
                    </div>
                    <div className="flex space-x-2">
                      <button
                        onClick={() => setPage(p => Math.max(0, p - 1))}
                        disabled={page === 0}
                        className="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Previous
                      </button>
                      <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={(page + 1) * pageSize >= total}
                        className="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
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
                  <div className="text-sm text-gray-500">Total Logins</div>
                  <div className="text-2xl font-bold text-gray-900">{stats.totalLogins}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-gray-500">Successful</div>
                  <div className="text-2xl font-bold text-green-600">{stats.successfulLogins}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-gray-500">Failed</div>
                  <div className="text-2xl font-bold text-red-600">{stats.failedLogins}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-gray-500">Unique Users</div>
                  <div className="text-2xl font-bold text-gray-900">{stats.uniqueUsers}</div>
                </div>
                <div className="card">
                  <div className="text-sm text-gray-500">Unique IPs</div>
                  <div className="text-2xl font-bold text-gray-900">{stats.uniqueIps}</div>
                </div>
              </div>

              {/* Charts Row */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Logins by Provider */}
                <div className="card">
                  <h3 className="text-lg font-medium text-gray-900 mb-4">Logins by Provider</h3>
                  <div className="space-y-3">
                    {Object.entries(stats.loginsByProvider).length === 0 ? (
                      <div className="text-gray-500 text-sm">No data available</div>
                    ) : (
                      Object.entries(stats.loginsByProvider).map(([provider, count]) => (
                        <div key={provider} className="flex items-center">
                          <div className="w-20 text-sm font-medium text-gray-700 uppercase">{provider}</div>
                          <div className="flex-1 mx-3">
                            <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                              <div
                                className={`h-full rounded-full ${
                                  provider === 'oidc' ? 'bg-blue-500' :
                                  provider === 'saml' ? 'bg-purple-500' :
                                  'bg-gray-500'
                                }`}
                                style={{ width: `${(count / stats.totalLogins) * 100}%` }}
                              />
                            </div>
                          </div>
                          <div className="w-16 text-sm text-gray-600 text-right">{count}</div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Logins by Country */}
                <div className="card">
                  <h3 className="text-lg font-medium text-gray-900 mb-4">Logins by Country</h3>
                  <div className="space-y-3">
                    {Object.entries(stats.loginsByCountry).length === 0 ? (
                      <div className="text-gray-500 text-sm">No data available</div>
                    ) : (
                      Object.entries(stats.loginsByCountry).slice(0, 10).map(([country, count]) => (
                        <div key={country} className="flex items-center">
                          <div className="w-24 text-sm font-medium text-gray-700 truncate">{country}</div>
                          <div className="flex-1 mx-3">
                            <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-primary-500 rounded-full"
                                style={{ width: `${(count / stats.totalLogins) * 100}%` }}
                              />
                            </div>
                          </div>
                          <div className="w-16 text-sm text-gray-600 text-right">{count}</div>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>

              {/* Recent Failures */}
              {stats.recentFailures.length > 0 && (
                <div className="card">
                  <h3 className="text-lg font-medium text-gray-900 mb-4">Recent Failed Logins</h3>
                  <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">IP Address</th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Reason</th>
                        </tr>
                      </thead>
                      <tbody className="bg-white divide-y divide-gray-200">
                        {stats.recentFailures.map((log) => (
                          <tr key={log.id}>
                            <td className="px-4 py-3 text-sm text-gray-500">{formatDate(log.createdAt)}</td>
                            <td className="px-4 py-3 text-sm text-gray-900">{log.userEmail}</td>
                            <td className="px-4 py-3 text-sm text-gray-900 font-mono">{log.ipAddress}</td>
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
                <h3 className="text-lg font-medium text-gray-900 mb-2">Log Retention</h3>
                <p className="text-sm text-gray-500 mb-4">
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
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <span className="text-gray-600">days</span>
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
                <h3 className="text-lg font-medium text-gray-900 mb-2">Manual Purge</h3>
                <p className="text-sm text-gray-500 mb-4">
                  Manually delete login logs older than a specified number of days.
                  This action cannot be undone.
                </p>
                <div className="flex items-center space-x-4">
                  <span className="text-gray-600">Delete logs older than</span>
                  <div className="w-24">
                    <input
                      type="number"
                      min="1"
                      value={purgeDays}
                      onChange={(e) => setPurgeDays(parseInt(e.target.value) || 1)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <span className="text-gray-600">days</span>
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
