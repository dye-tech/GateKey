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
          <h1 className="text-2xl font-bold text-gray-900">Web Access Applications</h1>
          <p className="text-gray-500 mt-1">Manage clientless web applications accessible via reverse proxy</p>
        </div>
        <button onClick={() => setShowAddModal(true)} className="btn btn-primary inline-flex items-center">
          <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Application
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : apps.length === 0 ? (
        <div className="card text-center py-12">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No applications</h3>
          <p className="mt-2 text-gray-500">Add a web application to enable clientless browser access</p>
          <button onClick={() => setShowAddModal(true)} className="mt-4 btn btn-primary inline-flex items-center">
            <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Application
          </button>
        </div>
      ) : (
        <div className="card p-0">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Slug / URL
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Internal URL
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
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
                        <div className="text-sm font-medium text-gray-900">{app.name}</div>
                        <div className="text-sm text-gray-500">{app.description || 'No description'}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      <code className="text-sm bg-gray-100 px-2 py-1 rounded">/proxy/{app.slug}/</code>
                      <button
                        onClick={() => copyProxyUrl(app.slug)}
                        className="text-gray-400 hover:text-gray-600"
                        title="Copy URL"
                      >
                        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                      </button>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm text-gray-900 font-mono">{app.internalUrl}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs font-semibold rounded-full ${
                      app.isActive ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
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
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <h2 className="text-xl font-semibold mb-4">
            {app ? 'Edit Application' : 'Add Application'}
          </h2>

          {error && (
            <div className="mb-4 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Name *</label>
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
              <label className="block text-sm font-medium text-gray-700">Slug *</label>
              <div className="mt-1 flex rounded-md shadow-sm">
                <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-gray-300 bg-gray-50 text-gray-500 text-sm">
                  /proxy/
                </span>
                <input
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                  className="flex-1 min-w-0 block w-full px-3 py-2 rounded-none rounded-r-md border border-gray-300 focus:outline-none focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
                  required
                  pattern="[a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9]"
                />
              </div>
              <p className="mt-1 text-xs text-gray-500">Lowercase letters, numbers, and dashes only</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Description</label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="input mt-1"
                placeholder="Brief description of the application"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Internal URL *</label>
              <input
                type="url"
                value={internalUrl}
                onChange={(e) => setInternalUrl(e.target.value)}
                className="input mt-1"
                placeholder="http://internal-app.local:8080"
                required
              />
              <p className="mt-1 text-xs text-gray-500">The internal URL of the application to proxy to</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Icon URL</label>
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
                className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
              />
              <label htmlFor="isActive" className="ml-2 text-sm text-gray-700">
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
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                  />
                  <label htmlFor="preserveHostHeader" className="ml-2 text-sm text-gray-700">
                    Preserve original Host header
                  </label>
                </div>

                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="stripPrefix"
                    checked={stripPrefix}
                    onChange={(e) => setStripPrefix(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                  />
                  <label htmlFor="stripPrefix" className="ml-2 text-sm text-gray-700">
                    Strip /proxy/slug prefix from requests
                  </label>
                </div>

                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="websocketEnabled"
                    checked={websocketEnabled}
                    onChange={(e) => setWebsocketEnabled(e.target.checked)}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                  />
                  <label htmlFor="websocketEnabled" className="ml-2 text-sm text-gray-700">
                    Enable WebSocket support
                  </label>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700">Timeout (seconds)</label>
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
  const [activeTab, setActiveTab] = useState<'users' | 'groups'>('users')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Current assignments
  const [assignedUserIds, setAssignedUserIds] = useState<string[]>([])
  const [assignedGroups, setAssignedGroups] = useState<string[]>([])

  // Available items
  const [allUsers, setAllUsers] = useState<SSOUser[]>([])
  const [allGroups, setAllGroups] = useState<Group[]>([])

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

  async function handleAssignUser(userId: string) {
    try {
      await assignProxyAppToUser(app.id, userId)
      setAssignedUserIds([...assignedUserIds, userId])
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

  async function handleAssignGroup(groupName: string) {
    try {
      await assignProxyAppToGroup(app.id, groupName)
      setAssignedGroups([...assignedGroups, groupName])
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

  const unassignedUsers = allUsers.filter(u => !assignedUserIds.includes(u.id))
  const unassignedGroups = allGroups.filter(g => !assignedGroups.includes(g.name))

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold">Access Control: {app.name}</h2>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <p className="text-sm text-gray-500 mb-4">
            Users can access this application if they are directly assigned or belong to an assigned group.
          </p>

          {error && (
            <div className="mb-4 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              {error}
            </div>
          )}

          {/* Tabs */}
          <div className="border-b border-gray-200 mb-4">
            <nav className="-mb-px flex space-x-8">
              {[
                { id: 'users', label: 'Users', count: assignedUserIds.length },
                { id: 'groups', label: 'Groups', count: assignedGroups.length },
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as 'users' | 'groups')}
                  className={`py-2 px-1 border-b-2 font-medium text-sm ${
                    activeTab === tab.id
                      ? 'border-primary-500 text-primary-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }`}
                >
                  {tab.label}
                  <span className={`ml-2 px-2 py-0.5 rounded-full text-xs ${
                    activeTab === tab.id ? 'bg-primary-100 text-primary-600' : 'bg-gray-100 text-gray-600'
                  }`}>
                    {tab.count}
                  </span>
                </button>
              ))}
            </nav>
          </div>

          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : (
            <>
              {activeTab === 'users' && (
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">Add User</label>
                    <select
                      className="input"
                      value=""
                      onChange={(e) => e.target.value && handleAssignUser(e.target.value)}
                    >
                      <option value="">Select a user to add...</option>
                      {unassignedUsers.map(u => (
                        <option key={u.id} value={u.id}>{u.email} ({u.name})</option>
                      ))}
                    </select>
                  </div>

                  {assignedUserIds.length > 0 ? (
                    <div className="border rounded-lg divide-y">
                      {assignedUserIds.map(userId => {
                        const user = allUsers.find(u => u.id === userId)
                        return (
                          <div key={userId} className="flex items-center justify-between px-4 py-3">
                            <div>
                              <div className="text-sm font-medium text-gray-900">{user?.email || userId}</div>
                              <div className="text-sm text-gray-500">{user?.name}</div>
                            </div>
                            <button
                              onClick={() => handleRemoveUser(userId)}
                              className="inline-flex items-center px-3 py-1.5 border border-red-300 text-sm font-medium rounded-md text-red-700 bg-white hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                            >
                              <svg className="h-4 w-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                              Remove
                            </button>
                          </div>
                        )
                      })}
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500 text-center py-4">No users directly assigned</p>
                  )}
                </div>
              )}

              {activeTab === 'groups' && (
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">Add Group</label>
                    <select
                      className="input"
                      value=""
                      onChange={(e) => e.target.value && handleAssignGroup(e.target.value)}
                    >
                      <option value="">Select a group to add...</option>
                      {unassignedGroups.map(g => (
                        <option key={g.name} value={g.name}>{g.name} ({g.memberCount} members)</option>
                      ))}
                    </select>
                  </div>

                  {assignedGroups.length > 0 ? (
                    <div className="border rounded-lg divide-y">
                      {assignedGroups.map(groupName => {
                        const group = allGroups.find(g => g.name === groupName)
                        return (
                          <div key={groupName} className="flex items-center justify-between px-4 py-3">
                            <div>
                              <div className="text-sm font-medium text-gray-900">{groupName}</div>
                              <div className="text-sm text-gray-500">{group?.memberCount || 0} members</div>
                            </div>
                            <button
                              onClick={() => handleRemoveGroup(groupName)}
                              className="inline-flex items-center px-3 py-1.5 border border-red-300 text-sm font-medium rounded-md text-red-700 bg-white hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                            >
                              <svg className="h-4 w-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                              Remove
                            </button>
                          </div>
                        )
                      })}
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500 text-center py-4">No groups assigned</p>
                  )}
                </div>
              )}
            </>
          )}

          <div className="flex justify-end mt-6">
            <button onClick={onClose} className="btn btn-secondary">
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
