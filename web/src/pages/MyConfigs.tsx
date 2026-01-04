import { useState, useEffect } from 'react'
import { getUserConfigs, revokeConfig, VPNConfig, getUserMeshConfigs, revokeMeshConfig, MeshVPNConfig } from '../api/client'

type ConfigType = 'gateway' | 'mesh'

interface UnifiedConfig {
  id: string
  type: ConfigType
  name: string // gatewayName or hubName
  fileName: string
  expiresAt: string
  createdAt: string
  isRevoked: boolean
  revokedAt: string | null
  downloaded: boolean
}

export default function MyConfigs() {
  const [gatewayConfigs, setGatewayConfigs] = useState<VPNConfig[]>([])
  const [meshConfigs, setMeshConfigs] = useState<MeshVPNConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [revoking, setRevoking] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'all' | 'gateway' | 'mesh'>('all')

  useEffect(() => {
    loadConfigs()
  }, [])

  async function loadConfigs() {
    try {
      setLoading(true)
      setError(null)
      const [gwData, meshData] = await Promise.all([
        getUserConfigs(),
        getUserMeshConfigs()
      ])
      setGatewayConfigs(gwData)
      setMeshConfigs(meshData)
    } catch (err) {
      setError('Failed to load configs')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleRevoke(configId: string, type: ConfigType) {
    if (!confirm('Are you sure you want to revoke this config? This will immediately disconnect any active VPN session using this config.')) {
      return
    }

    try {
      setRevoking(configId)
      if (type === 'gateway') {
        await revokeConfig(configId)
      } else {
        await revokeMeshConfig(configId)
      }
      await loadConfigs()
    } catch (err) {
      setError('Failed to revoke config')
      console.error(err)
    } finally {
      setRevoking(null)
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function isExpired(dateStr: string) {
    return new Date(dateStr) < new Date()
  }

  // Convert to unified format for display
  function toUnifiedConfigs(): UnifiedConfig[] {
    const gwUnified: UnifiedConfig[] = gatewayConfigs.map(c => ({
      id: c.id,
      type: 'gateway' as ConfigType,
      name: c.gatewayName,
      fileName: c.fileName,
      expiresAt: c.expiresAt,
      createdAt: c.createdAt,
      isRevoked: c.isRevoked,
      revokedAt: c.revokedAt,
      downloaded: c.downloaded,
    }))

    const meshUnified: UnifiedConfig[] = meshConfigs.map(c => ({
      id: c.id,
      type: 'mesh' as ConfigType,
      name: c.hubName,
      fileName: c.fileName,
      expiresAt: c.expiresAt,
      createdAt: c.createdAt,
      isRevoked: c.isRevoked,
      revokedAt: c.revokedAt,
      downloaded: c.downloaded,
    }))

    return [...gwUnified, ...meshUnified].sort((a, b) =>
      new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    )
  }

  const allConfigs = toUnifiedConfigs()
  const filteredConfigs = activeTab === 'all'
    ? allConfigs
    : allConfigs.filter(c => c.type === activeTab)

  const activeConfigs = filteredConfigs.filter(c => !c.isRevoked && !isExpired(c.expiresAt))
  const inactiveConfigs = filteredConfigs.filter(c => c.isRevoked || isExpired(c.expiresAt))

  const gatewayCount = gatewayConfigs.filter(c => !c.isRevoked && !isExpired(c.expiresAt)).length
  const meshCount = meshConfigs.filter(c => !c.isRevoked && !isExpired(c.expiresAt)).length

  return (
    <div className="space-y-6">
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">My VPN Configs</h1>
        <p className="text-theme-tertiary mt-1">
          Manage your VPN configurations. Revoke configs to immediately terminate access.
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <>
          {/* Filter Tabs */}
          <div className="card">
            <div className="border-b border-theme">
              <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                <button
                  onClick={() => setActiveTab('all')}
                  className={`${
                    activeTab === 'all'
                      ? 'border-primary-500 text-primary-600'
                      : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
                  } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm`}
                >
                  All Configs ({gatewayCount + meshCount})
                </button>
                <button
                  onClick={() => setActiveTab('gateway')}
                  className={`${
                    activeTab === 'gateway'
                      ? 'border-primary-500 text-primary-600'
                      : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
                  } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm`}
                >
                  Gateway ({gatewayCount})
                </button>
                <button
                  onClick={() => setActiveTab('mesh')}
                  className={`${
                    activeTab === 'mesh'
                      ? 'border-primary-500 text-primary-600'
                      : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
                  } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm`}
                >
                  Mesh Hub ({meshCount})
                </button>
              </nav>
            </div>
          </div>

          {/* Active Configs */}
          <div className="card">
            <h2 className="text-lg font-semibold text-theme-primary mb-4">
              Active Configs ({activeConfigs.length})
            </h2>
            {activeConfigs.length === 0 ? (
              <p className="text-theme-tertiary text-center py-8">No active configs</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">File</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Expires</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {activeConfigs.map((config) => (
                      <tr key={config.id}>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            config.type === 'gateway'
                              ? 'bg-blue-600 text-white'
                              : 'bg-purple-600 text-white'
                          }`}>
                            {config.type === 'gateway' ? 'Gateway' : 'Mesh Hub'}
                          </span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-theme-primary">{config.name}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {config.fileName}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {formatDate(config.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {formatDate(config.expiresAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-600 text-white">
                            Active
                          </span>
                          {config.downloaded && (
                            <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-600 text-white">
                              Downloaded
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-right">
                          <button
                            onClick={() => handleRevoke(config.id, config.type)}
                            disabled={revoking === config.id}
                            className="inline-flex items-center px-3 py-1.5 text-sm font-medium rounded-md border border-red-300 dark:border-red-700 text-red-700 dark:text-red-400 bg-transparent hover:bg-red-50 dark:hover:bg-red-900/20 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
                          >
                            <svg className="h-4 w-4 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                            </svg>
                            {revoking === config.id ? 'Revoking...' : 'Revoke'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Inactive Configs */}
          {inactiveConfigs.length > 0 && (
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">
                Revoked/Expired Configs ({inactiveConfigs.length})
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">File</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {inactiveConfigs.map((config) => (
                      <tr key={config.id} className="bg-theme-tertiary">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            config.type === 'gateway'
                              ? 'bg-blue-600 text-white'
                              : 'bg-purple-600 text-white'
                          }`}>
                            {config.type === 'gateway' ? 'Gateway' : 'Mesh Hub'}
                          </span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-theme-tertiary">{config.name}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-muted">
                          {config.fileName}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-muted">
                          {formatDate(config.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {config.isRevoked ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-600 text-white">
                              Revoked {config.revokedAt && `on ${formatDate(config.revokedAt)}`}
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300">
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
            <h3 className="info-box-title">About Config Revocation</h3>
            <p className="mt-1 info-box-text">
              Revoking a config immediately terminates any VPN session using that config.
              The embedded credentials become invalid and cannot be used to connect again.
              This is useful if you suspect your config file was compromised.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
