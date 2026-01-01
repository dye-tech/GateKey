import { useState, useEffect } from 'react'
import {
  getUsers,
  getLocalUsers,
  createLocalUser,
  deleteLocalUser,
  getGroups,
  getGroupMembers,
  getGroupAccessRules,
  getUserAccessRules,
  getUserGateways,
  assignUserGateway,
  removeUserGateway,
  getAdminGateways,
  getAdminUserConfigs,
  getAdminUserMeshConfigs,
  adminRevokeConfig,
  adminRevokeMeshConfig,
  adminRevokeUserConfigs,
  adminRevokeUserMeshConfigs,
  getAdminUserAPIKeys,
  revokeAdminAPIKey,
  adminDeleteUserAPIKeys,
  SSOUser,
  LocalUser,
  Group,
  GroupMember,
  AccessRule,
  UserGateway,
  AdminGateway,
  VPNConfig,
  MeshVPNConfig,
  APIKey,
} from '../api/client'
import ActionDropdown, { ActionItem } from '../components/ActionDropdown'

type TabType = 'sso' | 'local' | 'groups'

export default function AdminUsers() {
  const [activeTab, setActiveTab] = useState<TabType>('sso')
  const [ssoUsers, setSSOUsers] = useState<SSOUser[]>([])
  const [localUsers, setLocalUsers] = useState<LocalUser[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedUser, setSelectedUser] = useState<SSOUser | null>(null)
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null)
  const [showCreateUserModal, setShowCreateUserModal] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  async function loadData() {
    try {
      setLoading(true)
      const [sso, local, grps] = await Promise.all([
        getUsers(),
        getLocalUsers(),
        getGroups(),
      ])
      setSSOUsers(sso)
      setLocalUsers(local)
      setGroups(grps)
      setError(null)
    } catch (err) {
      setError('Failed to load user data')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">Users & Groups</h1>
        <p className="text-gray-500 mt-1">
          Manage SSO users, local admins, and groups for access control.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('sso')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'sso'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            SSO Users
            <span className="ml-2 px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-600">
              {ssoUsers.length}
            </span>
          </button>
          <button
            onClick={() => setActiveTab('local')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'local'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Local Users
            <span className="ml-2 px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-600">
              {localUsers.length}
            </span>
          </button>
          <button
            onClick={() => setActiveTab('groups')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'groups'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Groups
            <span className="ml-2 px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-600">
              {groups.length}
            </span>
          </button>
        </nav>
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}

      {/* Loading state */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <>
          {/* SSO Users Tab */}
          {activeTab === 'sso' && (
            <SSOUsersTab
              users={ssoUsers}
              onViewUser={setSelectedUser}
            />
          )}

          {/* Local Users Tab */}
          {activeTab === 'local' && (
            <LocalUsersTab
              users={localUsers}
              onCreateUser={() => setShowCreateUserModal(true)}
              onDeleteUser={async (id) => {
                if (!confirm('Are you sure you want to delete this user?')) return
                try {
                  await deleteLocalUser(id)
                  loadData()
                } catch (err) {
                  setError('Failed to delete user')
                }
              }}
            />
          )}

          {/* Groups Tab */}
          {activeTab === 'groups' && (
            <GroupsTab
              groups={groups}
              onViewGroup={setSelectedGroup}
            />
          )}
        </>
      )}

      {/* User Details Modal */}
      {selectedUser && (
        <UserDetailsModal
          user={selectedUser}
          onClose={() => setSelectedUser(null)}
        />
      )}

      {/* Group Details Modal */}
      {selectedGroup && (
        <GroupDetailsModal
          groupName={selectedGroup}
          onClose={() => setSelectedGroup(null)}
        />
      )}

      {/* Create User Modal */}
      {showCreateUserModal && (
        <CreateUserModal
          onClose={() => setShowCreateUserModal(false)}
          onSuccess={() => {
            setShowCreateUserModal(false)
            loadData()
          }}
        />
      )}
    </div>
  )
}

interface SSOUsersTabProps {
  users: SSOUser[]
  onViewUser: (user: SSOUser) => void
}

