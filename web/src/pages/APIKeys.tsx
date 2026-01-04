import { useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import {
  APIKey,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  getUserAPIKeys,
  createUserAPIKey,
  revokeUserAPIKey,
  getAdminAPIKeys,
  createAdminUserAPIKey,
  revokeAdminAPIKey,
  AdminAPIKey,
} from '../api/client'

export default function APIKeys() {
  const { user } = useAuth()
  const [keys, setKeys] = useState<APIKey[]>([])
  const [adminKeys, setAdminKeys] = useState<AdminAPIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [revoking, setRevoking] = useState<string | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showAdminCreateModal, setShowAdminCreateModal] = useState(false)
  const [newKey, setNewKey] = useState<CreateAPIKeyResponse | null>(null)
  const [viewMode, setViewMode] = useState<'user' | 'admin'>('user')

  useEffect(() => {
    loadKeys()
  }, [user?.isAdmin])

  async function loadKeys() {
    try {
      setLoading(true)
      setError(null)
      const userKeys = await getUserAPIKeys()
      setKeys(userKeys)

      if (user?.isAdmin) {
        const allKeys = await getAdminAPIKeys()
        setAdminKeys(allKeys)
      }
    } catch (err) {
      setError('Failed to load API keys')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleRevoke(keyId: string, isAdmin: boolean = false) {
    if (!confirm('Are you sure you want to revoke this API key? This action cannot be undone.')) {
      return
    }

    try {
      setRevoking(keyId)
      if (isAdmin) {
        await revokeAdminAPIKey(keyId)
      } else {
        await revokeUserAPIKey(keyId)
      }
      setSuccess('API key revoked successfully')
      await loadKeys()
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to revoke API key')
      console.error(err)
    } finally {
      setRevoking(null)
    }
  }

  function formatDate(dateStr: string | null) {
    if (!dateStr) return 'Never'
    return new Date(dateStr).toLocaleString()
  }

  function isExpired(dateStr: string | null) {
    if (!dateStr) return false
    return new Date(dateStr) < new Date()
  }

  const activeKeys = keys.filter(k => !k.isRevoked && !isExpired(k.expiresAt))
  const revokedKeys = keys.filter(k => k.isRevoked || isExpired(k.expiresAt))

  const activeAdminKeys = adminKeys.filter(k => !k.isRevoked && !isExpired(k.expiresAt))
  const revokedAdminKeys = adminKeys.filter(k => k.isRevoked || isExpired(k.expiresAt))

  return (
    <div className="space-y-6">
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">API Keys</h1>
        <p className="text-theme-tertiary mt-1">
          Manage API keys for CLI authentication and automation.
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

      {/* View toggle for admins */}
      {user?.isAdmin && (
        <div className="card">
          <div className="flex space-x-2">
            <button
              onClick={() => setViewMode('user')}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                viewMode === 'user'
                  ? 'bg-primary-600 text-white'
                  : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
              }`}
            >
              My API Keys
            </button>
            <button
              onClick={() => setViewMode('admin')}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                viewMode === 'admin'
                  ? 'bg-primary-600 text-white'
                  : 'bg-theme-tertiary text-theme-secondary hover:bg-theme-secondary'
              }`}
            >
              All Users' Keys
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : viewMode === 'user' ? (
        <>
          {/* User's own API Keys */}
          <div className="card">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-lg font-semibold text-theme-primary">
                Active API Keys ({activeKeys.length})
              </h2>
              <button
                onClick={() => setShowCreateModal(true)}
                className="btn btn-primary"
              >
                + Create API Key
              </button>
            </div>
            {activeKeys.length === 0 ? (
              <p className="text-theme-tertiary text-center py-8">No active API keys</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Expires</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Last Used</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {activeKeys.map((key) => (
                      <tr key={key.id} className="hover:bg-theme-tertiary transition-colors">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-theme-primary">{key.name}</span>
                          {key.description && (
                            <p className="text-xs text-theme-tertiary">{key.description}</p>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-theme-tertiary text-theme-secondary px-2 py-1 rounded">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {formatDate(key.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {key.expiresAt ? formatDate(key.expiresAt) : 'Never'}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {key.lastUsedAt ? (
                            <span title={key.lastUsedIp || undefined}>
                              {formatDate(key.lastUsedAt)}
                            </span>
                          ) : 'Never'}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-right">
                          <button
                            onClick={() => handleRevoke(key.id)}
                            disabled={revoking === key.id}
                            className="inline-flex items-center px-3 py-1.5 border border-red-300 dark:border-red-700 text-sm font-medium rounded-md text-red-700 dark:text-red-400 bg-theme-card hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                          >
                            <svg className="h-4 w-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                            </svg>
                            {revoking === key.id ? 'Revoking...' : 'Revoke'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Revoked Keys */}
          {revokedKeys.length > 0 && (
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">
                Revoked/Expired Keys ({revokedKeys.length})
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {revokedKeys.map((key) => (
                      <tr key={key.id} className="opacity-60">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-theme-tertiary">{key.name}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-theme-secondary px-2 py-1 rounded text-theme-muted">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-muted">
                          {formatDate(key.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {key.isRevoked ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-600 text-white">
                              Revoked {key.revokedAt && formatDate(key.revokedAt)}
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300">
                              Expired
                            </span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      ) : (
        <>
          {/* Admin View - All Users' Keys */}
          <div className="card">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-lg font-semibold text-theme-primary">
                All Active API Keys ({activeAdminKeys.length})
              </h2>
              <button
                onClick={() => setShowAdminCreateModal(true)}
                className="btn btn-primary"
              >
                + Create Key for User
              </button>
            </div>
            {activeAdminKeys.length === 0 ? (
              <p className="text-theme-tertiary text-center py-8">No active API keys in the system</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Last Used</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {activeAdminKeys.map((key) => (
                      <tr key={key.id} className="hover:bg-theme-tertiary transition-colors">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <div className="text-sm font-medium text-theme-primary">{key.userEmail}</div>
                          {key.userName && (
                            <div className="text-xs text-theme-tertiary">{key.userName}</div>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-theme-primary">{key.name}</span>
                          {key.isAdminProvisioned && (
                            <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-600 text-white">
                              Admin Created
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-theme-tertiary text-theme-secondary px-2 py-1 rounded">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {formatDate(key.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {key.lastUsedAt ? formatDate(key.lastUsedAt) : 'Never'}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-right space-x-2">
                          <button
                            onClick={() => handleRevoke(key.id, true)}
                            disabled={revoking === key.id}
                            className="inline-flex items-center px-3 py-1.5 border border-red-300 dark:border-red-700 text-sm font-medium rounded-md text-red-700 dark:text-red-400 bg-theme-card hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50"
                          >
                            <svg className="h-4 w-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                            </svg>
                            {revoking === key.id ? 'Revoking...' : 'Revoke'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Revoked Admin Keys */}
          {revokedAdminKeys.length > 0 && (
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">
                Revoked/Expired Keys ({revokedAdminKeys.length})
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {revokedAdminKeys.map((key) => (
                      <tr key={key.id} className="opacity-60">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <div className="text-sm text-theme-tertiary">{key.userEmail}</div>
                          {key.userName && (
                            <div className="text-xs text-theme-muted">{key.userName}</div>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-theme-tertiary">{key.name}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-theme-secondary px-2 py-1 rounded text-theme-muted">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {key.isRevoked ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-600 text-white">
                              Revoked
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300">
                              Expired
                            </span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {/* Info Box */}
      <div className="info-box">
        <div className="flex">
          <svg className="h-5 w-5 info-box-icon mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div className="ml-3">
            <h3 className="info-box-title">About API Keys</h3>
            <p className="mt-1 info-box-text">
              API keys allow you to authenticate with GateKey CLI tools without browser-based SSO.
              Use them for automation, CI/CD pipelines, or headless systems.
              The full API key is only shown once when created - save it securely!
            </p>
          </div>
        </div>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <CreateAPIKeyModal
          onClose={() => {
            setShowCreateModal(false)
            setNewKey(null)
          }}
          onCreated={(key) => {
            setNewKey(key)
            loadKeys()
          }}
          newKey={newKey}
        />
      )}

      {/* Admin Create Modal */}
      {showAdminCreateModal && (
        <AdminCreateAPIKeyModal
          onClose={() => {
            setShowAdminCreateModal(false)
            setNewKey(null)
          }}
          onCreated={(key) => {
            setNewKey(key)
            loadKeys()
          }}
          newKey={newKey}
        />
      )}
    </div>
  )
}

interface CreateAPIKeyModalProps {
  onClose: () => void
  onCreated: (key: CreateAPIKeyResponse) => void
  newKey: CreateAPIKeyResponse | null
}

function CreateAPIKeyModal({ onClose, onCreated, newKey }: CreateAPIKeyModalProps) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [expiresIn, setExpiresIn] = useState('90d')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    try {
      setCreating(true)
      setError(null)
      const req: CreateAPIKeyRequest = {
        name,
        description: description || undefined,
        expires_in: expiresIn === 'never' ? undefined : expiresIn,
      }
      const key = await createUserAPIKey(req)
      onCreated(key)
    } catch (err) {
      setError('Failed to create API key')
      console.error(err)
    } finally {
      setCreating(false)
    }
  }

  function copyToClipboard() {
    if (newKey?.rawKey) {
      navigator.clipboard.writeText(newKey.rawKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4">
        <div className="fixed inset-0 bg-black opacity-50" onClick={onClose}></div>
        <div className="relative bg-theme-card rounded-lg shadow-xl max-w-md w-full p-6 border border-theme">
          {newKey ? (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <svg className="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <h3 className="text-lg font-semibold text-theme-primary">API Key Created</h3>
              </div>
              <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg">
                <p className="text-sm text-yellow-800 dark:text-yellow-300 font-medium mb-2">
                  Copy your API key now. You won't be able to see it again!
                </p>
                <div className="flex items-center space-x-2">
                  <code className="flex-1 p-2 bg-theme-card border border-theme rounded text-sm font-mono break-all text-theme-primary">
                    {newKey.rawKey}
                  </code>
                  <button
                    onClick={copyToClipboard}
                    className="btn btn-secondary text-sm whitespace-nowrap"
                  >
                    {copied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
              </div>
              <div className="text-sm text-theme-tertiary">
                <p><strong className="text-theme-primary">Name:</strong> {newKey.name}</p>
                <p><strong className="text-theme-primary">Expires:</strong> {newKey.expiresAt ? new Date(newKey.expiresAt).toLocaleDateString() : 'Never'}</p>
              </div>
              <div className="flex justify-end pt-4">
                <button onClick={onClose} className="btn btn-primary">
                  Done
                </button>
              </div>
            </div>
          ) : (
            <form onSubmit={handleCreate} className="space-y-4">
              <h3 className="text-lg font-semibold text-theme-primary">Create API Key</h3>

              {error && (
                <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400 text-sm">
                  {error}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="input"
                  placeholder="My CLI Key"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">Description (optional)</label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="input"
                  placeholder="Used for CI/CD automation"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">Expires</label>
                <select
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(e.target.value)}
                  className="input"
                >
                  <option value="30d">30 days</option>
                  <option value="90d">90 days</option>
                  <option value="180d">180 days</option>
                  <option value="1y">1 year</option>
                  <option value="never">Never</option>
                </select>
              </div>

              <div className="flex justify-end space-x-3 pt-4">
                <button type="button" onClick={onClose} className="btn btn-secondary">
                  Cancel
                </button>
                <button type="submit" disabled={creating} className="btn btn-primary">
                  {creating ? 'Creating...' : 'Create Key'}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}

interface AdminCreateAPIKeyModalProps {
  onClose: () => void
  onCreated: (key: CreateAPIKeyResponse) => void
  newKey: CreateAPIKeyResponse | null
}

function AdminCreateAPIKeyModal({ onClose, onCreated, newKey }: AdminCreateAPIKeyModalProps) {
  const [userId, setUserId] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [expiresIn, setExpiresIn] = useState('90d')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    try {
      setCreating(true)
      setError(null)
      const req: CreateAPIKeyRequest = {
        name,
        description: description || undefined,
        expires_in: expiresIn === 'never' ? undefined : expiresIn,
      }
      const key = await createAdminUserAPIKey(userId, req)
      onCreated(key)
    } catch (err) {
      setError('Failed to create API key. Make sure the User ID is valid.')
      console.error(err)
    } finally {
      setCreating(false)
    }
  }

  function copyToClipboard() {
    if (newKey?.rawKey) {
      navigator.clipboard.writeText(newKey.rawKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4">
        <div className="fixed inset-0 bg-black opacity-50" onClick={onClose}></div>
        <div className="relative bg-theme-card rounded-lg shadow-xl max-w-md w-full p-6 border border-theme">
          {newKey ? (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <svg className="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <h3 className="text-lg font-semibold text-theme-primary">API Key Created</h3>
              </div>
              <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg">
                <p className="text-sm text-yellow-800 dark:text-yellow-300 font-medium mb-2">
                  Copy this API key now and share it with the user securely!
                </p>
                <div className="flex items-center space-x-2">
                  <code className="flex-1 p-2 bg-theme-card border border-theme rounded text-sm font-mono break-all text-theme-primary">
                    {newKey.rawKey}
                  </code>
                  <button
                    onClick={copyToClipboard}
                    className="btn btn-secondary text-sm whitespace-nowrap"
                  >
                    {copied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
              </div>
              <div className="text-sm text-theme-tertiary">
                <p><strong className="text-theme-primary">Name:</strong> {newKey.name}</p>
                <p><strong className="text-theme-primary">Expires:</strong> {newKey.expiresAt ? new Date(newKey.expiresAt).toLocaleDateString() : 'Never'}</p>
              </div>
              <div className="flex justify-end pt-4">
                <button onClick={onClose} className="btn btn-primary">
                  Done
                </button>
              </div>
            </div>
          ) : (
            <form onSubmit={handleCreate} className="space-y-4">
              <h3 className="text-lg font-semibold text-theme-primary">Create API Key for User</h3>

              {error && (
                <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400 text-sm">
                  {error}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">User ID</label>
                <input
                  type="text"
                  value={userId}
                  onChange={(e) => setUserId(e.target.value)}
                  className="input"
                  placeholder="user-uuid-here"
                  required
                />
                <p className="text-xs text-theme-muted mt-1">Get the User ID from the Users page</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">Key Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="input"
                  placeholder="Service Account Key"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">Description (optional)</label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="input"
                  placeholder="Created by admin for automation"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">Expires</label>
                <select
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(e.target.value)}
                  className="input"
                >
                  <option value="30d">30 days</option>
                  <option value="90d">90 days</option>
                  <option value="180d">180 days</option>
                  <option value="1y">1 year</option>
                  <option value="never">Never</option>
                </select>
              </div>

              <div className="flex justify-end space-x-3 pt-4">
                <button type="button" onClick={onClose} className="btn btn-secondary">
                  Cancel
                </button>
                <button type="submit" disabled={creating} className="btn btn-primary">
                  {creating ? 'Creating...' : 'Create Key'}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
