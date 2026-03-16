import { useState, useEffect } from 'react'
import {
  getNetworks,
  createNetwork,
  deleteNetwork,
  updateNetwork,
  getNetworkGateways,
  getNetworkAccessRules,
  getNetworkMeshHubs,
  getAdminGateways,
  assignGatewayToNetwork,
  removeGatewayFromNetwork,
  getMeshHubs,
  assignMeshHubNetwork,
  removeMeshHubNetwork,
  getNetworkDNSRule,
  upsertNetworkDNSRule,
  deleteNetworkDNSRule,
  getNetworkDNSRecords,
  createDNSRecord,
  deleteDNSRecord,
  exportDNSRecords,
  importDNSRecords,
  Network,
  Gateway,
  AdminGateway,
  NetworkAccessRule,
  NetworkMeshHub,
  MeshHub,
  DNSRule,
  DNSRecord,
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
  const [showMeshModal, setShowMeshModal] = useState(false)
  const [showDNSModal, setShowDNSModal] = useState(false)
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

  function handleManageMesh(network: Network) {
    setSelectedNetwork(network)
    setShowMeshModal(true)
  }

  function handleManageDNS(network: Network) {
    setSelectedNetwork(network)
    setShowDNSModal(true)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-theme-primary">Network Management</h1>
            <p className="text-theme-tertiary mt-1">
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
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Networks table */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : networks.length > 0 ? (
        <div className="card p-0 overflow-hidden">
          <table className="min-w-full divide-y divide-theme">
            <thead className="bg-theme-tertiary">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Network
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  CIDR
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-theme-card divide-y divide-theme">
              {networks.map((network) => (
                <tr key={network.id} className="hover:bg-theme-tertiary transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div>
                        <div className="text-sm font-medium text-theme-primary">{network.name}</div>
                        {network.description && (
                          <div className="text-sm text-theme-tertiary">{network.description}</div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="space-y-1">
                      {network.cidr && (
                        <div>
                          <code className="px-2 py-1 bg-theme-tertiary rounded text-sm font-mono text-theme-secondary">
                            {network.cidr}
                          </code>
                          {network.cidrV6 && <span className="ml-1 text-xs text-theme-muted">IPv4</span>}
                        </div>
                      )}
                      {network.cidrV6 && (
                        <div>
                          <code className="px-2 py-1 bg-purple-100 dark:bg-purple-900/30 rounded text-sm font-mono text-purple-700 dark:text-purple-300">
                            {network.cidrV6}
                          </code>
                          <span className="ml-1 text-xs text-theme-muted">IPv6</span>
                        </div>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      network.isActive
                        ? 'bg-green-600 text-white'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                    }`}>
                      {network.isActive ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                    {new Date(network.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Gateways', icon: 'gateway', onClick: () => handleManageGateways(network), color: 'primary' },
                        { label: 'Mesh Hubs', icon: 'mesh', onClick: () => handleManageMesh(network), color: 'purple' },
                        { label: 'Access Rules', icon: 'rules', onClick: () => handleManageAccessRules(network), color: 'green' },
                        { label: 'DNS', icon: 'rules', onClick: () => handleManageDNS(network), color: 'primary' },
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
          <svg className="mx-auto h-12 w-12 text-theme-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-theme-primary">No networks defined</h3>
          <p className="mt-2 text-theme-tertiary">
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

      {/* Mesh Hubs Modal */}
      {showMeshModal && selectedNetwork && (
        <MeshHubsModal
          network={selectedNetwork}
          onClose={() => {
            setShowMeshModal(false)
            setSelectedNetwork(null)
          }}
        />
      )}

      {/* DNS Configuration Modal */}
      {showDNSModal && selectedNetwork && (
        <DNSModal
          network={selectedNetwork}
          onClose={() => {
            setShowDNSModal(false)
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
  const [cidrV6, setCidrV6] = useState(network?.cidrV6 || '')
  const [isActive, setIsActive] = useState(network?.isActive ?? true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    // Validate that at least one CIDR is provided
    if (!cidr && !cidrV6) {
      setError('At least one of IPv4 or IPv6 CIDR is required')
      setSubmitting(false)
      return
    }

    try {
      const req = {
        name,
        description,
        cidr: cidr || undefined,
        cidrV6: cidrV6 || undefined,
        is_active: isActive
      }
      if (network) {
        await updateNetwork(network.id, req)
      } else {
        await createNetwork(req)
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
      <div className="bg-theme-card rounded-lg shadow-xl max-w-md w-full mx-4 p-6 border border-theme">
        <h2 className="text-xl font-semibold text-theme-primary mb-4">
          {network ? 'Edit Network' : 'Add New Network'}
        </h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Network Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="production-network"
              className="input"
              required
            />
          </div>

          <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg text-sm text-blue-700 dark:text-blue-300">
            Provide IPv4, IPv6, or both CIDR blocks. At least one is required.
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              IPv4 CIDR Block
            </label>
            <input
              type="text"
              value={cidr}
              onChange={(e) => setCidr(e.target.value)}
              placeholder="10.0.0.0/23"
              className="input font-mono"
            />
            <p className="text-xs text-theme-muted mt-1">e.g., 10.0.0.0/24, 192.168.1.0/24</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              IPv6 CIDR Block
            </label>
            <input
              type="text"
              value={cidrV6}
              onChange={(e) => setCidrV6(e.target.value)}
              placeholder="2001:db8::/32"
              className="input font-mono"
            />
            <p className="text-xs text-theme-muted mt-1">e.g., 2001:db8::/32, fd00::/8</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Production VPN network for internal services"
              rows={2}
              className="input"
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
      <div className="bg-theme-card rounded-lg shadow-xl max-w-lg w-full mx-4 p-6 border border-theme">
        <h2 className="text-xl font-semibold text-theme-primary mb-2">
          Manage Gateways
        </h2>
        <p className="text-sm text-theme-tertiary mb-4">
          Network: <span className="font-medium">{network.name}</span> ({network.cidr})
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
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
                <label className="block text-sm font-medium text-theme-secondary mb-1">
                  Add Gateway
                </label>
                <div className="flex space-x-2">
                  <select
                    value={selectedGateway}
                    onChange={(e) => setSelectedGateway(e.target.value)}
                    className="input flex-1"
                  >
                    <option value="">Select a gateway...</option>
                    {availableGateways.map((g) => (
                      <option key={g.id} value={g.id}>
                        {g.name}{g.hostname ? ` (${g.hostname})` : ''}
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
              <h3 className="text-sm font-medium text-theme-secondary mb-2">
                Assigned Gateways ({assignedGateways.length})
              </h3>
              {assignedGateways.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {assignedGateways.map((gateway) => (
                    <div key={gateway.id} className="flex items-center justify-between p-3">
                      <div>
                        <div className="text-sm font-medium text-theme-primary">{gateway.name}</div>
                        <div className="text-xs text-theme-tertiary">{gateway.hostname}</div>
                      </div>
                      <button
                        onClick={() => handleRemove(gateway.id)}
                        className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic">No gateways assigned to this network</p>
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
      <div className="bg-theme-card rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto border border-theme">
        <h2 className="text-xl font-semibold text-theme-primary mb-2">
          Access Rules
        </h2>
        <p className="text-sm text-theme-tertiary mb-4">
          Network: <span className="font-medium">{network.name}</span> ({network.cidr})
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        <div className="info-box mb-4">
          <div className="flex">
            <svg className="h-5 w-5 info-box-icon flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div className="ml-3 text-sm">
              <strong className="text-theme-primary">Note:</strong>{' '}
              <span className="info-box-text">Users/groups must be assigned to access rules to gain access.
              Rules can be created and assigned from the </span>
              <a href="/admin/access-rules" className="text-primary-600 dark:text-primary-400 hover:underline">Access Rules</a>
              <span className="info-box-text"> page.</span>
            </div>
          </div>
        </div>

        {loading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Network-specific rules */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2 flex items-center">
                <span className="bg-green-600 text-white px-2 py-0.5 rounded text-xs mr-2">Network-Specific</span>
                Rules for this network ({networkRules.length})
              </h3>
              {networkRules.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {networkRules.map((rule) => (
                    <RuleItem key={rule.id} rule={rule} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic py-2">No rules specifically assigned to this network</p>
              )}
            </div>

            {/* Global rules */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2 flex items-center">
                <span className="bg-violet-600 text-white px-2 py-0.5 rounded text-xs mr-2">Global</span>
                Rules applying to all networks ({globalRules.length})
              </h3>
              {globalRules.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {globalRules.map((rule) => (
                    <RuleItem key={rule.id} rule={rule} />
                  ))}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic py-2">No global rules defined</p>
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
          <div className="text-sm font-medium text-theme-primary">{rule.name}</div>
          <div className="text-xs text-theme-tertiary mt-1">
            <code className="bg-theme-tertiary px-1 rounded text-theme-secondary">{rule.value}</code>
            <span className="ml-2">({rule.ruleType})</span>
            {rule.portRange && <span className="ml-2">Port: {rule.portRange}</span>}
            {rule.protocol && <span className="ml-1">/ {rule.protocol.toUpperCase()}</span>}
          </div>
          {(rule.users.length > 0 || rule.groups.length > 0) && (
            <div className="text-xs mt-2">
              {rule.users.length > 0 && (
                <span className="text-purple-600 dark:text-purple-400 mr-3">
                  Users: {rule.users.length}
                </span>
              )}
              {rule.groups.length > 0 && (
                <span className="text-orange-600 dark:text-orange-400">
                  Groups: {rule.groups.join(', ')}
                </span>
              )}
            </div>
          )}
        </div>
        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
          rule.isActive ? 'bg-green-600 text-white' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
        }`}>
          {rule.isActive ? 'Active' : 'Inactive'}
        </span>
      </div>
    </div>
  )
}

interface MeshHubsModalProps {
  network: Network
  onClose: () => void
}

function MeshHubsModal({ network, onClose }: MeshHubsModalProps) {
  const [assignedHubs, setAssignedHubs] = useState<NetworkMeshHub[]>([])
  const [allHubs, setAllHubs] = useState<MeshHub[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedHub, setSelectedHub] = useState('')

  useEffect(() => {
    loadData()
  }, [network.id])

  async function loadData() {
    try {
      setLoading(true)
      const [assigned, all] = await Promise.all([
        getNetworkMeshHubs(network.id),
        getMeshHubs(),
      ])
      setAssignedHubs(assigned)
      setAllHubs(all)
      setError(null)
    } catch (err) {
      setError('Failed to load mesh hubs')
    } finally {
      setLoading(false)
    }
  }

  async function handleAssign() {
    if (!selectedHub) return

    try {
      await assignMeshHubNetwork(selectedHub, network.id)
      setSelectedHub('')
      await loadData()
    } catch (err) {
      setError('Failed to assign hub to network')
    }
  }

  async function handleRemove(hubId: string) {
    try {
      await removeMeshHubNetwork(hubId, network.id)
      await loadData()
    } catch (err) {
      setError('Failed to remove hub from network')
    }
  }

  const assignedIds = new Set(assignedHubs.map((h) => h.id))
  const availableHubs = allHubs.filter((h) => !assignedIds.has(h.id))

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-lg w-full mx-4 p-6 border border-theme">
        <h2 className="text-xl font-semibold text-theme-primary mb-2">
          Manage Mesh Hubs
        </h2>
        <p className="text-sm text-theme-tertiary mb-4">
          Network: <span className="font-medium">{network.name}</span> ({network.cidr})
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        <div className="info-box mb-4">
          <div className="flex">
            <svg className="h-5 w-5 info-box-icon flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div className="ml-3 text-sm info-box-text">
              <strong className="text-theme-primary">Zero-Trust Model:</strong>{' '}
              Assigning a network to a mesh hub allows users with access rules to route traffic through that hub.
            </div>
          </div>
        </div>

        {loading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <>
            {/* Assign new hub */}
            {availableHubs.length > 0 && (
              <div className="mb-4">
                <label className="block text-sm font-medium text-theme-secondary mb-1">
                  Add Mesh Hub
                </label>
                <div className="flex space-x-2">
                  <select
                    value={selectedHub}
                    onChange={(e) => setSelectedHub(e.target.value)}
                    className="input flex-1"
                  >
                    <option value="">Select a hub...</option>
                    {availableHubs.map((h) => (
                      <option key={h.id} value={h.id}>
                        {h.name} ({h.gatewayType === 'wireguard' ? 'WireGuard' : 'OpenVPN'})
                      </option>
                    ))}
                  </select>
                  <button
                    onClick={handleAssign}
                    disabled={!selectedHub}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>
            )}

            {/* Assigned hubs list */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2">
                Assigned Mesh Hubs ({assignedHubs.length})
              </h3>
              {assignedHubs.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {assignedHubs.map((hub) => (
                    <div key={hub.id} className="flex items-center justify-between p-3">
                      <div>
                        <div className="text-sm font-medium text-theme-primary flex items-center">
                          {hub.name}
                          <span className={`ml-2 px-1.5 py-0.5 rounded text-xs font-medium ${
                            hub.gatewayType === 'wireguard'
                              ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300'
                              : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                          }`}>
                            {hub.gatewayType === 'wireguard' ? 'WireGuard' : 'OpenVPN'}
                          </span>
                        </div>
                        <div className="text-xs text-theme-tertiary">{hub.publicEndpoint}</div>
                      </div>
                      <button
                        onClick={() => handleRemove(hub.id)}
                        className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic">No mesh hubs assigned to this network</p>
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

interface DNSModalProps {
  network: Network
  onClose: () => void
}

function DNSModal({ network, onClose }: DNSModalProps) {
  const [rule, setRule] = useState<DNSRule | null>(null)
  const [records, setRecords] = useState<DNSRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [dnsServerInput, setDnsServerInput] = useState('')
  const [searchDomainInput, setSearchDomainInput] = useState('')
  const [dnsServers, setDnsServers] = useState<string[]>([])
  const [searchDomains, setSearchDomains] = useState<string[]>([])
  const [saving, setSaving] = useState(false)

  // New record form
  const [newHostname, setNewHostname] = useState('')
  const [newIPAddress, setNewIPAddress] = useState('')
  const [newDescription, setNewDescription] = useState('')
  const [newRecordType, setNewRecordType] = useState('A')
  const [newIsWildcard, setNewIsWildcard] = useState(false)

  useEffect(() => {
    loadData()
  }, [network.id])

  async function loadData() {
    try {
      setLoading(true)
      const [dnsRule, dnsRecords] = await Promise.all([
        getNetworkDNSRule(network.id),
        getNetworkDNSRecords(network.id),
      ])
      setRule(dnsRule)
      setRecords(dnsRecords)
      if (dnsRule) {
        setDnsServers(dnsRule.dns_servers || [])
        setSearchDomains(dnsRule.search_domains || [])
      }
      setError(null)
    } catch {
      setError('Failed to load DNS configuration')
    } finally {
      setLoading(false)
    }
  }

  function handleAddDNSServer() {
    const val = dnsServerInput.trim()
    if (val && !dnsServers.includes(val)) {
      setDnsServers([...dnsServers, val])
    }
    setDnsServerInput('')
  }

  function handleRemoveDNSServer(srv: string) {
    setDnsServers(dnsServers.filter(s => s !== srv))
  }

  function handleAddSearchDomain() {
    const val = searchDomainInput.trim()
    if (val && !searchDomains.includes(val)) {
      setSearchDomains([...searchDomains, val])
    }
    setSearchDomainInput('')
  }

  function handleRemoveSearchDomain(dom: string) {
    setSearchDomains(searchDomains.filter(d => d !== dom))
  }

  async function handleSaveRule() {
    try {
      setSaving(true)
      await upsertNetworkDNSRule(network.id, {
        dns_servers: dnsServers,
        search_domains: searchDomains,
      })
      await loadData()
      setError(null)
    } catch {
      setError('Failed to save DNS rule')
    } finally {
      setSaving(false)
    }
  }

  async function handleDeleteRule() {
    if (!confirm('Delete DNS configuration for this network?')) return
    try {
      await deleteNetworkDNSRule(network.id)
      setDnsServers([])
      setSearchDomains([])
      setRule(null)
      await loadData()
    } catch {
      setError('Failed to delete DNS rule')
    }
  }

  async function handleAddRecord() {
    if (!newHostname.trim() || !newIPAddress.trim()) return
    try {
      await createDNSRecord(network.id, {
        hostname: newHostname.trim(),
        ip_address: newIPAddress.trim(),
        description: newDescription.trim() || undefined,
        record_type: newRecordType,
        is_wildcard: newIsWildcard,
      })
      setNewHostname('')
      setNewIPAddress('')
      setNewDescription('')
      setNewRecordType('A')
      setNewIsWildcard(false)
      await loadData()
    } catch {
      setError('Failed to create DNS record')
    }
  }

  async function handleExport() {
    try {
      const data = await exportDNSRecords()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'dns-records-export.json'
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      setError('Failed to export DNS records')
    }
  }

  async function handleImport() {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.json'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      try {
        const text = await file.text()
        const data = JSON.parse(text)
        const records = data.records || data
        if (!Array.isArray(records)) {
          setError('Invalid import file format')
          return
        }
        const result = await importDNSRecords(records)
        setError(null)
        await loadData()
        alert(`Imported ${result.imported} of ${result.total} records`)
      } catch {
        setError('Failed to import DNS records')
      }
    }
    input.click()
  }

  async function handleDeleteRecord(id: string) {
    try {
      await deleteDNSRecord(id)
      await loadData()
    } catch {
      setError('Failed to delete DNS record')
    }
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto border border-theme">
        <h2 className="text-xl font-semibold text-theme-primary mb-2">
          DNS Configuration
        </h2>
        <p className="text-sm text-theme-tertiary mb-4">
          Network: <span className="font-medium">{network.name}</span> ({network.cidr})
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <div className="space-y-6">
            {/* DNS Servers */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2">DNS Servers</h3>
              <div className="flex space-x-2 mb-2">
                <input
                  type="text"
                  value={dnsServerInput}
                  onChange={(e) => setDnsServerInput(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleAddDNSServer() } }}
                  placeholder="10.0.0.2"
                  className="input flex-1 font-mono"
                />
                <button onClick={handleAddDNSServer} className="btn btn-primary">Add</button>
              </div>
              {dnsServers.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {dnsServers.map((srv) => (
                    <span key={srv} className="inline-flex items-center px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded text-sm font-mono">
                      {srv}
                      <button onClick={() => handleRemoveDNSServer(srv)} className="ml-1 text-blue-500 hover:text-blue-700">&times;</button>
                    </span>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-theme-muted">No DNS servers configured</p>
              )}
            </div>

            {/* Search Domains */}
            <div>
              <h3 className="text-sm font-medium text-theme-secondary mb-2">Search Domains</h3>
              <div className="flex space-x-2 mb-2">
                <input
                  type="text"
                  value={searchDomainInput}
                  onChange={(e) => setSearchDomainInput(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleAddSearchDomain() } }}
                  placeholder=".internal"
                  className="input flex-1 font-mono"
                />
                <button onClick={handleAddSearchDomain} className="btn btn-primary">Add</button>
              </div>
              {searchDomains.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {searchDomains.map((dom) => (
                    <span key={dom} className="inline-flex items-center px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 rounded text-sm font-mono">
                      {dom}
                      <button onClick={() => handleRemoveSearchDomain(dom)} className="ml-1 text-green-500 hover:text-green-700">&times;</button>
                    </span>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-theme-muted">No search domains configured</p>
              )}
            </div>

            {/* Save / Delete DNS Rule */}
            <div className="flex space-x-2">
              <button onClick={handleSaveRule} disabled={saving} className="btn btn-primary">
                {saving ? 'Saving...' : 'Save DNS Rule'}
              </button>
              {rule && (
                <button onClick={handleDeleteRule} className="btn btn-secondary text-red-600 dark:text-red-400">
                  Delete Rule
                </button>
              )}
            </div>

            {/* Static DNS Records */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-medium text-theme-secondary">Static DNS Records</h3>
                <div className="flex space-x-2">
                  <button onClick={handleExport} className="btn btn-secondary text-xs px-2 py-1">Export</button>
                  <button onClick={handleImport} className="btn btn-secondary text-xs px-2 py-1">Import</button>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2 mb-2">
                <input
                  type="text"
                  value={newHostname}
                  onChange={(e) => setNewHostname(e.target.value)}
                  placeholder="db-prod"
                  className="input font-mono"
                />
                <input
                  type="text"
                  value={newIPAddress}
                  onChange={(e) => setNewIPAddress(e.target.value)}
                  placeholder="10.0.17.75"
                  className="input font-mono"
                />
                <input
                  type="text"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder="Description"
                  className="input"
                />
              </div>
              <div className="flex items-center space-x-3 mb-2">
                <select
                  value={newRecordType}
                  onChange={(e) => setNewRecordType(e.target.value)}
                  className="input w-28"
                >
                  <option value="A">A</option>
                  <option value="AAAA">AAAA</option>
                  <option value="CNAME">CNAME</option>
                </select>
                <label className="flex items-center space-x-1 text-sm text-theme-secondary">
                  <input
                    type="checkbox"
                    checked={newIsWildcard}
                    onChange={(e) => setNewIsWildcard(e.target.checked)}
                    className="rounded"
                  />
                  <span>Wildcard</span>
                </label>
                <button onClick={handleAddRecord} className="btn btn-primary">Add</button>
              </div>
              {records.length > 0 ? (
                <div className="border border-theme rounded-lg divide-y divide-theme">
                  {records.map((record) => (
                    <div key={record.id} className="flex items-center justify-between p-3">
                      <div>
                        <div className="flex items-center space-x-2">
                          <span className="text-sm font-medium text-theme-primary font-mono">{record.hostname}</span>
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
                            {record.record_type || 'A'}
                          </span>
                          {record.is_wildcard && (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300">
                              Wildcard
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-theme-tertiary font-mono">{record.ip_address}</div>
                        {record.description && (
                          <div className="text-xs text-theme-muted">{record.description}</div>
                        )}
                      </div>
                      <button
                        onClick={() => handleDeleteRecord(record.id)}
                        className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 text-sm"
                      >
                        Delete
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-theme-tertiary italic">No static DNS records</p>
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
