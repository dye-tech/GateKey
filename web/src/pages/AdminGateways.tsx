import { useState, useEffect } from 'react'
import {
  getAdminGateways, registerGateway, deleteGateway, updateGateway,
  getGatewayNetworks, getNetworks, assignGatewayToNetwork, removeGatewayFromNetwork,
  getGatewayUsers, assignUserToGateway, removeUserFromGateway,
  getGatewayGroups, assignGroupToGateway, removeGroupFromGateway,
  AdminGateway, RegisterGatewayResponse, Network, GatewayUser, GatewayGroup, CryptoProfile
} from '../api/client'
import ActionDropdown, { ActionItem } from '../components/ActionDropdown'

export default function AdminGateways() {
  const [gateways, setGateways] = useState<AdminGateway[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [showTokenModal, setShowTokenModal] = useState(false)
  const [newGateway, setNewGateway] = useState<RegisterGatewayResponse | null>(null)
  const [showInstaller, setShowInstaller] = useState(false)
  const [selectedGateway, setSelectedGateway] = useState<AdminGateway | null>(null)
  const [installerToken, setInstallerToken] = useState<string | null>(null)
  const [showEditModal, setShowEditModal] = useState(false)
  const [editingGateway, setEditingGateway] = useState<AdminGateway | null>(null)
  const [showAccessModal, setShowAccessModal] = useState(false)
  const [accessGateway, setAccessGateway] = useState<AdminGateway | null>(null)

  useEffect(() => {
    loadGateways()
  }, [])

  async function loadGateways() {
    try {
      setLoading(true)
      const data = await getAdminGateways()
      setGateways(data)
      setError(null)
    } catch (err) {
      setError('Failed to load gateways')
    } finally {
      setLoading(false)
    }
  }

  async function handleDelete(gateway: AdminGateway) {
    if (!confirm(`Are you sure you want to delete gateway "${gateway.name}"?`)) {
      return
    }

    try {
      await deleteGateway(gateway.id)
      await loadGateways()
    } catch (err) {
      setError('Failed to delete gateway')
    }
  }

  function handleShowInstaller(gateway: AdminGateway, token?: string) {
    setSelectedGateway(gateway)
    setInstallerToken(token || null)
    setShowInstaller(true)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Gateway Management</h1>
            <p className="text-gray-500 mt-1">
              Register and manage VPN gateways for your organization.
            </p>
          </div>
          <button
            onClick={() => setShowAddModal(true)}
            className="btn btn-primary"
          >
            <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Gateway
          </button>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}

      {/* Gateways table */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : gateways.length > 0 ? (
        <div className="card p-0">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Gateway
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  VPN Settings
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Last Heartbeat
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {gateways.map((gateway) => (
                <tr key={gateway.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div>
                        <div className="text-sm font-medium text-gray-900">{gateway.name}</div>
                        <div className="text-sm text-gray-500">{gateway.hostname}</div>
                        {gateway.publicIp && (
                          <div className="text-xs text-gray-400">{gateway.publicIp}</div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      gateway.isActive
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}>
                      {gateway.isActive ? 'Online' : 'Offline'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    <div>{gateway.vpnProtocol.toUpperCase()}:{gateway.vpnPort}</div>
                    <div className="text-xs">
                      <span className={`px-1.5 py-0.5 rounded ${
                        gateway.cryptoProfile === 'fips' ? 'bg-purple-100 text-purple-700' :
                        gateway.cryptoProfile === 'compatible' ? 'bg-yellow-100 text-yellow-700' :
                        'bg-blue-100 text-blue-700'
                      }`}>
                        {gateway.cryptoProfile === 'fips' ? 'FIPS' :
                         gateway.cryptoProfile === 'compatible' ? 'Compatible' : 'Modern'}
                      </span>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {gateway.lastHeartbeat
                      ? new Date(gateway.lastHeartbeat).toLocaleString()
                      : 'Never'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Edit', icon: 'edit', onClick: () => { setEditingGateway(gateway); setShowEditModal(true) }, color: 'gray' },
                        { label: 'Access', icon: 'access', onClick: () => { setAccessGateway(gateway); setShowAccessModal(true) }, color: 'purple' },
                        { label: 'Install', icon: 'install', onClick: () => handleShowInstaller(gateway), color: 'primary' },
                        { label: 'Delete', icon: 'delete', onClick: () => handleDelete(gateway), color: 'red' },
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
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No gateways registered</h3>
          <p className="mt-2 text-gray-500">
            Get started by adding a new VPN gateway.
          </p>
          <button
            onClick={() => setShowAddModal(true)}
            className="mt-4 btn btn-primary"
          >
            Add Gateway
          </button>
        </div>
      )}

      {/* Add Gateway Modal */}
      {showAddModal && (
        <AddGatewayModal
          onClose={() => setShowAddModal(false)}
          onSuccess={(gateway) => {
            setNewGateway(gateway)
            setShowAddModal(false)
            setShowTokenModal(true)
            loadGateways()
          }}
        />
      )}

      {/* Token Display Modal */}
      {showTokenModal && newGateway && (
        <TokenModal
          gateway={newGateway}
          onClose={() => {
            setShowTokenModal(false)
            setNewGateway(null)
          }}
          onShowInstaller={(token) => {
            // Convert RegisterGatewayResponse to AdminGateway format for installer
            const gatewayForInstaller: AdminGateway = {
              id: newGateway.id,
              name: newGateway.name,
              hostname: newGateway.hostname,
              publicIp: '',
              vpnPort: newGateway.vpnPort,
              vpnProtocol: newGateway.vpnProtocol,
              cryptoProfile: 'modern', // Default for new gateways
              vpnSubnet: '172.31.255.0/24', // Default VPN subnet
              tlsAuthEnabled: true, // Default for new gateways
              fullTunnelMode: false, // Default for new gateways
              pushDns: false, // Default for new gateways
              dnsServers: [], // Default for new gateways
              isActive: false,
              lastHeartbeat: null,
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            }
            setSelectedGateway(gatewayForInstaller)
            setInstallerToken(token)
            setShowTokenModal(false)
            setNewGateway(null)
            setShowInstaller(true)
          }}
        />
      )}

      {/* Installer Modal */}
      {showInstaller && selectedGateway && (
        <InstallerModal
          gateway={selectedGateway}
          token={installerToken}
          onClose={() => {
            setShowInstaller(false)
            setSelectedGateway(null)
            setInstallerToken(null)
          }}
        />
      )}

      {/* Edit Gateway Modal */}
      {showEditModal && editingGateway && (
        <EditGatewayModal
          gateway={editingGateway}
          onClose={() => { setShowEditModal(false); setEditingGateway(null) }}
          onSuccess={() => { setShowEditModal(false); setEditingGateway(null); loadGateways() }}
        />
      )}

      {/* Access Management Modal */}
      {showAccessModal && accessGateway && (
        <GatewayAccessModal
          gateway={accessGateway}
          onClose={() => { setShowAccessModal(false); setAccessGateway(null) }}
        />
      )}
    </div>
  )
}

interface AddGatewayModalProps {
  onClose: () => void
  onSuccess: (gateway: RegisterGatewayResponse) => void
}

function AddGatewayModal({ onClose, onSuccess }: AddGatewayModalProps) {
  const [name, setName] = useState('')
  const [hostname, setHostname] = useState('')
  const [publicIp, setPublicIp] = useState('')
  const [vpnPort, setVpnPort] = useState('1194')
  const [vpnProtocol, setVpnProtocol] = useState('udp')
  const [cryptoProfile, setCryptoProfile] = useState<CryptoProfile>('modern')
  const [vpnSubnet, setVpnSubnet] = useState('172.31.255.0/24')
  const [tlsAuthEnabled, setTlsAuthEnabled] = useState(true)
  const [fullTunnelMode, setFullTunnelMode] = useState(false)
  const [pushDns, setPushDns] = useState(false)
  const [dnsServers, setDnsServers] = useState('1.1.1.1, 8.8.8.8')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    // Validate that at least one of hostname or public IP is provided
    if (!hostname && !publicIp) {
      setError('Either hostname or public IP is required')
      setSubmitting(false)
      return
    }

    try {
      const gateway = await registerGateway({
        name,
        hostname: hostname || undefined,
        public_ip: publicIp || undefined,
        vpn_port: parseInt(vpnPort) || 1194,
        vpn_protocol: vpnProtocol,
        crypto_profile: cryptoProfile,
        vpn_subnet: vpnSubnet || '172.31.255.0/24',
        tls_auth_enabled: tlsAuthEnabled,
        full_tunnel_mode: fullTunnelMode,
        push_dns: pushDns,
        dns_servers: pushDns ? dnsServers.split(',').map(s => s.trim()).filter(s => s) : [],
      })
      onSuccess(gateway)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to register gateway')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Register New Gateway</h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Gateway Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="us-east-1"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              required
            />
          </div>

          <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg mb-2">
            <p className="text-sm text-blue-700">Provide either a hostname or IP address (at least one is required)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Hostname
            </label>
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="vpn.example.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Public IP
            </label>
            <input
              type="text"
              value={publicIp}
              onChange={(e) => setPublicIp(e.target.value)}
              placeholder="203.0.113.10"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                VPN Port
              </label>
              <input
                type="number"
                value={vpnPort}
                onChange={(e) => setVpnPort(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Protocol
              </label>
              <select
                value={vpnProtocol}
                onChange={(e) => setVpnProtocol(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                <option value="udp">UDP</option>
                <option value="tcp">TCP</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Crypto Profile
            </label>
            <select
              value={cryptoProfile}
              onChange={(e) => setCryptoProfile(e.target.value as CryptoProfile)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="modern">Modern (Recommended) - AES-256-GCM, CHACHA20-POLY1305</option>
              <option value="fips">FIPS 140-3 Compliant - AES-256-GCM, AES-128-GCM</option>
              <option value="compatible">Compatible - AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC</option>
            </select>
            <p className="mt-1 text-xs text-gray-500">
              {cryptoProfile === 'fips' && 'FIPS mode uses only FIPS 140-3 validated cryptographic algorithms (AES-GCM).'}
              {cryptoProfile === 'compatible' && 'Compatible mode supports older OpenVPN 2.3.x clients with CBC fallback.'}
              {cryptoProfile === 'modern' && 'Modern mode uses the latest secure ciphers including CHACHA20-POLY1305.'}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              VPN Subnet
            </label>
            <input
              type="text"
              value={vpnSubnet}
              onChange={(e) => setVpnSubnet(e.target.value)}
              placeholder="172.31.255.0/24"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
            <p className="mt-1 text-xs text-gray-500">
              The subnet used for VPN client IP addresses. Must be a valid CIDR block.
            </p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="tlsAuthEnabled"
              checked={tlsAuthEnabled}
              onChange={(e) => setTlsAuthEnabled(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="tlsAuthEnabled" className="ml-2 block text-sm text-gray-700">
              Enable TLS Authentication
            </label>
          </div>
          <p className="text-xs text-gray-500 -mt-2">
            TLS-Auth provides additional security. Disable for simpler direct IP connections.
          </p>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="fullTunnelMode"
              checked={fullTunnelMode}
              onChange={(e) => setFullTunnelMode(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="fullTunnelMode" className="ml-2 block text-sm text-gray-700">
              Full Tunnel Mode
            </label>
          </div>
          <p className="text-xs text-gray-500 -mt-2">
            Route all client traffic through VPN. When disabled (default), only routes for allowed networks are pushed.
          </p>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="pushDns"
              checked={pushDns}
              onChange={(e) => setPushDns(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="pushDns" className="ml-2 block text-sm text-gray-700">
              Push DNS Servers
            </label>
          </div>
          <p className="text-xs text-gray-500 -mt-2">
            Push DNS servers to clients. When disabled (default), client DNS settings are not changed.
          </p>

          {pushDns && (
            <div>
              <label htmlFor="dnsServers" className="block text-sm font-medium text-gray-700">
                DNS Servers
              </label>
              <input
                type="text"
                id="dnsServers"
                value={dnsServers}
                onChange={(e) => setDnsServers(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
                placeholder="1.1.1.1, 8.8.8.8"
              />
              <p className="mt-1 text-xs text-gray-500">
                Comma-separated list of DNS server IPs to push to clients.
              </p>
            </div>
          )}

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
              {submitting ? 'Registering...' : 'Register Gateway'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

interface TokenModalProps {
  gateway: RegisterGatewayResponse
  onClose: () => void
  onShowInstaller: (token: string) => void
}

function TokenModal({ gateway, onClose, onShowInstaller }: TokenModalProps) {
  const [copied, setCopied] = useState(false)

  function copyToken() {
    navigator.clipboard.writeText(gateway.token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
        <div className="text-center mb-4">
          <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100 mb-4">
            <svg className="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-gray-900">Gateway Registered!</h2>
        </div>

        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
          <div className="flex">
            <svg className="h-5 w-5 text-yellow-400 mr-2 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div>
              <h3 className="text-sm font-medium text-yellow-800">Save this token!</h3>
              <p className="text-sm text-yellow-700 mt-1">
                This token will only be shown once. You'll need it to configure the gateway agent.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Gateway Name</label>
            <div className="text-sm text-gray-900">{gateway.name}</div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Authentication Token</label>
            <div className="flex">
              <input
                type="text"
                readOnly
                value={gateway.token}
                className="flex-1 px-3 py-2 bg-gray-100 border border-gray-300 rounded-l-lg text-sm font-mono"
              />
              <button
                onClick={copyToken}
                className="px-4 py-2 bg-primary-600 text-white rounded-r-lg hover:bg-primary-700"
              >
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
          </div>
        </div>

        <div className="mt-6 flex justify-end space-x-3">
          <button onClick={onClose} className="btn btn-secondary">
            Done
          </button>
          <button
            onClick={() => onShowInstaller(gateway.token)}
            className="btn btn-primary inline-flex items-center"
          >
            <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            View Installer
          </button>
        </div>
      </div>
    </div>
  )
}

interface InstallerModalProps {
  gateway: AdminGateway
  token: string | null
  onClose: () => void
}

function InstallerModal({ gateway, token, onClose }: InstallerModalProps) {
  const serverUrl = window.location.origin
  const tokenValue = token || 'YOUR_GATEWAY_TOKEN'
  const hasToken = !!token

  const installCommand = `curl -sSL ${serverUrl}/scripts/install-gateway.sh | sudo bash -s -- \\
  --server ${serverUrl} \\
  --token ${tokenValue} \\
  --name ${gateway.name}`

  const [copied, setCopied] = useState(false)

  function copyCommand() {
    navigator.clipboard.writeText(installCommand)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Install Gateway: {gateway.name}
        </h2>

        <div className="space-y-4">
          {!hasToken && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <div className="flex">
                <svg className="h-5 w-5 text-yellow-400 mr-2 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div>
                  <h3 className="text-sm font-medium text-yellow-800">Token Required</h3>
                  <p className="text-sm text-yellow-700 mt-1">
                    Replace <code className="bg-yellow-100 px-1">YOUR_GATEWAY_TOKEN</code> with the token you received when registering this gateway.
                  </p>
                </div>
              </div>
            </div>
          )}

          {hasToken && (
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <div className="flex">
                <svg className="h-5 w-5 text-green-400 mr-2 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <div>
                  <h3 className="text-sm font-medium text-green-800">Token Included</h3>
                  <p className="text-sm text-green-700 mt-1">
                    The gateway token is already included in the command below. Copy and run it on your gateway server.
                  </p>
                </div>
              </div>
            </div>
          )}

          <div>
            <h3 className="text-sm font-medium text-gray-700 mb-2">Quick Install</h3>
            <p className="text-sm text-gray-500 mb-2">
              Run this command on your gateway server:
            </p>
            <div className="relative">
              <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto whitespace-pre-wrap break-all">
                {installCommand}
              </pre>
              <button
                onClick={copyCommand}
                className="absolute top-2 right-2 px-3 py-1.5 bg-gray-700 text-gray-200 text-xs rounded hover:bg-gray-600 flex items-center"
              >
                {copied ? (
                  <>
                    <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    Copied!
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                    Copy
                  </>
                )}
              </button>
            </div>
          </div>

          <div>
            <h3 className="text-sm font-medium text-gray-700 mb-2">Manual Installation</h3>
            <ol className="text-sm text-gray-600 space-y-2 list-decimal list-inside">
              <li>
                Download the gateway binary:
                <div className="mt-1 ml-5 space-x-2">
                  <a href="/downloads/gatekey-gateway-linux-amd64" className="text-primary-600 hover:underline text-xs">
                    Linux AMD64
                  </a>
                  <a href="/downloads/gatekey-gateway-linux-arm64" className="text-primary-600 hover:underline text-xs">
                    Linux ARM64
                  </a>
                </div>
              </li>
              <li>Move to <code className="bg-gray-100 px-1 text-xs">/usr/local/bin/gatekey-gateway</code></li>
              <li>Create config at <code className="bg-gray-100 px-1 text-xs">/etc/gatekey/gateway.yaml</code></li>
              <li>Configure systemd service and start</li>
            </ol>
          </div>

          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <h3 className="text-sm font-medium text-blue-800 mb-2">Requirements</h3>
            <ul className="text-sm text-blue-700 space-y-1">
              <li>- Ubuntu 20.04+, Debian 11+, RHEL 8+, or Fedora 35+</li>
              <li>- Root/sudo access</li>
              <li>- Outbound HTTPS access to the control plane</li>
              <li>- Inbound access on VPN port ({gateway.vpnPort}/{gateway.vpnProtocol})</li>
            </ul>
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <button onClick={onClose} className="btn btn-secondary">
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

interface EditGatewayModalProps {
  gateway: AdminGateway
  onClose: () => void
  onSuccess: () => void
}

function EditGatewayModal({ gateway, onClose, onSuccess }: EditGatewayModalProps) {
  const [name, setName] = useState(gateway.name)
  const [hostname, setHostname] = useState(gateway.hostname)
  const [publicIp, setPublicIp] = useState(gateway.publicIp || '')
  const [vpnPort, setVpnPort] = useState(gateway.vpnPort.toString())
  const [vpnProtocol, setVpnProtocol] = useState(gateway.vpnProtocol)
  const [cryptoProfile, setCryptoProfile] = useState<CryptoProfile>(gateway.cryptoProfile || 'modern')
  const [vpnSubnet, setVpnSubnet] = useState(gateway.vpnSubnet || '10.8.0.0/24')
  const [tlsAuthEnabled, setTlsAuthEnabled] = useState(gateway.tlsAuthEnabled ?? true)
  const [fullTunnelMode, setFullTunnelMode] = useState(gateway.fullTunnelMode ?? false)
  const [pushDns, setPushDns] = useState(gateway.pushDns ?? false)
  const [dnsServers, setDnsServers] = useState((gateway.dnsServers ?? []).join(', ') || '1.1.1.1, 8.8.8.8')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    // Validate that at least one of hostname or public IP is provided
    if (!hostname && !publicIp) {
      setError('Either hostname or public IP is required')
      setSubmitting(false)
      return
    }

    try {
      await updateGateway(gateway.id, {
        name,
        hostname: hostname || undefined,
        public_ip: publicIp || undefined,
        vpn_port: parseInt(vpnPort) || 1194,
        vpn_protocol: vpnProtocol,
        crypto_profile: cryptoProfile,
        vpn_subnet: vpnSubnet || '172.31.255.0/24',
        tls_auth_enabled: tlsAuthEnabled,
        full_tunnel_mode: fullTunnelMode,
        push_dns: pushDns,
        dns_servers: pushDns ? dnsServers.split(',').map(s => s.trim()).filter(s => s) : [],
      })
      onSuccess()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to update gateway')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Edit Gateway</h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Gateway Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              required
            />
          </div>

          <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg mb-2">
            <p className="text-sm text-blue-700">Provide either a hostname or IP address (at least one is required)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Hostname
            </label>
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Public IP
            </label>
            <input
              type="text"
              value={publicIp}
              onChange={(e) => setPublicIp(e.target.value)}
              placeholder="203.0.113.10"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                VPN Port
              </label>
              <input
                type="number"
                value={vpnPort}
                onChange={(e) => setVpnPort(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Protocol
              </label>
              <select
                value={vpnProtocol}
                onChange={(e) => setVpnProtocol(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                <option value="udp">UDP</option>
                <option value="tcp">TCP</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Crypto Profile
            </label>
            <select
              value={cryptoProfile}
              onChange={(e) => setCryptoProfile(e.target.value as CryptoProfile)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="modern">Modern (Recommended) - AES-256-GCM, CHACHA20-POLY1305</option>
              <option value="fips">FIPS 140-3 Compliant - AES-256-GCM, AES-128-GCM</option>
              <option value="compatible">Compatible - AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC</option>
            </select>
            <p className="mt-1 text-xs text-gray-500">
              {cryptoProfile === 'fips' && 'FIPS mode uses only FIPS 140-3 validated cryptographic algorithms (AES-GCM).'}
              {cryptoProfile === 'compatible' && 'Compatible mode supports older OpenVPN 2.3.x clients with CBC fallback.'}
              {cryptoProfile === 'modern' && 'Modern mode uses the latest secure ciphers including CHACHA20-POLY1305.'}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              VPN Subnet
            </label>
            <input
              type="text"
              value={vpnSubnet}
              onChange={(e) => setVpnSubnet(e.target.value)}
              placeholder="172.31.255.0/24"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
            <p className="mt-1 text-xs text-gray-500">
              The subnet used for VPN client IP addresses. Must be a valid CIDR block.
            </p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="editTlsAuthEnabled"
              checked={tlsAuthEnabled}
              onChange={(e) => setTlsAuthEnabled(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="editTlsAuthEnabled" className="ml-2 block text-sm text-gray-700">
              Enable TLS Authentication
            </label>
          </div>
          <p className="text-xs text-gray-500 -mt-2">
            TLS-Auth provides additional security. Disable for simpler direct IP connections.
          </p>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="editFullTunnelMode"
              checked={fullTunnelMode}
              onChange={(e) => setFullTunnelMode(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="editFullTunnelMode" className="ml-2 block text-sm text-gray-700">
              Full Tunnel Mode
            </label>
          </div>
          <p className="text-xs text-gray-500 -mt-2">
            Route all client traffic through VPN. When disabled (default), only routes for allowed networks are pushed.
          </p>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="editPushDns"
              checked={pushDns}
              onChange={(e) => setPushDns(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="editPushDns" className="ml-2 block text-sm text-gray-700">
              Push DNS Servers
            </label>
          </div>
          <p className="text-xs text-gray-500 -mt-2">
            Push DNS servers to clients. When disabled (default), client DNS settings are not changed.
          </p>

          {pushDns && (
            <div>
              <label htmlFor="editDnsServers" className="block text-sm font-medium text-gray-700">
                DNS Servers
              </label>
              <input
                type="text"
                id="editDnsServers"
                value={dnsServers}
                onChange={(e) => setDnsServers(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
                placeholder="1.1.1.1, 8.8.8.8"
              />
              <p className="mt-1 text-xs text-gray-500">
                Comma-separated list of DNS server IPs to push to clients.
              </p>
            </div>
          )}

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
              {submitting ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

interface GatewayAccessModalProps {
  gateway: AdminGateway
  onClose: () => void
}

function GatewayAccessModal({ gateway, onClose }: GatewayAccessModalProps) {
  const [activeTab, setActiveTab] = useState<'networks' | 'users' | 'groups'>('networks')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Networks state
  const [networks, setNetworks] = useState<Network[]>([])
  const [allNetworks, setAllNetworks] = useState<Network[]>([])
  const [selectedNetwork, setSelectedNetwork] = useState('')

  // Users state
  const [users, setUsers] = useState<GatewayUser[]>([])
  const [newUserId, setNewUserId] = useState('')

  // Groups state
  const [groups, setGroups] = useState<GatewayGroup[]>([])
  const [newGroupName, setNewGroupName] = useState('')

  useEffect(() => {
    loadData()
  }, [])

  async function loadData() {
    setLoading(true)
    setError(null)
    try {
      const [networksData, allNetworksData, usersData, groupsData] = await Promise.all([
        getGatewayNetworks(gateway.id),
        getNetworks(),
        getGatewayUsers(gateway.id),
        getGatewayGroups(gateway.id),
      ])
      setNetworks(networksData)
      setAllNetworks(allNetworksData)
      setUsers(usersData)
      setGroups(groupsData)
    } catch (err) {
      setError('Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  // Network functions
  const assignedNetworkIds = new Set(networks.map(n => n.id))
  const availableNetworks = allNetworks.filter(n => !assignedNetworkIds.has(n.id))

  async function handleAssignNetwork() {
    if (!selectedNetwork) return
    try {
      await assignGatewayToNetwork(gateway.id, selectedNetwork)
      setSelectedNetwork('')
      await loadData()
    } catch (err) {
      setError('Failed to assign network')
    }
  }

  async function handleRemoveNetwork(networkId: string) {
    try {
      await removeGatewayFromNetwork(gateway.id, networkId)
      await loadData()
    } catch (err) {
      setError('Failed to remove network')
    }
  }

  // User functions
  async function handleAssignUser() {
    if (!newUserId.trim()) return
    try {
      await assignUserToGateway(gateway.id, newUserId.trim())
      setNewUserId('')
      await loadData()
    } catch (err) {
      setError('Failed to assign user')
    }
  }

  async function handleRemoveUser(userId: string) {
    try {
      await removeUserFromGateway(gateway.id, userId)
      await loadData()
    } catch (err) {
      setError('Failed to remove user')
    }
  }

  // Group functions
  async function handleAssignGroup() {
    if (!newGroupName.trim()) return
    try {
      await assignGroupToGateway(gateway.id, newGroupName.trim())
      setNewGroupName('')
      await loadData()
    } catch (err) {
      setError('Failed to assign group')
    }
  }

  async function handleRemoveGroup(groupName: string) {
    try {
      await removeGroupFromGateway(gateway.id, groupName)
      await loadData()
    } catch (err) {
      setError('Failed to remove group')
    }
  }

  const tabs = [
    { id: 'networks' as const, label: 'Networks', count: networks.length },
    { id: 'users' as const, label: 'Users', count: users.length },
    { id: 'groups' as const, label: 'Groups', count: groups.length },
  ]

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Manage Access: {gateway.name}
        </h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-red-700 text-sm">
            {error}
          </div>
        )}

        {/* Tabs */}
        <div className="border-b border-gray-200 mb-4">
          <nav className="-mb-px flex space-x-8">
            {tabs.map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
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
          <div className="space-y-4">
            {/* Networks Tab */}
            {activeTab === 'networks' && (
              <>
                <div className="flex space-x-2">
                  <select
                    value={selectedNetwork}
                    onChange={(e) => setSelectedNetwork(e.target.value)}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                  >
                    <option value="">Select a network to add...</option>
                    {availableNetworks.map(n => (
                      <option key={n.id} value={n.id}>{n.name} ({n.cidr})</option>
                    ))}
                  </select>
                  <button
                    onClick={handleAssignNetwork}
                    disabled={!selectedNetwork}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
                <div className="space-y-2">
                  {networks.length === 0 ? (
                    <p className="text-gray-500 text-sm py-4 text-center">No networks assigned</p>
                  ) : (
                    networks.map(network => (
                      <div key={network.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                        <div>
                          <div className="font-medium text-sm">{network.name}</div>
                          <div className="text-xs text-gray-500">{network.cidr}</div>
                        </div>
                        <button
                          onClick={() => handleRemoveNetwork(network.id)}
                          className="text-red-600 hover:text-red-800 text-sm"
                        >
                          Remove
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </>
            )}

            {/* Users Tab */}
            {activeTab === 'users' && (
              <>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={newUserId}
                    onChange={(e) => setNewUserId(e.target.value)}
                    placeholder="Enter user ID or email..."
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                    onKeyPress={(e) => e.key === 'Enter' && handleAssignUser()}
                  />
                  <button
                    onClick={handleAssignUser}
                    disabled={!newUserId.trim()}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
                <div className="space-y-2">
                  {users.length === 0 ? (
                    <p className="text-gray-500 text-sm py-4 text-center">No users assigned</p>
                  ) : (
                    users.map(user => (
                      <div key={user.userId} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                        <div>
                          <div className="font-medium text-sm">{user.email || user.userId}</div>
                          {user.name && <div className="text-xs text-gray-500">{user.name}</div>}
                        </div>
                        <button
                          onClick={() => handleRemoveUser(user.userId)}
                          className="text-red-600 hover:text-red-800 text-sm"
                        >
                          Remove
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </>
            )}

            {/* Groups Tab */}
            {activeTab === 'groups' && (
              <>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={newGroupName}
                    onChange={(e) => setNewGroupName(e.target.value)}
                    placeholder="Enter group name..."
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                    onKeyPress={(e) => e.key === 'Enter' && handleAssignGroup()}
                  />
                  <button
                    onClick={handleAssignGroup}
                    disabled={!newGroupName.trim()}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
                <div className="space-y-2">
                  {groups.length === 0 ? (
                    <p className="text-gray-500 text-sm py-4 text-center">No groups assigned</p>
                  ) : (
                    groups.map(group => (
                      <div key={group.groupName} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                        <div className="font-medium text-sm">{group.groupName}</div>
                        <button
                          onClick={() => handleRemoveGroup(group.groupName)}
                          className="text-red-600 hover:text-red-800 text-sm"
                        >
                          Remove
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </>
            )}
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
