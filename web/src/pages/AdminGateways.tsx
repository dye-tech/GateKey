import { useState, useEffect } from 'react'
import {
  getAdminGateways, registerGateway, deleteGateway, updateGateway, reprovisionGateway, rotateGatewayToken,
  getGatewayNetworks, getNetworks, assignGatewayToNetwork, removeGatewayFromNetwork,
  getGatewayUsers, assignUserToGateway, removeUserFromGateway,
  getGatewayGroups, assignGroupToGateway, removeGroupFromGateway,
  AdminGateway, RegisterGatewayResponse, Network, GatewayUser, GatewayGroup, CryptoProfile, GatewayType, RotateTokenResponse
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
  const [showRotatedTokenModal, setShowRotatedTokenModal] = useState(false)
  const [rotatedToken, setRotatedToken] = useState<RotateTokenResponse | null>(null)

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

  async function handleReprovision(gateway: AdminGateway) {
    if (!confirm(
      `Re-provision gateway "${gateway.name}"?\n\n` +
      `WARNING: This will regenerate all certificates. ` +
      `Existing client VPN configs will stop working and users must download new configs.`
    )) {
      return
    }

    try {
      await reprovisionGateway(gateway.id)
      await loadGateways()
      setError(null)
    } catch (err) {
      setError('Failed to trigger reprovision')
    }
  }

  function handleShowInstaller(gateway: AdminGateway, token?: string) {
    setSelectedGateway(gateway)
    setInstallerToken(token || null)
    setShowInstaller(true)
  }

  async function handleRotateToken(gateway: AdminGateway) {
    if (!confirm(
      `Rotate token for gateway "${gateway.name}"?\n\n` +
      `WARNING: The current token will be immediately invalidated. ` +
      `You will need to update the gateway configuration with the new token.`
    )) {
      return
    }

    try {
      const response = await rotateGatewayToken(gateway.id)
      setRotatedToken(response)
      setSelectedGateway(gateway)
      setShowRotatedTokenModal(true)
      setError(null)
    } catch (err) {
      setError('Failed to rotate token')
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-theme-primary">Gateway Management</h1>
            <p className="text-theme-tertiary mt-1">
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
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-theme rounded-lg text-red-700 dark:text-red-400">
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
          <table className="min-w-full divide-y divide-theme">
            <thead className="bg-theme-tertiary">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Gateway
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  VPN Settings
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Last Heartbeat
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-theme-tertiary uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-theme-card divide-y divide-theme">
              {gateways.map((gateway) => (
                <tr key={gateway.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div>
                        <div className="text-sm font-medium text-theme-primary">{gateway.name}</div>
                        <div className="text-sm text-theme-tertiary">{gateway.hostname}</div>
                        {gateway.publicIp && (
                          <div className="text-xs text-theme-muted">{gateway.publicIp}</div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      gateway.gatewayType === 'wireguard'
                        ? 'bg-purple-600 text-white'
                        : 'bg-blue-600 text-white'
                    }`}>
                      {gateway.gatewayType === 'wireguard' ? 'WireGuard' : 'OpenVPN'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      gateway.isActive
                        ? 'bg-green-600 text-white'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300'
                    }`}>
                      {gateway.isActive ? 'Online' : 'Offline'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                    <div>{gateway.gatewayType === 'wireguard' ? 'UDP' : gateway.vpnProtocol.toUpperCase()}:{gateway.vpnPort}</div>
                    {gateway.gatewayType === 'openvpn' && (
                      <div className="text-xs">
                        <span className={`px-1.5 py-0.5 rounded ${
                          gateway.cryptoProfile === 'fips' ? 'bg-purple-600 text-white' :
                          gateway.cryptoProfile === 'compatible' ? 'bg-yellow-500 text-white' :
                          'bg-blue-600 text-white'
                        }`}>
                          {gateway.cryptoProfile === 'fips' ? 'FIPS' :
                           gateway.cryptoProfile === 'compatible' ? 'Compatible' : 'Modern'}
                        </span>
                      </div>
                    )}
                    {gateway.gatewayType === 'wireguard' && (
                      <div className="text-xs text-theme-muted">
                        ChaCha20-Poly1305
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                    {gateway.lastHeartbeat
                      ? new Date(gateway.lastHeartbeat).toLocaleString()
                      : 'Never'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <ActionDropdown
                      actions={[
                        { label: 'Edit', icon: 'edit', onClick: () => { setEditingGateway(gateway); setShowEditModal(true) }, color: 'gray' },
                        { label: 'Access', icon: 'access', onClick: () => { setAccessGateway(gateway); setShowAccessModal(true) }, color: 'purple' },
                        { label: 'Re-provision', icon: 'install', onClick: () => handleReprovision(gateway) },
                        { label: 'Rotate Token', icon: 'key', onClick: () => handleRotateToken(gateway), color: 'yellow' },
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
          <svg className="mx-auto h-12 w-12 text-theme-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-theme-primary">No gateways registered</h3>
          <p className="mt-2 text-theme-tertiary">
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
              sessionEnabled: true, // Default for new gateways
              gatewayType: newGateway.gatewayType || 'openvpn',
              wgPublicKey: newGateway.wgPublicKey,
              wgListenPort: newGateway.wgListenPort,
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

      {/* Rotated Token Modal */}
      {showRotatedTokenModal && rotatedToken && selectedGateway && (
        <RotatedTokenModal
          gatewayName={selectedGateway.name}
          token={rotatedToken.token}
          installScript={rotatedToken.installScript}
          onClose={() => {
            setShowRotatedTokenModal(false)
            setRotatedToken(null)
            setSelectedGateway(null)
          }}
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
  const [gatewayType, setGatewayType] = useState<GatewayType>('openvpn')
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
  const [sessionEnabled, setSessionEnabled] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Update port when gateway type changes
  useEffect(() => {
    if (gatewayType === 'wireguard') {
      setVpnPort('51820')
    } else {
      setVpnPort('1194')
    }
  }, [gatewayType])

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
        vpn_port: parseInt(vpnPort) || (gatewayType === 'wireguard' ? 51820 : 1194),
        vpn_protocol: gatewayType === 'wireguard' ? 'udp' : vpnProtocol,
        crypto_profile: gatewayType === 'wireguard' ? undefined : cryptoProfile,
        vpn_subnet: vpnSubnet || '172.31.255.0/24',
        tls_auth_enabled: gatewayType === 'wireguard' ? undefined : tlsAuthEnabled,
        full_tunnel_mode: fullTunnelMode,
        push_dns: pushDns,
        dns_servers: pushDns ? dnsServers.split(',').map(s => s.trim()).filter(s => s) : [],
        session_enabled: sessionEnabled,
        gateway_type: gatewayType,
        wg_listen_port: gatewayType === 'wireguard' ? parseInt(vpnPort) || 51820 : undefined,
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
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <h2 className="text-xl font-semibold text-theme-primary mb-4">Register New Gateway</h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-theme rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Gateway Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="us-east-1"
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Gateway Type *
            </label>
            <div className="flex space-x-2">
              <button
                type="button"
                onClick={() => setGatewayType('openvpn')}
                className={`flex-1 px-4 py-2 rounded-lg border font-medium text-sm transition-colors ${
                  gatewayType === 'openvpn'
                    ? 'bg-blue-600 text-white border-blue-600'
                    : 'bg-theme-card text-theme-secondary border-theme hover:bg-theme-tertiary'
                }`}
              >
                <svg className="w-4 h-4 inline mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
                OpenVPN
              </button>
              <button
                type="button"
                onClick={() => setGatewayType('wireguard')}
                className={`flex-1 px-4 py-2 rounded-lg border font-medium text-sm transition-colors ${
                  gatewayType === 'wireguard'
                    ? 'bg-purple-600 text-white border-purple-600'
                    : 'bg-theme-card text-theme-secondary border-theme hover:bg-theme-tertiary'
                }`}
              >
                <svg className="w-4 h-4 inline mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                WireGuard
              </button>
            </div>
            <p className="mt-1 text-xs text-theme-tertiary">
              {gatewayType === 'openvpn' && 'OpenVPN provides wide client compatibility with TCP/UDP options (FIPS compliant).'}
              {gatewayType === 'wireguard' && 'WireGuard is a modern, high-performance VPN protocol (faster, not FIPS compliant).'}
            </p>
          </div>

          <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg mb-2">
            <p className="text-sm text-gray-800 dark:text-blue-400">Provide either a hostname or IP address (at least one is required)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Hostname
            </label>
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="vpn.example.com"
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Public IP
            </label>
            <input
              type="text"
              value={publicIp}
              onChange={(e) => setPublicIp(e.target.value)}
              placeholder="203.0.113.10"
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div className={gatewayType === 'openvpn' ? 'grid grid-cols-2 gap-4' : ''}>
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-1">
                VPN Port
              </label>
              <input
                type="number"
                value={vpnPort}
                onChange={(e) => setVpnPort(e.target.value)}
                className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
              {gatewayType === 'wireguard' && (
                <p className="mt-1 text-xs text-theme-tertiary">
                  WireGuard uses UDP only. Default port is 51820.
                </p>
              )}
            </div>

            {gatewayType === 'openvpn' && (
              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">
                  Protocol
                </label>
                <select
                  value={vpnProtocol}
                  onChange={(e) => setVpnProtocol(e.target.value)}
                  className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="udp">UDP</option>
                  <option value="tcp">TCP</option>
                </select>
              </div>
            )}
          </div>

          {gatewayType === 'openvpn' && (
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-1">
                Crypto Profile
              </label>
              <select
                value={cryptoProfile}
                onChange={(e) => setCryptoProfile(e.target.value as CryptoProfile)}
                className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                <option value="modern">Modern (Recommended) - AES-256-GCM, CHACHA20-POLY1305</option>
                <option value="fips">FIPS 140-3 Compliant - AES-256-GCM, AES-128-GCM</option>
                <option value="compatible">Compatible - AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC</option>
              </select>
              <p className="mt-1 text-xs text-theme-tertiary">
                {cryptoProfile === 'fips' && 'FIPS mode uses only FIPS 140-3 validated cryptographic algorithms (AES-GCM).'}
                {cryptoProfile === 'compatible' && 'Compatible mode supports older OpenVPN 2.3.x clients with CBC fallback.'}
                {cryptoProfile === 'modern' && 'Modern mode uses the latest secure ciphers including CHACHA20-POLY1305.'}
              </p>
            </div>
          )}

          {gatewayType === 'wireguard' && (
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-1">
                Crypto Profile
              </label>
              <div className="w-full px-3 py-2 bg-theme-tertiary border border-theme rounded-lg text-theme-secondary">
                ChaCha20-Poly1305 (WireGuard Default)
              </div>
              <p className="mt-1 text-xs text-theme-tertiary">
                WireGuard uses a fixed cryptographic protocol with ChaCha20-Poly1305 for symmetric encryption, Curve25519 for key exchange, and BLAKE2s for hashing.
              </p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              VPN Subnet
            </label>
            <input
              type="text"
              value={vpnSubnet}
              onChange={(e) => setVpnSubnet(e.target.value)}
              placeholder="172.31.255.0/24"
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
            <p className="mt-1 text-xs text-theme-tertiary">
              The subnet used for VPN client IP addresses. Must be a valid CIDR block.
            </p>
          </div>

          {gatewayType === 'openvpn' && (
            <>
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="tlsAuthEnabled"
                  checked={tlsAuthEnabled}
                  onChange={(e) => setTlsAuthEnabled(e.target.checked)}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
                />
                <label htmlFor="tlsAuthEnabled" className="ml-2 block text-sm text-theme-secondary">
                  Enable TLS Authentication
                </label>
              </div>
              <p className="text-xs text-theme-tertiary -mt-2">
                TLS-Auth provides additional security. Disable for simpler direct IP connections.
              </p>
            </>
          )}

          <div className="flex items-center">
            <input
              type="checkbox"
              id="fullTunnelMode"
              checked={fullTunnelMode}
              onChange={(e) => setFullTunnelMode(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
            />
            <label htmlFor="fullTunnelMode" className="ml-2 block text-sm text-theme-secondary">
              Full Tunnel Mode
            </label>
          </div>
          <p className="text-xs text-theme-tertiary -mt-2">
            Route all client traffic through VPN. When disabled (default), only routes for allowed networks are pushed.
          </p>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="pushDns"
              checked={pushDns}
              onChange={(e) => setPushDns(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
            />
            <label htmlFor="pushDns" className="ml-2 block text-sm text-theme-secondary">
              Push DNS Servers
            </label>
          </div>
          <p className="text-xs text-theme-tertiary -mt-2">
            Push DNS servers to clients. When disabled (default), client DNS settings are not changed.
          </p>

          {pushDns && (
            <div>
              <label htmlFor="dnsServers" className="block text-sm font-medium text-theme-secondary">
                DNS Servers
              </label>
              <input
                type="text"
                id="dnsServers"
                value={dnsServers}
                onChange={(e) => setDnsServers(e.target.value)}
                className="mt-1 block w-full rounded-md border-theme shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
                placeholder="1.1.1.1, 8.8.8.8"
              />
              <p className="mt-1 text-xs text-theme-tertiary">
                Comma-separated list of DNS server IPs to push to clients.
              </p>
            </div>
          )}

          <div className="flex items-center">
            <input
              type="checkbox"
              id="sessionEnabled"
              checked={sessionEnabled}
              onChange={(e) => setSessionEnabled(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
            />
            <label htmlFor="sessionEnabled" className="ml-2 block text-sm text-theme-secondary">
              Enable Remote Sessions
            </label>
          </div>
          <p className="text-xs text-theme-tertiary -mt-2">
            Allow administrators to run commands on this gateway via the Remote Sessions page.
          </p>

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
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
        <div className="text-center mb-4">
          <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-blue-100 dark:bg-green-900/30 mb-4">
            <svg className="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-theme-primary">Gateway Registered!</h2>
        </div>

        <div className="bg-slate-100 dark:bg-slate-800 border-l-4 border-l-indigo-500 dark:border-l-indigo-400 rounded-lg p-4 mb-4">
          <div className="flex">
            <svg className="h-5 w-5 text-indigo-600 dark:text-indigo-400 mr-2 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div>
              <h3 className="text-sm font-medium text-slate-900 dark:text-slate-100">Save this token!</h3>
              <p className="text-sm text-slate-600 dark:text-slate-300 mt-1">
                This token will only be shown once. You'll need it to configure the gateway agent.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">Gateway Name</label>
            <div className="text-sm text-theme-primary">{gateway.name}</div>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">Authentication Token</label>
            <div className="flex">
              <input
                type="text"
                readOnly
                value={gateway.token}
                className="flex-1 px-3 py-2 bg-gray-100 dark:bg-gray-800 border border-theme rounded-l-lg text-sm font-mono text-gray-900 dark:text-gray-100"
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
  const isWireGuard = gateway.gatewayType === 'wireguard'

  const installScript = isWireGuard ? 'install-wireguard-gateway.sh' : 'install-gateway.sh'
  const installCommand = `curl -sSL ${serverUrl}/scripts/${installScript} | sudo bash -s -- \\
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
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-theme-primary mb-4">
          Install Gateway: {gateway.name}
        </h2>

        <div className="space-y-4">
          {!hasToken && (
            <div className="instruction-box">
              <div className="flex">
                <svg className="h-5 w-5 flex-shrink-0 mt-0.5" style={{ color: 'var(--instruction-accent)' }} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div className="ml-3">
                  <h3 className="text-sm font-medium instruction-box-title" style={{ color: 'var(--instruction-title)' }}>Token Required</h3>
                  <p className="text-sm mt-1 instruction-box-text">
                    Replace <code className="bg-theme-tertiary px-1.5 py-0.5 rounded text-theme-primary font-mono text-xs">YOUR_GATEWAY_TOKEN</code> with the token you received when registering this gateway.
                  </p>
                </div>
              </div>
            </div>
          )}

          {hasToken && (
            <div className="bg-gray-50 dark:bg-gray-800 border-l-4 border-l-slate-400 dark:border-l-slate-500 rounded-lg p-4">
              <div className="flex">
                <svg className="h-5 w-5 text-slate-600 dark:text-slate-400 mr-2 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <div>
                  <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Token Included</h3>
                  <p className="text-sm text-gray-600 dark:text-gray-300 mt-1">
                    The gateway token is already included in the command below. Copy and run it on your gateway server.
                  </p>
                </div>
              </div>
            </div>
          )}

          <div>
            <h3 className="text-sm font-medium text-theme-secondary mb-2">Quick Install</h3>
            <p className="text-sm text-theme-tertiary mb-2">
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
            <h3 className="text-sm font-medium text-theme-secondary mb-2">Manual Installation</h3>
            <ol className="text-sm text-theme-secondary space-y-2 list-decimal list-inside">
              <li>
                Download the {isWireGuard ? 'WireGuard gateway' : 'gateway'} binary:
                <div className="mt-1 ml-5 space-x-2">
                  <a href={`/downloads/gatekey-${isWireGuard ? 'wireguard-gateway' : 'gateway'}-linux-amd64`} className="text-primary-600 hover:underline text-xs">
                    Linux AMD64
                  </a>
                  <a href={`/downloads/gatekey-${isWireGuard ? 'wireguard-gateway' : 'gateway'}-linux-arm64`} className="text-primary-600 hover:underline text-xs">
                    Linux ARM64
                  </a>
                </div>
              </li>
              <li>Move to <code className="bg-theme-tertiary px-1.5 py-0.5 rounded text-theme-primary font-mono text-xs">/usr/local/bin/gatekey-{isWireGuard ? 'wireguard-gateway' : 'gateway'}</code></li>
              <li>Create config at <code className="bg-theme-tertiary px-1.5 py-0.5 rounded text-theme-primary font-mono text-xs">/etc/gatekey/{isWireGuard ? 'wireguard-gateway' : 'gateway'}.yaml</code></li>
              <li>Configure systemd service and start</li>
            </ol>
          </div>

          <div className="info-box">
            <h3 className="text-sm font-semibold text-theme-primary mb-2">Requirements</h3>
            <ul className="text-sm info-box-text space-y-1">
              <li>• Ubuntu 20.04+, Debian 11+, RHEL 8+, or Fedora 35+</li>
              <li>• Root/sudo access</li>
              {isWireGuard && <li>• wireguard-tools package (wg, wg-quick)</li>}
              <li>• Outbound HTTPS access to the control plane</li>
              <li>• Inbound access on VPN port ({gateway.vpnPort}/{isWireGuard ? 'UDP' : gateway.vpnProtocol})</li>
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
  const [sessionEnabled, setSessionEnabled] = useState(gateway.sessionEnabled ?? true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isWireGuard = gateway.gatewayType === 'wireguard'

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
        session_enabled: sessionEnabled,
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
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <h2 className="text-xl font-semibold text-theme-primary mb-4">Edit Gateway</h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-theme rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Gateway Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              required
            />
          </div>

          <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg mb-2">
            <p className="text-sm text-gray-800 dark:text-blue-400">Provide either a hostname or IP address (at least one is required)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Hostname
            </label>
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              Public IP
            </label>
            <input
              type="text"
              value={publicIp}
              onChange={(e) => setPublicIp(e.target.value)}
              placeholder="203.0.113.10"
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div className={isWireGuard ? '' : 'grid grid-cols-2 gap-4'}>
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-1">
                VPN Port
              </label>
              <input
                type="number"
                value={vpnPort}
                onChange={(e) => setVpnPort(e.target.value)}
                className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
              {isWireGuard && (
                <p className="mt-1 text-xs text-theme-tertiary">
                  WireGuard uses UDP only. Default port is 51820.
                </p>
              )}
            </div>

            {!isWireGuard && (
              <div>
                <label className="block text-sm font-medium text-theme-secondary mb-1">
                  Protocol
                </label>
                <select
                  value={vpnProtocol}
                  onChange={(e) => setVpnProtocol(e.target.value)}
                  className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="udp">UDP</option>
                  <option value="tcp">TCP</option>
                </select>
              </div>
            )}
          </div>

          {isWireGuard ? (
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-1">
                Crypto Profile
              </label>
              <div className="w-full px-3 py-2 bg-theme-tertiary border border-theme rounded-lg text-theme-secondary">
                ChaCha20-Poly1305 (WireGuard Default)
              </div>
              <p className="mt-1 text-xs text-theme-tertiary">
                WireGuard uses a fixed cryptographic protocol with ChaCha20-Poly1305 for symmetric encryption, Curve25519 for key exchange, and BLAKE2s for hashing.
              </p>
            </div>
          ) : (
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-1">
                Crypto Profile
              </label>
              <select
                value={cryptoProfile}
                onChange={(e) => setCryptoProfile(e.target.value as CryptoProfile)}
                className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                <option value="modern">Modern (Recommended) - AES-256-GCM, CHACHA20-POLY1305</option>
                <option value="fips">FIPS 140-3 Compliant - AES-256-GCM, AES-128-GCM</option>
                <option value="compatible">Compatible - AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC</option>
              </select>
              <p className="mt-1 text-xs text-theme-tertiary">
                {cryptoProfile === 'fips' && 'FIPS mode uses only FIPS 140-3 validated cryptographic algorithms (AES-GCM).'}
                {cryptoProfile === 'compatible' && 'Compatible mode supports older OpenVPN 2.3.x clients with CBC fallback.'}
                {cryptoProfile === 'modern' && 'Modern mode uses the latest secure ciphers including CHACHA20-POLY1305.'}
              </p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">
              VPN Subnet
            </label>
            <input
              type="text"
              value={vpnSubnet}
              onChange={(e) => setVpnSubnet(e.target.value)}
              placeholder="172.31.255.0/24"
              className="w-full px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
            <p className="mt-1 text-xs text-theme-tertiary">
              The subnet used for VPN client IP addresses. Must be a valid CIDR block.
            </p>
          </div>

          {!isWireGuard && (
            <>
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="editTlsAuthEnabled"
                  checked={tlsAuthEnabled}
                  onChange={(e) => setTlsAuthEnabled(e.target.checked)}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
                />
                <label htmlFor="editTlsAuthEnabled" className="ml-2 block text-sm text-theme-secondary">
                  Enable TLS Authentication
                </label>
              </div>
              <p className="text-xs text-theme-tertiary -mt-2">
                TLS-Auth provides additional security. Disable for simpler direct IP connections.
              </p>
            </>
          )}

          <div className="flex items-center">
            <input
              type="checkbox"
              id="editFullTunnelMode"
              checked={fullTunnelMode}
              onChange={(e) => setFullTunnelMode(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
            />
            <label htmlFor="editFullTunnelMode" className="ml-2 block text-sm text-theme-secondary">
              Full Tunnel Mode
            </label>
          </div>
          <p className="text-xs text-theme-tertiary -mt-2">
            Route all client traffic through VPN. When disabled (default), only routes for allowed networks are pushed.
          </p>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="editPushDns"
              checked={pushDns}
              onChange={(e) => setPushDns(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
            />
            <label htmlFor="editPushDns" className="ml-2 block text-sm text-theme-secondary">
              Push DNS Servers
            </label>
          </div>
          <p className="text-xs text-theme-tertiary -mt-2">
            Push DNS servers to clients. When disabled (default), client DNS settings are not changed.
          </p>

          {pushDns && (
            <div>
              <label htmlFor="editDnsServers" className="block text-sm font-medium text-theme-secondary">
                DNS Servers
              </label>
              <input
                type="text"
                id="editDnsServers"
                value={dnsServers}
                onChange={(e) => setDnsServers(e.target.value)}
                className="mt-1 block w-full rounded-md border-theme shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
                placeholder="1.1.1.1, 8.8.8.8"
              />
              <p className="mt-1 text-xs text-theme-tertiary">
                Comma-separated list of DNS server IPs to push to clients.
              </p>
            </div>
          )}

          <div className="flex items-center">
            <input
              type="checkbox"
              id="editSessionEnabled"
              checked={sessionEnabled}
              onChange={(e) => setSessionEnabled(e.target.checked)}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-theme rounded"
            />
            <label htmlFor="editSessionEnabled" className="ml-2 block text-sm text-theme-secondary">
              Enable Remote Sessions
            </label>
          </div>
          <p className="text-xs text-theme-tertiary -mt-2">
            Allow administrators to run commands on this gateway via the Remote Sessions page.
          </p>

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
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold text-theme-primary mb-4">
          Manage Access: {gateway.name}
        </h2>

        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-theme rounded text-red-700 dark:text-red-400 text-sm">
            {error}
          </div>
        )}

        {/* Tabs */}
        <div className="border-b border-theme mb-4">
          <nav className="-mb-px flex space-x-8">
            {tabs.map(tab => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === tab.id
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
                }`}
              >
                {tab.label}
                <span className={`ml-2 px-2 py-0.5 rounded-full text-xs ${
                  activeTab === tab.id ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-600' : 'bg-gray-100 dark:bg-gray-700 text-theme-secondary'
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
                    className="flex-1 px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500"
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
                    <p className="text-theme-tertiary text-sm py-4 text-center">No networks assigned</p>
                  ) : (
                    networks.map(network => (
                      <div key={network.id} className="flex items-center justify-between p-3 bg-theme-tertiary rounded-lg">
                        <div>
                          <div className="font-medium text-sm">{network.name}</div>
                          <div className="text-xs text-theme-tertiary">{network.cidr}</div>
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
                    className="flex-1 px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500"
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
                    <p className="text-theme-tertiary text-sm py-4 text-center">No users assigned</p>
                  ) : (
                    users.map(user => (
                      <div key={user.userId} className="flex items-center justify-between p-3 bg-theme-tertiary rounded-lg">
                        <div>
                          <div className="font-medium text-sm">{user.email || user.userId}</div>
                          {user.name && <div className="text-xs text-theme-tertiary">{user.name}</div>}
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
                    className="flex-1 px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500"
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
                    <p className="text-theme-tertiary text-sm py-4 text-center">No groups assigned</p>
                  ) : (
                    groups.map(group => (
                      <div key={group.groupName} className="flex items-center justify-between p-3 bg-theme-tertiary rounded-lg">
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

interface RotatedTokenModalProps {
  gatewayName: string
  token: string
  installScript: string
  onClose: () => void
}

function RotatedTokenModal({ gatewayName, token, installScript, onClose }: RotatedTokenModalProps) {
  const [copiedToken, setCopiedToken] = useState(false)
  const [copiedScript, setCopiedScript] = useState(false)

  function copyToken() {
    navigator.clipboard.writeText(token)
    setCopiedToken(true)
    setTimeout(() => setCopiedToken(false), 2000)
  }

  function copyScript() {
    navigator.clipboard.writeText(installScript)
    setCopiedScript(true)
    setTimeout(() => setCopiedScript(false), 2000)
  }

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="bg-theme-card rounded-lg shadow-xl max-w-2xl w-full mx-4 p-6">
        <div className="text-center mb-4">
          <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-yellow-100 dark:bg-yellow-900/30 mb-4">
            <svg className="h-6 w-6 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-theme-primary">Token Rotated</h2>
          <p className="text-theme-secondary mt-1">Gateway: {gatewayName}</p>
        </div>

        <div className="bg-slate-100 dark:bg-slate-800 border-l-4 border-l-indigo-500 dark:border-l-indigo-400 rounded-lg p-4 mb-4">
          <div className="flex">
            <svg className="h-5 w-5 text-indigo-600 dark:text-indigo-400 mr-2 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div>
              <h3 className="text-sm font-medium text-slate-900 dark:text-slate-100">Save this token!</h3>
              <p className="text-sm text-slate-600 dark:text-slate-300 mt-1">
                The old token is now invalid. This new token will only be shown once.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">New Authentication Token</label>
            <div className="flex">
              <input
                type="text"
                readOnly
                value={token}
                className="flex-1 px-3 py-2 bg-theme-tertiary border border-theme rounded-l-lg text-sm font-mono text-theme-primary"
              />
              <button
                onClick={copyToken}
                className="px-4 py-2 bg-primary-600 text-white rounded-r-lg hover:bg-primary-700"
              >
                {copiedToken ? 'Copied!' : 'Copy'}
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-theme-secondary mb-1">Install Script</label>
            <div className="relative">
              <pre className="bg-theme-tertiary border border-theme rounded-lg p-3 text-sm font-mono text-theme-primary overflow-x-auto whitespace-pre-wrap break-all">
                {installScript}
              </pre>
              <button
                onClick={copyScript}
                className="absolute top-2 right-2 px-3 py-1 bg-primary-600 text-white text-sm rounded hover:bg-primary-700"
              >
                {copiedScript ? 'Copied!' : 'Copy'}
              </button>
            </div>
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <button onClick={onClose} className="btn btn-primary">
            Done
          </button>
        </div>
      </div>
    </div>
  )
}
