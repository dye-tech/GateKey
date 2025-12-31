import { useState, useEffect } from 'react'
import { getUserConfigs, revokeConfig, VPNConfig } from '../api/client'

export default function MyConfigs() {
  const [configs, setConfigs] = useState<VPNConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [revoking, setRevoking] = useState<string | null>(null)

  useEffect(() => {
    loadConfigs()
  }, [])

  async function loadConfigs() {
    try {
      setLoading(true)
      setError(null)
      const data = await getUserConfigs()
      setConfigs(data)
    } catch (err) {
      setError('Failed to load configs')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleRevoke(configId: string) {
    if (!confirm('Are you sure you want to revoke this config? This will immediately disconnect any active VPN session using this config.')) {
      return
    }

    try {
      setRevoking(configId)
      await revokeConfig(configId)
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

  const activeConfigs = configs.filter(c => !c.isRevoked && !isExpired(c.expiresAt))
  const inactiveConfigs = configs.filter(c => c.isRevoked || isExpired(c.expiresAt))

  return (
    <div className="space-y-6">
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">My VPN Configs</h1>
        <p className="text-gray-500 mt-1">
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
          {/* Active Configs */}
          <div className="card">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              Active Configs ({activeConfigs.length})
            </h2>
            {activeConfigs.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No active configs</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Gateway</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">File</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Expires</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                      <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {activeConfigs.map((config) => (
                      <tr key={config.id}>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-gray-900">{config.gatewayName}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {config.fileName}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(config.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(config.expiresAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                            Active
                          </span>
                          {config.downloaded && (
                            <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                              Downloaded
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-right">
                          <button
                            onClick={() => handleRevoke(config.id)}
                            disabled={revoking === config.id}
                            className="inline-flex items-center px-3 py-1.5 border border-red-300 text-sm font-medium rounded-md text-red-700 bg-white hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
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
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Revoked/Expired Configs ({inactiveConfigs.length})
              </h2>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Gateway</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">File</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {inactiveConfigs.map((config) => (
                      <tr key={config.id} className="bg-gray-50">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className="font-medium text-gray-500">{config.gatewayName}</span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-400">
                          {config.fileName}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-400">
                          {formatDate(config.createdAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          {config.isRevoked ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                              Revoked {config.revokedAt && `on ${formatDate(config.revokedAt)}`}
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
            <h3 className="text-sm font-medium text-blue-800">About Config Revocation</h3>
            <p className="mt-1 text-sm text-blue-700">
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
