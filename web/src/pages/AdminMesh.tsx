import { useState, useEffect } from 'react'
import {
  getMeshHubs, createMeshHub, deleteMeshHub, provisionMeshHub, getMeshHubInstallScript, updateMeshHub,
  getMeshSpokes, createMeshSpoke, deleteMeshSpoke, provisionMeshSpoke, getMeshSpokeInstallScript, updateMeshSpoke,
  getMeshHubUsers, assignMeshHubUser, removeMeshHubUser,
  getMeshHubGroups, assignMeshHubGroup, removeMeshHubGroup,
  getMeshHubNetworks, assignMeshHubNetwork, removeMeshHubNetwork, MeshHubNetwork,
  getMeshSpokeUsers, assignMeshSpokeUser, removeMeshSpokeUser,
  getMeshSpokeGroups, assignMeshSpokeGroup, removeMeshSpokeGroup,
  getUsers, getGroups, getNetworks, Network,
  MeshHub, MeshHubWithToken, MeshSpoke, MeshSpokeWithToken,
  CreateMeshHubRequest, CreateMeshSpokeRequest, CryptoProfile
} from '../api/client'
import ActionDropdown from '../components/ActionDropdown'

type Tab = 'hubs' | 'spokes'

export default function AdminMesh() {
  const [activeTab, setActiveTab] = useState<Tab>('hubs')
  const [hubs, setHubs] = useState<MeshHub[]>([])
  const [selectedHub, setSelectedHub] = useState<MeshHub | null>(null)
  const [spokes, setSpokes] = useState<MeshSpoke[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Modal states
  const [showAddHubModal, setShowAddHubModal] = useState(false)
  const [showAddSpokeModal, setShowAddSpokeModal] = useState(false)
  const [showTokenModal, setShowTokenModal] = useState(false)
  const [showInstallScriptModal, setShowInstallScriptModal] = useState(false)
  const [showAccessModal, setShowAccessModal] = useState(false)
  const [accessHub, setAccessHub] = useState<MeshHub | null>(null)
  const [showSpokeAccessModal, setShowSpokeAccessModal] = useState(false)
  const [accessSpoke, setAccessSpoke] = useState<MeshSpoke | null>(null)
  const [installScript, setInstallScript] = useState<{ type: 'hub' | 'spoke'; name: string; script: string } | null>(null)
  const [newHub, setNewHub] = useState<MeshHubWithToken | null>(null)
  const [newSpoke, setNewSpoke] = useState<MeshSpokeWithToken | null>(null)
  const [editingHub, setEditingHub] = useState<MeshHub | null>(null)
  const [editingSpoke, setEditingSpoke] = useState<MeshSpoke | null>(null)

  useEffect(() => {
    loadHubs()
  }, [])

  useEffect(() => {
    if (selectedHub) {
      loadSpokes(selectedHub.id)
    }
  }, [selectedHub])

  async function loadHubs() {
    try {
      setLoading(true)
      const data = await getMeshHubs()
      setHubs(data)
      if (data.length > 0 && !selectedHub) {
        setSelectedHub(data[0])
      }
      setError(null)
    } catch (err) {
      setError('Failed to load mesh hubs')
    } finally {
      setLoading(false)
    }
  }

  async function loadSpokes(hubId: string) {
    try {
      const data = await getMeshSpokes(hubId)
      setSpokes(data)
    } catch (err) {
      setError('Failed to load spokes')
    }
  }

  async function handleDeleteHub(hub: MeshHub) {
    if (!confirm(`Are you sure you want to delete hub "${hub.name}"? This will also delete all associated spokes.`)) {
      return
    }
    try {
      await deleteMeshHub(hub.id)
      await loadHubs()
      if (selectedHub?.id === hub.id) {
        setSelectedHub(hubs.length > 1 ? hubs.find(h => h.id !== hub.id) || null : null)
      }
    } catch (err) {
      setError('Failed to delete hub')
    }
  }

  async function handleProvisionHub(hub: MeshHub) {
    try {
      await provisionMeshHub(hub.id)
      await loadHubs()
      setError(null)
    } catch (err) {
      setError('Failed to provision hub')
    }
  }

  async function handleDeleteSpoke(spoke: MeshSpoke) {
    if (!confirm(`Are you sure you want to delete spoke "${spoke.name}"?`)) {
      return
    }
    try {
      await deleteMeshSpoke(spoke.id)
      if (selectedHub) {
        await loadSpokes(selectedHub.id)
      }
    } catch (err) {
      setError('Failed to delete spoke')
    }
  }

  async function handleProvisionSpoke(spoke: MeshSpoke) {
    try {
      await provisionMeshSpoke(spoke.id)
      if (selectedHub) {
        await loadSpokes(selectedHub.id)
      }
      setError(null)
    } catch (err) {
      setError('Failed to provision spoke')
    }
  }

  async function handleShowHubInstallScript(hub: MeshHub) {
    try {
      const script = await getMeshHubInstallScript(hub.id)
      setInstallScript({ type: 'hub', name: hub.name, script })
      setShowInstallScriptModal(true)
    } catch (err) {
      setError('Failed to get install script')
    }
  }

  async function handleShowSpokeInstallScript(spoke: MeshSpoke) {
    try {
      const script = await getMeshSpokeInstallScript(spoke.id)
      setInstallScript({ type: 'spoke', name: spoke.name, script })
      setShowInstallScriptModal(true)
    } catch (err) {
      setError('Failed to get install script')
    }
  }

  function getStatusColor(status: string) {
    switch (status) {
      case 'online':
      case 'connected':
        return 'bg-green-600 text-white'
      case 'offline':
      case 'disconnected':
        return 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300'
      case 'error':
        return 'bg-red-600 text-white'
      default:
        return 'bg-yellow-500 text-white'
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-theme-primary">Mesh Networking</h1>
            <p className="text-theme-tertiary mt-1">
              Manage mesh hubs and spokes for hub-and-spoke VPN connectivity.
            </p>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-theme">
        <nav className="-mb-px flex space-x-8" aria-label="Tabs">
          <button
            onClick={() => setActiveTab('hubs')}
            className={`whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'hubs'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Hubs
          </button>
          <button
            onClick={() => setActiveTab('spokes')}
            className={`whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'spokes'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Spokes
          </button>
        </nav>
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
          {error}
          <button onClick={() => setError(null)} className="ml-4 text-red-500 hover:text-red-700">×</button>
        </div>
      )}

      {/* Hubs Tab */}
      {activeTab === 'hubs' && (
        <div className="space-y-6">
          <div className="flex justify-end">
            <button
              onClick={() => setShowAddHubModal(true)}
              className="btn btn-primary"
            >
              <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add Hub
            </button>
          </div>

          {loading ? (
            <div className="card flex justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : hubs.length > 0 ? (
            <div className="card p-0">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Hub</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Status</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Endpoint</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Connections</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Last Heartbeat</th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-theme-tertiary uppercase tracking-wider">Actions</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme">
                  {hubs.map((hub) => (
                    <tr key={hub.id} className={selectedHub?.id === hub.id ? 'row-selected' : 'row-hover'}>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div>
                          <div className="text-sm font-medium text-theme-primary">{hub.name}</div>
                          <div className="text-sm text-theme-tertiary">{hub.description}</div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(hub.status)}`}>
                          {hub.status.charAt(0).toUpperCase() + hub.status.slice(1)}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                        <div>{hub.publicEndpoint}</div>
                        <div className="text-xs">{hub.vpnProtocol.toUpperCase()}:{hub.vpnPort}</div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                        <div>{hub.connectedSpokes} spokes</div>
                        <div>{hub.connectedClients} clients</div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                        {hub.lastHeartbeat
                          ? new Date(hub.lastHeartbeat).toLocaleString()
                          : 'Never'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <ActionDropdown
                          actions={[
                            { label: 'View Spokes', icon: 'gateway' as const, onClick: () => { setSelectedHub(hub); setActiveTab('spokes') } },
                            { label: 'Edit', icon: 'edit' as const, onClick: () => setEditingHub(hub) },
                            { label: 'Manage Access', icon: 'access' as const, onClick: () => { setAccessHub(hub); setShowAccessModal(true) } },
                            { label: 'Re-provision', icon: 'install' as const, onClick: () => handleProvisionHub(hub) },
                            { label: 'Install Script', icon: 'view' as const, onClick: () => handleShowHubInstallScript(hub), color: 'primary' as const },
                            { label: 'Delete', icon: 'delete' as const, onClick: () => handleDeleteHub(hub), color: 'red' as const },
                          ]}
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
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
              </svg>
              <h3 className="mt-2 text-sm font-medium text-theme-primary">No mesh hubs</h3>
              <p className="mt-1 text-sm text-theme-tertiary">Get started by creating a new mesh hub.</p>
              <div className="mt-6">
                <button onClick={() => setShowAddHubModal(true)} className="btn btn-primary">
                  Add Hub
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Spokes Tab */}
      {activeTab === 'spokes' && (
        <div className="space-y-6">
          {/* Hub Selector */}
          <div className="card">
            <label className="block text-sm font-medium text-theme-secondary mb-2">Select Hub</label>
            <select
              value={selectedHub?.id || ''}
              onChange={(e) => setSelectedHub(hubs.find(h => h.id === e.target.value) || null)}
              className="input"
            >
              <option value="">-- Select a hub --</option>
              {hubs.map((hub) => (
                <option key={hub.id} value={hub.id}>{hub.name}</option>
              ))}
            </select>
          </div>

          {selectedHub && (
            <>
              <div className="flex justify-end">
                <button
                  onClick={() => setShowAddSpokeModal(true)}
                  className="btn btn-primary"
                >
                  <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Add Spoke
                </button>
              </div>

              {spokes.length > 0 ? (
                <div className="card p-0">
                  <table className="min-w-full divide-y divide-theme">
                    <thead className="bg-theme-tertiary">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Spoke</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Status</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Networks</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Tunnel IP</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-theme-tertiary uppercase tracking-wider">Last Seen</th>
                        <th className="px-6 py-3 text-right text-xs font-medium text-theme-tertiary uppercase tracking-wider">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="bg-theme-card divide-y divide-theme">
                      {spokes.map((spoke) => (
                        <tr key={spoke.id}>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div>
                              <div className="text-sm font-medium text-theme-primary">{spoke.name}</div>
                              <div className="text-sm text-theme-tertiary">{spoke.description}</div>
                              {spoke.remoteIp && <div className="text-xs text-theme-muted">Remote: {spoke.remoteIp}</div>}
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(spoke.status)}`}>
                              {spoke.status.charAt(0).toUpperCase() + spoke.status.slice(1)}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                            {spoke.localNetworks.length > 0 ? (
                              <div className="flex flex-wrap gap-1">
                                {spoke.localNetworks.map((net, i) => (
                                  <span key={i} className="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-theme-secondary rounded text-xs">{net}</span>
                                ))}
                              </div>
                            ) : (
                              <span className="text-theme-muted">No networks</span>
                            )}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                            {spoke.tunnelIp || <span className="text-theme-muted">Not assigned</span>}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-tertiary">
                            {spoke.lastSeen
                              ? new Date(spoke.lastSeen).toLocaleString()
                              : 'Never'}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                            <ActionDropdown
                              actions={[
                                { label: 'Edit', icon: 'edit' as const, onClick: () => setEditingSpoke(spoke) },
                                { label: 'Manage Access', icon: 'access' as const, onClick: () => { setAccessSpoke(spoke); setShowSpokeAccessModal(true) } },
                                { label: 'Re-provision', icon: 'install' as const, onClick: () => handleProvisionSpoke(spoke) },
                                { label: 'Install Script', icon: 'view' as const, onClick: () => handleShowSpokeInstallScript(spoke), color: 'primary' as const },
                                { label: 'Delete', icon: 'delete' as const, onClick: () => handleDeleteSpoke(spoke), color: 'red' as const },
                              ]}
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
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
                  </svg>
                  <h3 className="mt-2 text-sm font-medium text-theme-primary">No spokes</h3>
                  <p className="mt-1 text-sm text-theme-tertiary">Add spokes to connect remote sites to this hub.</p>
                  <div className="mt-6">
                    <button onClick={() => setShowAddSpokeModal(true)} className="btn btn-primary">
                      Add Spoke
                    </button>
                  </div>
                </div>
              )}
            </>
          )}

          {!selectedHub && (
            <div className="card text-center py-12 text-theme-tertiary">
              Select a hub to view its spokes
            </div>
          )}
        </div>
      )}

      {/* Add Hub Modal */}
      {showAddHubModal && (
        <AddHubModal
          onClose={() => setShowAddHubModal(false)}
          onSuccess={(hub) => {
            setNewHub(hub)
            setShowAddHubModal(false)
            setShowTokenModal(true)
            loadHubs()
          }}
        />
      )}

      {/* Add Spoke Modal */}
      {showAddSpokeModal && selectedHub && (
        <AddSpokeModal
          hubId={selectedHub.id}
          onClose={() => setShowAddSpokeModal(false)}
          onSuccess={(spoke) => {
            setNewSpoke(spoke)
            setShowAddSpokeModal(false)
            setShowTokenModal(true)
            loadSpokes(selectedHub.id)
          }}
        />
      )}

      {/* Token Modal */}
      {showTokenModal && (newHub || newSpoke) && (
        <TokenModal
          type={newHub ? 'hub' : 'spoke'}
          name={newHub?.name || newSpoke?.name || ''}
          token={newHub?.apiToken || newSpoke?.token || ''}
          controlPlaneUrl={newHub?.controlPlaneUrl || window.location.origin}
          onClose={() => {
            setShowTokenModal(false)
            setNewHub(null)
            setNewSpoke(null)
          }}
        />
      )}

      {/* Install Script Modal */}
      {showInstallScriptModal && installScript && (
        <InstallScriptModal
          type={installScript.type}
          name={installScript.name}
          script={installScript.script}
          onClose={() => {
            setShowInstallScriptModal(false)
            setInstallScript(null)
          }}
        />
      )}

      {/* Manage Access Modal */}
      {showAccessModal && accessHub && (
        <ManageAccessModal
          hub={accessHub}
          onClose={() => {
            setShowAccessModal(false)
            setAccessHub(null)
          }}
        />
      )}

      {/* Manage Spoke Access Modal */}
      {showSpokeAccessModal && accessSpoke && (
        <ManageSpokeAccessModal
          spoke={accessSpoke}
          onClose={() => {
            setShowSpokeAccessModal(false)
            setAccessSpoke(null)
          }}
        />
      )}

      {/* Edit Hub Modal */}
      {editingHub && (
        <EditHubModal
          hub={editingHub}
          onClose={() => setEditingHub(null)}
          onSuccess={() => {
            setEditingHub(null)
            loadHubs()
          }}
        />
      )}

      {/* Edit Spoke Modal */}
      {editingSpoke && selectedHub && (
        <EditSpokeModal
          spoke={editingSpoke}
          onClose={() => setEditingSpoke(null)}
          onSuccess={() => {
            setEditingSpoke(null)
            loadSpokes(selectedHub.id)
          }}
        />
      )}
    </div>
  )
}

