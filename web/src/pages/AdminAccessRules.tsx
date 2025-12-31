import { useState, useEffect } from 'react'
import {
  getAccessRules,
  getAccessRule,
  createAccessRule,
  deleteAccessRule,
  updateAccessRule,
  getNetworks,
  getUsers,
  getGroups,
  assignRuleToUser,
  removeRuleFromUser,
  assignRuleToGroup,
  removeRuleFromGroup,
  AccessRule,
  AccessRuleType,
  Network,
  SSOUser,
  Group,
} from '../api/client'
import ActionDropdown, { ActionItem } from '../components/ActionDropdown'

const RULE_TYPE_LABELS: Record<AccessRuleType, string> = {
  ip: 'IP Address',
  cidr: 'CIDR Range',
  hostname: 'Hostname',
  hostname_wildcard: 'Hostname Wildcard',
}

const RULE_TYPE_EXAMPLES: Record<AccessRuleType, string> = {
  ip: '192.168.1.100',
  cidr: '10.0.0.0/24',
  hostname: 'api.example.com',
  hostname_wildcard: '*.example.com',
}

export default function AdminAccessRules() {
  const [rules, setRules] = useState<AccessRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editingRule, setEditingRule] = useState<AccessRule | null>(null)
  const [showAssignModal, setShowAssignModal] = useState(false)
  const [selectedRule, setSelectedRule] = useState<AccessRule | null>(null)

  useEffect(() => {
    loadRules()
  }, [])

  async function loadRules() {
    try {
      setLoading(true)
      const data = await getAccessRules()
      setRules(data)
      setError(null)
    } catch (err) {
      setError('Failed to load access rules')
    } finally {
      setLoading(false)
    }
  }

  async function handleDelete(rule: AccessRule) {
    if (!confirm(`Are you sure you want to delete access rule "${rule.name}"?`)) {
      return
    }

    try {
      await deleteAccessRule(rule.id)
      await loadRules()
    } catch (err) {
      setError('Failed to delete access rule')
    }
  }

  async function handleManageAssignments(rule: AccessRule) {
    try {
      const fullRule = await getAccessRule(rule.id)
      setSelectedRule(fullRule)
      setShowAssignModal(true)
    } catch (err) {
      setError('Failed to load rule details')
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Access Rules</h1>
            <p className="text-gray-500 mt-1">
              Define IP addresses, CIDR ranges, and hostnames that users/groups can access.
            </p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="btn btn-primary"
          >
            <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Rule
          </button>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}

      {/* Rules table */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : rules.length > 0 ? (
        <div className="card p-0">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Rule
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Value
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Port/Protocol
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
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div>
                        <div className="text-sm font-medium text-gray-900">{rule.name}</div>
                        {rule.description && (
                          <div className="text-sm text-gray-500">{rule.description}</div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="px-2 py-1 inline-flex text-xs leading-5 font-medium rounded-full bg-blue-100 text-blue-800">
                      {RULE_TYPE_LABELS[rule.ruleType]}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <code className="px-2 py-1 bg-gray-100 rounded text-sm font-mono">
                      {rule.value}
                    </code>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {rule.portRange || '*'} / {rule.protocol?.toUpperCase() || '*'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      rule.isActive
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {rule.isActive ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Assign', icon: 'assign', onClick: () => handleManageAssignments(rule), color: 'purple' },
                        { label: 'Edit', icon: 'edit', onClick: () => setEditingRule(rule), color: 'gray' },
                        { label: 'Delete', icon: 'delete', onClick: () => handleDelete(rule), color: 'red' },
                      ] as ActionItem[]}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="card text-center py-12">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No access rules defined</h3>
          <p className="mt-2 text-gray-500">
            Get started by adding an IP, CIDR, or hostname whitelist rule.
          </p>
          <button
            onClick={() => setShowAddModal(true)}
            className="mt-4 btn btn-primary"
          >
            Add Rule
          </button>
        </div>
      )}

      {/* Add Rule Modal */}
      {showAddModal && (
        <RuleModal
          onClose={() => setShowAddModal(false)}
          onSuccess={() => {
            setShowAddModal(false)
            loadRules()
          }}
        />
      )}

      {/* Edit Rule Modal */}
      {editingRule && (
        <RuleModal
          rule={editingRule}
          onClose={() => setEditingRule(null)}
          onSuccess={() => {
            setEditingRule(null)
            loadRules()
          }}
        />
      )}

      {/* Assignment Modal */}
      {showAssignModal && selectedRule && (
        <AssignmentModal
          rule={selectedRule}
          onClose={() => {
            setShowAssignModal(false)
            setSelectedRule(null)
          }}
          onUpdate={async () => {
            const updated = await getAccessRule(selectedRule.id)
            setSelectedRule(updated)
          }}
        />
      )}
    </div>
  )
}

interface RuleModalProps {
  rule?: AccessRule
  onClose: () => void
  onSuccess: () => void
}

function RuleModal({ rule, onClose, onSuccess }: RuleModalProps) {
  const [name, setName] = useState(rule?.name || '')
  const [description, setDescription] = useState(rule?.description || '')
  const [ruleType, setRuleType] = useState<AccessRuleType>(rule?.ruleType || 'ip')
  const [value, setValue] = useState(rule?.value || '')
  const [portRange, setPortRange] = useState(rule?.portRange || '')
  const [protocol, setProtocol] = useState(rule?.protocol || '')
  const [networkId, setNetworkId] = useState(rule?.networkId || '')
  const [isActive, setIsActive] = useState(rule?.isActive ?? true)
  const [networks, setNetworks] = useState<Network[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getNetworks().then(setNetworks).catch(() => {})
  }, [])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      const req = {
        name,
        description: description || undefined,
        rule_type: ruleType,
        value,
        port_range: portRange || undefined,
        protocol: protocol || undefined,
        network_id: networkId || undefined,
        is_active: isActive,
      }
      if (rule) {
        await updateAccessRule(rule.id, req)
      } else {
        await createAccessRule(req)
      }
      onSuccess()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || `Failed to ${rule ? 'update' : 'create'} access rule`)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          {rule ? 'Edit Access Rule' : 'Add New Access Rule'}
        </h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Rule Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="internal-api-access"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Rule Type *
            </label>
            <select
              value={ruleType}
              onChange={(e) => {
                setRuleType(e.target.value as AccessRuleType)
                setValue('')
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="ip">IP Address</option>
              <option value="cidr">CIDR Range</option>
              <option value="hostname">Hostname</option>
              <option value="hostname_wildcard">Hostname Wildcard</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Value *
            </label>
            <input
              type="text"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={RULE_TYPE_EXAMPLES[ruleType]}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 font-mono"
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Example: {RULE_TYPE_EXAMPLES[ruleType]}
            </p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Port Range
              </label>
              <input
                type="text"
                value={portRange}
                onChange={(e) => setPortRange(e.target.value)}
                placeholder="443 or 8000-9000"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
              <p className="text-xs text-gray-500 mt-1">Leave empty for all ports</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Protocol
              </label>
              <select
                value={protocol}
                onChange={(e) => setProtocol(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                <option value="">All Protocols</option>
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
                <option value="icmp">ICMP</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Restrict to Network
            </label>
            <select
              value={networkId}
              onChange={(e) => setNetworkId(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="">All Networks</option>
              {networks.map((n) => (
                <option key={n.id} value={n.id}>
                  {n.name} ({n.cidr})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Allow access to internal API server"
              rows={2}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
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
              Active
            </label>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="btn btn-secondary"
              disabled={submitting}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={submitting}
            >
              {submitting ? 'Saving...' : rule ? 'Update Rule' : 'Create Rule'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

interface AssignmentModalProps {
  rule: AccessRule
  onClose: () => void
  onUpdate: () => void
}

function AssignmentModal({ rule, onClose, onUpdate }: AssignmentModalProps) {
  const [selectedUserId, setSelectedUserId] = useState('')
  const [selectedGroupName, setSelectedGroupName] = useState('')
  const [customUserId, setCustomUserId] = useState('')
  const [customGroupName, setCustomGroupName] = useState('')
  const [users, setUsers] = useState<SSOUser[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([getUsers(), getGroups()])
      .then(([u, g]) => {
        setUsers(u)
        setGroups(g)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  // Filter out already assigned users
  const availableUsers = users.filter(
    (u) => !rule.users?.includes(u.id) && !rule.users?.includes(u.email)
  )

  // Filter out already assigned groups
  const availableGroups = groups.filter(
    (g) => !rule.groups?.includes(g.name)
  )

  async function handleAddUser() {
    const userId = selectedUserId || customUserId.trim()
    if (!userId) return
    try {
      await assignRuleToUser(rule.id, userId)
      setSelectedUserId('')
      setCustomUserId('')
      onUpdate()
    } catch (err) {
      setError('Failed to assign rule to user')
    }
  }

  async function handleRemoveUser(userId: string) {
    try {
      await removeRuleFromUser(rule.id, userId)
      onUpdate()
    } catch (err) {
      setError('Failed to remove rule from user')
    }
  }

  async function handleAddGroup() {
    const groupName = selectedGroupName || customGroupName.trim()
    if (!groupName) return
    try {
      await assignRuleToGroup(rule.id, groupName)
      setSelectedGroupName('')
      setCustomGroupName('')
      onUpdate()
    } catch (err) {
      setError('Failed to assign rule to group')
    }
  }

  async function handleRemoveGroup(groupName: string) {
    try {
      await removeRuleFromGroup(rule.id, groupName)
      onUpdate()
    } catch (err) {
      setError('Failed to remove rule from group')
    }
  }

  // Find user details by ID for display
  function getUserDisplay(userId: string) {
    const user = users.find((u) => u.id === userId || u.email === userId)
    if (user) {
      return { name: user.name || user.email, email: user.email, provider: user.provider }
    }
    return { name: userId, email: '', provider: '' }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">
          Assign Access Rule
        </h2>
        <p className="text-sm text-gray-500 mb-4">
          Rule: <span className="font-medium">{rule.name}</span> ({RULE_TYPE_LABELS[rule.ruleType]}: <code className="bg-gray-100 px-1 rounded">{rule.value}</code>)
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
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
              <h3 className="text-sm font-medium text-gray-700 mb-2">Assigned Users</h3>

              {/* Add user controls */}
              <div className="mb-3 p-3 bg-gray-50 rounded-lg space-y-2">
                {availableUsers.length > 0 && (
                  <div className="flex space-x-2">
                    <select
                      value={selectedUserId}
                      onChange={(e) => {
                        setSelectedUserId(e.target.value)
                        setCustomUserId('')
                      }}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
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
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
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
              {rule.users && rule.users.length > 0 ? (
                <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                  {rule.users.map((userId) => {
                    const userInfo = getUserDisplay(userId)
                    return (
                      <div key={userId} className="flex items-center justify-between p-3">
                        <div className="flex items-center">
                          <div className="h-8 w-8 rounded-full bg-primary-100 flex items-center justify-center">
                            <span className="text-primary-700 font-medium text-xs">
                              {userInfo.name.charAt(0).toUpperCase()}
                            </span>
                          </div>
                          <div className="ml-3">
                            <p className="text-sm font-medium text-gray-900">{userInfo.name}</p>
                            {userInfo.email && userInfo.email !== userInfo.name && (
                              <p className="text-xs text-gray-500">{userInfo.email}</p>
                            )}
                          </div>
                          {userInfo.provider && (
                            <span className="ml-2 px-2 py-0.5 text-xs bg-blue-100 text-blue-800 rounded">
                              {userInfo.provider}
                            </span>
                          )}
                        </div>
                        <button
                          onClick={() => handleRemoveUser(userId)}
                          className="p-1.5 rounded-full bg-red-100 text-red-600 hover:bg-red-200 hover:text-red-800 transition-colors"
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
                <p className="text-sm text-gray-500 italic">No users assigned</p>
              )}
            </div>

            {/* Groups section */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">Assigned Groups</h3>

              {/* Add group controls */}
              <div className="mb-3 p-3 bg-gray-50 rounded-lg space-y-2">
                {availableGroups.length > 0 && (
                  <div className="flex space-x-2">
                    <select
                      value={selectedGroupName}
                      onChange={(e) => {
                        setSelectedGroupName(e.target.value)
                        setCustomGroupName('')
                      }}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
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
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500"
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
              {rule.groups && rule.groups.length > 0 ? (
                <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                  {rule.groups.map((groupName) => {
                    const group = groups.find((g) => g.name === groupName)
                    return (
                      <div key={groupName} className="flex items-center justify-between p-3">
                        <div className="flex items-center">
                          <div className="h-8 w-8 rounded-full bg-purple-100 flex items-center justify-center">
                            <svg className="h-4 w-4 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                            </svg>
                          </div>
                          <div className="ml-3">
                            <p className="text-sm font-medium text-gray-900">{groupName}</p>
                            {group && (
                              <p className="text-xs text-gray-500">{group.memberCount} members</p>
                            )}
                          </div>
                        </div>
                        <button
                          onClick={() => handleRemoveGroup(groupName)}
                          className="p-1.5 rounded-full bg-red-100 text-red-600 hover:bg-red-200 hover:text-red-800 transition-colors"
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
                <p className="text-sm text-gray-500 italic">No groups assigned</p>
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
