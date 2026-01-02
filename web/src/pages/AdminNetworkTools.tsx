import { useState, useEffect } from 'react'
import {
  getTopology,
  getNetworkToolsInfo,
  executeNetworkTool,
  TopologyResponse,
  NetworkToolInfo,
  NetworkToolResult,
  TopologyMeshSpoke,
} from '../api/client'
import { XMarkIcon } from '@heroicons/react/24/outline'

export default function AdminNetworkTools() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [errorDetails, setErrorDetails] = useState<string | null>(null)

  // Topology state for locations
  const [topology, setTopology] = useState<TopologyResponse | null>(null)

  // Network tools state
  const [tools, setTools] = useState<NetworkToolInfo[]>([])
  const [selectedTool, setSelectedTool] = useState('ping')
  const [target, setTarget] = useState('')
  const [port, setPort] = useState('')
  const [ports, setPorts] = useState('')
  const [selectedLocation, setSelectedLocation] = useState('control-plane')
  const [toolRunning, setToolRunning] = useState(false)
  const [toolResult, setToolResult] = useState<NetworkToolResult | null>(null)
  const [toolsContext, setToolsContext] = useState<string>('')

  useEffect(() => {
    loadData()
  }, [])

  async function loadData() {
    try {
      setLoading(true)
      setError(null)
      const [info, topo] = await Promise.all([
        getNetworkToolsInfo(),
        getTopology()
      ])
      setTools(info.tools)
      setTopology(topo)
    } catch (err) {
      setError('Failed to load data')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleRunTool() {
    if (!target.trim()) {
      setError('Target is required')
      setErrorDetails(null)
      return
    }

    try {
      setToolRunning(true)
      setToolResult(null)
      setError(null)
      setErrorDetails(null)

      const result = await executeNetworkTool({
        tool: selectedTool,
        target: target.trim(),
        port: port ? parseInt(port) : undefined,
        ports: ports || undefined,
        location: selectedLocation,
      })

      setToolResult(result)
    } catch (err: unknown) {
      console.error(err)
      const axiosError = err as { response?: { data?: { error?: string; details?: string } } }
      if (axiosError.response?.data?.error) {
        setError(axiosError.response.data.error)
        setErrorDetails(axiosError.response.data.details || null)
      } else {
        setError('Failed to execute tool')
        setErrorDetails(null)
      }
    } finally {
      setToolRunning(false)
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function getSpokesByHub(hubId: string): TopologyMeshSpoke[] {
    if (!topology) return []
    return topology.meshSpokes.filter(spoke => spoke.hubId === hubId)
  }

  function getToolLocations(): { id: string; name: string; type: string }[] {
    const locs: { id: string; name: string; type: string }[] = [
      { id: 'control-plane', name: 'Control Plane', type: 'control-plane' }
    ]

    if (!toolsContext || !topology) return locs

    if (toolsContext.startsWith('hub-')) {
      const hubId = toolsContext.replace('hub-', '')
      const hub = topology.meshHubs.find(h => h.id === hubId)
      if (hub) {
        locs.push({ id: `hub:${hub.id}`, name: `Hub: ${hub.name}`, type: 'hub' })
        const spokes = getSpokesByHub(hubId)
        for (const spoke of spokes) {
          if (spoke.status === 'connected') {
            locs.push({ id: `spoke:${spoke.id}`, name: `Spoke: ${spoke.name}`, type: 'spoke' })
          }
        }
      }
    } else if (toolsContext.startsWith('gateway-')) {
      const gwId = toolsContext.replace('gateway-', '')
      const gw = topology.gateways.find(g => g.id === gwId)
      if (gw) {
        locs.push({ id: `gateway:${gw.id}`, name: `Gateway: ${gw.name}`, type: 'gateway' })
      }
    }

    return locs
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">Network Tools</h1>
        <p className="text-gray-500 mt-1">
          Run network diagnostic tools from the control plane or remote agents.
        </p>
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
        <div className="space-y-6">
          {/* Hub/Gateway Context Selector */}
          <div className="card">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Select Hub or Gateway Context
            </label>
            <select
              value={toolsContext}
              onChange={(e) => {
                setToolsContext(e.target.value)
                setSelectedLocation('control-plane')
              }}
              className="w-full max-w-md px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="">-- Select a Hub or Gateway --</option>
              {topology?.meshHubs && topology.meshHubs.length > 0 && (
                <optgroup label="Mesh Hubs">
                  {topology.meshHubs.map((hub) => (
                    <option key={`hub-${hub.id}`} value={`hub-${hub.id}`}>
                      {hub.name} ({hub.status}) - {hub.connectedSpokes} spokes
                    </option>
                  ))}
                </optgroup>
              )}
              {topology?.gateways && topology.gateways.length > 0 && (
                <optgroup label="Gateways">
                  {topology.gateways.map((gw) => (
                    <option key={`gateway-${gw.id}`} value={`gateway-${gw.id}`}>
                      {gw.name} ({gw.isActive ? 'active' : 'inactive'})
                    </option>
                  ))}
                </optgroup>
              )}
            </select>
            <p className="text-xs text-gray-500 mt-1">
              Select a hub or gateway to enable execution from that node and its connected spokes.
            </p>
          </div>

          {/* Tool Configuration */}
          <div className="card">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Network Diagnostic Tools</h3>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-4">
              {/* Tool Selection */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Tool</label>
                <select
                  value={selectedTool}
                  onChange={(e) => setSelectedTool(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  {tools.map((tool) => (
                    <option key={tool.name} value={tool.name}>
                      {tool.name} - {tool.description}
                    </option>
                  ))}
                </select>
              </div>

              {/* Target */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Target</label>
                <input
                  type="text"
                  value={target}
                  onChange={(e) => setTarget(e.target.value)}
                  placeholder="e.g., 8.8.8.8 or google.com"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              {/* Port (for nc) */}
              {selectedTool === 'nc' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Port</label>
                  <input
                    type="number"
                    value={port}
                    onChange={(e) => setPort(e.target.value)}
                    placeholder="e.g., 443"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>
              )}

              {/* Ports (for nmap) */}
              {selectedTool === 'nmap' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Ports</label>
                  <input
                    type="text"
                    value={ports}
                    onChange={(e) => setPorts(e.target.value)}
                    placeholder="e.g., 22,80,443"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>
              )}

              {/* Location */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Execute From</label>
                <select
                  value={selectedLocation}
                  onChange={(e) => setSelectedLocation(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  {getToolLocations().map((loc) => (
                    <option key={loc.id} value={loc.id}>
                      {loc.name} {loc.id !== 'control-plane' ? '(requires agent)' : ''}
                    </option>
                  ))}
                </select>
                {selectedLocation !== 'control-plane' && (
                  <p className="text-xs text-amber-600 mt-1">
                    Remote execution requires deployed agent binaries
                  </p>
                )}
              </div>
            </div>

            <button
              onClick={handleRunTool}
              disabled={toolRunning || !target.trim()}
              className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {toolRunning ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                  Running...
                </>
              ) : (
                'Run Tool'
              )}
            </button>
          </div>

          {/* Tool Output */}
          {toolResult && (
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-medium text-gray-900">
                  {toolResult.tool} to {toolResult.target}
                </h3>
                <div className="flex items-center gap-4 text-sm">
                  <span className={`px-2 py-1 rounded ${
                    toolResult.status === 'success'
                      ? 'bg-green-100 text-green-800'
                      : toolResult.status === 'timeout'
                      ? 'bg-yellow-100 text-yellow-800'
                      : 'bg-red-100 text-red-800'
                  }`}>
                    {toolResult.status}
                  </span>
                  <span className="text-gray-500">Duration: {toolResult.duration}</span>
                </div>
              </div>

              {toolResult.error && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
                  {toolResult.error}
                </div>
              )}

              <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto text-sm font-mono whitespace-pre-wrap">
                {toolResult.output || 'No output'}
              </pre>

              <div className="mt-3 text-xs text-gray-500">
                Executed from: {toolResult.location} at {formatDate(toolResult.startedAt)}
              </div>
            </div>
          )}

          {/* Tool Descriptions */}
          <div className="card">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Available Tools</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {tools.map((tool) => (
                <div key={tool.name} className="p-3 bg-gray-50 rounded-lg">
                  <div className="font-medium text-gray-900">{tool.name}</div>
                  <div className="text-sm text-gray-600 mt-1">{tool.description}</div>
                  {tool.required && tool.required.length > 0 && (
                    <div className="text-xs text-gray-500 mt-2">
                      Required: {tool.required.join(', ')}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
