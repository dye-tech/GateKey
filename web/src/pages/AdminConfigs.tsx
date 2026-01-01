import { useState, useEffect } from 'react'
import {
  adminListAllConfigs,
  adminListMeshConfigs,
  adminRevokeConfig,
  adminRevokeMeshConfig,
  AdminVPNConfig,
  AdminMeshVPNConfig,
} from '../api/client'

type TabType = 'gateway' | 'mesh'

export default function AdminConfigs() {
  const [activeTab, setActiveTab] = useState<TabType>('gateway')
  const [gatewayConfigs, setGatewayConfigs] = useState<AdminVPNConfig[]>([])
  const [meshConfigs, setMeshConfigs] = useState<AdminMeshVPNConfig[]>([])
  const [gatewayTotal, setGatewayTotal] = useState(0)
  const [meshTotal, setMeshTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // Filters
  const [filterUser, setFilterUser] = useState('')
  const [filterStatus, setFilterStatus] = useState<'all' | 'active' | 'revoked' | 'expired'>('all')

  // Load both datasets on initial mount
  useEffect(() => {
    loadAllData()
  }, [])

  // Reload current tab data when tab changes (after initial load)
  useEffect(() => {
    if (gatewayConfigs.length > 0 || meshConfigs.length > 0 || gatewayTotal > 0 || meshTotal > 0) {
      // Only reload if we've already loaded initial data
      loadTabData()
    }
  }, [activeTab])

  async function loadAllData() {
    try {
      setLoading(true)
      setError(null)

      // Load both gateway and mesh configs in parallel
      const [gatewayResult, meshResult] = await Promise.all([
        adminListAllConfigs(),
        adminListMeshConfigs()
      ])

      setGatewayConfigs(gatewayResult.configs)
      setGatewayTotal(gatewayResult.total)
      setMeshConfigs(meshResult.configs)
      setMeshTotal(meshResult.total)
    } catch (err) {
      setError('Failed to load configs')
    } finally {
      setLoading(false)
    }
  }

  async function loadTabData() {
    try {
      setLoading(true)
      setError(null)

      if (activeTab === 'gateway') {
        const result = await adminListAllConfigs()
        setGatewayConfigs(result.configs)
        setGatewayTotal(result.total)
      } else {
        const result = await adminListMeshConfigs()
        setMeshConfigs(result.configs)
        setMeshTotal(result.total)
      }
    } catch (err) {
      setError('Failed to load configs')
    } finally {
      setLoading(false)
    }
  }

  async function handleRevokeConfig(configId: string, isMesh: boolean) {
    const reason = prompt('Enter reason for revocation (optional):')
    if (reason === null) return // User cancelled

    try {
      if (isMesh) {
        await adminRevokeMeshConfig(configId, reason || 'Revoked by admin')
      } else {
        await adminRevokeConfig(configId, reason || 'Revoked by admin')
      }
      setSuccess('Config revoked successfully')
      setTimeout(() => setSuccess(null), 3000)
      loadAllData()
    } catch (err) {
      setError('Failed to revoke config')
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function isExpired(expiresAt: string) {
    return new Date(expiresAt) < new Date()
  }

  function getStatus(config: AdminVPNConfig | AdminMeshVPNConfig): 'active' | 'revoked' | 'expired' {
    if (config.isRevoked) return 'revoked'
    if (isExpired(config.expiresAt)) return 'expired'
    return 'active'
  }

  function filterConfigs<T extends AdminVPNConfig | AdminMeshVPNConfig>(configs: T[]): T[] {
    return configs.filter(config => {
      // User filter
      if (filterUser) {
        const userMatch =
          config.userEmail?.toLowerCase().includes(filterUser.toLowerCase()) ||
          config.userName?.toLowerCase().includes(filterUser.toLowerCase())
        if (!userMatch) return false
      }

      // Status filter
      if (filterStatus !== 'all') {
        const status = getStatus(config)
        if (status !== filterStatus) return false
      }

      return true
    })
  }

  const filteredGatewayConfigs = filterConfigs(gatewayConfigs)
  const filteredMeshConfigs = filterConfigs(meshConfigs)

  function clearFilters() {
    setFilterUser('')
    setFilterStatus('all')
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">VPN Configurations</h1>
        <p className="text-gray-500 mt-1">
          View and manage all VPN configurations across all users.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('gateway')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'gateway'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Gateway Configs
            <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">
              {gatewayTotal}
            </span>
          </button>
          <button
            onClick={() => setActiveTab('mesh')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'mesh'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Mesh Configs
            <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">
              {meshTotal}
            </span>
          </button>
        </nav>
      </div>

      {/* Messages */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}
      {success && (
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
          {success}
        </div>
      )}

      {/* Filters */}
      <div className="card">
        <h3 className="text-sm font-medium text-gray-700 mb-3">Filters</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">User</label>
            <input
              type="text"
              value={filterUser}
              onChange={(e) => setFilterUser(e.target.value)}
              placeholder="Search by email or name..."
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Status</label>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value as typeof filterStatus)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="all">All</option>
              <option value="active">Active</option>
              <option value="revoked">Revoked</option>
              <option value="expired">Expired</option>
            </select>
          </div>
        </div>
        {(filterUser || filterStatus !== 'all') && (
          <button
            onClick={clearFilters}
            className="mt-3 text-sm text-primary-600 hover:text-primary-800"
          >
            Clear filters
          </button>
        )}
      </div>

      {/* Loading */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <>
          {/* Gateway Configs Tab */}
          {activeTab === 'gateway' && (
            <div className="card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Gateway</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Expires</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {filteredGatewayConfigs.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="px-4 py-8 text-center text-gray-500">
                          No gateway configs found
                        </td>
                      </tr>
                    ) : (
                      filteredGatewayConfigs.map((config) => {
                        const status = getStatus(config)
                        return (
                          <tr key={config.id} className="hover:bg-gray-50">
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-medium text-gray-900">{config.userEmail}</div>
                              {config.userName && (
                                <div className="text-xs text-gray-500">{config.userName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                                {config.gatewayName}
                              </span>
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 font-mono">
                              {config.fileName}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {formatDate(config.createdAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {formatDate(config.expiresAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              {status === 'active' && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                                  Active
                                </span>
                              )}
                              {status === 'revoked' && (
                                <div>
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">
                                    Revoked
                                  </span>
                                  {config.revokedReason && (
                                    <div className="text-xs text-gray-500 mt-0.5">{config.revokedReason}</div>
                                  )}
                                </div>
                              )}
                              {status === 'expired' && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">
                                  Expired
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm">
                              {status === 'active' && (
                                <button
                                  onClick={() => handleRevokeConfig(config.id, false)}
                                  className="text-red-600 hover:text-red-800"
                                >
                                  Revoke
                                </button>
                              )}
                            </td>
                          </tr>
                        )
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Mesh Configs Tab */}
          {activeTab === 'mesh' && (
            <div className="card overflow-hidden">
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Mesh Hub</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Expires</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {filteredMeshConfigs.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="px-4 py-8 text-center text-gray-500">
                          No mesh configs found
                        </td>
                      </tr>
                    ) : (
                      filteredMeshConfigs.map((config) => {
                        const status = getStatus(config)
                        return (
                          <tr key={config.id} className="hover:bg-gray-50">
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-medium text-gray-900">{config.userEmail}</div>
                              {config.userName && (
                                <div className="text-xs text-gray-500">{config.userName}</div>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
                                {config.hubName}
                              </span>
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 font-mono">
                              {config.fileName}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {formatDate(config.createdAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {formatDate(config.expiresAt)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              {status === 'active' && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                                  Active
                                </span>
                              )}
                              {status === 'revoked' && (
                                <div>
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">
                                    Revoked
                                  </span>
                                  {config.revokedReason && (
                                    <div className="text-xs text-gray-500 mt-0.5">{config.revokedReason}</div>
                                  )}
                                </div>
                              )}
                              {status === 'expired' && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">
                                  Expired
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm">
                              {status === 'active' && (
                                <button
                                  onClick={() => handleRevokeConfig(config.id, true)}
                                  className="text-red-600 hover:text-red-800"
                                >
                                  Revoke
                                </button>
                              )}
                            </td>
                          </tr>
                        )
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