function SSOUsersTab({ users, onViewUser }: SSOUsersTabProps) {
  if (users.length === 0) {
    return (
      <div className="card text-center py-12">
        <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
        </svg>
        <h3 className="mt-4 text-lg font-medium text-gray-900">No SSO users yet</h3>
        <p className="mt-2 text-gray-500">
          SSO users will appear here after they log in via OIDC or SAML.
        </p>
      </div>
    )
  }

  return (
    <div className="card p-0">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              User
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Provider
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Groups
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Last Login
            </th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {users.map((user) => (
            <tr key={user.id}>
              <td className="px-6 py-4 whitespace-nowrap">
                <div className="flex items-center">
                  <div className="h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center">
                    <span className="text-primary-700 font-medium text-sm">
                      {user.name ? user.name.charAt(0).toUpperCase() : user.email.charAt(0).toUpperCase()}
                    </span>
                  </div>
                  <div className="ml-4">
                    <div className="text-sm font-medium text-gray-900">{user.name || 'Unnamed'}</div>
                    <div className="text-sm text-gray-500">{user.email}</div>
                  </div>
                </div>
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className="px-2 py-1 inline-flex text-xs leading-5 font-medium rounded-full bg-blue-100 text-blue-800">
                  {user.provider}
                </span>
              </td>
              <td className="px-6 py-4">
                <div className="flex flex-wrap gap-1">
                  {user.groups.length > 0 ? (
                    user.groups.slice(0, 3).map((group) => (
                      <span key={group} className="px-2 py-0.5 text-xs bg-gray-100 text-gray-700 rounded">
                        {group}
                      </span>
                    ))
                  ) : (
                    <span className="text-xs text-gray-400 italic">No groups</span>
                  )}
                  {user.groups.length > 3 && (
                    <span className="px-2 py-0.5 text-xs bg-gray-100 text-gray-500 rounded">
                      +{user.groups.length - 3} more
                    </span>
                  )}
                </div>
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                  user.isActive
                    ? 'bg-green-100 text-green-800'
                    : 'bg-gray-100 text-gray-800'
                }`}>
                  {user.isActive ? 'Active' : 'Inactive'}
                </span>
                {user.isAdmin && (
                  <span className="ml-1 px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full bg-orange-100 text-orange-800">
                    Admin
                  </span>
                )}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {user.lastLoginAt
                  ? new Date(user.lastLoginAt).toLocaleString()
                  : 'Never'}
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <ActionDropdown
                  actions={[
                    { label: 'View Details', icon: 'view', onClick: () => onViewUser(user), color: 'primary' },
                  ] as ActionItem[]}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

interface LocalUsersTabProps {
  users: LocalUser[]
  onCreateUser: () => void
  onDeleteUser: (id: string) => void
}

function LocalUsersTab({ users, onCreateUser, onDeleteUser }: LocalUsersTabProps) {
  return (
    <div className="space-y-4">
      {/* Add User Button */}
      <div className="flex justify-end">
        <button onClick={onCreateUser} className="btn btn-primary">
          <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Local User
        </button>
      </div>

      {users.length === 0 ? (
        <div className="card text-center py-12">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No local users</h3>
          <p className="mt-2 text-gray-500">
            Click "Add Local User" to create a local account.
          </p>
        </div>
      ) : (
        <div className="card p-0">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Username
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Email
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Role
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Last Login
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {users.map((user) => (
                <tr key={user.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div className="h-10 w-10 rounded-full bg-orange-100 flex items-center justify-center">
                        <span className="text-orange-700 font-medium text-sm">
                          {user.username.charAt(0).toUpperCase()}
                        </span>
                      </div>
                      <div className="ml-4">
                        <div className="text-sm font-medium text-gray-900">{user.username}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {user.email}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      user.isAdmin
                        ? 'bg-orange-100 text-orange-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {user.isAdmin ? 'Admin' : 'User'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {user.lastLoginAt
                      ? new Date(user.lastLoginAt).toLocaleString()
                      : 'Never'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(user.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Delete', icon: 'delete', onClick: () => onDeleteUser(user.id), color: 'red', disabled: user.username === 'admin' },
                      ] as ActionItem[]}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

interface GroupsTabProps {
  groups: Group[]
  onViewGroup: (name: string) => void
}

function GroupsTab({ groups, onViewGroup }: GroupsTabProps) {
  if (groups.length === 0) {
    return (
      <div className="card text-center py-12">
        <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
        </svg>
        <h3 className="mt-4 text-lg font-medium text-gray-900">No groups found</h3>
        <p className="mt-2 text-gray-500">
          Groups are synced from your identity provider when users log in, or created when assigning access rules.
        </p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {groups.map((group) => (
        <div
          key={group.name}
          className="card hover:shadow-md transition-shadow cursor-pointer"
          onClick={() => onViewGroup(group.name)}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <div className="h-10 w-10 rounded-full bg-purple-100 flex items-center justify-center">
                <svg className="h-5 w-5 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
              </div>
              <div className="ml-3">
                <div className="text-sm font-medium text-gray-900">{group.name}</div>
                <div className="text-xs text-gray-500">{group.memberCount} members</div>
              </div>
            </div>
            <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </div>
        </div>
      ))}
    </div>
  )
}

interface UserDetailsModalProps {
  user: SSOUser
  onClose: () => void
}

type UserModalTab = 'info' | 'gateways' | 'configs' | 'apikeys'

function UserDetailsModal({ user, onClose }: UserDetailsModalProps) {
  const [accessRules, setAccessRules] = useState<AccessRule[]>([])
  const [userGateways, setUserGateways] = useState<UserGateway[]>([])
  const [allGateways, setAllGateways] = useState<AdminGateway[]>([])
  const [gatewayConfigs, setGatewayConfigs] = useState<VPNConfig[]>([])
  const [meshConfigs, setMeshConfigs] = useState<MeshVPNConfig[]>([])
  const [apiKeys, setAPIKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<UserModalTab>('info')
  const [selectedGateway, setSelectedGateway] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [revoking, setRevoking] = useState<string | null>(null)

  useEffect(() => {
    loadData()
  }, [user.id])

  async function loadData() {
    setLoading(true)
    try {
      const [rules, gateways, allGw, gwConfigs, mConfigs, keys] = await Promise.all([
        getUserAccessRules(user.id),
        getUserGateways(user.id),
        getAdminGateways(),
        getAdminUserConfigs(user.id),
        getAdminUserMeshConfigs(user.id),
        getAdminUserAPIKeys(user.id),
      ])
      setAccessRules(rules)
      setUserGateways(gateways)
      setAllGateways(allGw)
      setGatewayConfigs(gwConfigs)
      setMeshConfigs(mConfigs)
      setAPIKeys(keys)
    } catch {
      setError('Failed to load user data')
    } finally {
      setLoading(false)
    }
  }

  const assignedGatewayIds = new Set(userGateways.map(g => g.id))
  const availableGateways = allGateways.filter(g => !assignedGatewayIds.has(g.id))

  // Config helpers
  const isExpired = (dateStr: string) => new Date(dateStr) < new Date()
  const activeGwConfigs = gatewayConfigs.filter(c => !c.isRevoked && !isExpired(c.expiresAt))
  const activeMeshConfigs = meshConfigs.filter(c => !c.isRevoked && !isExpired(c.expiresAt))
  const activeAPIKeys = apiKeys.filter(k => !k.isRevoked && (!k.expiresAt || !isExpired(k.expiresAt)))

  async function handleAssignGateway() {
    if (!selectedGateway) return
    try {
      await assignUserGateway(user.id, selectedGateway)
      setSelectedGateway('')
      await loadData()
    } catch {
      setError('Failed to assign gateway')
    }
  }

  async function handleRemoveGateway(gatewayId: string) {
    try {
      await removeUserGateway(user.id, gatewayId)
      await loadData()
    } catch {
      setError('Failed to remove gateway')
    }
  }

  async function handleRevokeConfig(configId: string, type: 'gateway' | 'mesh') {
    if (!confirm('Are you sure you want to revoke this config? This will immediately disconnect any active VPN session.')) return
    setRevoking(configId)
    try {
      if (type === 'gateway') {
        await adminRevokeConfig(configId, 'revoked by admin')
      } else {
        await adminRevokeMeshConfig(configId, 'revoked by admin')
      }
      await loadData()
    } catch {
      setError('Failed to revoke config')
    } finally {
      setRevoking(null)
    }
  }

  async function handleRevokeAllConfigs() {
    if (!confirm('Are you sure you want to revoke ALL configs for this user? This will disconnect all active VPN sessions.')) return
    try {
      await Promise.all([
        adminRevokeUserConfigs(user.id, 'all configs revoked by admin'),
        adminRevokeUserMeshConfigs(user.id, 'all configs revoked by admin'),
      ])
      await loadData()
    } catch {
      setError('Failed to revoke configs')
    }
  }

  async function handleRevokeAPIKey(keyId: string) {
    if (!confirm('Are you sure you want to revoke this API key?')) return
    setRevoking(keyId)
    try {
      await revokeAdminAPIKey(keyId)
      await loadData()
    } catch {
      setError('Failed to revoke API key')
    } finally {
      setRevoking(null)
    }
  }

  async function handleDeleteAllAPIKeys() {
    if (!confirm('Are you sure you want to PERMANENTLY DELETE all API keys for this user? This cannot be undone.')) return
    try {
      await adminDeleteUserAPIKeys(user.id)
      await loadData()
    } catch {
      setError('Failed to delete API keys')
    }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full mx-4 max-h-[90vh] overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <div className="h-12 w-12 rounded-full bg-primary-100 flex items-center justify-center">
                <span className="text-primary-700 font-medium text-lg">
                  {user.name ? user.name.charAt(0).toUpperCase() : user.email.charAt(0).toUpperCase()}
                </span>
              </div>
              <div className="ml-4">
                <h2 className="text-xl font-semibold text-gray-900">{user.name || 'Unnamed User'}</h2>
                <p className="text-sm text-gray-500">{user.email}</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
            >
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b border-gray-200 px-6">
          <nav className="-mb-px flex space-x-6 overflow-x-auto">
            <button
              onClick={() => setActiveTab('info')}
              className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
                activeTab === 'info'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Info
            </button>
            <button
              onClick={() => setActiveTab('gateways')}
              className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
                activeTab === 'gateways'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Gateways ({userGateways.length})
            </button>
            <button
              onClick={() => setActiveTab('configs')}
              className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
                activeTab === 'configs'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              VPN Configs ({activeGwConfigs.length + activeMeshConfigs.length})
            </button>
            <button
              onClick={() => setActiveTab('apikeys')}
              className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
                activeTab === 'apikeys'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              API Keys ({activeAPIKeys.length})
            </button>
          </nav>
        </div>

        {error && (
          <div className="mx-6 mt-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
            <button onClick={() => setError(null)} className="ml-2 text-red-500 hover:text-red-700">&times;</button>
          </div>
        )}

        <div className="p-6 overflow-y-auto max-h-[50vh]">
          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : activeTab === 'info' ? (
            <>
              {/* User Info */}
              <div className="grid grid-cols-2 gap-4 mb-6">
                <div>
                  <label className="text-xs text-gray-500 uppercase">Provider</label>
                  <p className="text-sm font-medium text-gray-900">{user.provider}</p>
                </div>
                <div>
                  <label className="text-xs text-gray-500 uppercase">External ID</label>
                  <p className="text-sm font-medium text-gray-900 font-mono text-xs break-all">{user.externalId}</p>
                </div>
                <div>
                  <label className="text-xs text-gray-500 uppercase">Status</label>
                  <p className="text-sm">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                      user.isActive ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                    }`}>
                      {user.isActive ? 'Active' : 'Inactive'}
                    </span>
                    {user.isAdmin && (
                      <span className="ml-1 px-2 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
                        Admin
                      </span>
                    )}
                  </p>
                </div>
                <div>
                  <label className="text-xs text-gray-500 uppercase">Last Login</label>
                  <p className="text-sm text-gray-900">
                    {user.lastLoginAt ? new Date(user.lastLoginAt).toLocaleString() : 'Never'}
                  </p>
                </div>
              </div>

              {/* Groups */}
              <div className="mb-6">
                <label className="text-xs text-gray-500 uppercase mb-2 block">Groups ({user.groups.length})</label>
                <div className="flex flex-wrap gap-2">
                  {user.groups.length > 0 ? (
                    user.groups.map((group) => (
                      <span key={group} className="px-3 py-1 bg-purple-100 text-purple-800 rounded-full text-sm">
                        {group}
                      </span>
                    ))
                  ) : (
                    <span className="text-sm text-gray-400 italic">No groups assigned</span>
                  )}
                </div>
              </div>

              {/* Access Rules */}
              <div>
                <label className="text-xs text-gray-500 uppercase mb-2 block">
                  Access Rules ({accessRules.length})
                </label>
                {accessRules.length > 0 ? (
                  <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                    {accessRules.map((rule) => (
                      <div key={rule.id} className="p-3 flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium text-gray-900">{rule.name}</p>
                          <p className="text-xs text-gray-500">
                            <span className="font-mono bg-gray-100 px-1 rounded">{rule.value}</span>
                            <span className="ml-2">({rule.ruleType})</span>
                          </p>
                        </div>
                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                          rule.isActive ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                          {rule.isActive ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-gray-400 italic">No access rules assigned (directly or via groups)</p>
                )}
              </div>
            </>
          ) : activeTab === 'gateways' ? (
            /* Gateways Tab */
            <div className="space-y-4">
              {/* Add Gateway */}
              <div className="flex space-x-2">
                <select
                  value={selectedGateway}
                  onChange={(e) => setSelectedGateway(e.target.value)}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                >
                  <option value="">Select a gateway to assign...</option>
                  {availableGateways.map(g => (
                    <option key={g.id} value={g.id}>{g.name} ({g.hostname || g.publicIp})</option>
                  ))}
                </select>
                <button
                  onClick={handleAssignGateway}
                  disabled={!selectedGateway}
                  className="btn btn-primary"
                >
                  Assign
                </button>
              </div>

              {/* Assigned Gateways */}
              <div className="space-y-2">
                {userGateways.length === 0 ? (
                  <p className="text-gray-500 text-sm py-4 text-center">No gateways assigned to this user</p>
                ) : (
                  userGateways.map(gateway => (
                    <div key={gateway.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                      <div>
                        <div className="font-medium text-sm flex items-center">
                          {gateway.name}
                          <span className={`ml-2 px-1.5 py-0.5 text-xs rounded-full ${
                            gateway.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
                          }`}>
                            {gateway.isActive ? 'Online' : 'Offline'}
                          </span>
                        </div>
                        <div className="text-xs text-gray-500">
                          {gateway.hostname || gateway.publicIp} - {gateway.vpnProtocol.toUpperCase()}:{gateway.vpnPort}
                        </div>
                      </div>
                      <button
                        onClick={() => handleRemoveGateway(gateway.id)}
                        className="text-red-600 hover:text-red-800 text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          ) : activeTab === 'configs' ? (
            /* VPN Configs Tab */
            <div className="space-y-4">
              {/* Revoke All Button */}
              {(activeGwConfigs.length > 0 || activeMeshConfigs.length > 0) && (
                <div className="flex justify-end">
                  <button
                    onClick={handleRevokeAllConfigs}
                    className="btn bg-red-100 text-red-700 hover:bg-red-200 text-sm"
                  >
                    Revoke All Active Configs
                  </button>
                </div>
              )}

              {/* Gateway Configs */}
              <div>
                <h4 className="text-sm font-medium text-gray-700 mb-2">Gateway Configs ({gatewayConfigs.length})</h4>
                {gatewayConfigs.length === 0 ? (
                  <p className="text-gray-400 text-sm italic">No gateway configs</p>
                ) : (
                  <div className="border rounded-lg divide-y">
                    {gatewayConfigs.map(cfg => (
                      <div key={cfg.id} className={`p-3 flex items-center justify-between ${cfg.isRevoked || isExpired(cfg.expiresAt) ? 'bg-gray-50' : ''}`}>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">{cfg.gatewayName}</span>
                            <span className="px-1.5 py-0.5 text-xs rounded bg-blue-100 text-blue-700">Gateway</span>
                            {cfg.isRevoked ? (
                              <span className="px-1.5 py-0.5 text-xs rounded bg-red-100 text-red-700">Revoked</span>
                            ) : isExpired(cfg.expiresAt) ? (
                              <span className="px-1.5 py-0.5 text-xs rounded bg-gray-100 text-gray-600">Expired</span>
                            ) : (
                              <span className="px-1.5 py-0.5 text-xs rounded bg-green-100 text-green-700">Active</span>
                            )}
                          </div>
                          <div className="text-xs text-gray-500 mt-1">
                            {cfg.fileName} &bull; Expires: {new Date(cfg.expiresAt).toLocaleString()}
                          </div>
                        </div>
                        {!cfg.isRevoked && !isExpired(cfg.expiresAt) && (
                          <button
                            onClick={() => handleRevokeConfig(cfg.id, 'gateway')}
                            disabled={revoking === cfg.id}
                            className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
                          >
                            {revoking === cfg.id ? 'Revoking...' : 'Revoke'}
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Mesh Configs */}
              <div>
                <h4 className="text-sm font-medium text-gray-700 mb-2">Mesh Hub Configs ({meshConfigs.length})</h4>
                {meshConfigs.length === 0 ? (
                  <p className="text-gray-400 text-sm italic">No mesh configs</p>
                ) : (
                  <div className="border rounded-lg divide-y">
                    {meshConfigs.map(cfg => (
                      <div key={cfg.id} className={`p-3 flex items-center justify-between ${cfg.isRevoked || isExpired(cfg.expiresAt) ? 'bg-gray-50' : ''}`}>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">{cfg.hubName}</span>
                            <span className="px-1.5 py-0.5 text-xs rounded bg-purple-100 text-purple-700">Mesh Hub</span>
                            {cfg.isRevoked ? (
                              <span className="px-1.5 py-0.5 text-xs rounded bg-red-100 text-red-700">Revoked</span>
                            ) : isExpired(cfg.expiresAt) ? (
                              <span className="px-1.5 py-0.5 text-xs rounded bg-gray-100 text-gray-600">Expired</span>
                            ) : (
                              <span className="px-1.5 py-0.5 text-xs rounded bg-green-100 text-green-700">Active</span>
                            )}
                          </div>
                          <div className="text-xs text-gray-500 mt-1">
                            {cfg.fileName} &bull; Expires: {new Date(cfg.expiresAt).toLocaleString()}
                          </div>
                        </div>
                        {!cfg.isRevoked && !isExpired(cfg.expiresAt) && (
                          <button
                            onClick={() => handleRevokeConfig(cfg.id, 'mesh')}
                            disabled={revoking === cfg.id}
                            className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
                          >
                            {revoking === cfg.id ? 'Revoking...' : 'Revoke'}
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            /* API Keys Tab */
            <div className="space-y-4">
              {/* Delete All Button */}
              {apiKeys.length > 0 && (
                <div className="flex justify-end">
                  <button
                    onClick={handleDeleteAllAPIKeys}
                    className="btn bg-red-100 text-red-700 hover:bg-red-200 text-sm"
                  >
                    Delete All API Keys
                  </button>
                </div>
              )}

              {apiKeys.length === 0 ? (
                <p className="text-gray-400 text-sm italic text-center py-4">No API keys</p>
              ) : (
                <div className="border rounded-lg divide-y">
                  {apiKeys.map(key => (
                    <div key={key.id} className={`p-3 flex items-center justify-between ${key.isRevoked ? 'bg-gray-50' : ''}`}>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{key.name}</span>
                          <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{key.keyPrefix}...</code>
                          {key.isRevoked ? (
                            <span className="px-1.5 py-0.5 text-xs rounded bg-red-100 text-red-700">Revoked</span>
                          ) : key.expiresAt && isExpired(key.expiresAt) ? (
                            <span className="px-1.5 py-0.5 text-xs rounded bg-gray-100 text-gray-600">Expired</span>
                          ) : (
                            <span className="px-1.5 py-0.5 text-xs rounded bg-green-100 text-green-700">Active</span>
                          )}
                          {key.isAdminProvisioned && (
                            <span className="px-1.5 py-0.5 text-xs rounded bg-orange-100 text-orange-700">Admin</span>
                          )}
                        </div>
                        <div className="text-xs text-gray-500 mt-1">
                          Created: {new Date(key.createdAt).toLocaleString()}
                          {key.lastUsedAt && ` • Last used: ${new Date(key.lastUsedAt).toLocaleString()}`}
                          {key.expiresAt && ` • Expires: ${new Date(key.expiresAt).toLocaleString()}`}
                        </div>
                        {key.description && (
                          <div className="text-xs text-gray-400 mt-1">{key.description}</div>
                        )}
                      </div>
                      {!key.isRevoked && (
                        <button
                          onClick={() => handleRevokeAPIKey(key.id)}
                          disabled={revoking === key.id}
                          className="text-red-600 hover:text-red-800 text-sm disabled:opacity-50"
                        >
                          {revoking === key.id ? 'Revoking...' : 'Revoke'}
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="p-4 border-t border-gray-200 flex justify-end">
          <button onClick={onClose} className="btn btn-secondary">
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

interface GroupDetailsModalProps {
  groupName: string
  onClose: () => void
}

function GroupDetailsModal({ groupName, onClose }: GroupDetailsModalProps) {
  const [members, setMembers] = useState<GroupMember[]>([])
  const [accessRules, setAccessRules] = useState<AccessRule[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'members' | 'rules'>('members')

  useEffect(() => {
    Promise.all([
      getGroupMembers(groupName),
      getGroupAccessRules(groupName),
    ])
      .then(([m, r]) => {
        setMembers(m)
        setAccessRules(r)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [groupName])

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <div className="h-12 w-12 rounded-full bg-purple-100 flex items-center justify-center">
                <svg className="h-6 w-6 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
              </div>
              <div className="ml-4">
                <h2 className="text-xl font-semibold text-gray-900">{groupName}</h2>
                <p className="text-sm text-gray-500">{members.length} members</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
            >
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b border-gray-200 px-6">
          <nav className="-mb-px flex space-x-8">
            <button
              onClick={() => setActiveTab('members')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'members'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Members ({members.length})
            </button>
            <button
              onClick={() => setActiveTab('rules')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'rules'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Access Rules ({accessRules.length})
            </button>
          </nav>
        </div>

        <div className="p-6 overflow-y-auto max-h-[50vh]">
          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : activeTab === 'members' ? (
            members.length > 0 ? (
              <div className="space-y-2">
                {members.map((member) => (
                  <div key={member.id} className="flex items-center p-3 bg-gray-50 rounded-lg">
                    <div className="h-8 w-8 rounded-full bg-primary-100 flex items-center justify-center">
                      <span className="text-primary-700 font-medium text-xs">
                        {member.name ? member.name.charAt(0).toUpperCase() : member.email.charAt(0).toUpperCase()}
                      </span>
                    </div>
                    <div className="ml-3">
                      <p className="text-sm font-medium text-gray-900">{member.name || member.email}</p>
                      {member.name && <p className="text-xs text-gray-500">{member.email}</p>}
                    </div>
                    <span className="ml-auto px-2 py-0.5 text-xs bg-blue-100 text-blue-800 rounded">
                      {member.provider}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-center text-gray-500 py-8">No members in this group</p>
            )
          ) : accessRules.length > 0 ? (
            <div className="space-y-2">
              {accessRules.map((rule) => (
                <div key={rule.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{rule.name}</p>
                    <p className="text-xs text-gray-500">
                      <span className="font-mono bg-white px-1 rounded border">{rule.value}</span>
                      <span className="ml-2">({rule.ruleType})</span>
                    </p>
                  </div>
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    rule.isActive ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                  }`}>
                    {rule.isActive ? 'Active' : 'Inactive'}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-center text-gray-500 py-8">No access rules assigned to this group</p>
          )}
        </div>

        <div className="p-4 border-t border-gray-200 flex justify-end">
          <button onClick={onClose} className="btn btn-secondary">
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

interface CreateUserModalProps {
  onClose: () => void
  onSuccess: () => void
}

function CreateUserModal({ onClose, onSuccess }: CreateUserModalProps) {
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isAdmin, setIsAdmin] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    if (password !== confirmPassword) {
      setError('Passwords do not match')
      return
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }

    setLoading(true)
    setError(null)

    try {
      await createLocalUser({
        username,
        email,
        password,
        is_admin: isAdmin,
      })
      onSuccess()
    } catch (err) {
      setError('Failed to create user. Username may already exist.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <form onSubmit={handleSubmit}>
          <div className="p-6 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h2 className="text-xl font-semibold text-gray-900">Create Local User</h2>
              <button
                type="button"
                onClick={onClose}
                className="text-gray-400 hover:text-gray-600"
              >
                <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          <div className="p-6 space-y-4">
            {error && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Username
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="input"
                placeholder="Enter username"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Email
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="input"
                placeholder="Enter email address"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Password
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="input"
                placeholder="Enter password"
                required
                minLength={8}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Confirm Password
              </label>
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="input"
                placeholder="Confirm password"
                required
              />
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="isAdmin"
                checked={isAdmin}
                onChange={(e) => setIsAdmin(e.target.checked)}
                className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
              />
              <label htmlFor="isAdmin" className="ml-2 block text-sm text-gray-700">
                Administrator
                <span className="block text-xs text-gray-500">
                  Admins have full access to manage gateways, users, and settings
                </span>
              </label>
            </div>
          </div>

          <div className="p-4 border-t border-gray-200 flex justify-end gap-3">
            <button type="button" onClick={onClose} className="btn btn-secondary">
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="btn btn-primary"
            >
              {loading ? 'Creating...' : 'Create User'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
