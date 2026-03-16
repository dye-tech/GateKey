import { useState, useEffect, useCallback } from 'react'
import {
  JITResource,
  JITRequest,
  JITGrant,
  getJITResources,
  createJITRequest,
  getMyJITRequests,
  cancelJITRequest,
  getMyJITGrants,
} from '../api/client'

const DURATION_OPTIONS = [
  { value: 15, label: '15 minutes' },
  { value: 30, label: '30 minutes' },
  { value: 60, label: '1 hour' },
  { value: 120, label: '2 hours' },
  { value: 240, label: '4 hours' },
  { value: 480, label: '8 hours' },
]

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

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString()
}

function useCountdown(expiresAt: string) {
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

  return remaining
}

function GrantCountdown({ expiresAt }: { expiresAt: string }) {
  const remaining = useCountdown(expiresAt)
  const isExpired = remaining === 'Expired'
  return (
    <span className={`font-mono text-sm ${isExpired ? 'text-red-500' : 'text-green-600 dark:text-green-400'}`}>
      {remaining}
    </span>
  )
}

export default function AccessRequests() {
  const [tab, setTab] = useState<'request' | 'my-requests'>('request')
  const [resources, setResources] = useState<JITResource[]>([])
  const [requests, setRequests] = useState<JITRequest[]>([])
  const [grants, setGrants] = useState<JITGrant[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // Request form state
  const [selectedResource, setSelectedResource] = useState<JITResource | null>(null)
  const [duration, setDuration] = useState(60)
  const [justification, setJustification] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [canceling, setCanceling] = useState<string | null>(null)

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const results = await Promise.allSettled([
        getJITResources(),
        getMyJITRequests(),
        getMyJITGrants(),
      ])
      if (results[0].status === 'fulfilled') setResources(results[0].value)
      if (results[1].status === 'fulfilled') setRequests(results[1].value)
      if (results[2].status === 'fulfilled') setGrants(results[2].value)

      const failed = results.filter(r => r.status === 'rejected')
      if (failed.length === results.length) {
        setError('Failed to load data')
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  async function handleSubmitRequest(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedResource) return
    try {
      setSubmitting(true)
      setError(null)
      await createJITRequest({
        resource_type: selectedResource.type,
        resource_id: selectedResource.id,
        justification,
        duration_minutes: duration,
      })
      setSuccess('Access request submitted successfully')
      setSelectedResource(null)
      setJustification('')
      setDuration(60)
      await loadData()
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to submit access request')
      console.error(err)
    } finally {
      setSubmitting(false)
    }
  }

  async function handleCancel(id: string) {
    if (!confirm('Cancel this access request?')) return
    try {
      setCanceling(id)
      await cancelJITRequest(id)
      setSuccess('Request canceled')
      await loadData()
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to cancel request')
      console.error(err)
    } finally {
      setCanceling(null)
    }
  }

  // Group resources by type
  const groupedResources = resources.reduce<Record<string, JITResource[]>>((acc, r) => {
    if (!acc[r.type]) acc[r.type] = []
    acc[r.type].push(r)
    return acc
  }, {})

  const activeGrants = grants.filter(g => g.is_active && new Date(g.expires_at) > new Date())

  return (
    <div className="space-y-6">
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">Access Requests</h1>
        <p className="text-theme-tertiary mt-1">
          Request just-in-time access to resources. Access is time-limited and requires approval.
        </p>
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
            onClick={() => setTab('request')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              tab === 'request'
                ? 'bg-primary-600 text-white'
                : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
            }`}
          >
            Request Access
          </button>
          <button
            onClick={() => setTab('my-requests')}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              tab === 'my-requests'
                ? 'bg-primary-600 text-white'
                : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
            }`}
          >
            My Requests
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : tab === 'request' ? (
        <>
          {/* Active Grants */}
          {activeGrants.length > 0 && (
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">
                Active Access ({activeGrants.length})
              </h2>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {activeGrants.map((grant) => (
                  <div key={grant.id} className="p-4 border border-green-200 dark:border-green-800 rounded-lg bg-green-50/50 dark:bg-green-900/10">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm font-medium text-theme-primary">{grant.resource_id}</span>
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
                        Active
                      </span>
                    </div>
                    <div className="text-xs text-theme-tertiary mb-1">{grant.resource_type}</div>
                    <div className="flex items-center space-x-1 text-sm">
                      <span className="text-theme-tertiary">Expires in:</span>
                      <GrantCountdown expiresAt={grant.expires_at} />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Request Form */}
          {selectedResource ? (
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">
                Request Access: {selectedResource.name}
              </h2>
              <form onSubmit={handleSubmitRequest} className="space-y-4 max-w-lg">
                <div>
                  <label className="block text-sm font-medium text-theme-secondary mb-1">Resource</label>
                  <div className="p-3 bg-theme-tertiary rounded-lg">
                    <div className="font-medium text-theme-primary">{selectedResource.name}</div>
                    <div className="text-sm text-theme-tertiary">{selectedResource.description}</div>
                    <div className="text-xs text-theme-muted mt-1">{TYPE_LABELS[selectedResource.type] || selectedResource.type}</div>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-theme-secondary mb-1">Duration</label>
                  <select
                    value={duration}
                    onChange={(e) => setDuration(Number(e.target.value))}
                    className="input"
                  >
                    {DURATION_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>{opt.label}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-theme-secondary mb-1">Justification</label>
                  <textarea
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    className="input"
                    rows={3}
                    placeholder="Why do you need access to this resource?"
                    required
                  />
                </div>

                <div className="flex space-x-3 pt-2">
                  <button type="submit" disabled={submitting} className="btn btn-primary">
                    {submitting ? 'Submitting...' : 'Submit Request'}
                  </button>
                  <button
                    type="button"
                    onClick={() => setSelectedResource(null)}
                    className="btn btn-secondary"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            </div>
          ) : (
            <>
              {/* Available Resources */}
              {Object.keys(groupedResources).length === 0 ? (
                <div className="card">
                  <p className="text-theme-tertiary text-center py-8">No resources available for JIT access</p>
                </div>
              ) : (
                Object.entries(groupedResources).map(([type, items]) => (
                  <div key={type} className="card">
                    <h2 className="text-lg font-semibold text-theme-primary mb-4">
                      {TYPE_LABELS[type] || type}
                    </h2>
                    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                      {items.map((resource) => (
                        <div key={resource.id} className="p-4 border border-theme rounded-lg hover:border-primary-300 dark:hover:border-primary-700 transition-colors">
                          <div className="font-medium text-theme-primary mb-1">{resource.name}</div>
                          <p className="text-sm text-theme-tertiary mb-3">{resource.description}</p>
                          <button
                            onClick={() => setSelectedResource(resource)}
                            className="btn btn-primary text-sm"
                          >
                            Request Access
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              )}
            </>
          )}
        </>
      ) : (
        /* My Requests tab */
        <div className="card">
          <h2 className="text-lg font-semibold text-theme-primary mb-4">
            My Requests ({requests.length})
          </h2>
          {requests.length === 0 ? (
            <p className="text-theme-tertiary text-center py-8">No access requests yet</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Resource</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Duration</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Submitted</th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme">
                  {requests.map((req) => (
                    <tr key={req.id} className="hover:bg-theme-tertiary transition-colors">
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span className="font-medium text-theme-primary">{req.resource_name}</span>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                        {TYPE_LABELS[req.resource_type] || req.resource_type}
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
                      <td className="px-4 py-3 whitespace-nowrap text-right">
                        {req.status === 'pending' && (
                          <button
                            onClick={() => handleCancel(req.id)}
                            disabled={canceling === req.id}
                            className="inline-flex items-center px-3 py-1.5 border border-red-300 dark:border-red-700 text-sm font-medium rounded-md text-red-700 dark:text-red-400 bg-theme-card hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                          >
                            {canceling === req.id ? 'Canceling...' : 'Cancel'}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