// Add Hub Modal Component
function AddHubModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: (hub: MeshHubWithToken) => void }) {
  const [form, setForm] = useState<CreateMeshHubRequest>({
    name: '',
    description: '',
    publicEndpoint: '',
    vpnPort: 1194,
    vpnProtocol: 'udp',
    vpnSubnet: '172.30.0.0/16',
    cryptoProfile: 'fips' as CryptoProfile,
    tlsAuthEnabled: true,
    fullTunnelMode: false,
    pushDns: false,
    dnsServers: [],
    sessionEnabled: true,
  })
  const [dnsInput, setDnsInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const hub = await createMeshHub(form)
      onSuccess(hub)
    } catch (err) {
      setError('Failed to create hub')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-lg w-full p-6">
          <h2 className="text-lg font-semibold mb-4 text-theme-primary">Add Mesh Hub</h2>

          {error && <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded">{error}</div>}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-theme-secondary">Name</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="input"
                required
                placeholder="e.g., main-hub"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Description</label>
              <input
                type="text"
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                className="input"
                placeholder="Optional description"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Public Endpoint</label>
              <input
                type="text"
                value={form.publicEndpoint}
                onChange={(e) => setForm({ ...form, publicEndpoint: e.target.value })}
                className="input"
                required
                placeholder="e.g., hub.example.com or 1.2.3.4"
              />
              <p className="text-xs text-theme-tertiary mt-1">The public hostname or IP where spokes will connect</p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-theme-secondary">VPN Port</label>
                <input
                  type="number"
                  value={form.vpnPort}
                  onChange={(e) => setForm({ ...form, vpnPort: parseInt(e.target.value) })}
                  className="input"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-theme-secondary">Protocol</label>
                <select
                  value={form.vpnProtocol}
                  onChange={(e) => setForm({ ...form, vpnProtocol: e.target.value })}
                  className="input"
                >
                  <option value="udp">UDP</option>
                  <option value="tcp">TCP</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">VPN Subnet</label>
              <input
                type="text"
                value={form.vpnSubnet}
                onChange={(e) => setForm({ ...form, vpnSubnet: e.target.value })}
                className="input"
                placeholder="172.30.0.0/16"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Crypto Profile</label>
              <select
                value={form.cryptoProfile}
                onChange={(e) => setForm({ ...form, cryptoProfile: e.target.value as CryptoProfile })}
                className="input"
              >
                <option value="fips">FIPS (AES-256-GCM, AES-128-GCM)</option>
                <option value="modern">Modern (AES-256-GCM, ChaCha20)</option>
                <option value="compatible">Compatible (includes CBC ciphers)</option>
              </select>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="tlsAuth"
                checked={form.tlsAuthEnabled}
                onChange={(e) => setForm({ ...form, tlsAuthEnabled: e.target.checked })}
                className="rounded border-theme text-primary-600 focus:ring-primary-500"
              />
              <label htmlFor="tlsAuth" className="ml-2 text-sm text-theme-secondary">
                Enable TLS-Auth (additional HMAC layer)
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="fullTunnel"
                checked={form.fullTunnelMode}
                onChange={(e) => setForm({ ...form, fullTunnelMode: e.target.checked })}
                className="rounded border-theme text-primary-600 focus:ring-primary-500"
              />
              <label htmlFor="fullTunnel" className="ml-2 text-sm text-theme-secondary">
                Full Tunnel Mode (route all client traffic through VPN)
              </label>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="pushDns"
                checked={form.pushDns}
                onChange={(e) => setForm({ ...form, pushDns: e.target.checked })}
                className="rounded border-theme text-primary-600 focus:ring-primary-500"
              />
              <label htmlFor="pushDns" className="ml-2 text-sm text-theme-secondary">
                Push DNS servers to clients
              </label>
            </div>

            {form.pushDns && (
              <div>
                <label className="block text-sm font-medium text-theme-secondary">DNS Servers</label>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={dnsInput}
                    onChange={(e) => setDnsInput(e.target.value)}
                    className="input flex-1"
                    placeholder="e.g., 1.1.1.1"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        if (dnsInput.trim()) {
                          setForm({ ...form, dnsServers: [...(form.dnsServers || []), dnsInput.trim()] })
                          setDnsInput('')
                        }
                      }
                    }}
                  />
                  <button
                    type="button"
                    onClick={() => {
                      if (dnsInput.trim()) {
                        setForm({ ...form, dnsServers: [...(form.dnsServers || []), dnsInput.trim()] })
                        setDnsInput('')
                      }
                    }}
                    className="btn btn-secondary"
                  >
                    Add
                  </button>
                </div>
                <p className="text-xs text-theme-tertiary mt-1">Leave empty to use defaults (1.1.1.1, 8.8.8.8)</p>
                {form.dnsServers && form.dnsServers.length > 0 && (
                  <div className="flex flex-wrap gap-2 mt-2">
                    {form.dnsServers.map((dns, idx) => (
                      <span key={idx} className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-theme-secondary rounded text-sm flex items-center">
                        {dns}
                        <button
                          type="button"
                          onClick={() => setForm({ ...form, dnsServers: form.dnsServers?.filter((_, i) => i !== idx) })}
                          className="ml-1 text-theme-muted hover:text-red-600"
                        >
                          ×
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )}

            <div className="flex items-center">
              <input
                type="checkbox"
                id="hubSessionEnabled"
                checked={form.sessionEnabled ?? true}
                onChange={(e) => setForm({ ...form, sessionEnabled: e.target.checked })}
                className="rounded border-theme text-primary-600 focus:ring-primary-500"
              />
              <label htmlFor="hubSessionEnabled" className="ml-2 text-sm text-theme-secondary">
                Enable Remote Sessions
              </label>
            </div>
            <p className="text-xs text-theme-tertiary -mt-2">
              Allow administrators to run commands on this hub via the Remote Sessions page.
            </p>

            <div className="flex justify-end space-x-3 pt-4">
              <button type="button" onClick={onClose} className="btn btn-secondary">Cancel</button>
              <button type="submit" disabled={loading} className="btn btn-primary">
                {loading ? 'Creating...' : 'Create Hub'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

// Add Spoke Modal Component
function AddSpokeModal({ hubId, onClose, onSuccess }: { hubId: string; onClose: () => void; onSuccess: (spoke: MeshSpokeWithToken) => void }) {
  const [form, setForm] = useState<CreateMeshSpokeRequest>({
    name: '',
    description: '',
    localNetworks: [],
    sessionEnabled: true,
  })
  const [networkInput, setNetworkInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function addNetwork() {
    if (networkInput && !form.localNetworks.includes(networkInput)) {
      setForm({ ...form, localNetworks: [...form.localNetworks, networkInput] })
      setNetworkInput('')
    }
  }

  function removeNetwork(net: string) {
    setForm({ ...form, localNetworks: form.localNetworks.filter(n => n !== net) })
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const spoke = await createMeshSpoke(hubId, form)
      onSuccess(spoke)
    } catch (err) {
      setError('Failed to create spoke')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-lg w-full p-6">
          <h2 className="text-lg font-semibold mb-4 text-theme-primary">Add Mesh Spoke</h2>

          {error && <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded">{error}</div>}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-theme-secondary">Name</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="input"
                required
                placeholder="e.g., home-lab"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Description</label>
              <input
                type="text"
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                className="input"
                placeholder="Optional description"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">Local Networks</label>
              <div className="flex space-x-2">
                <input
                  type="text"
                  value={networkInput}
                  onChange={(e) => setNetworkInput(e.target.value)}
                  className="input flex-1"
                  placeholder="e.g., 10.0.0.0/8"
                  onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addNetwork())}
                />
                <button type="button" onClick={addNetwork} className="btn btn-secondary">Add</button>
              </div>
              <p className="text-xs text-theme-tertiary mt-1">Networks behind this spoke that should be routable via the hub</p>
              {form.localNetworks.length > 0 && (
                <div className="flex flex-wrap gap-2 mt-2">
                  {form.localNetworks.map((net) => (
                    <span key={net} className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-theme-secondary rounded text-sm flex items-center">
                      {net}
                      <button type="button" onClick={() => removeNetwork(net)} className="ml-1 text-theme-muted hover:text-red-600">×</button>
                    </span>
                  ))}
                </div>
              )}
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="spokeSessionEnabled"
                checked={form.sessionEnabled ?? true}
                onChange={(e) => setForm({ ...form, sessionEnabled: e.target.checked })}
                className="rounded border-theme text-primary-600 focus:ring-primary-500"
              />
              <label htmlFor="spokeSessionEnabled" className="ml-2 text-sm text-theme-secondary">
                Enable Remote Sessions
              </label>
            </div>
            <p className="text-xs text-theme-tertiary -mt-2">
              Allow administrators to run commands on this spoke via the Remote Sessions page.
            </p>

            <div className="flex justify-end space-x-3 pt-4">
              <button type="button" onClick={onClose} className="btn btn-secondary">Cancel</button>
              <button type="submit" disabled={loading} className="btn btn-primary">
                {loading ? 'Creating...' : 'Create Spoke'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

// Token Modal Component
function TokenModal({ type, name, token, controlPlaneUrl, onClose }: { type: 'hub' | 'spoke'; name: string; token: string; controlPlaneUrl?: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false)

  function copyToClipboard() {
    navigator.clipboard.writeText(token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const setupCommand = type === 'hub'
    ? `curl -sSL ${controlPlaneUrl}/scripts/install-hub.sh | sudo bash -s -- \\
  --token "${token}" \\
  --control-plane "${controlPlaneUrl}"`
    : `curl -sSL ${controlPlaneUrl}/scripts/install-mesh-spoke.sh | sudo bash -s -- \\
  --token "${token}" \\
  --control-plane "${controlPlaneUrl}"`

  const [commandCopied, setCommandCopied] = useState(false)

  function copyCommand() {
    navigator.clipboard.writeText(setupCommand)
    setCommandCopied(true)
    setTimeout(() => setCommandCopied(false), 2000)
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-2xl w-full p-6">
          <div className="flex items-center mb-4">
            <div className="flex-shrink-0 w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
              <svg className="w-6 h-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h2 className="ml-3 text-lg font-semibold text-theme-primary">{type === 'hub' ? 'Hub' : 'Spoke'} Created</h2>
          </div>

          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4 mb-4">
            <div className="flex">
              <svg className="h-5 w-5 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-yellow-800 dark:text-yellow-300">Save this token!</h3>
                <p className="text-sm text-yellow-700 dark:text-yellow-400 mt-1">
                  This token will only be shown once. You'll need it to set up the {type}.
                </p>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-theme-secondary">Name</label>
              <div className="text-theme-primary">{name}</div>
            </div>

            <div>
              <label className="block text-sm font-medium text-theme-secondary">
                {type === 'hub' ? 'API Token' : 'Spoke Token'}
              </label>
              <div className="flex items-center space-x-2">
                <code className="flex-1 p-2 bg-gray-100 dark:bg-gray-700 rounded text-sm font-mono break-all text-theme-primary">{token}</code>
                <button onClick={copyToClipboard} className="btn btn-secondary">
                  {copied ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>

            {controlPlaneUrl && (
              <div>
                <label className="block text-sm font-medium text-theme-secondary">Control Plane URL</label>
                <code className="block p-2 bg-gray-100 dark:bg-gray-700 rounded text-sm font-mono text-theme-primary">{controlPlaneUrl}</code>
              </div>
            )}

            {/* Setup Command */}
            <div>
              <label className="block text-sm font-medium text-theme-secondary mb-2">
                Quick Setup Command
              </label>
              <p className="text-xs text-theme-tertiary mb-2">
                Run this command on your {type === 'hub' ? 'hub server' : 'spoke server'} to install and configure:
              </p>
              <div className="relative">
                <pre className="p-3 bg-gray-900 text-green-400 rounded-lg text-sm overflow-x-auto font-mono">
                  {setupCommand}
                </pre>
                <button
                  onClick={copyCommand}
                  className="absolute top-2 right-2 px-3 py-1 bg-gray-700 hover:bg-gray-600 text-white text-xs rounded"
                >
                  {commandCopied ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          </div>

          <div className="mt-6 flex justify-end">
            <button onClick={onClose} className="btn btn-primary">Done</button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Install Script Modal Component
function InstallScriptModal({ type, name, script, onClose }: { type: 'hub' | 'spoke'; name: string; script: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false)

  function copyToClipboard() {
    navigator.clipboard.writeText(script)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  function downloadScript() {
    const blob = new Blob([script], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `install-${type}-${name.replace(/[^a-zA-Z0-9-_]/g, '-')}.sh`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-3xl w-full p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-theme-primary">
              Install Script - {type === 'hub' ? 'Hub' : 'Spoke'}: {name}
            </h2>
            <button onClick={onClose} className="text-theme-muted hover:text-theme-tertiary">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-4">
            <div className="flex items-start">
              <svg className="h-5 w-5 text-blue-500 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-gray-800 dark:text-blue-300">Installation Instructions</h3>
                <p className="text-sm text-gray-700 dark:text-blue-400 mt-1">
                  Run this script on your {type === 'hub' ? 'hub server' : 'spoke server'} with root privileges:
                </p>
                <code className="block mt-2 text-sm bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-theme-secondary">
                  sudo bash install-{type}.sh
                </code>
              </div>
            </div>
          </div>

          <div className="relative">
            <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto max-h-96 overflow-y-auto">
              <code>{script}</code>
            </pre>
          </div>

          <div className="mt-4 flex justify-end space-x-3">
            <button onClick={downloadScript} className="btn btn-secondary">
              <svg className="w-4 h-4 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              Download
            </button>
            <button onClick={copyToClipboard} className="btn btn-primary">
              {copied ? 'Copied!' : 'Copy to Clipboard'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Manage Access Modal Component
function ManageAccessModal({ hub, onClose }: { hub: MeshHub; onClose: () => void }) {
  const [activeTab, setActiveTab] = useState<'users' | 'groups' | 'networks'>('users')
  const [users, setUsers] = useState<string[]>([])
  const [groups, setGroups] = useState<string[]>([])
  const [networks, setNetworks] = useState<MeshHubNetwork[]>([])
  const [allUsers, setAllUsers] = useState<{ id: string; email: string; name: string }[]>([])
  const [allGroups, setAllGroups] = useState<string[]>([])
  const [allNetworks, setAllNetworks] = useState<Network[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedUser, setSelectedUser] = useState('')
  const [customUserInput, setCustomUserInput] = useState('')
  const [selectedGroup, setSelectedGroup] = useState('')
  const [customGroupInput, setCustomGroupInput] = useState('')
  const [selectedNetwork, setSelectedNetwork] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadData()
  }, [hub.id])

  async function loadData() {
    try {
      setLoading(true)
      const [hubUsers, hubGroups, hubNetworks, userList, groupList, networkList] = await Promise.all([
        getMeshHubUsers(hub.id),
        getMeshHubGroups(hub.id),
        getMeshHubNetworks(hub.id),
        getUsers(),
        getGroups(),
        getNetworks(),
      ])
      setUsers(hubUsers)
      setGroups(hubGroups)
      setNetworks(hubNetworks)
      setAllUsers(userList)
      setAllGroups(groupList.map(g => g.name))
      setAllNetworks(networkList)
    } catch (err) {
      setError('Failed to load access data')
    } finally {
      setLoading(false)
    }
  }

  async function handleAddUser() {
    const userToAdd = selectedUser || customUserInput.trim()
    if (!userToAdd) return
    if (users.includes(userToAdd)) {
      setError('User already assigned')
      return
    }
    try {
      await assignMeshHubUser(hub.id, userToAdd)
      setUsers([...users, userToAdd])
      setSelectedUser('')
      setCustomUserInput('')
      setError(null)
    } catch (err) {
      setError('Failed to add user')
    }
  }

  async function handleRemoveUser(userId: string) {
    try {
      await removeMeshHubUser(hub.id, userId)
      setUsers(users.filter(u => u !== userId))
    } catch (err) {
      setError('Failed to remove user')
    }
  }

  async function handleAddGroup() {
    const groupToAdd = selectedGroup || customGroupInput.trim()
    if (!groupToAdd) return
    if (groups.includes(groupToAdd)) {
      setError('Group already assigned')
      return
    }
    try {
      await assignMeshHubGroup(hub.id, groupToAdd)
      setGroups([...groups, groupToAdd])
      setSelectedGroup('')
      setCustomGroupInput('')
      setError(null)
    } catch (err) {
      setError('Failed to add group')
    }
  }

  async function handleRemoveGroup(groupName: string) {
    try {
      await removeMeshHubGroup(hub.id, groupName)
      setGroups(groups.filter(g => g !== groupName))
    } catch (err) {
      setError('Failed to remove group')
    }
  }

  async function handleAddNetwork() {
    if (!selectedNetwork) return
    try {
      await assignMeshHubNetwork(hub.id, selectedNetwork)
      const network = allNetworks.find(n => n.id === selectedNetwork)
      if (network) {
        setNetworks([...networks, { id: network.id, name: network.name, description: network.description || '', cidr: network.cidr, isActive: network.isActive }])
      }
      setSelectedNetwork('')
    } catch (err) {
      setError('Failed to add network')
    }
  }

  async function handleRemoveNetwork(networkId: string) {
    try {
      await removeMeshHubNetwork(hub.id, networkId)
      setNetworks(networks.filter(n => n.id !== networkId))
    } catch (err) {
      setError('Failed to remove network')
    }
  }

  const availableUsers = allUsers.filter(u => !users.includes(u.id))
  const availableGroups = allGroups.filter(g => !groups.includes(g))
  const availableNetworks = allNetworks.filter(n => !networks.find(hn => hn.id === n.id))

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-2xl w-full p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-theme-primary">Manage Access - {hub.name}</h2>
            <button onClick={onClose} className="text-theme-muted hover:text-theme-tertiary">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
              {error}
            </div>
          )}

          {/* Zero-Trust Info */}
          <div className="info-box mb-4">
            <strong className="text-theme-primary">Zero-Trust Model:</strong>{' '}
            <span className="info-box-text">Users only get routes to networks they have explicit access rules for.
            Assign networks to this hub, then ensure users have access rules within those networks.</span>
          </div>

          {/* Tabs */}
          <div className="border-b border-theme mb-4">
            <nav className="-mb-px flex space-x-8">
              <button
                onClick={() => setActiveTab('users')}
                className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'users'
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary'
                }`}
              >
                Users ({users.length})
              </button>
              <button
                onClick={() => setActiveTab('groups')}
                className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'groups'
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary'
                }`}
              >
                Groups ({groups.length})
              </button>
              <button
                onClick={() => setActiveTab('networks')}
                className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'networks'
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary'
                }`}
              >
                Networks ({networks.length})
              </button>
            </nav>
          </div>

          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : activeTab === 'networks' ? (
            <div className="space-y-4">
              {/* Add Network */}
              <div className="flex space-x-2">
                <select
                  value={selectedNetwork}
                  onChange={(e) => setSelectedNetwork(e.target.value)}
                  className="input flex-1"
                >
                  <option value="">Select a network...</option>
                  {availableNetworks.map((network) => (
                    <option key={network.id} value={network.id}>
                      {network.name} ({network.cidr})
                    </option>
                  ))}
                </select>
                <button
                  onClick={handleAddNetwork}
                  disabled={!selectedNetwork}
                  className="btn btn-primary"
                >
                  Add
                </button>
              </div>

              {/* Network List */}
              <div className="border border-theme rounded-lg divide-y divide-theme">
                {networks.length === 0 ? (
                  <div className="p-4 text-center text-theme-tertiary">
                    No networks assigned to this hub. Add networks to enable zero-trust access control.
                  </div>
                ) : (
                  networks.map((network) => (
                    <div key={network.id} className="p-3 flex items-center justify-between bg-theme-card">
                      <div>
                        <div className="font-medium text-theme-primary">{network.name}</div>
                        <div className="text-sm text-theme-tertiary">{network.cidr}</div>
                        {network.description && (
                          <div className="text-xs text-theme-muted">{network.description}</div>
                        )}
                      </div>
                      <button
                        onClick={() => handleRemoveNetwork(network.id)}
                        className="text-red-600 hover:text-red-800"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          ) : activeTab === 'users' ? (
            <div className="space-y-4">
              {/* Add User */}
              <div className="space-y-2">
                <div className="flex space-x-2">
                  <select
                    value={selectedUser}
                    onChange={(e) => {
                      setSelectedUser(e.target.value)
                      setCustomUserInput('')
                    }}
                    className="input flex-1 text-sm"
                  >
                    <option value="">Select from list...</option>
                    {availableUsers.map((user) => (
                      <option key={user.id} value={user.id}>
                        {user.email} {user.name && `(${user.name})`}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-theme-tertiary">or type:</span>
                  <input
                    type="text"
                    value={customUserInput}
                    onChange={(e) => {
                      setCustomUserInput(e.target.value)
                      setSelectedUser('')
                    }}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddUser())}
                    placeholder="username or email"
                    className="input flex-1 text-sm"
                  />
                  <button
                    onClick={handleAddUser}
                    disabled={!selectedUser && !customUserInput.trim()}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* User List */}
              <div className="border border-theme rounded-lg divide-y divide-theme">
                {users.length === 0 ? (
                  <div className="p-4 text-center text-theme-tertiary">
                    No users assigned to this hub
                  </div>
                ) : (
                  users.map((userId) => {
                    const user = allUsers.find(u => u.id === userId)
                    return (
                      <div key={userId} className="p-3 flex items-center justify-between bg-theme-card">
                        <div>
                          <div className="font-medium text-theme-primary">
                            {user?.email || userId}
                          </div>
                          {user?.name && (
                            <div className="text-sm text-theme-tertiary">{user.name}</div>
                          )}
                        </div>
                        <button
                          onClick={() => handleRemoveUser(userId)}
                          className="text-red-600 hover:text-red-800"
                        >
                          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                    )
                  })
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Add Group */}
              <div className="space-y-2">
                <div className="flex space-x-2">
                  <select
                    value={selectedGroup}
                    onChange={(e) => {
                      setSelectedGroup(e.target.value)
                      setCustomGroupInput('')
                    }}
                    className="input flex-1 text-sm"
                  >
                    <option value="">Select from list...</option>
                    {availableGroups.map((group) => (
                      <option key={group} value={group}>
                        {group}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-theme-tertiary">or type:</span>
                  <input
                    type="text"
                    value={customGroupInput}
                    onChange={(e) => {
                      setCustomGroupInput(e.target.value)
                      setSelectedGroup('')
                    }}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddGroup())}
                    placeholder="group name"
                    className="input flex-1 text-sm"
                  />
                  <button
                    onClick={handleAddGroup}
                    disabled={!selectedGroup && !customGroupInput.trim()}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* Group List */}
              <div className="border border-theme rounded-lg divide-y divide-theme">
                {groups.length === 0 ? (
                  <div className="p-4 text-center text-theme-tertiary">
                    No groups assigned to this hub
                  </div>
                ) : (
                  groups.map((group) => (
                    <div key={group} className="p-3 flex items-center justify-between bg-theme-card">
                      <div className="font-medium text-theme-primary">{group}</div>
                      <button
                        onClick={() => handleRemoveGroup(group)}
                        className="text-red-600 hover:text-red-800"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))
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
    </div>
  )
}

// Manage Spoke Access Modal Component
function ManageSpokeAccessModal({ spoke, onClose }: { spoke: MeshSpoke; onClose: () => void }) {
  const [activeTab, setActiveTab] = useState<'users' | 'groups'>('users')
  const [users, setUsers] = useState<string[]>([])
  const [groups, setGroups] = useState<string[]>([])
  const [allUsers, setAllUsers] = useState<{ id: string; email: string; name: string }[]>([])
  const [allGroups, setAllGroups] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedUser, setSelectedUser] = useState('')
  const [customUserInput, setCustomUserInput] = useState('')
  const [selectedGroup, setSelectedGroup] = useState('')
  const [customGroupInput, setCustomGroupInput] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadData()
  }, [spoke.id])

  async function loadData() {
    try {
      setLoading(true)
      const [spokeUsers, spokeGroups, userList, groupList] = await Promise.all([
        getMeshSpokeUsers(spoke.id),
        getMeshSpokeGroups(spoke.id),
        getUsers(),
        getGroups(),
      ])
      setUsers(spokeUsers)
      setGroups(spokeGroups)
      setAllUsers(userList)
      setAllGroups(groupList.map(g => g.name))
    } catch (err) {
      setError('Failed to load access data')
    } finally {
      setLoading(false)
    }
  }

  async function handleAddUser() {
    const userToAdd = selectedUser || customUserInput.trim()
    if (!userToAdd) return
    if (users.includes(userToAdd)) {
      setError('User already assigned')
      return
    }
    try {
      await assignMeshSpokeUser(spoke.id, userToAdd)
      setUsers([...users, userToAdd])
      setSelectedUser('')
      setCustomUserInput('')
      setError(null)
    } catch (err) {
      setError('Failed to add user')
    }
  }

  async function handleRemoveUser(userId: string) {
    try {
      await removeMeshSpokeUser(spoke.id, userId)
      setUsers(users.filter(u => u !== userId))
    } catch (err) {
      setError('Failed to remove user')
    }
  }

  async function handleAddGroup() {
    const groupToAdd = selectedGroup || customGroupInput.trim()
    if (!groupToAdd) return
    if (groups.includes(groupToAdd)) {
      setError('Group already assigned')
      return
    }
    try {
      await assignMeshSpokeGroup(spoke.id, groupToAdd)
      setGroups([...groups, groupToAdd])
      setSelectedGroup('')
      setCustomGroupInput('')
      setError(null)
    } catch (err) {
      setError('Failed to add group')
    }
  }

  async function handleRemoveGroup(groupName: string) {
    try {
      await removeMeshSpokeGroup(spoke.id, groupName)
      setGroups(groups.filter(g => g !== groupName))
    } catch (err) {
      setError('Failed to remove group')
    }
  }

  const availableUsers = allUsers.filter(u => !users.includes(u.id))
  const availableGroups = allGroups.filter(g => !groups.includes(g))

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-2xl w-full p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-theme-primary">Manage Spoke Access - {spoke.name}</h2>
              <p className="text-sm text-theme-tertiary mt-1">
                Control which users/groups can route traffic to networks behind this spoke
              </p>
            </div>
            <button onClick={onClose} className="text-theme-muted hover:text-theme-tertiary">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Show spoke networks */}
          {spoke.localNetworks.length > 0 && (
            <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
              <div className="text-sm font-medium text-gray-800 dark:text-blue-300 mb-1">Networks accessible via this spoke:</div>
              <div className="flex flex-wrap gap-2">
                {spoke.localNetworks.map((net, i) => (
                  <span key={i} className="px-2 py-1 bg-blue-600 text-white rounded text-sm font-mono">{net}</span>
                ))}
              </div>
            </div>
          )}

          {error && (
            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-400 text-sm">
              {error}
            </div>
          )}

          {/* Tabs */}
          <div className="border-b border-theme mb-4">
            <nav className="-mb-px flex space-x-8">
              <button
                onClick={() => setActiveTab('users')}
                className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'users'
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary'
                }`}
              >
                Users ({users.length})
              </button>
              <button
                onClick={() => setActiveTab('groups')}
                className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'groups'
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary'
                }`}
              >
                Groups ({groups.length})
              </button>
            </nav>
          </div>

          {loading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : activeTab === 'users' ? (
            <div className="space-y-4">
              {/* Add User */}
              <div className="space-y-2">
                <div className="flex space-x-2">
                  <select
                    value={selectedUser}
                    onChange={(e) => {
                      setSelectedUser(e.target.value)
                      setCustomUserInput('')
                    }}
                    className="input flex-1 text-sm"
                  >
                    <option value="">Select from list...</option>
                    {availableUsers.map((user) => (
                      <option key={user.id} value={user.id}>
                        {user.email} {user.name && `(${user.name})`}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-theme-tertiary">or type:</span>
                  <input
                    type="text"
                    value={customUserInput}
                    onChange={(e) => {
                      setCustomUserInput(e.target.value)
                      setSelectedUser('')
                    }}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddUser())}
                    placeholder="username or email"
                    className="input flex-1 text-sm"
                  />
                  <button
                    onClick={handleAddUser}
                    disabled={!selectedUser && !customUserInput.trim()}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* User List */}
              <div className="border border-theme rounded-lg divide-y divide-theme">
                {users.length === 0 ? (
                  <div className="p-4 text-center text-theme-tertiary">
                    No users have access to this spoke's networks
                  </div>
                ) : (
                  users.map((userId) => {
                    const user = allUsers.find(u => u.id === userId)
                    return (
                      <div key={userId} className="p-3 flex items-center justify-between bg-theme-card">
                        <div>
                          <div className="font-medium text-theme-primary">
                            {user?.email || userId}
                          </div>
                          {user?.name && (
                            <div className="text-sm text-theme-tertiary">{user.name}</div>
                          )}
                        </div>
                        <button
                          onClick={() => handleRemoveUser(userId)}
                          className="text-red-600 hover:text-red-800"
                        >
                          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                    )
                  })
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Add Group */}
              <div className="space-y-2">
                <div className="flex space-x-2">
                  <select
                    value={selectedGroup}
                    onChange={(e) => {
                      setSelectedGroup(e.target.value)
                      setCustomGroupInput('')
                    }}
                    className="input flex-1 text-sm"
                  >
                    <option value="">Select from list...</option>
                    {availableGroups.map((group) => (
                      <option key={group} value={group}>
                        {group}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-theme-tertiary">or type:</span>
                  <input
                    type="text"
                    value={customGroupInput}
                    onChange={(e) => {
                      setCustomGroupInput(e.target.value)
                      setSelectedGroup('')
                    }}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddGroup())}
                    placeholder="group name"
                    className="input flex-1 text-sm"
                  />
                  <button
                    onClick={handleAddGroup}
                    disabled={!selectedGroup && !customGroupInput.trim()}
                    className="btn btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* Group List */}
              <div className="border border-theme rounded-lg divide-y divide-theme">
                {groups.length === 0 ? (
                  <div className="p-4 text-center text-theme-tertiary">
                    No groups have access to this spoke's networks
                  </div>
                ) : (
                  groups.map((group) => (
                    <div key={group} className="p-3 flex items-center justify-between bg-theme-card">
                      <div className="font-medium text-theme-primary">{group}</div>
                      <button
                        onClick={() => handleRemoveGroup(group)}
                        className="text-red-600 hover:text-red-800"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))
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
    </div>
  )
}

// Edit Hub Modal Component
function EditHubModal({ hub, onClose, onSuccess }: { hub: MeshHub; onClose: () => void; onSuccess: () => void }) {
  const [form, setForm] = useState({ name: hub.name, description: hub.description || '', publicEndpoint: hub.publicEndpoint || '', vpnPort: hub.vpnPort || 1194, vpnProtocol: hub.vpnProtocol || 'udp', vpnSubnet: hub.vpnSubnet || '172.30.0.0/16', cryptoProfile: hub.cryptoProfile || 'modern' as CryptoProfile, tlsAuthEnabled: hub.tlsAuthEnabled ?? true, fullTunnelMode: hub.fullTunnelMode ?? false, pushDns: hub.pushDns ?? false, dnsServers: hub.dnsServers || [], localNetworks: hub.localNetworks || [], sessionEnabled: hub.sessionEnabled ?? true })
  const [dnsInput, setDnsInput] = useState('')
  const [networkInput, setNetworkInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setLoading(true); setError(null)
    try { await updateMeshHub(hub.id, form as any); onSuccess() } catch { setError('Failed to update hub') } finally { setLoading(false) }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-lg w-full p-6 max-h-[90vh] overflow-y-auto">
          <h2 className="text-lg font-semibold mb-4 text-theme-primary">Edit Hub: {hub.name}</h2>
          {error && <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded">{error}</div>}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div><label className="block text-sm font-medium text-theme-secondary">Name</label><input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="input" required /></div>
            <div><label className="block text-sm font-medium text-theme-secondary">Description</label><input type="text" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} className="input" /></div>
            <div><label className="block text-sm font-medium text-theme-secondary">Public Endpoint</label><input type="text" value={form.publicEndpoint} onChange={(e) => setForm({ ...form, publicEndpoint: e.target.value })} className="input" /></div>
            <div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium text-theme-secondary">Port</label><input type="number" value={form.vpnPort} onChange={(e) => setForm({ ...form, vpnPort: parseInt(e.target.value) })} className="input" /></div><div><label className="block text-sm font-medium text-theme-secondary">Protocol</label><select value={form.vpnProtocol} onChange={(e) => setForm({ ...form, vpnProtocol: e.target.value })} className="input"><option value="udp">UDP</option><option value="tcp">TCP</option></select></div></div>
            <div><label className="block text-sm font-medium text-theme-secondary">VPN Subnet</label><input type="text" value={form.vpnSubnet} onChange={(e) => setForm({ ...form, vpnSubnet: e.target.value })} className="input" /></div>
            <div><label className="block text-sm font-medium text-theme-secondary">Crypto Profile</label><select value={form.cryptoProfile} onChange={(e) => setForm({ ...form, cryptoProfile: e.target.value as CryptoProfile })} className="input"><option value="modern">Modern</option><option value="fips">FIPS</option><option value="compatible">Compatible</option></select></div>
            <div className="space-y-2"><label className="flex items-center"><input type="checkbox" checked={form.tlsAuthEnabled} onChange={(e) => setForm({ ...form, tlsAuthEnabled: e.target.checked })} className="mr-2" /><span className="text-sm">TLS-Auth</span></label><label className="flex items-center"><input type="checkbox" checked={form.fullTunnelMode} onChange={(e) => setForm({ ...form, fullTunnelMode: e.target.checked })} className="mr-2" /><span className="text-sm">Full Tunnel</span></label><label className="flex items-center"><input type="checkbox" checked={form.pushDns} onChange={(e) => setForm({ ...form, pushDns: e.target.checked })} className="mr-2" /><span className="text-sm">Push DNS</span></label></div>
            {form.pushDns && <div><label className="block text-sm font-medium text-theme-secondary">DNS Servers</label><div className="flex space-x-2"><input type="text" value={dnsInput} onChange={(e) => setDnsInput(e.target.value)} className="input flex-1" placeholder="1.1.1.1" onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); if (dnsInput) { setForm({ ...form, dnsServers: [...form.dnsServers, dnsInput] }); setDnsInput('') } } }} /><button type="button" onClick={() => { if (dnsInput) { setForm({ ...form, dnsServers: [...form.dnsServers, dnsInput] }); setDnsInput('') } }} className="btn btn-secondary">Add</button></div>{form.dnsServers.length > 0 && <div className="flex flex-wrap gap-2 mt-2">{form.dnsServers.map((d) => <span key={d} className="px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm text-theme-primary">{d}<button type="button" onClick={() => setForm({ ...form, dnsServers: form.dnsServers.filter(x => x !== d) })} className="ml-1 text-red-600">×</button></span>)}</div>}</div>}
            <div><label className="block text-sm font-medium text-theme-secondary">Local Networks</label><div className="flex space-x-2"><input type="text" value={networkInput} onChange={(e) => setNetworkInput(e.target.value)} className="input flex-1" placeholder="192.168.1.0/24" onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); if (networkInput) { setForm({ ...form, localNetworks: [...form.localNetworks, networkInput] }); setNetworkInput('') } } }} /><button type="button" onClick={() => { if (networkInput) { setForm({ ...form, localNetworks: [...form.localNetworks, networkInput] }); setNetworkInput('') } }} className="btn btn-secondary">Add</button></div>{form.localNetworks.length > 0 && <div className="flex flex-wrap gap-2 mt-2">{form.localNetworks.map((n) => <span key={n} className="px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm text-theme-primary">{n}<button type="button" onClick={() => setForm({ ...form, localNetworks: form.localNetworks.filter(x => x !== n) })} className="ml-1 text-red-600">×</button></span>)}</div>}</div>
            <div className="flex items-center"><input type="checkbox" id="editHubSessionEnabled" checked={form.sessionEnabled ?? true} onChange={(e) => setForm({ ...form, sessionEnabled: e.target.checked })} className="rounded border-theme text-primary-600 focus:ring-primary-500" /><label htmlFor="editHubSessionEnabled" className="ml-2 text-sm text-theme-secondary">Enable Remote Sessions</label></div>
            <p className="text-xs text-theme-tertiary -mt-2">Allow administrators to run commands on this hub via the Remote Sessions page.</p>
            <div className="flex justify-end space-x-3 pt-4"><button type="button" onClick={onClose} className="btn btn-secondary">Cancel</button><button type="submit" disabled={loading} className="btn btn-primary">{loading ? 'Saving...' : 'Save'}</button></div>
          </form>
        </div>
      </div>
    </div>
  )
}

// Edit Spoke Modal Component
function EditSpokeModal({ spoke, onClose, onSuccess }: { spoke: MeshSpoke; onClose: () => void; onSuccess: () => void }) {
  const [form, setForm] = useState({ name: spoke.name, description: spoke.description || '', localNetworks: spoke.localNetworks || [], sessionEnabled: spoke.sessionEnabled ?? true })
  const [networkInput, setNetworkInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault(); setLoading(true); setError(null)
    try { await updateMeshSpoke(spoke.id, form); onSuccess() } catch { setError('Failed to update spoke') } finally { setLoading(false) }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="relative modal-content max-w-lg w-full p-6">
          <h2 className="text-lg font-semibold mb-4 text-theme-primary">Edit Spoke: {spoke.name}</h2>
          {error && <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded">{error}</div>}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div><label className="block text-sm font-medium text-theme-secondary">Name</label><input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="input" required /></div>
            <div><label className="block text-sm font-medium text-theme-secondary">Description</label><input type="text" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} className="input" /></div>
            <div><label className="block text-sm font-medium text-theme-secondary">Local Networks</label><div className="flex space-x-2"><input type="text" value={networkInput} onChange={(e) => setNetworkInput(e.target.value)} className="input flex-1" placeholder="e.g., 10.0.0.0/24" onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); if (networkInput && !form.localNetworks.includes(networkInput)) { setForm({ ...form, localNetworks: [...form.localNetworks, networkInput] }); setNetworkInput('') } } }} /><button type="button" onClick={() => { if (networkInput && !form.localNetworks.includes(networkInput)) { setForm({ ...form, localNetworks: [...form.localNetworks, networkInput] }); setNetworkInput('') } }} className="btn btn-secondary">Add</button></div><p className="text-xs text-theme-tertiary mt-1">Networks behind this spoke routable via hub</p>{form.localNetworks.length > 0 && <div className="flex flex-wrap gap-2 mt-2">{form.localNetworks.map((net) => <span key={net} className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-theme-secondary rounded text-sm flex items-center">{net}<button type="button" onClick={() => setForm({ ...form, localNetworks: form.localNetworks.filter(n => n !== net) })} className="ml-1 text-theme-muted hover:text-red-600">×</button></span>)}</div>}</div>
            <div className="flex items-center"><input type="checkbox" id="editSpokeSessionEnabled" checked={form.sessionEnabled ?? true} onChange={(e) => setForm({ ...form, sessionEnabled: e.target.checked })} className="rounded border-theme text-primary-600 focus:ring-primary-500" /><label htmlFor="editSpokeSessionEnabled" className="ml-2 text-sm text-theme-secondary">Enable Remote Sessions</label></div>
            <p className="text-xs text-theme-tertiary -mt-2">Allow administrators to run commands on this spoke via the Remote Sessions page.</p>
            <div className="flex justify-end space-x-3 pt-4"><button type="button" onClick={onClose} className="btn btn-secondary">Cancel</button><button type="submit" disabled={loading} className="btn btn-primary">{loading ? 'Saving...' : 'Save'}</button></div>
          </form>
        </div>
      </div>
    </div>
  )
}
