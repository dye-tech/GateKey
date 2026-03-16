import { useState, useEffect, useCallback } from 'react'
import {
  JITRequest,
  JITStats,
  getAdminJITRequests,
  approveJITRequest,
  denyJITRequest,
  revokeJITGrant,
  getJITStats,
  getSettings,
  updateSettings,
} from '../api/client'

const TYPE_LABELS: Record<string, string> = {
  access_rule: 'Access Rules',
  proxy_app: 'Proxy Apps',
  mesh_hub: 'Mesh Hubs',
}

function statusBadge(status: string) {
  const colors: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
    approved: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
    denied: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
    expired: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
    canceled: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
    revoked: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
  }
  const cls = colors[status] || colors.expired
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${cls}`}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  )
}

function typeBadge(type: string) {
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-primary-100 text-primary-800 dark:bg-primary-900/30 dark:text-primary-400">
      {TYPE_LABELS[type] || type}
    </span>
  )
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString()
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

function CountdownTimer({ expiresAt }: { expiresAt: string }) {
  const [remaining, setRemaining] = useState('')

  useEffect(() => {
    function update() {
      const diff = new Date(expiresAt).getTime() - Date.now()
      if (diff <= 0) {
        setRemaining('Expired')
        return
      }
      const hours = Math.floor(diff / 3600000)
      const mins = Math.floor((diff % 3600000) / 60000)
      const secs = Math.floor((diff % 60000) / 1000)
      if (hours > 0) {
        setRemaining(`${hours}h ${mins}m ${secs}s`)
      } else if (mins > 0) {
        setRemaining(`${mins}m ${secs}s`)
      } else {
        setRemaining(`${secs}s`)
      }
    }
    update()
    const interval = setInterval(update, 1000)
    return () => clearInterval(interval)
  }, [expiresAt])

  const isExpired = remaining === 'Expired'
  return (
    <span className={`font-mono text-sm ${isExpired ? 'text-red-500' : 'text-green-600 dark:text-green-400'}`}>
      {remaining}
    </span>
  )
}

export default function AdminJITAccess() {
  const [tab, setTab] = useState<'pending' | 'active' | 'history' | 'settings'>('pending')
  const [pendingRequests, setPendingRequests] = useState<JITRequest[]>([])
  const [approvedRequests, setApprovedRequests] = useState<JITRequest[]>([])
  const [allRequests, setAllRequests] = useState<JITRequest[]>([])
  const [stats, setStats] = useState<JITStats>({ pending_requests: 0, active_grants: 0 })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [notes, setNotes] = useState<Record<string, string>>({})
  const [webhookEnabled, setWebhookEnabled] = useState(false)
  const [webhookURL, setWebhookURL] = useState('')
  const [appURL, setAppURL] = useState('')
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [settingsSaved, setSettingsSaved] = useState(false)

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const results = await Promise.allSettled([
        getAdminJITRequests('pending'),
        getAdminJITRequests('approved'),
        getAdminJITRequests(),
        getJITStats(),
        getSettings(),
      ])
      if (results[0].status === 'fulfilled') setPendingRequests(results[0].value)
      if (results[1].status === 'fulfilled') setApprovedRequests(results[1].value)
      if (results[2].status === 'fulfilled') setAllRequests(results[2].value)
      if (results[3].status === 'fulfilled') setStats(results[3].value)
      if (results[4].status === 'fulfilled') {
        const s = results[4].value
        setWebhookEnabled(s['jit_webhook_enabled'] === 'true')
        setWebhookURL(s['jit_webhook_url'] || '')
        setAppURL(s['jit_app_url'] || '')
      }

      const failed = results.filter(r => r.status === 'rejected')
      if (failed.length === results.length) {
        setError('Failed to load JIT access data')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  async function handleApprove(id: string) {
    try {
      setActionLoading(id)
      setError(null)
      await approveJITRequest(id, notes[id])
      setSuccess('Request approved')
      setNotes((prev) => ({ ...prev, [id]: '' }))
      await loadData()
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to approve request')
      console.error(err)
    } finally {
      setActionLoading(null)
    }
  }

  async function handleDeny(id: string) {
    try {
      setActionLoading(id)
      setError(null)
      await denyJITRequest(id, notes[id])
      setSuccess('Request denied')
      setNotes((prev) => ({ ...prev, [id]: '' }))
      await loadData()
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to deny request')
      console.error(err)
    } finally {
      setActionLoading(null)
    }
  }

  async function handleRevoke(requestId: string) {
    if (!confirm('Revoke this access grant? The user will lose access immediately.')) return
    try {
      setActionLoading(requestId)
      setError(null)
      await revokeJITGrant(requestId, notes[requestId])
      setSuccess('Grant revoked')
      await loadData()
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to revoke grant')
      console.error(err)
    } finally {
      setActionLoading(null)
    }
  }

  async function handleSaveSettings() {
    try {
      setSettingsLoading(true)
      setError(null)
      await updateSettings({
        jit_webhook_enabled: webhookEnabled ? 'true' : 'false',
        jit_webhook_url: webhookURL,
        jit_app_url: appURL,
      })
      setSettingsSaved(true)
      setTimeout(() => setSettingsSaved(false), 3000)
    } catch (err) {
      setError('Failed to save webhook settings')
      console.error(err)
    } finally {
      setSettingsLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">JIT Access Management</h1>
        <p className="text-theme-tertiary mt-1">
          Review and manage just-in-time access requests and active grants.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="card">
          <div className="flex items-center space-x-3">
            <div className="flex-shrink-0 p-3 bg-yellow-100 dark:bg-yellow-900/30 rounded-lg">
              <svg className="h-6 w-6 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-theme-tertiary">Pending Requests</p>
              <p className="text-2xl font-bold text-theme-primary">{stats.pending_requests}</p>
            </div>
          </div>
        </div>
        <div className="card">
          <div className="flex items-center space-x-3">
            <div className="flex-shrink-0 p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
              <svg className="h-6 w-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-theme-tertiary">Active Grants</p>
              <p className="text-2xl font-bold text-theme-primary">{stats.active_grants}</p>
            </div>
          </div>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 text-red-700 dark:text-red-400 flex justify-between items-center">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-500 hover:text-red-700">&times;</button>
        </div>
      )}

      {success && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 text-green-700 dark:text-green-400 flex justify-between items-center">
          <span>{success}</span>
          <button onClick={() => setSuccess(null)} className="text-green-500 hover:text-green-700">&times;</button>
        </div>
      )}

      {/* Tabs */}
      <div className="card">
        <div className="flex space-x-2">
          <button
            onClick={() => setTab('pending')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              tab === 'pending'
                ? 'bg-primary-600 text-white'
                : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
            }`}
          >
            Pending Requests
            {stats.pending_requests > 0 && (
              <span className="ml-2 inline-flex items-center justify-center px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-500 text-white">
                {stats.pending_requests}
              </span>
            )}
          </button>
          <button
            onClick={() => setTab('active')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              tab === 'active'
                ? 'bg-primary-600 text-white'
                : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
            }`}
          >
            Active Grants
          </button>
          <button
            onClick={() => setTab('history')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              tab === 'history'
                ? 'bg-primary-600 text-white'
                : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
            }`}
          >
            History
          </button>
          <button
            onClick={() => setTab('settings')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              tab === 'settings'
                ? 'bg-primary-600 text-white'
                : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
            }`}
          >
            Settings
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : tab === 'pending' ? (
        /* Pending Requests */
        <div className="space-y-4">
          {pendingRequests.length === 0 ? (
            <div className="card">
              <p className="text-theme-tertiary text-center py-8">No pending requests</p>
            </div>
          ) : (
            pendingRequests.map((req) => (
              <div key={req.id} className="card">
                <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                  <div className="flex-1 space-y-2">
                    <div className="flex items-center space-x-3">
                      <span className="font-medium text-theme-primary">{req.requester_email || 'Unknown user'}</span>
                      <span className="text-xs text-theme-muted">{timeAgo(req.created_at)}</span>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className="text-sm text-theme-secondary">{req.resource_name}</span>
                      {typeBadge(req.resource_type)}
                    </div>
                    <div className="text-sm text-theme-tertiary">
                      <span className="font-medium">Justification:</span> {req.justification}
                    </div>
                    <div className="text-sm text-theme-muted">
                      Duration: {req.duration_minutes >= 60 ? `${req.duration_minutes / 60}h` : `${req.duration_minutes}m`}
                    </div>
                    <div>
                      <input
                        type="text"
                        value={notes[req.id] || ''}
                        onChange={(e) => setNotes((prev) => ({ ...prev, [req.id]: e.target.value }))}
                        className="input text-sm"
                        placeholder="Optional note..."
                      />
                    </div>
                  </div>
                  <div className="flex space-x-2 sm:flex-col sm:space-x-0 sm:space-y-2">
                    <button
                      onClick={() => handleApprove(req.id)}
                      disabled={actionLoading === req.id}
                      className="btn btn-primary text-sm"
                    >
                      {actionLoading === req.id ? 'Processing...' : 'Approve'}
                    </button>
                    <button
                      onClick={() => handleDeny(req.id)}
                      disabled={actionLoading === req.id}
                      className="inline-flex items-center px-3 py-1.5 border border-red-300 dark:border-red-700 text-sm font-medium rounded-md text-red-700 dark:text-red-400 bg-theme-card hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                    >
                      Deny
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      ) : tab === 'active' ? (
        /* Active Grants */
        <div className="card">
          <h2 className="text-lg font-semibold text-theme-primary mb-4">
            Active Grants ({approvedRequests.length})
          </h2>
          {approvedRequests.length === 0 ? (
            <p className="text-theme-tertiary text-center py-8">No active grants</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">User</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Resource</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Granted</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Expires</th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme">
                  {approvedRequests.map((req) => (
                    <tr key={req.id} className="hover:bg-theme-tertiary transition-colors">
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-primary">
                        {req.requester_email || 'Unknown'}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span className="font-medium text-theme-primary">{req.resource_name}</span>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        {typeBadge(req.resource_type)}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                        {req.decided_at ? formatDate(req.decided_at) : formatDate(req.created_at)}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <CountdownTimer expiresAt={req.expires_at} />
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-right">
                        <button
                          onClick={() => handleRevoke(req.id)}
                          disabled={actionLoading === req.id}
                          className="inline-flex items-center px-3 py-1.5 border border-red-300 dark:border-red-700 text-sm font-medium rounded-md text-red-700 dark:text-red-400 bg-theme-card hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                        >
                          {actionLoading === req.id ? 'Revoking...' : 'Revoke'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      ) : tab === 'history' ? (
        /* History */
        <div className="card">
          <h2 className="text-lg font-semibold text-theme-primary mb-4">
            Request History ({allRequests.length})
          </h2>
          {allRequests.length === 0 ? (
            <p className="text-theme-tertiary text-center py-8">No request history</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">User</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Resource</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Duration</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Submitted</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Decided By</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme">
                  {allRequests.map((req) => (
                    <tr key={req.id} className="hover:bg-theme-tertiary transition-colors">
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-primary">
                        {req.requester_email || 'Unknown'}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span className="font-medium text-theme-primary">{req.resource_name}</span>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        {typeBadge(req.resource_type)}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        {statusBadge(req.status)}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                        {req.duration_minutes >= 60 ? `${req.duration_minutes / 60}h` : `${req.duration_minutes}m`}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                        {formatDate(req.created_at)}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                        {req.approver_email || '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      ) : (
        /* Settings */
        <div className="card">
          <h2 className="text-lg font-semibold text-theme-primary mb-4">Webhook Notifications</h2>
          <p className="text-sm text-theme-tertiary mb-6">
            Configure webhook notifications for JIT access requests. Supports Slack and Microsoft Teams incoming webhooks.
          </p>
          <div className="space-y-4 max-w-lg">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-theme-primary">Enable Webhook Notifications</label>
              <button
                type="button"
                role="switch"
                aria-checked={webhookEnabled}
                onClick={() => setWebhookEnabled(!webhookEnabled)}
                className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 ${
                  webhookEnabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    webhookEnabled ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
            </div>
            <div>
              <label className="block text-sm font-medium text-theme-primary mb-1">Webhook URL</label>
              <input
                type="url"
                value={webhookURL}
                onChange={(e) => setWebhookURL(e.target.value)}
                className="input w-full"
                placeholder="https://hooks.slack.com/services/..."
              />
              <p className="text-xs text-theme-muted mt-1">Slack or Teams incoming webhook URL</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-theme-primary mb-1">Application URL</label>
              <input
                type="url"
                value={appURL}
                onChange={(e) => setAppURL(e.target.value)}
                className="input w-full"
                placeholder="https://gatekey.example.com"
              />
              <p className="text-xs text-theme-muted mt-1">Base URL for approval links in notifications</p>
            </div>
            <div className="pt-2">
              <button
                onClick={handleSaveSettings}
                disabled={settingsLoading}
                className="btn btn-primary"
              >
                {settingsLoading ? 'Saving...' : 'Save Settings'}
              </button>
              {settingsSaved && (
                <span className="ml-3 text-sm text-green-600 dark:text-green-400">Settings saved</span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
