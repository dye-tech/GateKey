import { useState, useEffect } from 'react'
import {
  getNetworks,
  createNetwork,
  deleteNetwork,
  updateNetwork,
  getNetworkGateways,
  getNetworkAccessRules,
  getAdminGateways,
  assignGatewayToNetwork,
  removeGatewayFromNetwork,
  Network,
  Gateway,
  AdminGateway,
  NetworkAccessRule,
} from '../api/client'
import ActionDropdown, { ActionItem } from '../components/ActionDropdown'

export default function AdminNetworks() {
  const [networks, setNetworks] = useState<Network[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editingNetwork, setEditingNetwork] = useState<Network | null>(null)
  const [showGatewaysModal, setShowGatewaysModal] = useState(false)
  const [showAccessRulesModal, setShowAccessRulesModal] = useState(false)
  const [selectedNetwork, setSelectedNetwork] = useState<Network | null>(null)

  useEffect(() => {
    loadNetworks()
  }, [])

  async function loadNetworks() {
    try {
      setLoading(true)
      const data = await getNetworks()
      setNetworks(data)
      setError(null)
    } catch (err) {
      setError('Failed to load networks')
    } finally {
      setLoading(false)
    }
  }

  async function handleDelete(network: Network) {
    if (!confirm(`Are you sure you want to delete network "${network.name}"?`)) {
      return
    }

    try {
      await deleteNetwork(network.id)
      await loadNetworks()
    } catch (err) {
      setError('Failed to delete network')
    }
  }

  function handleManageGateways(network: Network) {
    setSelectedNetwork(network)
    setShowGatewaysModal(true)
  }

  function handleManageAccessRules(network: Network) {
    setSelectedNetwork(network)
    setShowAccessRulesModal(true)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Network Management</h1>
            <p className="text-gray-500 mt-1">
              Define CIDR network blocks and assign gateways to serve them.
            </p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="btn btn-primary"
          >
            <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Network
          </button>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}

      {/* Networks table */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : networks.length > 0 ? (
        <div className="card p-0">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Network
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  CIDR
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
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
              {networks.map((network) => (
                <tr key={network.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div>
                        <div className="text-sm font-medium text-gray-900">{network.name}</div>
                        {network.description && (
                          <div className="text-sm text-gray-500">{network.description}</div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <code className="px-2 py-1 bg-gray-100 rounded text-sm font-mono">
                      {network.cidr}
                    </code>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      network.isActive
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {network.isActive ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(network.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Gateways', icon: 'gateway', onClick: () => handleManageGateways(network), color: 'primary' },
                        { label: 'Access Rules', icon: 'rules', onClick: () => handleManageAccessRules(network), color: 'green' },
                        { label: 'Edit', icon: 'edit', onClick: () => setEditingNetwork(network), color: 'gray' },
                        { label: 'Delete', icon: 'delete', onClick: () => handleDelete(network), color: 'red' },
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
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No networks defined</h3>
          <p className="mt-2 text-gray-500">
            Get started by adding a network CIDR block.
          </p>
          <button
            onClick={() => setShowAddModal(true)}
            className="mt-4 btn btn-primary"
          >
            Add Network
          </button>
        </div>
      )}

      {/* Add Network Modal */}
      {showAddModal && (
        <NetworkModal
          onClose={() => setShowAddModal(false)}
          onSuccess={() => {
            setShowAddModal(false)
            loadNetworks()
          }}
        />
      )}

      {/* Edit Network Modal */}
      {editingNetwork && (
        <NetworkModal
          network={editingNetwork}
          onClose={() => setEditingNetwork(null)}
          onSuccess={() => {
            setEditingNetwork(null)
            loadNetworks()
          }}
        />
      )}

      {/* Manage Gateways Modal */}
      {showGatewaysModal && selectedNetwork && (
        <GatewaysModal
          network={selectedNetwork}
          onClose={() => {
            setShowGatewaysModal(false)
            setSelectedNetwork(null)
          }}
        />
      )}

      {/* Access Rules Modal */}
      {showAccessRulesModal && selectedNetwork && (
        <AccessRulesModal
          network={selectedNetwork}
          onClose={() => {
            setShowAccessRulesModal(false)
            setSelectedNetwork(null)
          }}
        />
      )}
    </div>
  )
}

interface NetworkModalProps {
  network?: Network
  onClose: () => void
  onSuccess: () => void
}

function NetworkModal({ network, onClose, onSuccess }: NetworkModalProps) {
  const [name, setName] = useState(network?.name || '')
  const [description, setDescription] = useState(network?.description || '')
  const [cidr, setCidr] = useState(network?.cidr || '')
  const [isActive, setIsActive] = useState(network?.isActive ?? true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      if (network) {
        await updateNetwork(network.id, { name, description, cidr, is_active: isActive })
      } else {
        await createNetwork({ name, description, cidr, is_active: isActive })
      }
      onSuccess()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || `Failed to ${network ? 'update' : 'create'} network`)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          {network ? 'Edit Network' : 'Add New Network'}
        </h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Network Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="production-network"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              CIDR Block *
            </label>
            <input
              type="text"
              value={cidr}
              onChange={(e) => setCidr(e.target.value)}
              placeholder="10.0.0.0/23"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 font-mono"
              required
            />
            <p className="text-xs text-gray-500 mt-1">e.g., 10.0.0.0/24, 192.168.1.0/24</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Production VPN network for internal services"
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
              {submitting ? 'Saving...' : network ? 'Update Network' : 'Create Network'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

interface GatewaysModalProps {
  network: Network
  onClose: () => void
}

function GatewaysModal({ network, onClose }: GatewaysModalProps) {
  const [assignedGateways, setAssignedGateways] = useState<Gateway[]>([])
  const [allGateways, setAllGateways] = useState<AdminGateway[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedGateway, setSelectedGateway] = useState('')

  useEffect(() => {
    loadData()
  }, [network.id])

  async function loadData() {
    try {
      setLoading(true)
      const [assigned, all] = await Promise.all([
        getNetworkGateways(network.id),
        getAdminGateways(),
      ])
      setAssignedGateways(assigned)
      setAllGateways(all)
      setError(null)
    } catch (err) {
      setError('Failed to load gateways')
    } finally {
      setLoading(false)
    }
  }

  async function handleAssign() {
    if (!selectedGateway) return

    try {
      await assignGatewayToNetwork(selectedGateway, network.id)
      setSelectedGateway('')
      await loadData()
    } catch (err) {
      setError('Failed to assign gateway')
    }
  }

  async function handleRemove(gatewayId: string) {
    try {
      await removeGatewayFromNetwork(gatewayId, network.id)
      await loadData()
    } catch (err) {
      setError('Failed to remove gateway')
    }
  }

  const assignedIds = new Set(assignedGateways.map((g) => g.id))
  const availableGateways = allGateways.filter((g) => !assignedIds.has(g.id))

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">
          Manage Gateways
        </h2>
        <p className="text-sm text-gray-500 mb-4">
          Network: <span className="font-medium">{network.name}</span> ({network.cidr})
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
          <>
            {/* Assign new gateway */}
            {availableGateways.length > 0 && (
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Add Gateway
                </label>
                <div className="flex space-x-2">
                  <select
                    value={selectedGateway}
                    onChange={(e) => setSelectedGateway(e.target.value)}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  >
                    <option value="">Select a gateway...</option>
                    {availableGateways.map((g) => (
                      <option key={g.id} value={g.id}>
                        {g.name} ({g.hostname})
                      </option>
                    ))}
                  </select>
                  <button
                    onClick={handleAssign}
                    disabled={!selectedGateway}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>
            )}

            {/* Assigned gateways list */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                Assigned Gateways ({assignedGateways.length})
              </h3>
              {assignedGateways.length > 0 ? (
                <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                  {assignedGateways.map((gateway) => (
                    <div key={gateway.id} className="flex items-center justify-between p-3">
                      <div>
                        <div className="text-sm font-medium text-gray-900">{gateway.name}</div>
                        <div className="text-xs text-gray-500">{gateway.hostname}</div>
                      </div>
                      <button
                        onClick={() => handleRemove(gateway.id)}
                        className="text-red-600 hover:text-red-800 text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500 italic">No gateways assigned to this network</p>
              )}
            </div>
          </>
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

interface AccessRulesModalProps {
  network: Network
  onClose: () => void
}

function AccessRulesModal({ network, onClose }: AccessRulesModalProps) {
  const [rules, setRules] = useState<NetworkAccessRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadRules()
  }, [network.id])

  async function loadRules() {
    try {
      setLoading(true)
      const data = await getNetworkAccessRules(network.id)
      setRules(data)
      setError(null)
    } catch (err) {
      setError('Failed to load access rules')
    } finally {
      setLoading(false)
    }
  }

  const networkRules = rules.filter(r => r.networkId === network.id)
  const globalRules = rules.filter(r => !r.networkId)

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">
          Access Rules
        </h2>
        <p className="text-sm text-gray-500 mb-4">
          Network: <span className="font-medium">{network.name}</span> ({network.cidr})
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
          </div>
        )}

        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded text-yellow-800 text-sm">
          <strong>Note:</strong> Users/groups must be assigned to access rules to gain access.
          Rules can be created and assigned from the <a href="/admin/access-rules" className="underline">Access Rules</a> page.
        </div>

        {loading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Network-specific rules */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2 flex items-center">
                <span className="bg-green-100 text-green-800 px-2 py-0.5 rounded text-xs mr-2">Network-Specific</span>
                Rules for this network ({networkRules.length})
              </h3>
              {networkRules.length > 0 ? (
                <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                  {networkRules.map((rule) => (
                    <RuleItem key={rule.id} rule={rule} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500 italic py-2">No rules specifically assigned to this network</p>
              )}
            </div>

            {/* Global rules */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2 flex items-center">
                <span className="bg-blue-100 text-blue-800 px-2 py-0.5 rounded text-xs mr-2">Global</span>
                Rules applying to all networks ({globalRules.length})
              </h3>
              {globalRules.length > 0 ? (
                <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                  {globalRules.map((rule) => (
                    <RuleItem key={rule.id} rule={rule} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500 italic py-2">No global rules defined</p>
              )}
            </div>
          </div>
        )}

        <div className="mt-6 flex justify-between">
          <a href="/admin/access-rules" className="btn btn-primary">
            Manage Access Rules
          </a>
          <button onClick={onClose} className="btn btn-secondary">
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

function RuleItem({ rule }: { rule: NetworkAccessRule }) {
  return (
    <div className="p-3">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-sm font-medium text-gray-900">{rule.name}</div>
          <div className="text-xs text-gray-500 mt-1">
            <code className="bg-gray-100 px-1 rounded">{rule.value}</code>
            <span className="ml-2">({rule.ruleType})</span>
            {rule.portRange && <span className="ml-2">Port: {rule.portRange}</span>}
            {rule.protocol && <span className="ml-1">/ {rule.protocol.toUpperCase()}</span>}
          </div>
          {(rule.users.length > 0 || rule.groups.length > 0) && (
            <div className="text-xs mt-2">
              {rule.users.length > 0 && (
                <span className="text-purple-600 mr-3">
                  Users: {rule.users.length}
                </span>
              )}
              {rule.groups.length > 0 && (
                <span className="text-orange-600">
                  Groups: {rule.groups.join(', ')}
                </span>
              )}
            </div>
          )}
        </div>
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
          rule.isActive ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
        }`}>
          {rule.isActive ? 'Active' : 'Inactive'}
        </span>
      </div>
    </div>
  )
}
