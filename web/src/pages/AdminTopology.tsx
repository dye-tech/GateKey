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
        <h1 className="text-2xl font-bold text-gray-900">Network Topology</h1>
        <p className="text-gray-500 mt-1">
          View your network topology and active VPN sessions.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('topology')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'topology'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Topology Map
          </button>
          <button
            onClick={() => setActiveTab('sessions')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'sessions'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Active Sessions
          </button>
        </nav>
      </div>

      {/* Error Message */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
          <div className="flex items-start">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">{error}</h3>
              {errorDetails && (
                <div className="mt-2 text-sm text-red-700 whitespace-pre-line">
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
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Select Hub or Gateway to View
                </label>
                <div className="flex gap-4">
                  <select
                    value={selectedHubOrGateway}
                    onChange={(e) => {
                      setSelectedHubOrGateway(e.target.value)
                      setSelectedNode(null)
                    }}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
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
                <div className="card text-center py-12 text-gray-500">
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
                        <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
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
                              className="w-full p-4 rounded-lg border-2 border-green-300 bg-green-50 hover:border-green-500 transition-colors cursor-pointer text-left"
                            >
                              <div className="flex items-center gap-2 mb-2">
                                <ComputerDesktopIcon className="w-5 h-5 text-green-600" />
                                <span className="font-medium text-green-800">Users</span>
                              </div>
                              <div className="text-2xl font-bold text-green-700">{hub.connectedUsers}</div>
                              <div className="text-xs text-green-600">connected clients</div>
                            </button>
                          </div>

                          {/* Arrow */}
                          <div className="flex-shrink-0 flex flex-col items-center justify-center pt-8">
                            <ArrowRightIcon className="w-8 h-8 text-gray-400" />
                            <div className="text-xs text-gray-400 mt-1">VPN</div>
                          </div>

                          {/* Hub Column */}
                          <div className="flex-shrink-0 w-64">
                            <button
                              onClick={() => setSelectedNode({ type: 'hub', data: hub })}
                              className={`w-full p-4 rounded-lg border-2 transition-colors cursor-pointer text-left ${
                                isOnline
                                  ? 'border-blue-400 bg-blue-50 hover:border-blue-600'
                                  : 'border-gray-300 bg-gray-50 hover:border-gray-400'
                              }`}
                            >
                              <div className="flex items-center gap-2 mb-2">
                                <div className={`w-3 h-3 rounded-full ${isOnline ? 'bg-blue-500' : 'bg-gray-400'}`} />
                                <span className={`font-medium ${isOnline ? 'text-blue-800' : 'text-gray-700'}`}>
                                  {hub.name}
                                </span>
                              </div>
                              <div className="text-xs text-gray-600 mb-2">Mesh Hub</div>

                              {/* Hub IPs */}
                              <div className="space-y-1 text-xs">
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Public:</span>
                                  <span className="font-mono text-gray-700">{hub.publicIp}:{hub.vpnPort}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Tunnel:</span>
                                  <span className="font-mono text-blue-700">{hub.serverTunnelIp}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Subnet:</span>
                                  <span className="font-mono text-gray-600">{hub.vpnSubnet}</span>
                                </div>
                              </div>

                              {/* Status */}
                              <div className="mt-3 pt-2 border-t border-gray-200">
                                <div className="flex justify-between text-xs">
                                  <span className="text-gray-500">Spokes:</span>
                                  <span className="font-medium">{hub.connectedSpokes}</span>
                                </div>
                              </div>
                            </button>
                          </div>

                          {/* Arrow */}
                          <div className="flex-shrink-0 flex flex-col items-center justify-center pt-8">
                            <ArrowRightIcon className="w-8 h-8 text-gray-400" />
                            <div className="text-xs text-gray-400 mt-1">Mesh</div>
                          </div>

                          {/* Spokes Column */}
                          <div className="flex-1 space-y-3">
                            {spokes.length === 0 ? (
                              <div className="p-4 rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 text-center text-gray-500">
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
                                        ? 'border-purple-400 bg-purple-50 hover:border-purple-600'
                                        : 'border-gray-300 bg-gray-50 hover:border-gray-400'
                                    }`}
                                  >
                                    <div className="flex items-center gap-2 mb-2">
                                      <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-purple-500' : 'bg-gray-400'}`} />
                                      <span className={`font-medium ${isConnected ? 'text-purple-800' : 'text-gray-700'}`}>
                                        {spoke.name}
                                      </span>
                                      <span className={`ml-auto text-xs ${isConnected ? 'text-purple-600' : 'text-gray-500'}`}>
                                        {spoke.status}
                                      </span>
                                    </div>

                                    {/* Spoke IPs */}
                                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                                      <div>
                                        <span className="text-gray-500">Tunnel: </span>
                                        <span className="font-mono text-purple-700">{spoke.tunnelIp || 'N/A'}</span>
                                      </div>
                                      <div>
                                        <span className="text-gray-500">Remote: </span>
                                        <span className="font-mono text-gray-700">{spoke.remoteIp || 'N/A'}</span>
                                      </div>
                                    </div>

                                    {/* Local Networks */}
                                    {spoke.localNetworks.length > 0 && (
                                      <div className="mt-2 pt-2 border-t border-gray-200">
                                        <div className="text-xs text-gray-500 mb-1">Routes:</div>
                                        <div className="flex flex-wrap gap-1">
                                          {spoke.localNetworks.slice(0, 3).map((net, i) => (
                                            <span
                                              key={i}
                                              className="px-1.5 py-0.5 bg-purple-100 text-purple-700 text-xs font-mono rounded"
                                            >
                                              {net}
                                            </span>
                                          ))}
                                          {spoke.localNetworks.length > 3 && (
                                            <span className="px-1.5 py-0.5 bg-gray-100 text-gray-600 text-xs rounded">
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
                        <h3 className="text-lg font-medium text-gray-900">Node Details</h3>
                        <button
                          onClick={() => setSelectedNode(null)}
                          className="p-1 hover:bg-gray-100 rounded"
                        >
                          <XMarkIcon className="w-5 h-5 text-gray-500" />
                        </button>
                      </div>

                      {/* User Details */}
                      {selectedNode.type === 'user' && (
                        <div className="space-y-4">
                          <div className="p-3 bg-green-50 rounded-lg">
                            <div className="flex items-center gap-2">
                              <ComputerDesktopIcon className="w-6 h-6 text-green-600" />
                              <span className="text-lg font-medium text-green-800">Connected Users</span>
                            </div>
                          </div>
                          <div className="space-y-3">
                            <div>
                              <label className="text-xs text-gray-500">Hub</label>
                              <div className="text-sm font-medium">{(selectedNode.data as { hubName: string }).hubName}</div>
                            </div>
                            <div>
                              <label className="text-xs text-gray-500">Active Connections</label>
                              <div className="text-2xl font-bold text-green-700">{(selectedNode.data as { count: number }).count}</div>
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
                          <div className={`p-3 rounded-lg ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'bg-blue-50' : 'bg-gray-50'}`}>
                            <div className="flex items-center gap-2">
                              <ServerIcon className={`w-6 h-6 ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-blue-600' : 'text-gray-500'}`} />
                              <span className={`text-lg font-medium ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-blue-800' : 'text-gray-700'}`}>
                                {(selectedNode.data as TopologyMeshHub).name}
                              </span>
                            </div>
                            <div className="text-xs text-gray-500 mt-1">Mesh Hub</div>
                          </div>
                          <div className="space-y-3">
                            <div className="p-3 bg-gray-50 rounded-lg">
                              <div className="text-xs text-gray-500 mb-2">Network Endpoints</div>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Public:</span>
                                  <span className="font-mono">{(selectedNode.data as TopologyMeshHub).publicEndpoint}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Server Tunnel IP:</span>
                                  <span className="font-mono text-blue-700">{(selectedNode.data as TopologyMeshHub).serverTunnelIp}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-600">VPN Subnet:</span>
                                  <span className="font-mono">{(selectedNode.data as TopologyMeshHub).vpnSubnet}</span>
                                </div>
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-gray-500">Status</label>
                              <div className={`text-sm ${(selectedNode.data as TopologyMeshHub).status === 'online' ? 'text-blue-600' : 'text-gray-500'}`}>
                                {(selectedNode.data as TopologyMeshHub).status}
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-gray-500">Connected Spokes</label>
                              <div className="text-sm font-medium">{(selectedNode.data as TopologyMeshHub).connectedSpokes}</div>
                            </div>
                            <div>
                              <label className="text-xs text-gray-500">Connected Users</label>
                              <div className="text-sm font-medium">{(selectedNode.data as TopologyMeshHub).connectedUsers}</div>
                            </div>
                            {(selectedNode.data as TopologyMeshHub).localNetworks?.length > 0 && (
                              <div>
                                <label className="text-xs text-gray-500">Hub Local Networks</label>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {(selectedNode.data as TopologyMeshHub).localNetworks.map((net, i) => (
                                    <span key={i} className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-mono rounded">{net}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {(selectedNode.data as TopologyMeshHub).lastHeartbeat && (
                              <div>
                                <label className="text-xs text-gray-500">Last Heartbeat</label>
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
                          <div className={`p-3 rounded-lg ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'bg-purple-50' : 'bg-gray-50'}`}>
                            <div className="flex items-center gap-2">
                              <ServerIcon className={`w-6 h-6 ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-purple-600' : 'text-gray-500'}`} />
                              <span className={`text-lg font-medium ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-purple-800' : 'text-gray-700'}`}>
                                {(selectedNode.data as TopologyMeshSpoke).name}
                              </span>
                            </div>
                            <div className="text-xs text-gray-500 mt-1">Mesh Spoke</div>
                          </div>
                          <div className="space-y-3">
                            <div className="p-3 bg-gray-50 rounded-lg">
                              <div className="text-xs text-gray-500 mb-2">Network Endpoints</div>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Tunnel IP:</span>
                                  <span className="font-mono text-purple-700">{(selectedNode.data as TopologyMeshSpoke).tunnelIp || 'N/A'}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Remote Public IP:</span>
                                  <span className="font-mono">{(selectedNode.data as TopologyMeshSpoke).remoteIp || 'N/A'}</span>
                                </div>
                              </div>
                            </div>
                            <div>
                              <label className="text-xs text-gray-500">Status</label>
                              <div className={`text-sm ${(selectedNode.data as TopologyMeshSpoke).status === 'connected' ? 'text-purple-600' : 'text-gray-500'}`}>
                                {(selectedNode.data as TopologyMeshSpoke).status}
                              </div>
                            </div>
                            {(selectedNode.data as TopologyMeshSpoke).localNetworks.length > 0 && (
                              <div>
                                <label className="text-xs text-gray-500">Advertised Routes</label>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {(selectedNode.data as TopologyMeshSpoke).localNetworks.map((net, i) => (
                                    <span key={i} className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-mono rounded">{net}</span>
                                  ))}
                                </div>
                              </div>
                            )}
                            {(selectedNode.data as TopologyMeshSpoke).lastSeen && (
                              <div>
                                <label className="text-xs text-gray-500">Last Seen</label>
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
                            <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
                              <GlobeAltIcon className="w-5 h-5 text-green-600" />
                              Gateway: {gw.name}
                            </h3>

                            {/* Flow Diagram: Users -> Gateway */}
                            <div className="flex items-start gap-4">
                              {/* Users Column */}
                              <div className="flex-shrink-0 w-48">
                                <div className="w-full p-4 rounded-lg border-2 border-green-300 bg-green-50 text-left">
                                  <div className="flex items-center gap-2 mb-2">
                                    <ComputerDesktopIcon className="w-5 h-5 text-green-600" />
                                    <span className="font-medium text-green-800">Clients</span>
                                  </div>
                                  <div className="text-2xl font-bold text-green-700">{gw.clientCount}</div>
                                  <div className="text-xs text-green-600">connected clients</div>
                                </div>
                              </div>

                              {/* Arrow */}
                              <div className="flex-shrink-0 flex flex-col items-center justify-center pt-8">
                                <ArrowRightIcon className="w-8 h-8 text-gray-400" />
                                <div className="text-xs text-gray-400 mt-1">VPN</div>
                              </div>

                              {/* Gateway Column */}
                              <div className="flex-shrink-0 w-64">
                                <button
                                  onClick={() => setSelectedNode({ type: 'gateway', data: gw })}
                                  className={`w-full p-4 rounded-lg border-2 transition-colors cursor-pointer text-left ${
                                    gw.isActive
                                      ? 'border-green-400 bg-green-50 hover:border-green-600'
                                      : 'border-gray-300 bg-gray-50 hover:border-gray-400'
                                  }`}
                                >
                                  <div className="flex items-center gap-2 mb-2">
                                    <div className={`w-3 h-3 rounded-full ${gw.isActive ? 'bg-green-500' : 'bg-gray-400'}`} />
                                    <span className={`font-medium ${gw.isActive ? 'text-green-800' : 'text-gray-700'}`}>
                                      {gw.name}
                                    </span>
                                  </div>
                                  <div className="text-xs text-gray-600 mb-2">Gateway</div>
                                  <div className="space-y-1 text-xs">
                                    <div className="flex justify-between">
                                      <span className="text-gray-500">Public:</span>
                                      <span className="font-mono text-gray-700">{gw.publicIp}:{gw.vpnPort}</span>
                                    </div>
                                    <div className="flex justify-between">
                                      <span className="text-gray-500">Protocol:</span>
                                      <span className="text-gray-700">{gw.vpnProtocol?.toUpperCase()}</span>
                                    </div>
                                    <div className="flex justify-between">
                                      <span className="text-gray-500">Status:</span>
                                      <span className={gw.isActive ? 'text-green-600' : 'text-gray-500'}>
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
                        <h3 className="text-lg font-medium text-gray-900">Gateway Details</h3>
                        <button onClick={() => setSelectedNode(null)} className="p-1 hover:bg-gray-100 rounded">
                          <XMarkIcon className="w-5 h-5 text-gray-500" />
                        </button>
                      </div>
                      <div className="space-y-4">
                        <div className={`p-3 rounded-lg ${(selectedNode.data as TopologyGateway).isActive ? 'bg-green-50' : 'bg-gray-50'}`}>
                          <div className="flex items-center gap-2">
                            <GlobeAltIcon className={`w-6 h-6 ${(selectedNode.data as TopologyGateway).isActive ? 'text-green-600' : 'text-gray-500'}`} />
                            <span className={`text-lg font-medium ${(selectedNode.data as TopologyGateway).isActive ? 'text-green-800' : 'text-gray-700'}`}>
                              {(selectedNode.data as TopologyGateway).name}
                            </span>
                          </div>
                          <div className="text-xs text-gray-500 mt-1">Gateway</div>
                        </div>
                        <div className="space-y-3">
                          <div>
                            <label className="text-xs text-gray-500">Hostname</label>
                            <div className="text-sm font-mono">{(selectedNode.data as TopologyGateway).hostname || 'N/A'}</div>
                          </div>
                          <div>
                            <label className="text-xs text-gray-500">Public IP</label>
                            <div className="text-sm font-mono">{(selectedNode.data as TopologyGateway).publicIp}</div>
                          </div>
                          <div>
                            <label className="text-xs text-gray-500">VPN Port</label>
                            <div className="text-sm">{(selectedNode.data as TopologyGateway).vpnPort} ({(selectedNode.data as TopologyGateway).vpnProtocol?.toUpperCase()})</div>
                          </div>
                          <div>
                            <label className="text-xs text-gray-500">Status</label>
                            <div className={`text-sm ${(selectedNode.data as TopologyGateway).isActive ? 'text-green-600' : 'text-gray-500'}`}>
                              {(selectedNode.data as TopologyGateway).isActive ? 'Active' : 'Inactive'}
                            </div>
                          </div>
                          <div>
                            <label className="text-xs text-gray-500">Connected Clients</label>
                            <div className="text-sm font-medium">{(selectedNode.data as TopologyGateway).clientCount}</div>
                          </div>
                          {(selectedNode.data as TopologyGateway).lastHeartbeat && (
                            <div>
                              <label className="text-xs text-gray-500">Last Heartbeat</label>
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
              <div className="px-4 py-3 border-b border-gray-200 flex items-center justify-between">
                <h3 className="font-medium text-gray-900">Active VPN Sessions ({sessionsTotal})</h3>
                <button
                  onClick={loadData}
                  className="text-sm text-primary-600 hover:text-primary-800"
                >
                  Refresh
                </button>
              </div>

              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Hub</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Client IP</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">VPN Address</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Connected</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Traffic</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {!sessions || sessions.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                          No active sessions
                        </td>
                      </tr>
                    ) : (
                      (sessions || []).map((session) => (
                        <tr key={session.id} className="hover:bg-gray-50">
                          <td className="px-4 py-3 whitespace-nowrap">
                            <div className="text-sm font-medium text-gray-900">{session.userEmail}</div>
                            {session.userName && (
                              <div className="text-xs text-gray-500">{session.userName}</div>
                            )}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                            {session.gatewayName}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-gray-600">
                            {session.clientIp}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm font-mono text-gray-600">
                            {session.vpnAddress}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                            {formatDate(session.connectedAt)}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
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
