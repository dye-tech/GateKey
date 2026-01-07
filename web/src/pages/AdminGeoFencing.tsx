import { useState, useEffect } from 'react'
import {
  getGeoFenceSettings,
  updateGeoFenceSettings,
  getGeoFenceRules,
  createGeoFenceRule,
  updateGeoFenceRule,
  deleteGeoFenceRule,
  getGlobalGeoRules,
  addGlobalGeoRule,
  removeGlobalGeoRule,
  getUsers,
  getLocalUsers,
  getGroups,
  getUserGeoRules,
  addUserGeoRule,
  removeUserGeoRule,
  getGroupGeoRules,
  addGroupGeoRule,
  removeGroupGeoRule,
  GeoFenceSettings,
  GeoFenceRule,
  Group,
} from '../api/client'
import ActionDropdown from '../components/ActionDropdown'

type TabType = 'settings' | 'rules' | 'global' | 'assignments'

interface User {
  id: string
  email: string
  name: string
}

export default function AdminGeoFencing() {
  const [activeTab, setActiveTab] = useState<TabType>('settings')
  const [settings, setSettings] = useState<GeoFenceSettings>({ enabled: false, enforceMode: 'audit' })
  const [rules, setRules] = useState<GeoFenceRule[]>([])
  const [globalRules, setGlobalRules] = useState<GeoFenceRule[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // Modal states
  const [showRuleModal, setShowRuleModal] = useState(false)
  const [editingRule, setEditingRule] = useState<GeoFenceRule | null>(null)
  const [showAssignGlobalModal, setShowAssignGlobalModal] = useState(false)
  const [showUserRulesModal, setShowUserRulesModal] = useState(false)
  const [showGroupRulesModal, setShowGroupRulesModal] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null)
  const [userRules, setUserRules] = useState<GeoFenceRule[]>([])
  const [groupRules, setGroupRules] = useState<GeoFenceRule[]>([])

  useEffect(() => {
    loadAll()
  }, [])

  async function loadAll() {
    setLoading(true)
    try {
      const [settingsData, rulesData, globalData, usersData, localUsersData, groupsData] = await Promise.all([
        getGeoFenceSettings(),
        getGeoFenceRules(),
        getGlobalGeoRules(),
        getUsers(),
        getLocalUsers(),
        getGroups(),
      ])
      setSettings(settingsData)
      setRules(rulesData)
      setGlobalRules(globalData)
      // Combine SSO and local users
      const allUsers = [
        ...usersData.map(u => ({ id: u.id, email: u.email, name: u.name || u.email })),
        ...localUsersData.map(u => ({ id: u.id, email: u.email, name: u.username })),
      ]
      setUsers(allUsers)
      setGroups(groupsData)
      setError(null)
    } catch (err) {
      setError('Failed to load geo-fencing data')
    } finally {
      setLoading(false)
    }
  }

  async function handleSaveSettings() {
    setSaving(true)
    try {
      await updateGeoFenceSettings(settings)
      setSuccess('Settings saved successfully')
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  async function handleSaveRule(data: { name: string; description: string; ipRange: string; isActive?: boolean }) {
    try {
      if (editingRule) {
        await updateGeoFenceRule(editingRule.id, {
          name: data.name,
          description: data.description,
          ipRange: data.ipRange,
          isActive: data.isActive ?? true,
        })
      } else {
        await createGeoFenceRule(data)
      }
      setShowRuleModal(false)
      setEditingRule(null)
      await loadAll()
    } catch (err) {
      setError('Failed to save rule')
    }
  }

  async function handleDeleteRule(rule: GeoFenceRule) {
    if (!confirm(`Are you sure you want to delete rule "${rule.name}"? This will remove it from all assignments.`)) {
      return
    }
    try {
      await deleteGeoFenceRule(rule.id)
      await loadAll()
    } catch (err) {
      setError('Failed to delete rule')
    }
  }

  async function handleAddGlobalRule(ruleId: string) {
    try {
      await addGlobalGeoRule(ruleId)
      setShowAssignGlobalModal(false)
      const globalData = await getGlobalGeoRules()
      setGlobalRules(globalData)
    } catch (err) {
      setError('Failed to add global rule')
    }
  }

  async function handleRemoveGlobalRule(ruleId: string) {
    try {
      await removeGlobalGeoRule(ruleId)
      const globalData = await getGlobalGeoRules()
      setGlobalRules(globalData)
    } catch (err) {
      setError('Failed to remove global rule')
    }
  }

  async function handleViewUserRules(user: User) {
    try {
      const rules = await getUserGeoRules(user.id)
      setUserRules(rules)
      setSelectedUser(user)
      setShowUserRulesModal(true)
    } catch (err) {
      setError('Failed to load user rules')
    }
  }

  async function handleAddUserRule(ruleId: string) {
    if (!selectedUser) return
    try {
      await addUserGeoRule(selectedUser.id, ruleId)
      const rules = await getUserGeoRules(selectedUser.id)
      setUserRules(rules)
      await loadAll()
    } catch (err) {
      setError('Failed to add user rule')
    }
  }

  async function handleRemoveUserRule(ruleId: string) {
    if (!selectedUser) return
    try {
      await removeUserGeoRule(selectedUser.id, ruleId)
      const rules = await getUserGeoRules(selectedUser.id)
      setUserRules(rules)
      await loadAll()
    } catch (err) {
      setError('Failed to remove user rule')
    }
  }

  async function handleViewGroupRules(group: Group) {
    try {
      const rules = await getGroupGeoRules(group.name)
      setGroupRules(rules)
      setSelectedGroup(group)
      setShowGroupRulesModal(true)
    } catch (err) {
      setError('Failed to load group rules')
    }
  }

  async function handleAddGroupRule(ruleId: string) {
    if (!selectedGroup) return
    try {
      await addGroupGeoRule(selectedGroup.name, ruleId)
      const rules = await getGroupGeoRules(selectedGroup.name)
      setGroupRules(rules)
      await loadAll()
    } catch (err) {
      setError('Failed to add group rule')
    }
  }

  async function handleRemoveGroupRule(ruleId: string) {
    if (!selectedGroup) return
    try {
      await removeGroupGeoRule(selectedGroup.name, ruleId)
      const rules = await getGroupGeoRules(selectedGroup.name)
      setGroupRules(rules)
      await loadAll()
    } catch (err) {
      setError('Failed to remove group rule')
    }
  }

  const tabs = [
    { id: 'settings' as TabType, label: 'Settings', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z' },
    { id: 'rules' as TabType, label: 'IP Rules', icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
    { id: 'global' as TabType, label: 'Global Rules', icon: 'M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z' },
    { id: 'assignments' as TabType, label: 'User/Group Overrides', icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z' },
  ]

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-theme-primary">Geo-Fencing</h1>
            <p className="text-theme-tertiary mt-1">
              Restrict VPN access based on client IP address. Only whitelisted IPs can connect.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span className={`px-3 py-1 rounded-full text-sm font-medium ${
              settings.enabled
                ? 'bg-emerald-600 text-white'
                : 'bg-slate-500 text-white'
            }`}>
              {settings.enabled ? 'Enabled' : 'Disabled'}
            </span>
            <span className={`px-3 py-1 rounded-full text-sm font-medium ${
              settings.enforceMode === 'enforce'
                ? 'bg-red-600 text-white'
                : 'bg-amber-500 text-white'
            }`}>
              {settings.enforceMode === 'enforce' ? 'Enforcing' : 'Audit Only'}
            </span>
          </div>
        </div>
      </div>

      {/* Messages */}
      {error && (
        <div className="alert-danger">
          <div className="alert-danger-title">
            <svg className="w-5 h-5 alert-danger-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Error
            <button onClick={() => setError(null)} className="ml-auto text-current opacity-70 hover:opacity-100">&times;</button>
          </div>
          <p className="alert-danger-text">{error}</p>
        </div>
      )}
      {success && (
        <div className="alert-success">
          <div className="alert-success-title">
            <svg className="w-5 h-5 alert-success-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Success
          </div>
          <p className="alert-success-text">{success}</p>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-theme">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`py-4 px-1 border-b-2 font-medium text-sm flex items-center gap-2 ${
                activeTab === tab.id
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-theme-tertiary hover:text-theme-primary hover:border-gray-300'
              }`}
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={tab.icon} />
              </svg>
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === 'settings' && (
        <div className="card space-y-6">
          <h2 className="text-lg font-semibold text-theme-primary">Geo-Fencing Settings</h2>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <label className="font-medium text-theme-primary">Enable Geo-Fencing</label>
                <p className="text-sm text-theme-tertiary">When enabled, only IPs matching your rules can connect</p>
              </div>
              <button
                onClick={() => setSettings(s => ({ ...s, enabled: !s.enabled }))}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  settings.enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'
                }`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  settings.enabled ? 'translate-x-6' : 'translate-x-1'
                }`} />
              </button>
            </div>

            <div>
              <label className="font-medium text-theme-primary block mb-2">Enforcement Mode</label>
              <select
                value={settings.enforceMode}
                onChange={(e) => setSettings(s => ({ ...s, enforceMode: e.target.value as 'enforce' | 'audit' }))}
                className="input w-full max-w-xs"
              >
                <option value="audit">Audit Only (log but allow)</option>
                <option value="enforce">Enforce (block connections)</option>
              </select>
              <p className="text-sm text-theme-tertiary mt-1">
                {settings.enforceMode === 'audit'
                  ? 'Connections from non-whitelisted IPs will be logged but allowed'
                  : 'Connections from non-whitelisted IPs will be blocked'}
              </p>
            </div>

            <div className="pt-4 border-t border-theme">
              <button
                onClick={handleSaveSettings}
                disabled={saving}
                className="btn btn-primary"
              >
                {saving ? 'Saving...' : 'Save Settings'}
              </button>
            </div>
          </div>

          <div className="alert-warning mt-6">
            <h3 className="alert-warning-title">
              <svg className="w-5 h-5 alert-warning-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              Important: Whitelist Model
            </h3>
            <p className="alert-warning-text">
              Geo-fencing uses a whitelist model. When enabled with no rules, <strong>all connections will be blocked</strong>.
              Make sure to add rules for your allowed IP ranges before enabling enforcement.
            </p>
          </div>
        </div>
      )}

      {activeTab === 'rules' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => { setEditingRule(null); setShowRuleModal(true); }}
              className="btn btn-primary"
            >
              <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add IP Rule
            </button>
          </div>

          {rules.length === 0 ? (
            <div className="card text-center py-12 text-theme-tertiary">
              <svg className="w-12 h-12 mx-auto mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <p>No IP rules defined yet</p>
              <p className="text-sm mt-1">Create rules to whitelist IP ranges</p>
            </div>
          ) : (
            <div className="card p-0">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">IP Range (CIDR)</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Assignments</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-theme">
                  {rules.map((rule) => (
                    <tr key={rule.id} className="hover:bg-theme-secondary">
                      <td className="px-6 py-4">
                        <div className="font-medium text-theme-primary">{rule.name}</div>
                        {rule.description && (
                          <div className="text-sm text-theme-tertiary">{rule.description}</div>
                        )}
                      </td>
                      <td className="px-6 py-4 font-mono text-theme-secondary">{rule.ipRange}</td>
                      <td className="px-6 py-4">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                          rule.isActive
                            ? 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400'
                            : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400'
                        }`}>
                          {rule.isActive ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-theme-tertiary">
                        {rule.isGlobal && <span className="mr-2">Global</span>}
                        {(rule.userCount || 0) > 0 && <span className="mr-2">{rule.userCount} user{rule.userCount !== 1 ? 's' : ''}</span>}
                        {(rule.groupCount || 0) > 0 && <span>{rule.groupCount} group{rule.groupCount !== 1 ? 's' : ''}</span>}
                        {!rule.isGlobal && !rule.userCount && !rule.groupCount && <span className="text-yellow-600">Not assigned</span>}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <ActionDropdown
                          actions={[
                            { label: 'Edit', icon: 'edit', onClick: () => { setEditingRule(rule); setShowRuleModal(true); } },
                            { label: 'Delete', icon: 'delete', onClick: () => handleDeleteRule(rule), color: 'red' },
                          ]}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {activeTab === 'global' && (
        <div className="space-y-4">
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold text-theme-primary">Global Rules</h2>
                <p className="text-sm text-theme-tertiary">
                  Rules applied to all users unless they have user or group-specific overrides
                </p>
              </div>
              <button
                onClick={() => setShowAssignGlobalModal(true)}
                className="btn btn-primary"
              >
                Add Rule
              </button>
            </div>

            {globalRules.length === 0 ? (
              <div className="text-center py-8 text-theme-tertiary">
                <p>No global rules assigned</p>
                <p className="text-sm mt-1">Add rules to apply to all users by default</p>
              </div>
            ) : (
              <div className="space-y-2">
                {globalRules.map((rule) => (
                  <div key={rule.id} className="flex items-center justify-between p-3 bg-theme-secondary rounded-lg">
                    <div>
                      <div className="font-medium text-theme-primary">{rule.name}</div>
                      <div className="text-sm text-theme-tertiary font-mono">{rule.ipRange}</div>
                    </div>
                    <button
                      onClick={() => handleRemoveGlobalRule(rule.id)}
                      className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'assignments' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Users */}
          <div className="card">
            <h2 className="text-lg font-semibold text-theme-primary mb-4">User Overrides</h2>
            <p className="text-sm text-theme-tertiary mb-4">
              Assign rules to specific users (overrides global and group rules)
            </p>

            {users.length === 0 ? (
              <p className="text-theme-tertiary">No users found</p>
            ) : (
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {users.map((user) => (
                  <div key={user.id} className="flex items-center justify-between p-3 bg-theme-secondary rounded-lg">
                    <div>
                      <div className="font-medium text-theme-primary">{user.name}</div>
                      <div className="text-sm text-theme-tertiary">{user.email}</div>
                    </div>
                    <button
                      onClick={() => handleViewUserRules(user)}
                      className="btn btn-secondary text-sm"
                    >
                      Manage Rules
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Groups */}
          <div className="card">
            <h2 className="text-lg font-semibold text-theme-primary mb-4">Group Overrides</h2>
            <p className="text-sm text-theme-tertiary mb-4">
              Assign rules to groups (overrides global rules, overridden by user rules)
            </p>

            {groups.length === 0 ? (
              <p className="text-theme-tertiary">No groups found</p>
            ) : (
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {groups.map((group) => (
                  <div key={group.name} className="flex items-center justify-between p-3 bg-theme-secondary rounded-lg">
                    <div>
                      <div className="font-medium text-theme-primary">{group.name}</div>
                      <div className="text-sm text-theme-tertiary">{group.memberCount} member{group.memberCount !== 1 ? 's' : ''}</div>
                    </div>
                    <button
                      onClick={() => handleViewGroupRules(group)}
                      className="btn btn-secondary text-sm"
                    >
                      Manage Rules
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Rule Modal */}
      {showRuleModal && (
        <RuleModal
          rule={editingRule}
          onSave={handleSaveRule}
          onClose={() => { setShowRuleModal(false); setEditingRule(null); }}
        />
      )}

      {/* Assign Global Rule Modal */}
      {showAssignGlobalModal && (
        <AssignRuleModal
          title="Add Global Rule"
          rules={rules.filter(r => !globalRules.find(g => g.id === r.id))}
          onAssign={handleAddGlobalRule}
          onClose={() => setShowAssignGlobalModal(false)}
        />
      )}

      {/* User Rules Modal */}
      {showUserRulesModal && selectedUser && (
        <ManageRulesModal
          title={`Geo-Fence Rules for ${selectedUser.name}`}
          assignedRules={userRules}
          allRules={rules}
          onAdd={handleAddUserRule}
          onRemove={handleRemoveUserRule}
          onClose={() => { setShowUserRulesModal(false); setSelectedUser(null); }}
        />
      )}

      {/* Group Rules Modal */}
      {showGroupRulesModal && selectedGroup && (
        <ManageRulesModal
          title={`Geo-Fence Rules for Group: ${selectedGroup.name}`}
          assignedRules={groupRules}
          allRules={rules}
          onAdd={handleAddGroupRule}
          onRemove={handleRemoveGroupRule}
          onClose={() => { setShowGroupRulesModal(false); setSelectedGroup(null); }}
        />
      )}
    </div>
  )
}

// Rule creation/edit modal
function RuleModal({ rule, onSave, onClose }: {
  rule: GeoFenceRule | null
  onSave: (data: { name: string; description: string; ipRange: string; isActive?: boolean }) => void
  onClose: () => void
}) {
  const [name, setName] = useState(rule?.name || '')
  const [description, setDescription] = useState(rule?.description || '')
  const [ipRange, setIpRange] = useState(rule?.ipRange || '')
  const [isActive, setIsActive] = useState(rule?.isActive ?? true)
  const [error, setError] = useState<string | null>(null)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    // Basic CIDR validation
    const cidrRegex = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/
    if (!cidrRegex.test(ipRange)) {
      setError('Invalid CIDR format. Example: 192.168.0.0/24')
      return
    }

    onSave({ name, description, ipRange, isActive })
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="modal-content p-6 w-full max-w-md">
        <h2 className="text-xl font-bold text-theme-primary mb-4">
          {rule ? 'Edit IP Rule' : 'Add IP Rule'}
        </h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-theme-primary mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="input w-full"
              placeholder="e.g., US East Coast"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-primary mb-1">IP Range (CIDR)</label>
            <input
              type="text"
              value={ipRange}
              onChange={(e) => { setIpRange(e.target.value); setError(null); }}
              className="input w-full font-mono"
              placeholder="e.g., 192.168.0.0/24"
              required
            />
            <p className="text-xs text-theme-tertiary mt-1">
              Enter an IP range in CIDR notation
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-primary mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="input w-full"
              placeholder="Optional description"
              rows={2}
            />
          </div>

          {rule && (
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-theme-primary">Active</label>
              <button
                type="button"
                onClick={() => setIsActive(!isActive)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  isActive ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'
                }`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  isActive ? 'translate-x-6' : 'translate-x-1'
                }`} />
              </button>
            </div>
          )}

          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary">
              {rule ? 'Save Changes' : 'Create Rule'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// Modal to assign a rule
function AssignRuleModal({ title, rules, onAssign, onClose }: {
  title: string
  rules: GeoFenceRule[]
  onAssign: (ruleId: string) => void
  onClose: () => void
}) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="modal-content p-6 w-full max-w-md">
        <h2 className="text-xl font-bold text-theme-primary mb-4">{title}</h2>

        {rules.length === 0 ? (
          <div className="text-center py-4">
            <p className="text-theme-tertiary mb-2">No available rules to assign</p>
            <p className="text-sm text-theme-tertiary">
              Create IP rules in the <strong>IP Rules</strong> tab first, then assign them here.
            </p>
          </div>
        ) : (
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {rules.map((rule) => (
              <button
                key={rule.id}
                onClick={() => onAssign(rule.id)}
                className="w-full text-left p-3 bg-theme-secondary hover:bg-theme-tertiary rounded-lg transition-colors"
              >
                <div className="font-medium text-theme-primary">{rule.name}</div>
                <div className="text-sm text-theme-tertiary font-mono">{rule.ipRange}</div>
              </button>
            ))}
          </div>
        )}

        <div className="flex justify-end mt-4">
          <button onClick={onClose} className="btn btn-secondary">Close</button>
        </div>
      </div>
    </div>
  )
}

// Modal to manage assigned rules
function ManageRulesModal({ title, assignedRules, allRules, onAdd, onRemove, onClose }: {
  title: string
  assignedRules: GeoFenceRule[]
  allRules: GeoFenceRule[]
  onAdd: (ruleId: string) => void
  onRemove: (ruleId: string) => void
  onClose: () => void
}) {
  const availableRules = allRules.filter(r => !assignedRules.find(a => a.id === r.id))
  const [showAddDropdown, setShowAddDropdown] = useState(false)

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="modal-content p-6 w-full max-w-lg">
        <h2 className="text-xl font-bold text-theme-primary mb-4">{title}</h2>

        <div className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <h3 className="font-medium text-theme-primary">Assigned Rules</h3>
            <div className="relative">
              <button
                onClick={() => setShowAddDropdown(!showAddDropdown)}
                className="btn btn-primary text-sm"
                disabled={availableRules.length === 0}
              >
                Add Rule
              </button>
              {showAddDropdown && availableRules.length > 0 && (
                <div className="absolute right-0 mt-1 w-64 dropdown-menu max-h-48 overflow-y-auto">
                  {availableRules.map((rule) => (
                    <button
                      key={rule.id}
                      onClick={() => { onAdd(rule.id); setShowAddDropdown(false); }}
                      className="w-full text-left p-3 dropdown-item"
                    >
                      <div className="font-medium text-theme-primary">{rule.name}</div>
                      <div className="text-sm text-theme-tertiary font-mono">{rule.ipRange}</div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {assignedRules.length === 0 ? (
            <p className="text-theme-tertiary text-sm p-4 bg-theme-secondary rounded-lg">
              No rules assigned. Global rules will apply.
            </p>
          ) : (
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {assignedRules.map((rule) => (
                <div key={rule.id} className="flex items-center justify-between p-3 bg-theme-secondary rounded-lg">
                  <div>
                    <div className="font-medium text-theme-primary">{rule.name}</div>
                    <div className="text-sm text-theme-tertiary font-mono">{rule.ipRange}</div>
                  </div>
                  <button
                    onClick={() => onRemove(rule.id)}
                    className="text-red-600 hover:text-red-800 dark:text-red-400"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="flex justify-end">
          <button onClick={onClose} className="btn btn-secondary">Close</button>
        </div>
      </div>
    </div>
  )
}
