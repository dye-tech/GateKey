import { useState, useEffect } from 'react'
import {
  ProxyApplication,
  getProxyApps,
  createProxyApp,
  updateProxyApp,
  deleteProxyApp,
  getProxyAppUsers,
  assignProxyAppToUser,
  removeProxyAppFromUser,
  getProxyAppGroups,
  assignProxyAppToGroup,
  removeProxyAppFromGroup,
  getUsers,
  getGroups,
  SSOUser,
  Group,
  CreateProxyAppRequest,
} from '../api/client'
import ActionDropdown, { ActionItem } from '../components/ActionDropdown'

export default function AdminProxyApps() {
  const [apps, setApps] = useState<ProxyApplication[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editingApp, setEditingApp] = useState<ProxyApplication | null>(null)
  const [assigningApp, setAssigningApp] = useState<ProxyApplication | null>(null)

  useEffect(() => {
    loadApps()
  }, [])

  async function loadApps() {
    try {
      setLoading(true)
      const data = await getProxyApps()
      setApps(data)
      setError(null)
    } catch {
      setError('Failed to load proxy applications')
    } finally {
      setLoading(false)
    }
  }

  async function handleDelete(app: ProxyApplication) {
    if (!confirm(`Delete "${app.name}"? This will remove all user access.`)) return
    try {
      await deleteProxyApp(app.id)
      await loadApps()
    } catch {
      setError('Failed to delete application')
    }
  }

  function copyProxyUrl(slug: string) {
    const url = `${window.location.origin}/proxy/${slug}/`
    navigator.clipboard.writeText(url)
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-theme-primary">Web Access Applications</h1>
          <p className="text-theme-tertiary mt-1">Manage clientless web applications accessible via reverse proxy</p>
        </div>
        <button onClick={() => setShowAddModal(true)} className="btn btn-primary inline-flex items-center">
          <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Application
        </button>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-theme text-red-700 dark:text-red-400 px-4 py-3 rounded">
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
          <h3 className="mt-4 text-lg font-medium text-theme-primary">No applications</h3>
          <p className="mt-2 text-theme-tertiary">Add a web application to enable clientless browser access</p>
          <button onClick={() => setShowAddModal(true)} className="mt-4 btn btn-primary inline-flex items-center">
            <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Application
          </button>
        </div>
      ) : (
        <div className="card p-0">
          <table className="min-w-full divide-y divide-theme">
            <thead className="bg-theme-tertiary">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Slug / URL
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Internal URL
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-theme-card divide-y divide-theme">
              {apps.map((app) => (
                <tr key={app.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      {app.iconUrl ? (
                        <img src={app.iconUrl} alt="" className="h-8 w-8 rounded mr-3" />
                      ) : (
                        <div className="h-8 w-8 rounded bg-primary-100 text-primary-600 flex items-center justify-center mr-3">
                          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                          </svg>
                        </div>
                      )}
                      <div>
                        <div className="text-sm font-medium text-theme-primary">{app.name}</div>
                        <div className="text-sm text-theme-tertiary">{app.description || 'No description'}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      <code className="text-sm bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-gray-200 px-2 py-1 rounded">/proxy/{app.slug}/</code>
                      <button
                        onClick={() => copyProxyUrl(app.slug)}
                        className="text-theme-muted hover:text-theme-secondary"
                        title="Copy URL"
                      >
                        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                      </button>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm text-theme-primary font-mono">{app.internalUrl}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs font-semibold rounded-full ${
                      app.isActive ? 'bg-green-600 text-white' : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300'
                    }`}>
                      {app.isActive ? 'Active' : 'Disabled'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Open', icon: 'open', onClick: () => window.open(`/proxy/${app.slug}/`, '_blank'), color: 'green' },
                        { label: 'Access', icon: 'access', onClick: () => setAssigningApp(app), color: 'purple' },
                        { label: 'Edit', icon: 'edit', onClick: () => setEditingApp(app), color: 'gray' },
                        { label: 'Delete', icon: 'delete', onClick: () => handleDelete(app), color: 'red' },
                      ] as ActionItem[]}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showAddModal && (
        <AddEditModal
          onClose={() => setShowAddModal(false)}
          onSuccess={() => {
            setShowAddModal(false)
            loadApps()
          }}
        />
      )}

      {editingApp && (
        <AddEditModal
          app={editingApp}
          onClose={() => setEditingApp(null)}
          onSuccess={() => {
            setEditingApp(null)
            loadApps()
          }}
        />
      )}

      {assigningApp && (
        <AssignAccessModal
          app={assigningApp}
          onClose={() => setAssigningApp(null)}
        />
      )}
    </div>
  )
}

interface AddEditModalProps {
  app?: ProxyApplication
  onClose: () => void
  onSuccess: () => void
}

function AddEditModal({ app, onClose, onSuccess }: AddEditModalProps) {
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showAdvanced, setShowAdvanced] = useState(false)

  const [name, setName] = useState(app?.name || '')
  const [slug, setSlug] = useState(app?.slug || '')
  const [description, setDescription] = useState(app?.description || '')
  const [internalUrl, setInternalUrl] = useState(app?.internalUrl || '')
  const [iconUrl, setIconUrl] = useState(app?.iconUrl || '')
  const [isActive, setIsActive] = useState(app?.isActive ?? true)
  const [preserveHostHeader, setPreserveHostHeader] = useState(app?.preserveHostHeader ?? false)
  const [stripPrefix, setStripPrefix] = useState(app?.stripPrefix ?? true)
  const [websocketEnabled, setWebsocketEnabled] = useState(app?.websocketEnabled ?? true)
  const [timeoutSeconds, setTimeoutSeconds] = useState(app?.timeoutSeconds ?? 30)

  function generateSlug(text: string) {
    return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    const req: CreateProxyAppRequest = {
      name,
      slug,
      description,
      internal_url: internalUrl,
      icon_url: iconUrl || undefined,
      is_active: isActive,
      preserve_host_header: preserveHostHeader,
      strip_prefix: stripPrefix,
      websocket_enabled: websocketEnabled,
      timeout_seconds: timeoutSeconds,
    }

    try {
      if (app) {
        await updateProxyApp(app.id, req)
      } else {
        await createProxyApp(req)
      }
      onSuccess()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to save application')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <h2 className="text-xl font-semibold mb-4">
            {app ? 'Edit Application' : 'Add Application'}
          </h2>

          {error && (
            <div className="mb-4 bg-red-50 dark:bg-red-900/20 border border-theme text-red-700 dark:text-red-400 px-4 py-3 rounded">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-theme-secondary">Name *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => {
                  setName(e.target.value)
                  if (!app) setSlug(generateSlug(e.target.value))
                }}
                className="input mt-1"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Slug *</label>
              <div className="mt-1 flex rounded-md shadow-sm">
                <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-theme bg-theme-tertiary text-theme-tertiary text-sm">
                  /proxy/
                </span>
                <input
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                  className="flex-1 min-w-0 block w-full px-3 py-2 rounded-none rounded-r-md border border-theme focus:outline-none focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
                  required
                  pattern="[a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9]"
                />
              </div>
              <p className="mt-1 text-xs text-theme-tertiary">Lowercase letters, numbers, and dashes only</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Description</label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="input mt-1"
                placeholder="Brief description of the application"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Internal URL *</label>
              <input
                type="url"
                value={internalUrl}
                onChange={(e) => setInternalUrl(e.target.value)}
                className="input mt-1"
                placeholder="http://internal-app.local:8080"
                required
              />
              <p className="mt-1 text-xs text-theme-tertiary">The internal URL of the application to proxy to</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Icon URL</label>
              <input
                type="url"
                value={iconUrl}
                onChange={(e) => setIconUrl(e.target.value)}
                className="input mt-1"
                placeholder="https://example.com/icon.png"
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="isActive"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
              />
              <label htmlFor="isActive" className="ml-2 text-sm text-theme-secondary">
                Application is active
              </label>
            </div>

            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-sm text-primary-600 hover:text-primary-800"
            >
              {showAdvanced ? 'Hide' : 'Show'} Advanced Options
            </button>

            {showAdvanced && (
              <div className="space-y-4 border-t pt-4">
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="preserveHostHeader"
                    checked={preserveHostHeader}
                    onChange={(e) => setPreserveHostHeader(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
                  />
                  <label htmlFor="preserveHostHeader" className="ml-2 text-sm text-theme-secondary">
                    Preserve original Host header
                  </label>
                </div>

                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="stripPrefix"
                    checked={stripPrefix}
                    onChange={(e) => setStripPrefix(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
                  />
                  <label htmlFor="stripPrefix" className="ml-2 text-sm text-theme-secondary">
                    Strip /proxy/slug prefix from requests
                  </label>
                </div>

                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="websocketEnabled"
                    checked={websocketEnabled}
                    onChange={(e) => setWebsocketEnabled(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
                  />
                  <label htmlFor="websocketEnabled" className="ml-2 text-sm text-theme-secondary">
                    Enable WebSocket support
                  </label>
                </div>

                <div>
                  <label className="block text-sm font-medium text-theme-secondary">Timeout (seconds)</label>
                  <input
                    type="number"
                    value={timeoutSeconds}
                    onChange={(e) => setTimeoutSeconds(parseInt(e.target.value) || 30)}
                    min={1}
                    max={300}
                    className="input mt-1 w-32"
                  />
                </div>
              </div>
            )}

            <div className="flex justify-end space-x-3 pt-4">
              <button type="button" onClick={onClose} className="btn btn-secondary">
                Cancel
              </button>
              <button type="submit" disabled={submitting} className="btn btn-primary">
                {submitting ? 'Saving...' : (app ? 'Update' : 'Create')}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

interface AssignAccessModalProps {
  app: ProxyApplication
  onClose: () => void
}

function AssignAccessModal({ app, onClose }: AssignAccessModalProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Current assignments
  const [assignedUserIds, setAssignedUserIds] = useState<string[]>([])
  const [assignedGroups, setAssignedGroups] = useState<string[]>([])

  // Available items
  const [allUsers, setAllUsers] = useState<SSOUser[]>([])
  const [allGroups, setAllGroups] = useState<Group[]>([])

  // Selection and manual input fields
  const [selectedUserId, setSelectedUserId] = useState('')
  const [customUserId, setCustomUserId] = useState('')
  const [selectedGroupName, setSelectedGroupName] = useState('')
  const [customGroupName, setCustomGroupName] = useState('')

  useEffect(() => {
    loadData()
  }, [app.id])

  async function loadData() {
    try {
      setLoading(true)
      const [users, groups, appUsers, appGroups] = await Promise.all([
        getUsers(),
        getGroups(),
        getProxyAppUsers(app.id),
        getProxyAppGroups(app.id),
      ])
      setAllUsers(users)
      setAllGroups(groups)
      setAssignedUserIds(appUsers)
      setAssignedGroups(appGroups)
      setError(null)
    } catch {
      setError('Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  async function handleAddUser() {
    const userId = selectedUserId || customUserId.trim()
    if (!userId) return
    try {
      // Find user by email or id to get the actual user id
      const user = allUsers.find(u => u.email === userId || u.id === userId)
      const idToUse = user ? user.id : userId
      await assignProxyAppToUser(app.id, idToUse)
      setAssignedUserIds([...assignedUserIds, idToUse])
      setSelectedUserId('')
      setCustomUserId('')
    } catch {
      setError('Failed to assign user')
    }
  }

  async function handleRemoveUser(userId: string) {
    try {
      await removeProxyAppFromUser(app.id, userId)
      setAssignedUserIds(assignedUserIds.filter(id => id !== userId))
    } catch {
      setError('Failed to remove user')
    }
  }

  async function handleAddGroup() {
    const groupName = selectedGroupName || customGroupName.trim()
    if (!groupName) return
    try {
      await assignProxyAppToGroup(app.id, groupName)
      setAssignedGroups([...assignedGroups, groupName])
      setSelectedGroupName('')
      setCustomGroupName('')
    } catch {
      setError('Failed to assign group')
    }
  }

  async function handleRemoveGroup(groupName: string) {
    try {
      await removeProxyAppFromGroup(app.id, groupName)
      setAssignedGroups(assignedGroups.filter(g => g !== groupName))
    } catch {
      setError('Failed to remove group')
    }
  }

  // Filter out already assigned users
  const availableUsers = allUsers.filter(
    u => !assignedUserIds.includes(u.id) && !assignedUserIds.includes(u.email)
  )

  // Filter out already assigned groups
  const availableGroups = allGroups.filter(
    g => !assignedGroups.includes(g.name)
  )

  // Find user details by ID for display
  function getUserDisplay(userId: string) {
    const user = allUsers.find(u => u.id === userId || u.email === userId)
    if (user) {
      return { name: user.name || user.email, email: user.email, provider: user.provider }
    }
    return { name: userId, email: '', provider: '' }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-theme-primary mb-2">
          Access Control: {app.name}
        </h2>
        <p className="text-sm text-theme-tertiary mb-4">
          Users can access this application if they are directly assigned or belong to an assigned group.
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-theme rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Users section */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2">Assigned Users</h3>

              {/* Add user controls */}
              <div className="mb-3 p-3 bg-theme-tertiary rounded-lg space-y-2">
                {availableUsers.length > 0 && (
                  <div className="flex space-x-2">
                    <select
                      value={selectedUserId}
                      onChange={(e) => {
                        setSelectedUserId(e.target.value)
                        setCustomUserId('')
                      }}
                      className="flex-1 px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                    >
                      <option value="">Select a user...</option>
                      {availableUsers.map((u) => (
                        <option key={u.id} value={u.id}>
                          {u.name || u.email} ({u.email})
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={handleAddUser}
                      disabled={!selectedUserId}
                      className="btn btn-primary text-sm"
                    >
                      Add
                    </button>
                  </div>
                )}
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={customUserId}
                    onChange={(e) => {
                      setCustomUserId(e.target.value)
                      setSelectedUserId('')
                    }}
                    placeholder="Or enter user ID/email manually..."
                    className="flex-1 px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && customUserId.trim()) {
                        e.preventDefault()
                        handleAddUser()
                      }
                    }}
                  />
                  <button
                    onClick={handleAddUser}
                    disabled={!customUserId.trim()}
                    className="btn btn-secondary text-sm"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* Assigned users list */}
              {assignedUserIds.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {assignedUserIds.map((userId) => {
                    const userInfo = getUserDisplay(userId)
                    return (
                      <div key={userId} className="flex items-center justify-between p-3">
                        <div className="flex items-center">
                          <div className="h-8 w-8 rounded-full bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
                            <span className="text-primary-700 dark:text-primary-300 font-medium text-xs">
                              {userInfo.name.charAt(0).toUpperCase()}
                            </span>
                          </div>
                          <div className="ml-3">
                            <p className="text-sm font-medium text-theme-primary">{userInfo.name}</p>
                            {userInfo.email && userInfo.email !== userInfo.name && (
                              <p className="text-xs text-theme-tertiary">{userInfo.email}</p>
                            )}
                          </div>
                          {userInfo.provider && (
                            <span className="ml-2 px-2 py-0.5 text-xs bg-blue-600 text-white rounded">
                              {userInfo.provider}
                            </span>
                          )}
                        </div>
                        <button
                          onClick={() => handleRemoveUser(userId)}
                          className="p-1.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 hover:bg-red-200 dark:hover:bg-red-900/50 hover:text-red-800 dark:hover:text-red-300 transition-colors"
                          title="Remove user"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic">No users assigned</p>
              )}
            </div>

            {/* Groups section */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2">Assigned Groups</h3>

              {/* Add group controls */}
              <div className="mb-3 p-3 bg-theme-tertiary rounded-lg space-y-2">
                {availableGroups.length > 0 && (
                  <div className="flex space-x-2">
                    <select
                      value={selectedGroupName}
                      onChange={(e) => {
                        setSelectedGroupName(e.target.value)
                        setCustomGroupName('')
                      }}
                      className="flex-1 px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                    >
                      <option value="">Select a group...</option>
                      {availableGroups.map((g) => (
                        <option key={g.name} value={g.name}>
                          {g.name} ({g.memberCount} members)
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={handleAddGroup}
                      disabled={!selectedGroupName}
                      className="btn btn-primary text-sm"
                    >
                      Add
                    </button>
                  </div>
                )}
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={customGroupName}
                    onChange={(e) => {
                      setCustomGroupName(e.target.value)
                      setSelectedGroupName('')
                    }}
                    placeholder="Or enter group name manually..."
                    className="flex-1 px-3 py-2 border border-theme rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && customGroupName.trim()) {
                        e.preventDefault()
                        handleAddGroup()
                      }
                    }}
                  />
                  <button
                    onClick={handleAddGroup}
                    disabled={!customGroupName.trim()}
                    className="btn btn-secondary text-sm"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* Assigned groups list */}
              {assignedGroups.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {assignedGroups.map((groupName) => {
                    const group = allGroups.find(g => g.name === groupName)
                    return (
                      <div key={groupName} className="flex items-center justify-between p-3">
                        <div className="flex items-center">
                          <div className="h-8 w-8 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
                            <svg className="h-4 w-4 text-purple-600 dark:text-purple-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                            </svg>
                          </div>
                          <div className="ml-3">
                            <p className="text-sm font-medium text-theme-primary">{groupName}</p>
                            {group && (
                              <p className="text-xs text-theme-tertiary">{group.memberCount} members</p>
                            )}
                          </div>
                        </div>
                        <button
                          onClick={() => handleRemoveGroup(groupName)}
                          className="p-1.5 rounded-full bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 hover:bg-red-200 dark:hover:bg-red-900/50 hover:text-red-800 dark:hover:text-red-300 transition-colors"
                          title="Remove group"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic">No groups assigned</p>
              )}
            </div>
          </div>
        )}

        <div className="mt-6 flex justify-end">
          <button onClick={onClose} className="btn btn-secondary">
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
