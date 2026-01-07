import { useState, useEffect } from 'react'
import {
  getTopology,
  getActiveSessions,
  TopologyResponse,
  TopologyGateway,
  TopologyMeshHub,
  TopologyMeshSpoke,
  ActiveSession,
} from '../api/client'
import {
  ComputerDesktopIcon,
  ServerStackIcon,
  GlobeAltIcon,
  ArrowRightIcon,
  XMarkIcon,
  UserGroupIcon,
  CubeTransparentIcon,
} from '@heroicons/react/24/outline'

type TabType = 'topology' | 'sessions'

// Node type for details panel
type SelectedNode = {
  type: 'gateway' | 'hub' | 'spoke' | 'user'
  data: TopologyGateway | TopologyMeshHub | TopologyMeshSpoke | { hubId: string; hubName: string; count: number }
}

// VPN Type Badge Component
function VpnTypeBadge({ type }: { type: string }) {
  const isWireGuard = type === 'wireguard'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${
      isWireGuard
        ? 'bg-violet-600 text-white'
        : 'bg-emerald-600 text-white'
    }`}>
      {isWireGuard ? 'WireGuard' : 'OpenVPN'}
    </span>
  )
}

export default function AdminTopology() {
  const [activeTab, setActiveTab] = useState<TabType>('topology')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [errorDetails, setErrorDetails] = useState<string | null>(null)

  // Topology state
  const [topology, setTopology] = useState<TopologyResponse | null>(null)
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null)
  const [selectedHubOrGateway, setSelectedHubOrGateway] = useState<string>('')

  // Sessions state
  const [sessions, setSessions] = useState<ActiveSession[]>([])
  const [sessionsTotal, setSessionsTotal] = useState(0)

  useEffect(() => {
    loadData()
  }, [activeTab])

  async function loadData() {
    try {
      setLoading(true)
      setError(null)

      if (activeTab === 'topology') {
        const topo = await getTopology()
        setTopology(topo)
      } else if (activeTab === 'sessions') {
        const result = await getActiveSessions()
        setSessions(result.sessions)
        setSessionsTotal(result.total)
      }
    } catch (err) {
      setError('Failed to load data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  // Group spokes by hub
  function getSpokesByHub(hubId: string): TopologyMeshSpoke[] {
    if (!topology) return []
    return topology.meshSpokes.filter(spoke => spoke.hubId === hubId)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">Network Topology</h1>
        <p className="text-theme-tertiary mt-1">
          View your network topology and active VPN sessions.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-theme">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('topology')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'topology'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Topology Map
          </button>
          <button
            onClick={() => setActiveTab('sessions')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'sessions'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
            }`}
          >
            Active Sessions
          </button>
        </nav>
      </div>

      {/* Error Message */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <div className="flex items-start">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800 dark:text-red-300">{error}</h3>
              {errorDetails && (
                <div className="mt-2 text-sm text-red-700 dark:text-red-400 whitespace-pre-line">
                  {errorDetails}
                </div>
              )}
            </div>
            <div className="ml-auto pl-3">
              <button
                onClick={() => { setError(null); setErrorDetails(null); }}
                className="inline-flex rounded-md bg-red-50 p-1.5 text-red-500 hover:bg-red-100"
              >
                <XMarkIcon className="h-5 w-5" />
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <>
          {/* Topology Map Tab - Static Left-to-Right Layout */}
          {activeTab === 'topology' && topology && (
            <div className="space-y-6">
              {/* Hub/Gateway Selector */}
              <div className="card">
                <label className="block text-sm font-medium text-theme-secondary mb-2">
                  Select Hub or Gateway to View
                </label>
                <div className="flex gap-4">
                  <select
                    value={selectedHubOrGateway}
                    onChange={(e) => {
                      setSelectedHubOrGateway(e.target.value)
                      setSelectedNode(null)
                    }}
                    className="flex-1 px-3 py-2 border border-theme rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 bg-theme-card text-theme-primary"
                  >
                    <option value="">-- Select a Hub or Gateway --</option>
                    {topology.meshHubs.length > 0 && (
                      <optgroup label="Mesh Hubs">
                        {topology.meshHubs.map((hub) => (
                          <option key={`hub-${hub.id}`} value={`hub-${hub.id}`}>
                            {hub.name} ({hub.status}) - {hub.connectedSpokes} spokes, {hub.connectedUsers} users
                          </option>
                        ))}
                      </optgroup>
                    )}
                    {topology.gateways.length > 0 && (
                      <optgroup label="Gateways">
                        {topology.gateways.map((gw) => (
                          <option key={`gateway-${gw.id}`} value={`gateway-${gw.id}`}>
                            {gw.name} ({gw.isActive ? 'active' : 'inactive'}) - {gw.clientCount} clients
                          </option>
                        ))}
                      </optgroup>
                    )}
                  </select>
                </div>
              </div>

              {/* Empty state when no selection */}
              {!selectedHubOrGateway && (
                <div className="card text-center py-12 text-theme-tertiary">
                  <ServerStackIcon className="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
                  <p>Select a hub or gateway above to view its topology</p>
                </div>
              )}

              {/* Show selected hub topology */}
              {selectedHubOrGateway.startsWith('hub-') && (
                <div className="flex gap-6">
                  {/* Main topology diagram */}
                  <div className="flex-1 card overflow-x-auto">
                    <div className="min-w-[800px]">
                      {topology.meshHubs
                        .filter((hub) => `hub-${hub.id}` === selectedHubOrGateway)
                        .map((hub) => {
                    const spokes = getSpokesByHub(hub.id)
                    const isOnline = hub.status === 'online'

                    return (
                      <div key={hub.id} className="mb-8 last:mb-0">
                        {/* Hub Label */}
                        <div className="flex items-center gap-3 mb-4">
                          <ServerStackIcon className="w-6 h-6 text-slate-600 dark:text-slate-400" />
                          <h3 className="text-lg font-semibold text-theme-primary">Mesh: {hub.name}</h3>
                          <VpnTypeBadge type={hub.gatewayType || 'openvpn'} />
                        </div>

                        {/* Flow Diagram: Users -> Hub -> Spokes */}
                        <div className="flex items-start gap-4">
                          {/* Users Column */}
                          <div className="flex-shrink-0 w-52">
                            <button
                              onClick={() => setSelectedNode({
                                type: 'user',
                                data: { hubId: hub.id, hubName: hub.name, count: hub.connectedUsers }
                              })}
                              className="w-full p-4 rounded-xl border-2 border-cyan-400 bg-white dark:border-slate-600 dark:bg-slate-800 hover:border-cyan-500 dark:hover:border-cyan-400 transition-all duration-200 cursor-pointer text-left shadow-sm hover:shadow-md"
                            >
                              <div className="flex items-center gap-2 mb-3">
                                <div className="p-2 rounded-lg bg-cyan-100 dark:bg-cyan-900/50">
                                  <UserGroupIcon className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
                                </div>
                                <span className="font-semibold text-slate-700 dark:text-slate-200">Users</span>
                              </div>
                              <div className="text-3xl font-bold text-slate-900 dark:text-white">{hub.connectedUsers}</div>
                              <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">connected clients</div>
                            </button>
                          </div>

                          {/* Arrow */}
                          <div className="flex-shrink-0 flex flex-col items-center justify-center pt-10">
                            <ArrowRightIcon className="w-6 h-6 text-slate-400 dark:text-slate-500" />
                            <div className="text-xs text-slate-400 dark:text-slate-500 mt-1 font-medium">VPN</div>
                          </div>

                          {/* Hub Column */}
                          <div className="flex-shrink-0 w-72">
                            <button
                              onClick={() => setSelectedNode({ type: 'hub', data: hub })}
                              className={`w-full p-4 rounded-xl border-2 bg-white dark:bg-slate-800 transition-all duration-200 cursor-pointer text-left shadow-sm hover:shadow-md ${
                                isOnline
                                  ? 'border-indigo-400 dark:border-indigo-500 hover:border-indigo-500 dark:hover:border-indigo-400'
                                  : 'border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500'
                              }`}
                            >
                              <div className="flex items-center gap-2 mb-2">
                                <div className={`w-2.5 h-2.5 rounded-full ${isOnline ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'}`} />
                                <span className="font-semibold text-slate-800 dark:text-slate-100">{hub.name}</span>
                              </div>
                              <div className="flex items-center gap-2 mb-3">
                                <div className={`p-1.5 rounded-lg ${isOnline ? 'bg-indigo-100 dark:bg-indigo-900/50' : 'bg-slate-200 dark:bg-slate-700'}`}>
                                  <ServerStackIcon className={`w-4 h-4 ${isOnline ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-500'}`} />
                                </div>
                                <span className="text-xs text-slate-500 dark:text-slate-400">Mesh Hub</span>
                                <VpnTypeBadge type={hub.gatewayType || 'openvpn'} />
                              </div>

                              {/* Hub IPs */}
                              <div className="space-y-1.5 text-xs border-t border-slate-200 dark:border-slate-700 pt-3 mt-2">
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Public:</span>
                                  <span className="font-mono text-slate-700 dark:text-slate-300">{hub.publicIp}:{hub.vpnPort}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Tunnel:</span>
                                  <span className="font-mono text-indigo-600 dark:text-indigo-400">{hub.serverTunnelIp}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Subnet:</span>
                                  <span className="font-mono text-slate-600 dark:text-slate-400">{hub.vpnSubnet}</span>
                                </div>
                              </div>

                              {/* Status */}
                              <div className="mt-3 pt-2 border-t border-slate-200 dark:border-slate-700">
                                <div className="flex justify-between text-xs">
                                  <span className="text-slate-500 dark:text-slate-400">Spokes:</span>
                                  <span className="font-semibold text-slate-700 dark:text-slate-200">{hub.connectedSpokes}</span>
                                </div>
                              </div>
                            </button>
                          </div>

                          {/* Arrow */}
                          <div className="flex-shrink-0 flex flex-col items-center justify-center pt-10">
                            <ArrowRightIcon className="w-6 h-6 text-slate-400 dark:text-slate-500" />
                            <div className="text-xs text-slate-400 dark:text-slate-500 mt-1 font-medium">Mesh</div>
                          </div>

                          {/* Spokes Column */}
                          <div className="flex-1 space-y-3">
                            {spokes.length === 0 ? (
                              <div className="p-4 rounded-xl border-2 border-dashed border-slate-300 bg-white dark:border-slate-600 dark:bg-slate-800 text-center text-slate-500 dark:text-slate-400">
                                No spokes connected
                              </div>
                            ) : (
                              spokes.map((spoke) => {
                                const isConnected = spoke.status === 'connected'
                                return (
                                  <button
                                    key={spoke.id}
                                    onClick={() => setSelectedNode({ type: 'spoke', data: spoke })}
                                    className={`w-full p-4 rounded-xl border-2 bg-white dark:bg-slate-800 transition-all duration-200 cursor-pointer text-left shadow-sm hover:shadow-md ${
                                      isConnected
                                        ? 'border-amber-400 dark:border-amber-500 hover:border-amber-500 dark:hover:border-amber-400'
                                        : 'border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500'
                                    }`}
                                  >
                                    <div className="flex items-center gap-2 mb-2">
                                      <div className={`w-2.5 h-2.5 rounded-full ${isConnected ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'}`} />
                                      <span className="font-semibold text-slate-800 dark:text-slate-100">{spoke.name}</span>
                                      <span className={`ml-auto text-xs px-2 py-0.5 rounded-full ${
                                        isConnected
                                          ? 'bg-emerald-100 dark:bg-emerald-900/40 text-emerald-700 dark:text-emerald-300'
                                          : 'bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-400'
                                      }`}>
                                        {spoke.status}
                                      </span>
                                    </div>

                                    <div className="flex items-center gap-2 mb-2">
                                      <div className={`p-1.5 rounded-lg ${isConnected ? 'bg-amber-100 dark:bg-amber-900/40' : 'bg-slate-200 dark:bg-slate-700'}`}>
                                        <CubeTransparentIcon className={`w-4 h-4 ${isConnected ? 'text-amber-600 dark:text-amber-400' : 'text-slate-500'}`} />
                                      </div>
                                      <span className="text-xs text-slate-500 dark:text-slate-400">Spoke</span>
                                      <VpnTypeBadge type={spoke.gatewayType || 'openvpn'} />
                                    </div>

                                    {/* Spoke IPs */}
                                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs border-t border-slate-200 dark:border-slate-700 pt-2 mt-2">
                                      <div>
                                        <span className="text-slate-500 dark:text-slate-400">Tunnel: </span>
                                        <span className="font-mono text-amber-600 dark:text-amber-400">{spoke.tunnelIp || 'N/A'}</span>
                                      </div>
                                      <div>
                                        <span className="text-slate-500 dark:text-slate-400">Remote: </span>
                                        <span className="font-mono text-slate-600 dark:text-slate-400">{spoke.remoteIp || 'N/A'}</span>
                                      </div>
                                    </div>

                                    {/* Local Networks */}
                                    {spoke.localNetworks.length > 0 && (
                                      <div className="mt-2 pt-2 border-t border-slate-200 dark:border-slate-700">
                                        <div className="text-xs text-emerald-600 dark:text-emerald-400 mb-1.5">Routes:</div>
                                        <div className="flex flex-wrap gap-1.5">
                                          {spoke.localNetworks.slice(0, 3).map((net, i) => (
                                            <span
                                              key={i}
                                              className="px-2 py-0.5 bg-emerald-100 dark:bg-emerald-900/40 text-emerald-700 dark:text-emerald-300 text-xs font-mono rounded-md"
                                            >
                                              {net}
                                            </span>
                                          ))}
                                          {spoke.localNetworks.length > 3 && (
                                            <span className="px-2 py-0.5 bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 text-xs rounded-md">
                                              +{spoke.localNetworks.length - 3} more
                                            </span>
                                          )}
                                        </div>
                                      </div>
                                    )}
                                  </button>
                                )
                              })
                            )}
                          </div>
                        </div>
                      </div>
                    )
                  })}

                    </div>
                  </div>

                  {/* Details Panel for Hub */}
                  {selectedNode && (
                    <div className="w-96 card">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-medium text-theme-primary">Node Details</h3>
                        <button
                          onClick={() => setSelectedNode(null)}
                          className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
                        >
                          <XMarkIcon className="w-5 h-5 text-slate-400" />
                        </button>
                      </div>

                      {/* User Details */}
                      {selectedNode.type === 'user' && (
                        <div className="space-y-4">
                          <div className="p-3 bg-cyan-50 dark:bg-cyan-950 rounded-lg border border-cyan-200 dark:border-cyan-800">
                            <div className="flex items-center gap-2">
                              <UserGroupIcon className="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                              <span className="text-lg font-medium text-slate-800 dark:text-slate-200">Connected Users</span>
                            </div>
                          </div>
                          <div className="space-y-3">
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Hub</label>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{(selectedNode.data as { hubName: string }).hubName}</div>
                            </div>
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Active Connections</label>
                              <div className="text-2xl font-bold text-cyan-600 dark:text-cyan-400">{(selectedNode.data as { count: number }).count}</div>
                            </div>
                          </div>
                          <div className="pt-3 border-t border-slate-200 dark:border-slate-700">
                            <a href="#" onClick={(e) => { e.preventDefault(); setActiveTab('sessions') }} className="text-sm text-primary-600 hover:text-primary-800">
                              View Active Sessions →
                            </a>
                          </div>
                        </div>
                      )}

                      {/* Hub Details */}
                      {selectedNode.type === 'hub' && (
                        <div className="space-y-4">
                          <div className={`p-3 rounded-lg border-2 bg-white dark:bg-slate-800 ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'border-indigo-400 dark:border-indigo-500' : 'border-slate-300 dark:border-slate-600'}`}>
                            <div className="flex items-center gap-2">
                              <ServerStackIcon className={`w-6 h-6 ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-500'}`} />
                              <span className={`text-lg font-medium ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-slate-800 dark:text-slate-200' : 'text-slate-600 dark:text-slate-400'}`}>
                                {(selectedNode.data as TopologyMeshHub).name}
                              </span>
                            </div>
                            <div className="flex items-center gap-2 mt-1">
                              <span className="text-xs text-slate-500 dark:text-slate-400">Mesh Hub</span>
                              <VpnTypeBadge type={(selectedNode.data as TopologyMeshHub).gatewayType || 'openvpn'} />
                            </div>
                          </div>
                          <div className="space-y-3">
                            <div className="p-3 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg">
                              <div className="text-xs text-slate-500 dark:text-slate-400 mb-2">Network Endpoints</div>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Public:</span>
                                  <span className="font-mono text-slate-700 dark:text-slate-300">{(selectedNode.data as TopologyMeshHub).publicEndpoint}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Server Tunnel IP:</span>
                                  <span className="font-mono text-indigo-600 dark:text-indigo-400">{(selectedNode.data as TopologyMeshHub).serverTunnelIp}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">VPN Subnet:</span>
                                  <span className="font-mono text-slate-600 dark:text-slate-400">{(selectedNode.data as TopologyMeshHub).vpnSubnet}</span>
                                </div>
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Status</label>
                              <div className={`text-sm ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}`}>
                                {(selectedNode.data as TopologyMeshHub).status}
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Connected Spokes</label>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{(selectedNode.data as TopologyMeshHub).connectedSpokes}</div>
                            </div>
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Connected Users</label>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{(selectedNode.data as TopologyMeshHub).connectedUsers}</div>
                            </div>
                            {(selectedNode.data as TopologyMeshHub).localNetworks?.length > 0 && (
                              <div>
                                <label className="text-xs text-slate-500 dark:text-slate-400">Hub Local Networks</label>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {(selectedNode.data as TopologyMeshHub).localNetworks.map((net, i) => (
                                    <span key={i} className="px-2 py-0.5 bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 text-xs font-mono rounded-md">{net}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {(selectedNode.data as TopologyMeshHub).lastHeartbeat && (
                              <div>
                                <label className="text-xs text-slate-500 dark:text-slate-400">Last Heartbeat</label>
                                <div className="text-sm text-slate-700 dark:text-slate-300">{formatDate((selectedNode.data as TopologyMeshHub).lastHeartbeat!)}</div>
                              </div>
                            )}
                          </div>
                          <div className="pt-3 border-t border-slate-200 dark:border-slate-700">
                            <a href="/admin/mesh" className="block text-sm text-primary-600 hover:text-primary-800">Manage Mesh →</a>
                          </div>
                        </div>
                      )}

                      {/* Spoke Details */}
                      {selectedNode.type === 'spoke' && (
                        <div className="space-y-4">
                          <div className={`p-3 rounded-lg border-2 bg-white dark:bg-slate-800 ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'border-amber-400 dark:border-amber-500' : 'border-slate-300 dark:border-slate-600'}`}>
                            <div className="flex items-center gap-2">
                              <CubeTransparentIcon className={`w-6 h-6 ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-amber-600 dark:text-amber-400' : 'text-slate-500'}`} />
                              <span className={`text-lg font-medium ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-slate-800 dark:text-slate-200' : 'text-slate-600 dark:text-slate-400'}`}>
                                {(selectedNode.data as TopologyMeshSpoke).name}
                              </span>
                            </div>
                            <div className="flex items-center gap-2 mt-1">
                              <span className="text-xs text-slate-500 dark:text-slate-400">Mesh Spoke</span>
                              <VpnTypeBadge type={(selectedNode.data as TopologyMeshSpoke).gatewayType || 'openvpn'} />
                            </div>
                          </div>
                          <div className="space-y-3">
                            <div className="p-3 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg">
                              <div className="text-xs text-slate-500 dark:text-slate-400 mb-2">Network Endpoints</div>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Tunnel IP:</span>
                                  <span className="font-mono text-amber-600 dark:text-amber-400">{(selectedNode.data as TopologyMeshSpoke).tunnelIp || 'N/A'}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-slate-500 dark:text-slate-400">Remote Public IP:</span>
                                  <span className="font-mono text-slate-600 dark:text-slate-400">{(selectedNode.data as TopologyMeshSpoke).remoteIp || 'N/A'}</span>
                                </div>
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Status</label>
                              <div className={`text-sm ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}`}>
                                {(selectedNode.data as TopologyMeshSpoke).status}
                              </div>
                            </div>
                            {(selectedNode.data as TopologyMeshSpoke).localNetworks.length > 0 && (
                              <div>
                                <label className="text-xs text-slate-500 dark:text-slate-400">Advertised Routes</label>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {(selectedNode.data as TopologyMeshSpoke).localNetworks.map((net, i) => (
                                    <span key={i} className="px-2 py-0.5 bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300 text-xs font-mono rounded-md">{net}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {(selectedNode.data as TopologyMeshSpoke).lastSeen && (
                              <div>
                                <label className="text-xs text-slate-500 dark:text-slate-400">Last Seen</label>
                                <div className="text-sm text-slate-700 dark:text-slate-300">{formatDate((selectedNode.data as TopologyMeshSpoke).lastSeen!)}</div>
                              </div>
                            )}
                          </div>
                          <div className="pt-3 border-t border-slate-200 dark:border-slate-700">
                            <a href="/admin/mesh" className="block text-sm text-primary-600 hover:text-primary-800">Manage Mesh →</a>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}

              {/* Show selected gateway topology */}
              {selectedHubOrGateway.startsWith('gateway-') && (
                <div className="flex gap-6">
                  <div className="flex-1 card overflow-x-auto">
                    <div className="min-w-[600px]">
                      {topology.gateways
                        .filter((gw) => `gateway-${gw.id}` === selectedHubOrGateway)
                        .map((gw) => (
                          <div key={gw.id}>
                            <div className="flex items-center gap-3 mb-4">
                              <GlobeAltIcon className="w-6 h-6 text-slate-600 dark:text-slate-400" />
                              <h3 className="text-lg font-semibold text-theme-primary">Gateway: {gw.name}</h3>
                            </div>

                            {/* Flow Diagram: Users -> Gateway */}
                            <div className="flex items-start gap-4">
                              {/* Users Column */}
                              <div className="flex-shrink-0 w-52">
                                <div className="w-full p-4 rounded-xl border-2 border-cyan-400 bg-white dark:border-slate-600 dark:bg-slate-800 shadow-sm text-left">
                                  <div className="flex items-center gap-2 mb-3">
                                    <div className="p-2 rounded-lg bg-cyan-100 dark:bg-cyan-900/50">
                                      <ComputerDesktopIcon className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
                                    </div>
                                    <span className="font-semibold text-slate-700 dark:text-slate-200">Clients</span>
                                  </div>
                                  <div className="text-3xl font-bold text-slate-900 dark:text-white">{gw.clientCount}</div>
                                  <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">connected clients</div>
                                </div>
                              </div>

                              {/* Arrow */}
                              <div className="flex-shrink-0 flex flex-col items-center justify-center pt-10">
                                <ArrowRightIcon className="w-6 h-6 text-slate-400 dark:text-slate-500" />
                                <div className="text-xs text-slate-400 dark:text-slate-500 mt-1 font-medium">VPN</div>
                              </div>

                              {/* Gateway Column */}
                              <div className="flex-shrink-0 w-72">
                                <button
                                  onClick={() => setSelectedNode({ type: 'gateway', data: gw })}
                                  className={`w-full p-4 rounded-xl border-2 bg-white dark:bg-slate-800 transition-all duration-200 cursor-pointer text-left shadow-sm hover:shadow-md ${
                                    gw.isActive
                                      ? 'border-emerald-400 dark:border-emerald-500 hover:border-emerald-500 dark:hover:border-emerald-400'
                                      : 'border-slate-300 dark:border-slate-600 hover:border-slate-400 dark:hover:border-slate-500'
                                  }`}
                                >
                                  <div className="flex items-center gap-2 mb-2">
                                    <div className={`w-2.5 h-2.5 rounded-full ${gw.isActive ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'}`} />
                                    <span className="font-semibold text-slate-800 dark:text-slate-100">{gw.name}</span>
                                  </div>
                                  <div className="flex items-center gap-2 mb-3">
                                    <div className={`p-1.5 rounded-lg ${gw.isActive ? 'bg-emerald-100 dark:bg-emerald-900/50' : 'bg-slate-200 dark:bg-slate-700'}`}>
                                      <GlobeAltIcon className={`w-4 h-4 ${gw.isActive ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}`} />
                                    </div>
                                    <span className="text-xs text-slate-500 dark:text-slate-400">Gateway</span>
                                  </div>
                                  <div className="space-y-1.5 text-xs border-t border-slate-200 dark:border-slate-700 pt-3 mt-2">
                                    <div className="flex justify-between">
                                      <span className="text-slate-500 dark:text-slate-400">Public:</span>
                                      <span className="font-mono text-slate-700 dark:text-slate-300">{gw.publicIp}:{gw.vpnPort}</span>
                                    </div>
                                    <div className="flex justify-between">
                                      <span className="text-slate-500 dark:text-slate-400">Protocol:</span>
                                      <span className="text-slate-700 dark:text-slate-300">{gw.vpnProtocol?.toUpperCase()}</span>
                                    </div>
                                    <div className="flex justify-between">
                                      <span className="text-slate-500 dark:text-slate-400">Status:</span>
                                      <span className={gw.isActive ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}>
                                        {gw.isActive ? 'Active' : 'Inactive'}
                                      </span>
                                    </div>
                                  </div>
                                </button>
                              </div>
                            </div>
                          </div>
                        ))}
                    </div>
                  </div>

                  {/* Details Panel for Gateway */}
                  {selectedNode && selectedNode.type === 'gateway' && (
                    <div className="w-96 card">
                      <div className="flex items-center justify-between mb-4">
                        <h3 className="text-lg font-medium text-theme-primary">Gateway Details</h3>
                        <button onClick={() => setSelectedNode(null)} className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
                          <XMarkIcon className="w-5 h-5 text-slate-400" />
                        </button>
                      </div>
                      <div className="space-y-4">
                        <div className={`p-3 rounded-lg border-2 bg-white dark:bg-slate-800 ${(selectedNode.data as TopologyGateway).isActive ? 'border-emerald-400 dark:border-emerald-500' : 'border-slate-300 dark:border-slate-600'}`}>
                          <div className="flex items-center gap-2">
                            <GlobeAltIcon className={`w-6 h-6 ${(selectedNode.data as TopologyGateway).isActive ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}`} />
                            <span className={`text-lg font-medium ${(selectedNode.data as TopologyGateway).isActive ? 'text-slate-800 dark:text-slate-200' : 'text-slate-600 dark:text-slate-400'}`}>
                              {(selectedNode.data as TopologyGateway).name}
                            </span>
                          </div>
                          <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">Gateway</div>
                        </div>
                        <div className="space-y-3">
                          <div>
                            <label className="text-xs text-slate-500 dark:text-slate-400">Hostname</label>
                            <div className="text-sm font-mono text-slate-700 dark:text-slate-300">{(selectedNode.data as TopologyGateway).hostname || 'N/A'}</div>
                          </div>
                          <div>
                            <label className="text-xs text-slate-500 dark:text-slate-400">Public IP</label>
                            <div className="text-sm font-mono text-slate-700 dark:text-slate-300">{(selectedNode.data as TopologyGateway).publicIp}</div>
                          </div>
                          <div>
                            <label className="text-xs text-slate-500 dark:text-slate-400">VPN Port</label>
                            <div className="text-sm text-slate-700 dark:text-slate-300">{(selectedNode.data as TopologyGateway).vpnPort} ({(selectedNode.data as TopologyGateway).vpnProtocol?.toUpperCase()})</div>
                          </div>
                          <div>
                            <label className="text-xs text-slate-500 dark:text-slate-400">Status</label>
                            <div className={`text-sm ${(selectedNode.data as TopologyGateway).isActive ? 'text-emerald-600 dark:text-emerald-400' : 'text-slate-500'}`}>
                              {(selectedNode.data as TopologyGateway).isActive ? 'Active' : 'Inactive'}
                            </div>
                          </div>
                          <div>
                            <label className="text-xs text-slate-500 dark:text-slate-400">Connected Clients</label>
                            <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{(selectedNode.data as TopologyGateway).clientCount}</div>
                          </div>
                          {(selectedNode.data as TopologyGateway).lastHeartbeat && (
                            <div>
                              <label className="text-xs text-slate-500 dark:text-slate-400">Last Heartbeat</label>
                              <div className="text-sm text-slate-700 dark:text-slate-300">{formatDate((selectedNode.data as TopologyGateway).lastHeartbeat!)}</div>
                            </div>
                          )}
                        </div>
                        <div className="pt-3 border-t border-slate-200 dark:border-slate-700 space-y-2">
                          <a href={`/admin/access-rules?gateway=${(selectedNode.data as TopologyGateway).id}`} className="block text-sm text-primary-600 hover:text-primary-800">
                            View Access Rules →
                          </a>
                          <a href="/admin/gateways" className="block text-sm text-primary-600 hover:text-primary-800">
                            Manage Gateways →
                          </a>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )}

            </div>
          )}

          {/* Active Sessions Tab */}
          {activeTab === 'sessions' && (
            <div className="card overflow-hidden">
              <div className="px-4 py-3 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
                <h3 className="font-medium text-theme-primary">Active VPN Sessions ({sessionsTotal})</h3>
                <button
                  onClick={loadData}
                  className="text-sm text-primary-600 hover:text-primary-800"
                >
                  Refresh
                </button>
              </div>

              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-200 dark:divide-slate-700">
                  <thead className="bg-slate-50 dark:bg-slate-800">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Hub</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Client IP</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">VPN Address</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Connected</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">Traffic</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-slate-900 divide-y divide-slate-100 dark:divide-slate-700">
                    {!sessions || sessions.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-4 py-8 text-center text-slate-400 dark:text-slate-500">
                          No active sessions
                        </td>
                      </tr>
                    ) : (
                      (sessions || []).map((session) => (
                        <tr key={session.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                          <td className="px-4 py-3 whitespace-nowrap">
                            <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{session.userEmail}</div>
                            {session.userName && (
                              <div className="text-xs text-slate-500 dark:text-slate-400">{session.userName}</div>
                            )}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-700 dark:text-slate-300">
                            {session.gatewayName}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-slate-500 dark:text-slate-400">
                            {session.clientIp}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-slate-500 dark:text-slate-400">
                            {session.vpnAddress}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                            {formatDate(session.connectedAt)}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm">
                            <span className="text-emerald-600 dark:text-emerald-400">{formatBytes(session.bytesSent)}</span>
                            {' / '}
                            <span className="text-sky-600 dark:text-sky-400">{formatBytes(session.bytesRecv)}</span>
                          </td>
                        </tr>
                      ))
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
