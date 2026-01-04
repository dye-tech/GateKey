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
  ServerIcon,
  GlobeAltIcon,
  ArrowRightIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline'

type TabType = 'topology' | 'sessions'

// Node type for details panel
type SelectedNode = {
  type: 'gateway' | 'hub' | 'spoke' | 'user'
  data: TopologyGateway | TopologyMeshHub | TopologyMeshSpoke | { hubId: string; hubName: string; count: number }
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
                  <ServerIcon className="w-12 h-12 mx-auto mb-4 text-gray-300" />
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
                        <h3 className="text-lg font-semibold text-theme-primary mb-4 flex items-center gap-2">
                          <ServerIcon className="w-5 h-5 text-blue-600" />
                          Mesh: {hub.name}
                        </h3>

                        {/* Flow Diagram: Users -> Hub -> Spokes */}
                        <div className="flex items-start gap-4">
                          {/* Users Column */}
                          <div className="flex-shrink-0 w-48">
                            <button
                              onClick={() => setSelectedNode({
                                type: 'user',
                                data: { hubId: hub.id, hubName: hub.name, count: hub.connectedUsers }
                              })}
                              className="w-full p-4 rounded-lg border-2 border-green-500 bg-white dark:bg-green-900/20 hover:border-green-600 transition-colors cursor-pointer text-left"
                            >
                              <div className="flex items-center gap-2 mb-2">
                                <ComputerDesktopIcon className="w-5 h-5 text-green-600" />
                                <span className="font-medium text-gray-800 dark:text-green-300">Users</span>
                              </div>
                              <div className="text-2xl font-bold text-gray-900 dark:text-green-400">{hub.connectedUsers}</div>
                              <div className="text-xs text-gray-600 dark:text-green-500">connected clients</div>
                            </button>
                          </div>

                          {/* Arrow */}
                          <div className="flex-shrink-0 flex flex-col items-center justify-center pt-8">
                            <ArrowRightIcon className="w-8 h-8 text-theme-muted" />
                            <div className="text-xs text-theme-muted mt-1">VPN</div>
                          </div>

                          {/* Hub Column */}
                          <div className="flex-shrink-0 w-64">
                            <button
                              onClick={() => setSelectedNode({ type: 'hub', data: hub })}
                              className={`w-full p-4 rounded-lg border-2 transition-colors cursor-pointer text-left ${
                                isOnline
                                  ? 'border-blue-500 bg-white dark:bg-blue-900/20 hover:border-blue-600'
                                  : 'border-theme bg-theme-tertiary hover:border-gray-400'
                              }`}
                            >
                              <div className="flex items-center gap-2 mb-2">
                                <div className={`w-3 h-3 rounded-full ${isOnline ? 'bg-blue-500' : 'bg-gray-400'}`} />
                                <span className={`font-medium ${isOnline ? 'text-gray-800 dark:text-blue-300' : 'text-theme-secondary'}`}>
                                  {hub.name}
                                </span>
                              </div>
                              <div className="text-xs text-theme-tertiary mb-2">Mesh Hub</div>

                              {/* Hub IPs */}
                              <div className="space-y-1 text-xs">
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Public:</span>
                                  <span className="font-mono text-theme-secondary">{hub.publicIp}:{hub.vpnPort}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Tunnel:</span>
                                  <span className="font-mono text-blue-700 dark:text-blue-400">{hub.serverTunnelIp}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Subnet:</span>
                                  <span className="font-mono text-theme-tertiary">{hub.vpnSubnet}</span>
                                </div>
                              </div>

                              {/* Status */}
                              <div className="mt-3 pt-2 border-t border-theme">
                                <div className="flex justify-between text-xs">
                                  <span className="text-theme-tertiary">Spokes:</span>
                                  <span className="font-medium">{hub.connectedSpokes}</span>
                                </div>
                              </div>
                            </button>
                          </div>

                          {/* Arrow */}
                          <div className="flex-shrink-0 flex flex-col items-center justify-center pt-8">
                            <ArrowRightIcon className="w-8 h-8 text-theme-muted" />
                            <div className="text-xs text-theme-muted mt-1">Mesh</div>
                          </div>

                          {/* Spokes Column */}
                          <div className="flex-1 space-y-3">
                            {spokes.length === 0 ? (
                              <div className="p-4 rounded-lg border-2 border-dashed border-theme bg-theme-tertiary text-center text-theme-tertiary">
                                No spokes connected
                              </div>
                            ) : (
                              spokes.map((spoke) => {
                                const isConnected = spoke.status === 'connected'
                                return (
                                  <button
                                    key={spoke.id}
                                    onClick={() => setSelectedNode({ type: 'spoke', data: spoke })}
                                    className={`w-full p-4 rounded-lg border-2 transition-colors cursor-pointer text-left ${
                                      isConnected
                                        ? 'border-purple-500 bg-white dark:bg-purple-900/20 hover:border-purple-600'
                                        : 'border-theme bg-theme-tertiary hover:border-gray-400'
                                    }`}
                                  >
                                    <div className="flex items-center gap-2 mb-2">
                                      <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-purple-500' : 'bg-gray-400'}`} />
                                      <span className={`font-medium ${isConnected ? 'text-gray-800 dark:text-purple-300' : 'text-theme-secondary'}`}>
                                        {spoke.name}
                                      </span>
                                      <span className={`ml-auto text-xs ${isConnected ? 'text-gray-600 dark:text-purple-400' : 'text-theme-tertiary'}`}>
                                        {spoke.status}
                                      </span>
                                    </div>

                                    {/* Spoke IPs */}
                                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                                      <div>
                                        <span className="text-theme-tertiary">Tunnel: </span>
                                        <span className="font-mono text-gray-700 dark:text-purple-400">{spoke.tunnelIp || 'N/A'}</span>
                                      </div>
                                      <div>
                                        <span className="text-theme-tertiary">Remote: </span>
                                        <span className="font-mono text-theme-secondary">{spoke.remoteIp || 'N/A'}</span>
                                      </div>
                                    </div>

                                    {/* Local Networks */}
                                    {spoke.localNetworks.length > 0 && (
                                      <div className="mt-2 pt-2 border-t border-theme">
                                        <div className="text-xs text-theme-tertiary mb-1">Routes:</div>
                                        <div className="flex flex-wrap gap-1">
                                          {spoke.localNetworks.slice(0, 3).map((net, i) => (
                                            <span
                                              key={i}
                                              className="px-1.5 py-0.5 bg-purple-600 text-white text-xs font-mono rounded"
                                            >
                                              {net}
                                            </span>
                                          ))}
                                          {spoke.localNetworks.length > 3 && (
                                            <span className="px-1.5 py-0.5 bg-gray-100 text-theme-tertiary text-xs rounded">
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
                          className="p-1 hover:bg-gray-100 rounded"
                        >
                          <XMarkIcon className="w-5 h-5 text-theme-tertiary" />
                        </button>
                      </div>

                      {/* User Details */}
                      {selectedNode.type === 'user' && (
                        <div className="space-y-4">
                          <div className="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                            <div className="flex items-center gap-2">
                              <ComputerDesktopIcon className="w-6 h-6 text-green-600" />
                              <span className="text-lg font-medium text-green-800 dark:text-green-300">Connected Users</span>
                            </div>
                          </div>
                          <div className="space-y-3">
                            <div>
                              <label className="text-xs text-theme-tertiary">Hub</label>
                              <div className="text-sm font-medium">{(selectedNode.data as { hubName: string }).hubName}</div>
                            </div>
                            <div>
                              <label className="text-xs text-theme-tertiary">Active Connections</label>
                              <div className="text-2xl font-bold text-green-700 dark:text-green-400">{(selectedNode.data as { count: number }).count}</div>
                            </div>
                          </div>
                          <div className="pt-3 border-t">
                            <a href="#" onClick={(e) => { e.preventDefault(); setActiveTab('sessions') }} className="text-sm text-primary-600 hover:text-primary-800">
                              View Active Sessions →
                            </a>
                          </div>
                        </div>
                      )}

                      {/* Hub Details */}
                      {selectedNode.type === 'hub' && (
                        <div className="space-y-4">
                          <div className={`p-3 rounded-lg ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'selected-highlight' : 'bg-theme-tertiary'}`}>
                            <div className="flex items-center gap-2">
                              <ServerIcon className={`w-6 h-6 ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-blue-600' : 'text-theme-tertiary'}`} />
                              <span className={`text-lg font-medium ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-blue-800 dark:text-blue-300' : 'text-theme-secondary'}`}>
                                {(selectedNode.data as TopologyMeshHub).name}
                              </span>
                            </div>
                            <div className="text-xs text-theme-tertiary mt-1">Mesh Hub</div>
                          </div>
                          <div className="space-y-3">
                            <div className="p-3 bg-theme-tertiary rounded-lg">
                              <div className="text-xs text-theme-tertiary mb-2">Network Endpoints</div>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Public:</span>
                                  <span className="font-mono">{(selectedNode.data as TopologyMeshHub).publicEndpoint}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Server Tunnel IP:</span>
                                  <span className="font-mono text-blue-700 dark:text-blue-400">{(selectedNode.data as TopologyMeshHub).serverTunnelIp}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">VPN Subnet:</span>
                                  <span className="font-mono">{(selectedNode.data as TopologyMeshHub).vpnSubnet}</span>
                                </div>
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-theme-tertiary">Status</label>
                              <div className={`text-sm ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-blue-600' : 'text-theme-tertiary'}`}>
                                {(selectedNode.data as TopologyMeshHub).status}
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-theme-tertiary">Connected Spokes</label>
                              <div className="text-sm font-medium">{(selectedNode.data as TopologyMeshHub).connectedSpokes}</div>
                            </div>
                            <div>
                              <label className="text-xs text-theme-tertiary">Connected Users</label>
                              <div className="text-sm font-medium">{(selectedNode.data as TopologyMeshHub).connectedUsers}</div>
                            </div>
                            {(selectedNode.data as TopologyMeshHub).localNetworks?.length > 0 && (
                              <div>
                                <label className="text-xs text-theme-tertiary">Hub Local Networks</label>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {(selectedNode.data as TopologyMeshHub).localNetworks.map((net, i) => (
                                    <span key={i} className="px-2 py-0.5 bg-blue-600 text-white text-xs font-mono rounded">{net}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {(selectedNode.data as TopologyMeshHub).lastHeartbeat && (
                              <div>
                                <label className="text-xs text-theme-tertiary">Last Heartbeat</label>
                                <div className="text-sm">{formatDate((selectedNode.data as TopologyMeshHub).lastHeartbeat!)}</div>
                              </div>
                            )}
                          </div>
                          <div className="pt-3 border-t">
                            <a href="/admin/mesh" className="block text-sm text-primary-600 hover:text-primary-800">Manage Mesh →</a>
                          </div>
                        </div>
                      )}

                      {/* Spoke Details */}
                      {selectedNode.type === 'spoke' && (
                        <div className="space-y-4">
                          <div className={`p-3 rounded-lg ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'selected-highlight' : 'bg-theme-tertiary'}`}>
                            <div className="flex items-center gap-2">
                              <ServerIcon className={`w-6 h-6 ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-purple-600' : 'text-theme-tertiary'}`} />
                              <span className={`text-lg font-medium ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-purple-800 dark:text-purple-300' : 'text-theme-secondary'}`}>
                                {(selectedNode.data as TopologyMeshSpoke).name}
                              </span>
                            </div>
                            <div className="text-xs text-theme-tertiary mt-1">Mesh Spoke</div>
                          </div>
                          <div className="space-y-3">
                            <div className="p-3 bg-theme-tertiary rounded-lg">
                              <div className="text-xs text-theme-tertiary mb-2">Network Endpoints</div>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Tunnel IP:</span>
                                  <span className="font-mono text-purple-700 dark:text-purple-400">{(selectedNode.data as TopologyMeshSpoke).tunnelIp || 'N/A'}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-theme-tertiary">Remote Public IP:</span>
                                  <span className="font-mono">{(selectedNode.data as TopologyMeshSpoke).remoteIp || 'N/A'}</span>
                                </div>
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-theme-tertiary">Status</label>
                              <div className={`text-sm ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-purple-600' : 'text-theme-tertiary'}`}>
                                {(selectedNode.data as TopologyMeshSpoke).status}
                              </div>
                            </div>
                            {(selectedNode.data as TopologyMeshSpoke).localNetworks.length > 0 && (
                              <div>
                                <label className="text-xs text-theme-tertiary">Advertised Routes</label>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {(selectedNode.data as TopologyMeshSpoke).localNetworks.map((net, i) => (
                                    <span key={i} className="px-2 py-0.5 bg-purple-600 text-white text-xs font-mono rounded">{net}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {(selectedNode.data as TopologyMeshSpoke).lastSeen && (
                              <div>
                                <label className="text-xs text-theme-tertiary">Last Seen</label>
                                <div className="text-sm">{formatDate((selectedNode.data as TopologyMeshSpoke).lastSeen!)}</div>
                              </div>
                            )}
                          </div>
                          <div className="pt-3 border-t">
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
                            <h3 className="text-lg font-semibold text-theme-primary mb-4 flex items-center gap-2">
                              <GlobeAltIcon className="w-5 h-5 text-green-600" />
                              Gateway: {gw.name}
                            </h3>

                            {/* Flow Diagram: Users -> Gateway */}
                            <div className="flex items-start gap-4">
                              {/* Users Column */}
                              <div className="flex-shrink-0 w-48">
                                <div className="w-full p-4 rounded-lg border-2 border-green-500 bg-white dark:bg-green-900/20 text-left">
                                  <div className="flex items-center gap-2 mb-2">
                                    <ComputerDesktopIcon className="w-5 h-5 text-green-600" />
                                    <span className="font-medium text-gray-800 dark:text-green-300">Clients</span>
                                  </div>
                                  <div className="text-2xl font-bold text-gray-900 dark:text-green-400">{gw.clientCount}</div>
                                  <div className="text-xs text-gray-600 dark:text-green-500">connected clients</div>
                                </div>
                              </div>

                              {/* Arrow */}
                              <div className="flex-shrink-0 flex flex-col items-center justify-center pt-8">
                                <ArrowRightIcon className="w-8 h-8 text-theme-muted" />
                                <div className="text-xs text-theme-muted mt-1">VPN</div>
                              </div>

                              {/* Gateway Column */}
                              <div className="flex-shrink-0 w-64">
                                <button
                                  onClick={() => setSelectedNode({ type: 'gateway', data: gw })}
                                  className={`w-full p-4 rounded-lg border-2 transition-colors cursor-pointer text-left ${
                                    gw.isActive
                                      ? 'border-green-500 bg-white dark:bg-green-900/20 hover:border-green-600'
                                      : 'border-theme bg-theme-tertiary hover:border-gray-400'
                                  }`}
                                >
                                  <div className="flex items-center gap-2 mb-2">
                                    <div className={`w-3 h-3 rounded-full ${gw.isActive ? 'bg-green-500' : 'bg-gray-400'}`} />
                                    <span className={`font-medium ${gw.isActive ? 'text-gray-800 dark:text-green-300' : 'text-theme-secondary'}`}>
                                      {gw.name}
                                    </span>
                                  </div>
                                  <div className="text-xs text-theme-tertiary mb-2">Gateway</div>
                                  <div className="space-y-1 text-xs">
                                    <div className="flex justify-between">
                                      <span className="text-theme-tertiary">Public:</span>
                                      <span className="font-mono text-theme-secondary">{gw.publicIp}:{gw.vpnPort}</span>
                                    </div>
                                    <div className="flex justify-between">
                                      <span className="text-theme-tertiary">Protocol:</span>
                                      <span className="text-theme-secondary">{gw.vpnProtocol?.toUpperCase()}</span>
                                    </div>
                                    <div className="flex justify-between">
                                      <span className="text-theme-tertiary">Status:</span>
                                      <span className={gw.isActive ? 'text-green-600' : 'text-theme-tertiary'}>
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
                        <button onClick={() => setSelectedNode(null)} className="p-1 hover:bg-gray-100 rounded">
                          <XMarkIcon className="w-5 h-5 text-theme-tertiary" />
                        </button>
                      </div>
                      <div className="space-y-4">
                        <div className={`p-3 rounded-lg ${(selectedNode.data as TopologyGateway).isActive ? 'selected-highlight' : 'bg-theme-tertiary'}`}>
                          <div className="flex items-center gap-2">
                            <GlobeAltIcon className={`w-6 h-6 ${(selectedNode.data as TopologyGateway).isActive ? 'text-green-600' : 'text-theme-tertiary'}`} />
                            <span className={`text-lg font-medium ${(selectedNode.data as TopologyGateway).isActive ? 'text-green-800 dark:text-green-300' : 'text-theme-secondary'}`}>
                              {(selectedNode.data as TopologyGateway).name}
                            </span>
                          </div>
                          <div className="text-xs text-theme-tertiary mt-1">Gateway</div>
                        </div>
                        <div className="space-y-3">
                          <div>
                            <label className="text-xs text-theme-tertiary">Hostname</label>
                            <div className="text-sm font-mono">{(selectedNode.data as TopologyGateway).hostname || 'N/A'}</div>
                          </div>
                          <div>
                            <label className="text-xs text-theme-tertiary">Public IP</label>
                            <div className="text-sm font-mono">{(selectedNode.data as TopologyGateway).publicIp}</div>
                          </div>
                          <div>
                            <label className="text-xs text-theme-tertiary">VPN Port</label>
                            <div className="text-sm">{(selectedNode.data as TopologyGateway).vpnPort} ({(selectedNode.data as TopologyGateway).vpnProtocol?.toUpperCase()})</div>
                          </div>
                          <div>
                            <label className="text-xs text-theme-tertiary">Status</label>
                            <div className={`text-sm ${(selectedNode.data as TopologyGateway).isActive ? 'text-green-600' : 'text-theme-tertiary'}`}>
                              {(selectedNode.data as TopologyGateway).isActive ? 'Active' : 'Inactive'}
                            </div>
                          </div>
                          <div>
                            <label className="text-xs text-theme-tertiary">Connected Clients</label>
                            <div className="text-sm font-medium">{(selectedNode.data as TopologyGateway).clientCount}</div>
                          </div>
                          {(selectedNode.data as TopologyGateway).lastHeartbeat && (
                            <div>
                              <label className="text-xs text-theme-tertiary">Last Heartbeat</label>
                              <div className="text-sm">{formatDate((selectedNode.data as TopologyGateway).lastHeartbeat!)}</div>
                            </div>
                          )}
                        </div>
                        <div className="pt-3 border-t space-y-2">
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
              <div className="px-4 py-3 border-b border-theme flex items-center justify-between">
                <h3 className="font-medium text-theme-primary">Active VPN Sessions ({sessionsTotal})</h3>
                <button
                  onClick={loadData}
                  className="text-sm text-primary-600 hover:text-primary-800"
                >
                  Refresh
                </button>
              </div>

              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Hub</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Client IP</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">VPN Address</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Connected</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Traffic</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {!sessions || sessions.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-4 py-8 text-center text-theme-tertiary">
                          No active sessions
                        </td>
                      </tr>
                    ) : (
                      (sessions || []).map((session) => (
                        <tr key={session.id} className="hover:bg-theme-tertiary">
                          <td className="px-4 py-3 whitespace-nowrap">
                            <div className="text-sm font-medium text-theme-primary">{session.userEmail}</div>
                            {session.userName && (
                              <div className="text-xs text-theme-tertiary">{session.userName}</div>
                            )}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-primary">
                            {session.gatewayName}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-theme-tertiary">
                            {session.clientIp}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-theme-tertiary">
                            {session.vpnAddress}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                            {formatDate(session.connectedAt)}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                            <span className="text-green-600">{formatBytes(session.bytesSent)}</span>
                            {' / '}
                            <span className="text-blue-600">{formatBytes(session.bytesRecv)}</span>
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
