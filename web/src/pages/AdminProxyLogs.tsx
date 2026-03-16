import { useState, useEffect, useCallback } from 'react'
import {
  ProxyAccessLogEntry,
  getAllProxyLogs,
  getProxyApps,
  ProxyApplication,
} from '../api/client'

function methodBadge(method: string) {
  const colors: Record<string, string> = {
    GET: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
    POST: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    PUT: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
    PATCH: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
    DELETE: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  }
  const cls = colors[method] || 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {method}
    </span>
  )
}

function statusBadge(status: number) {
  let cls = 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
  if (status >= 200 && status < 300) {
    cls = 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
  } else if (status >= 300 && status < 400) {
    cls = 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
  } else if (status >= 400 && status < 500) {
    cls = 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
  } else if (status >= 500) {
    cls = 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
  }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {status}
    </span>
  )
}

function formatSize(bytes?: number) {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function AdminProxyLogs() {
  const [logs, setLogs] = useState<ProxyAccessLogEntry[]>([])
  const [apps, setApps] = useState<ProxyApplication[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // Filters
  const [filterAppId, setFilterAppId] = useState('')
  const [filterEmail, setFilterEmail] = useState('')
  const [filterMethod, setFilterMethod] = useState('')
  const [filterMinStatus, setFilterMinStatus] = useState('')
  const [filterMaxStatus, setFilterMaxStatus] = useState('')
  const [offset, setOffset] = useState(0)
  const limit = 100

  const loadApps = useCallback(async () => {
    try {
      const data = await getProxyApps()
      setApps(data)
    } catch {
      // non-critical
    }
  }, [])

  const loadLogs = useCallback(async (newOffset = 0) => {
    try {
      setLoading(true)
      setError('')
      const data = await getAllProxyLogs({
        app_id: filterAppId || undefined,
        user_email: filterEmail || undefined,
        method: filterMethod || undefined,
        min_status: filterMinStatus ? parseInt(filterMinStatus) : undefined,
        max_status: filterMaxStatus ? parseInt(filterMaxStatus) : undefined,
        limit,
        offset: newOffset,
      })
      if (newOffset > 0) {
        setLogs(prev => [...prev, ...data])
      } else {
        setLogs(data)
      }
      setOffset(newOffset)
    } catch {
      setError('Failed to load proxy logs')
    } finally {
      setLoading(false)
    }
  }, [filterAppId, filterEmail, filterMethod, filterMinStatus, filterMaxStatus])

  useEffect(() => {
    loadApps()
  }, [loadApps])

  useEffect(() => {
    loadLogs(0)
  }, [loadLogs])

  const handleSearch = () => {
    setOffset(0)
    loadLogs(0)
  }

  const handleLoadMore = () => {
    loadLogs(offset + limit)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-theme-primary">Proxy Access Logs</h1>
        <button
          onClick={() => loadLogs(0)}
          className="px-3 py-1.5 text-sm bg-theme-secondary text-theme-primary rounded-md hover:bg-theme-tertiary transition-colors"
        >
          Refresh
        </button>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-sm text-red-800 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Filter bar */}
      <div className="bg-theme-primary shadow rounded-lg p-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-3">
          <div>
            <label className="block text-xs font-medium text-theme-secondary mb-1">Application</label>
            <select
              value={filterAppId}
              onChange={e => setFilterAppId(e.target.value)}
              className="w-full px-2 py-1.5 text-sm border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All Apps</option>
              {apps.map(app => (
                <option key={app.id} value={app.id}>{app.name}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-theme-secondary mb-1">User Email</label>
            <input
              type="text"
              value={filterEmail}
              onChange={e => setFilterEmail(e.target.value)}
              placeholder="user@example.com"
              className="w-full px-2 py-1.5 text-sm border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-theme-secondary mb-1">Method</label>
            <select
              value={filterMethod}
              onChange={e => setFilterMethod(e.target.value)}
              className="w-full px-2 py-1.5 text-sm border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All</option>
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="PATCH">PATCH</option>
              <option value="DELETE">DELETE</option>
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-theme-secondary mb-1">Min Status</label>
            <input
              type="number"
              value={filterMinStatus}
              onChange={e => setFilterMinStatus(e.target.value)}
              placeholder="200"
              className="w-full px-2 py-1.5 text-sm border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-theme-secondary mb-1">Max Status</label>
            <input
              type="number"
              value={filterMaxStatus}
              onChange={e => setFilterMaxStatus(e.target.value)}
              placeholder="599"
              className="w-full px-2 py-1.5 text-sm border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div className="flex items-end">
            <button
              onClick={handleSearch}
              className="w-full px-4 py-1.5 text-sm bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors"
            >
              Search
            </button>
          </div>
        </div>
      </div>

      {/* Results table */}
      {loading && logs.length === 0 ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : logs.length === 0 ? (
        <div className="text-center py-12 text-theme-secondary">
          <p>No proxy access logs found.</p>
          <p className="text-sm mt-1">Logs are created when users access proxy applications.</p>
        </div>
      ) : (
        <>
          <div className="bg-theme-primary shadow rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-secondary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Time</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">App</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">User</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Method</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Path</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Time (ms)</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Size</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">IP</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-theme">
                  {logs.map((log) => (
                    <>
                      <tr
                        key={log.id}
                        className="hover:bg-theme-tertiary cursor-pointer"
                        onClick={() => setExpandedId(expandedId === log.id ? null : log.id)}
                      >
                        <td className="px-4 py-3 whitespace-nowrap text-xs text-theme-secondary">
                          {new Date(log.created_at).toLocaleString()}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-primary font-medium">
                          {log.app_name || '-'}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-secondary truncate max-w-[160px]" title={log.user_email}>
                          {log.user_email}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {methodBadge(log.request_method)}
                        </td>
                        <td className="px-4 py-3 text-sm text-theme-secondary truncate max-w-[250px]" title={log.request_path}>
                          {log.request_path}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {statusBadge(log.response_status)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-secondary">
                          {log.response_time_ms}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-secondary">
                          {formatSize(log.response_size_bytes)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-xs text-theme-secondary font-mono">
                          {log.client_ip}
                        </td>
                      </tr>
                      {expandedId === log.id && (
                        <tr key={`${log.id}-detail`}>
                          <td colSpan={9} className="px-4 py-3 bg-theme-secondary">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                              <div>
                                <p className="text-theme-secondary mb-1">
                                  <span className="font-medium text-theme-primary">User Agent:</span>{' '}
                                  <span className="text-xs break-all">{log.user_agent}</span>
                                </p>
                                <p className="text-theme-secondary">
                                  <span className="font-medium text-theme-primary">User ID:</span> {log.user_id}
                                </p>
                              </div>
                              {log.request_headers && Object.keys(log.request_headers).length > 0 && (
                                <div>
                                  <p className="font-medium text-theme-primary mb-1">Request Headers</p>
                                  <div className="bg-theme-primary rounded p-2 text-xs font-mono max-h-40 overflow-y-auto">
                                    {Object.entries(log.request_headers).map(([k, v]) => (
                                      <div key={k} className="text-theme-secondary">
                                        <span className="text-theme-primary">{k}:</span> {v}
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}
                              {log.response_headers && Object.keys(log.response_headers).length > 0 && (
                                <div>
                                  <p className="font-medium text-theme-primary mb-1">Response Headers</p>
                                  <div className="bg-theme-primary rounded p-2 text-xs font-mono max-h-40 overflow-y-auto">
                                    {Object.entries(log.response_headers).map(([k, v]) => (
                                      <div key={k} className="text-theme-secondary">
                                        <span className="text-theme-primary">{k}:</span> {v}
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Load more */}
          {logs.length >= offset + limit && (
            <div className="flex justify-center">
              <button
                onClick={handleLoadMore}
                disabled={loading}
                className="px-4 py-2 text-sm bg-theme-secondary text-theme-primary rounded-md hover:bg-theme-tertiary disabled:opacity-50 transition-colors"
              >
                {loading ? 'Loading...' : 'Load More'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
