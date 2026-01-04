import { useState, useEffect, useCallback } from 'react'
import {
  getRemoteSessionAgents,
  RemoteSessionAgent,
} from '../api/client'
import { CommandLineIcon, XMarkIcon } from '@heroicons/react/24/outline'
import RemoteTerminal from '../components/topology/RemoteTerminal'

interface TerminalTab {
  agent: RemoteSessionAgent
  openedAt: Date
}

const SESSION_TIMEOUT_MS = 60 * 60 * 1000 // 1 hour

export default function AdminRemoteSessions() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Remote session state - multiple tabs
  const [remoteAgents, setRemoteAgents] = useState<RemoteSessionAgent[]>([])
  const [terminalTabs, setTerminalTabs] = useState<TerminalTab[]>([])
  const [activeTabId, setActiveTabId] = useState<string | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  // Auto-disconnect timeout checker
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now()
      setTerminalTabs(prev => {
        const timedOut = prev.filter(tab => now - tab.openedAt.getTime() >= SESSION_TIMEOUT_MS)
        if (timedOut.length > 0) {
          // If active tab is timed out, switch to another or null
          const remaining = prev.filter(tab => now - tab.openedAt.getTime() < SESSION_TIMEOUT_MS)
          if (activeTabId && timedOut.some(t => t.agent.id === activeTabId)) {
            setActiveTabId(remaining.length > 0 ? remaining[0].agent.id : null)
          }
          return remaining
        }
        return prev
      })
    }, 60000) // Check every minute
    return () => clearInterval(interval)
  }, [activeTabId])

  async function loadData() {
    try {
      setLoading(true)
      setError(null)
      const agents = await getRemoteSessionAgents()
      setRemoteAgents(agents)
    } catch (err) {
      setError('Failed to load remote agents')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const openTerminal = useCallback((agent: RemoteSessionAgent) => {
    // Check if already open
    const existing = terminalTabs.find(t => t.agent.id === agent.id)
    if (existing) {
      setActiveTabId(agent.id)
      return
    }
    // Add new tab
    setTerminalTabs(prev => [...prev, { agent, openedAt: new Date() }])
    setActiveTabId(agent.id)
  }, [terminalTabs])

  const closeTerminal = useCallback((agentId: string) => {
    setTerminalTabs(prev => prev.filter(t => t.agent.id !== agentId))
    // If closing active tab, switch to another
    if (activeTabId === agentId) {
      const remaining = terminalTabs.filter(t => t.agent.id !== agentId)
      setActiveTabId(remaining.length > 0 ? remaining[0].agent.id : null)
    }
  }, [activeTabId, terminalTabs])

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function getTimeRemaining(openedAt: Date): string {
    const elapsed = Date.now() - openedAt.getTime()
    const remaining = SESSION_TIMEOUT_MS - elapsed
    if (remaining <= 0) return 'Expiring...'
    const minutes = Math.floor(remaining / 60000)
    if (minutes < 60) return `${minutes}m left`
    const hours = Math.floor(minutes / 60)
    return `${hours}h ${minutes % 60}m left`
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">Remote Sessions</h1>
        <p className="text-theme-tertiary mt-1">
          Connect to and run commands on remote hubs, gateways, and spokes.
        </p>
      </div>

      {/* Error Message */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : (
        <div className="space-y-6">
          {/* Terminal Tabs */}
          {terminalTabs.length > 0 && (
            <div className="card p-0 overflow-hidden">
              {/* Tab Bar */}
              <div className="flex items-center bg-gray-800 border-b border-gray-700 overflow-x-auto">
                {terminalTabs.map((tab) => (
                  <div
                    key={tab.agent.id}
                    className={`flex items-center gap-2 px-4 py-2 cursor-pointer border-r border-gray-700 min-w-0 ${
                      activeTabId === tab.agent.id
                        ? 'bg-gray-900 text-white'
                        : 'bg-gray-800 text-theme-muted hover:bg-gray-700 hover:text-gray-200'
                    }`}
                    onClick={() => setActiveTabId(tab.agent.id)}
                  >
                    <span className={`w-2 h-2 rounded-full flex-shrink-0 ${
                      tab.agent.nodeType === 'hub' ? 'bg-blue-400' :
                      tab.agent.nodeType === 'gateway' ? 'bg-green-400' :
                      'bg-purple-400'
                    }`} />
                    <span className="truncate text-sm font-medium">
                      {tab.agent.nodeType}: {tab.agent.nodeName}
                    </span>
                    <span className="text-xs text-theme-tertiary whitespace-nowrap">
                      ({getTimeRemaining(tab.openedAt)})
                    </span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        closeTerminal(tab.agent.id)
                      }}
                      className="p-0.5 hover:bg-gray-600 rounded ml-1 flex-shrink-0"
                      title="Disconnect"
                    >
                      <XMarkIcon className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>

              {/* Active Terminal Content */}
              <div className="h-[450px]">
                {terminalTabs.map((tab) => (
                  <div
                    key={tab.agent.id}
                    className={activeTabId === tab.agent.id ? 'h-full' : 'hidden'}
                  >
                    <RemoteTerminal
                      agent={tab.agent}
                      onClose={() => closeTerminal(tab.agent.id)}
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Connected Agents List */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium text-theme-primary">Connected Agents</h3>
              <button
                onClick={loadData}
                className="text-sm text-primary-600 hover:text-primary-800"
              >
                Refresh
              </button>
            </div>

            <p className="text-sm text-theme-tertiary mb-4">
              Remote sessions allow you to run shell commands on connected hubs, gateways, and spokes.
              Agents connect outbound to the control plane, so no inbound firewall rules are needed.
            </p>

            {remoteAgents.length === 0 ? (
              <div className="text-center py-8 text-theme-tertiary">
                <CommandLineIcon className="w-12 h-12 mx-auto mb-4 text-theme-muted" />
                <p className="font-medium">No agents connected</p>
                <p className="text-sm mt-1">
                  Enable remote sessions on your hubs, gateways, or spokes by setting <code className="bg-gray-300 dark:bg-gray-600 px-1 rounded text-gray-900 dark:text-gray-100">session_enabled: true</code> in their config.
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-theme">
                  <thead className="bg-theme-tertiary">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Name</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Connected</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-theme-card divide-y divide-theme">
                    {remoteAgents.map((agent) => (
                      <tr key={agent.id} className="hover:bg-theme-tertiary">
                        <td className="px-4 py-3 whitespace-nowrap">
                          <span className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${
                            agent.nodeType === 'hub' ? 'bg-blue-600 text-white' :
                            agent.nodeType === 'gateway' ? 'bg-green-600 text-white' :
                            'bg-purple-600 text-white'
                          }`}>
                            {agent.nodeType}
                          </span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-theme-primary">
                          {agent.nodeName}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-theme-tertiary">
                          {formatDate(agent.connectedAt)}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <button
                            onClick={() => openTerminal(agent)}
                            className={`inline-flex items-center gap-1 px-3 py-1 text-sm rounded transition-colors ${
                              terminalTabs.some(t => t.agent.id === agent.id)
                                ? 'text-green-600 hover:text-green-800 hover:bg-green-50'
                                : 'text-primary-600 hover:text-primary-800 hover:bg-primary-50'
                            }`}
                          >
                            <CommandLineIcon className="w-4 h-4" />
                            {terminalTabs.some(t => t.agent.id === agent.id) ? 'View' : 'Connect'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* CLI Instructions */}
          <div className="instruction-box">
            <h3 className="instruction-box-title">CLI Access</h3>
            <div className="instruction-box-text space-y-2">
              <p>You can also use remote sessions from the command line:</p>
              <div className="bg-gray-900 p-3 rounded font-mono text-xs space-y-1 text-gray-300">
                <div className="text-gray-500"># List connected agents</div>
                <div className="text-gray-100">gatekey-admin session list</div>
                <div className="mt-2 text-gray-500"># Execute a single command</div>
                <div className="text-gray-100">gatekey-admin session exec &lt;agent-id&gt; "ip addr"</div>
                <div className="mt-2 text-gray-500"># Start interactive session</div>
                <div className="text-gray-100">gatekey-admin session connect &lt;agent-id&gt;</div>
              </div>
            </div>
          </div>

          {/* Setup Instructions */}
          <div className="instruction-box">
            <h3 className="instruction-box-title">Setup Instructions</h3>
            <div className="instruction-box-text space-y-2">
              <p>To enable remote sessions on an agent:</p>
              <ol className="list-decimal list-inside space-y-1 ml-2">
                <li>Add <code className="bg-gray-300 dark:bg-gray-600 px-1 rounded text-gray-900 dark:text-gray-100">session_enabled: true</code> to the agent's config file</li>
                <li>Ensure the agent can reach the control plane URL</li>
                <li>Restart the agent service</li>
              </ol>
              <p className="mt-2">
                The agent will automatically connect to the control plane and appear in the list above.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
