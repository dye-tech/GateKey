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
        <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
        <p className="text-gray-500 mt-1">
          Manage API keys for CLI authentication and automation.
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700 flex justify-between items-center">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-500 hover:text-red-700">&times;</button>
        </div>
      )}

      {success && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 text-green-700 flex justify-between items-center">
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
                  ? 'bg-primary-100 text-primary-700'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              My API Keys
            </button>
            <button
              onClick={() => setViewMode('admin')}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                viewMode === 'admin'
                  ? 'bg-primary-100 text-primary-700'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
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
              <h2 className="text-lg font-semibold text-gray-900">
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
              <p className="text-gray-500 text-center py-8">No active API keys</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Expires</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Last Used</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {activeKeys.map((key) => (
                      <tr key={key.id}>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-gray-900">{key.name}</span>
                          {key.description && (
                            <p className="text-xs text-gray-500">{key.description}</p>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-gray-100 px-2 py-1 rounded">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(key.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {key.expiresAt ? formatDate(key.expiresAt) : 'Never'}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
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
                            className="inline-flex items-center px-3 py-1.5 border border-red-300 text-sm font-medium rounded-md text-red-700 bg-white hover:bg-red-50 disabled:opacity-50"
                          >
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
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Revoked/Expired Keys ({revokedKeys.length})
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {revokedKeys.map((key) => (
                      <tr key={key.id} className="bg-gray-50">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-gray-500">{key.name}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-gray-200 px-2 py-1 rounded text-gray-500">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-400">
                          {formatDate(key.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {key.isRevoked ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                              Revoked {key.revokedAt && formatDate(key.revokedAt)}
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
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
              <h2 className="text-lg font-semibold text-gray-900">
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
              <p className="text-gray-500 text-center py-8">No active API keys in the system</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Last Used</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {activeAdminKeys.map((key) => (
                      <tr key={key.id}>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="text-sm text-gray-900">{key.userEmail}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-gray-900">{key.name}</span>
                          {key.isAdminProvisioned && (
                            <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">
                              Admin Created
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-gray-100 px-2 py-1 rounded">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(key.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {key.lastUsedAt ? formatDate(key.lastUsedAt) : 'Never'}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-right space-x-2">
                          <button
                            onClick={() => handleRevoke(key.id, true)}
                            disabled={revoking === key.id}
                            className="inline-flex items-center px-3 py-1.5 border border-red-300 text-sm font-medium rounded-md text-red-700 bg-white hover:bg-red-50 disabled:opacity-50"
                          >
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
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Revoked/Expired Keys ({revokedAdminKeys.length})
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key Prefix</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {revokedAdminKeys.map((key) => (
                      <tr key={key.id} className="bg-gray-50">
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {key.userEmail}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-gray-500">{key.name}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <code className="text-sm bg-gray-200 px-2 py-1 rounded text-gray-500">{key.keyPrefix}...</code>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {key.isRevoked ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                              Revoked
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
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
      <div className="card bg-blue-50 border-blue-200">
        <div className="flex">
          <svg className="h-5 w-5 text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-blue-800">About API Keys</h3>
            <p className="mt-1 text-sm text-blue-700">
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
        <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full p-6">
          {newKey ? (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <svg className="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <h3 className="text-lg font-semibold text-gray-900">API Key Created</h3>
              </div>
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p className="text-sm text-yellow-800 font-medium mb-2">
                  Copy your API key now. You won't be able to see it again!
                </p>
                <div className="flex items-center space-x-2">
                  <code className="flex-1 p-2 bg-white border rounded text-sm font-mono break-all">
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
              <div className="text-sm text-gray-500">
                <p><strong>Name:</strong> {newKey.name}</p>
                <p><strong>Expires:</strong> {newKey.expiresAt ? new Date(newKey.expiresAt).toLocaleDateString() : 'Never'}</p>
              </div>
              <div className="flex justify-end pt-4">
                <button onClick={onClose} className="btn btn-primary">
                  Done
                </button>
              </div>
            </div>
          ) : (
            <form onSubmit={handleCreate} className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900">Create API Key</h3>

              {error && (
                <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                  {error}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  placeholder="My CLI Key"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Description (optional)</label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  placeholder="Used for CI/CD automation"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Expires</label>
                <select
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
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
        <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full p-6">
          {newKey ? (
            <div className="space-y-4">
              <div className="flex items-center space-x-2">
                <svg className="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <h3 className="text-lg font-semibold text-gray-900">API Key Created</h3>
              </div>
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p className="text-sm text-yellow-800 font-medium mb-2">
                  Copy this API key now and share it with the user securely!
                </p>
                <div className="flex items-center space-x-2">
                  <code className="flex-1 p-2 bg-white border rounded text-sm font-mono break-all">
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
              <div className="text-sm text-gray-500">
                <p><strong>Name:</strong> {newKey.name}</p>
                <p><strong>Expires:</strong> {newKey.expiresAt ? new Date(newKey.expiresAt).toLocaleDateString() : 'Never'}</p>
              </div>
              <div className="flex justify-end pt-4">
                <button onClick={onClose} className="btn btn-primary">
                  Done
                </button>
              </div>
            </div>
          ) : (
            <form onSubmit={handleCreate} className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900">Create API Key for User</h3>

              {error && (
                <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                  {error}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">User ID</label>
                <input
                  type="text"
                  value={userId}
                  onChange={(e) => setUserId(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  placeholder="user-uuid-here"
                  required
                />
                <p className="text-xs text-gray-500 mt-1">Get the User ID from the Users page</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Key Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  placeholder="Service Account Key"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Description (optional)</label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  placeholder="Created by admin for automation"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Expires</label>
                <select
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
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
