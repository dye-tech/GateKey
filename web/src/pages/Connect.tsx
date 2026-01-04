import { useState, useEffect } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import { getGateways, generateConfig, getUserMeshHubs, generateMeshClientConfig, Gateway, GeneratedConfig, UserMeshHub } from '../api/client'
import { useAuth } from '../contexts/AuthContext'

type ConnectionType = 'gateways' | 'mesh'

export default function Connect() {
  const { user } = useAuth()
  const [searchParams] = useSearchParams()
  const [connectionType, setConnectionType] = useState<ConnectionType>('gateways')
  const [gateways, setGateways] = useState<Gateway[]>([])
  const [meshHubs, setMeshHubs] = useState<UserMeshHub[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedGateway, setSelectedGateway] = useState<Gateway | null>(null)
  const [selectedMeshHub, setSelectedMeshHub] = useState<UserMeshHub | null>(null)
  const [generating, setGenerating] = useState(false)
  const [generatedConfig, setGeneratedConfig] = useState<GeneratedConfig | null>(null)
  const [meshConfig, setMeshConfig] = useState<{ hubname: string; config: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState<string | null>(null)
  const [showManualDownload, setShowManualDownload] = useState(false)

  // Check if CLI initiated login (cli_callback param will be set)
  const cliCallbackUrl = searchParams.get('cli_callback')
  const isCliFlow = !!cliCallbackUrl
  const isAdmin = user?.isAdmin ?? false

  // Get the server URL for CLI command
  const serverUrl = window.location.origin

  useEffect(() => {
    loadData()
  }, [])

  async function loadData() {
    try {
      const [gatewayData, meshData] = await Promise.all([
        getGateways(),
        getUserMeshHubs().catch(() => []) // Don't fail if mesh hubs unavailable
      ])
      setGateways(gatewayData)
      setMeshHubs(meshData)
      // Auto-select first online gateway if available
      const firstOnline = gatewayData.find(g => g.isActive)
      if (firstOnline) {
        setSelectedGateway(firstOnline)
      }
      // Auto-select first mesh hub if available
      if (meshData.length > 0) {
        setSelectedMeshHub(meshData[0])
      }
    } catch (err) {
      setError('Failed to load connection options')
    } finally {
      setLoading(false)
    }
  }

  async function handleConnect() {
    if (!selectedGateway) return

    setGenerating(true)
    setError(null)

    try {
      const config = await generateConfig(selectedGateway.id, cliCallbackUrl || undefined)
      setGeneratedConfig(config)
    } catch (err) {
      setError('Failed to generate configuration')
    } finally {
      setGenerating(false)
    }
  }

  async function handleMeshConnect() {
    if (!selectedMeshHub) return

    setGenerating(true)
    setError(null)

    try {
      const config = await generateMeshClientConfig(selectedMeshHub.id)
      setMeshConfig(config)
    } catch (err) {
      setError('Failed to generate mesh configuration')
    } finally {
      setGenerating(false)
    }
  }

  function handleMeshDownload() {
    if (!meshConfig) return
    const blob = new Blob([meshConfig.config], { type: 'application/x-openvpn-profile' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `mesh-${meshConfig.hubname}.ovpn`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    setMeshConfig(null)
  }

  function handleDownload() {
    if (!generatedConfig) return
    window.location.href = generatedConfig.downloadUrl
    setTimeout(() => {
      setGeneratedConfig(null)
      setShowManualDownload(false)
    }, 1000)
  }

  function handleCliRedirect() {
    if (!generatedConfig || !cliCallbackUrl) return
    const redirectUrl = `${generatedConfig.downloadUrl}?cli_redirect=true`
    window.location.href = redirectUrl
  }

  function copyCommand(command: string, id: string) {
    navigator.clipboard.writeText(command)
    setCopied(id)
    setTimeout(() => setCopied(null), 2000)
  }

  const cliConnectCommand = selectedGateway
    ? `gatekey connect --gateway ${selectedGateway.name}`
    : 'gatekey connect'

  const cliMeshConnectCommand = selectedMeshHub
    ? `gatekey connect --mesh ${selectedMeshHub.name}`
    : 'gatekey connect --mesh'

  const cliSetupCommand = `gatekey config init --server ${serverUrl}`
  const cliInstallCommand = `curl -sSL ${serverUrl}/scripts/install-client.sh | bash`

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-theme-primary">Connect to VPN</h1>
        <p className="text-theme-tertiary mt-1">
          Select a gateway or mesh hub and connect using the GateKey CLI.
        </p>
        {isCliFlow && (
          <div className="mt-4 info-box">
            <div className="flex items-center">
              <svg className="w-5 h-5 info-box-icon mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span className="info-box-title">CLI Mode</span>
            </div>
            <p className="info-box-text mt-1">
              Select a gateway and the configuration will be automatically sent to your CLI client.
            </p>
          </div>
        )}

        {/* Connection Type Tabs */}
        {(gateways.length > 0 || meshHubs.length > 0) && (
          <div className="mt-4 border-b border-theme">
            <nav className="-mb-px flex space-x-8">
              <button
                onClick={() => setConnectionType('gateways')}
                className={`whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm ${
                  connectionType === 'gateways'
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
                }`}
              >
                Gateways ({gateways.length})
              </button>
              {meshHubs.length > 0 && (
                <button
                  onClick={() => setConnectionType('mesh')}
                  className={`whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm ${
                    connectionType === 'mesh'
                      ? 'border-primary-500 text-primary-600'
                      : 'border-transparent text-theme-tertiary hover:text-theme-secondary hover:border-theme'
                  }`}
                >
                  Mesh Networks ({meshHubs.length})
                </button>
              )}
            </nav>
          </div>
        )}
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700">
          {error}
        </div>
      )}

      {/* Loading state */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : connectionType === 'gateways' && gateways.length === 0 ? (
        <div className="card text-center py-12">
          <svg className="mx-auto h-12 w-12 text-theme-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-theme-primary">No gateways available</h3>
          {isAdmin ? (
            <>
              <p className="mt-2 text-theme-tertiary">
                Get started by adding a VPN gateway.
              </p>
              <Link
                to="/admin/gateways"
                className="mt-4 inline-flex items-center btn btn-primary"
              >
                <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add Gateway
              </Link>
            </>
          ) : (
            <p className="mt-2 text-theme-tertiary">
              Contact your administrator to set up VPN gateways.
            </p>
          )}
        </div>
      ) : connectionType === 'gateways' ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Gateway Selection */}
          <div className="lg:col-span-1">
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">Select Gateway</h2>
              <div className="space-y-2">
                {gateways.map((gateway) => (
                  <button
                    key={gateway.id}
                    onClick={() => setSelectedGateway(gateway)}
                    disabled={!gateway.isActive}
                    className={`w-full text-left p-4 rounded-lg border-2 transition-colors ${
                      selectedGateway?.id === gateway.id
                        ? 'border-primary-600 selected-highlight'
                        : gateway.isActive
                        ? 'border-theme hover:border-primary-400 hover-theme'
                        : 'border-theme bg-theme-tertiary opacity-60 cursor-not-allowed'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-medium text-theme-primary">{gateway.name}</p>
                        <p className="text-sm text-theme-tertiary">{gateway.hostname || gateway.publicIp}</p>
                      </div>
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                        gateway.isActive
                          ? 'bg-green-600 text-white'
                          : 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
                      }`}>
                        {gateway.isActive ? 'Online' : 'Offline'}
                      </span>
                    </div>
                    <div className="mt-2 text-xs text-theme-muted">
                      {gateway.vpnProtocol.toUpperCase()}:{gateway.vpnPort}
                    </div>
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Connection Instructions */}
          <div className="lg:col-span-2 space-y-6">
            {/* CLI Connect - Primary Action */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-theme-primary">Connect with CLI</h2>
                <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-600 text-white">
                  Recommended
                </span>
              </div>

              <p className="text-theme-secondary mb-4">
                Use the GateKey CLI for the easiest connection experience. The CLI handles authentication,
                configuration, and connection automatically.
              </p>

              {/* Setup command (if not already configured) */}
              <div className="mb-4">
                <p className="text-sm font-medium text-theme-secondary mb-2">1. First time setup (run once):</p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-100 flex items-center justify-between">
                  <code className="break-all">{cliSetupCommand}</code>
                  <button
                    onClick={() => copyCommand(cliSetupCommand, 'setup')}
                    className="ml-4 text-theme-muted hover:text-white transition-colors flex-shrink-0"
                    title="Copy to clipboard"
                  >
                    {copied === 'setup' ? (
                      <svg className="w-5 h-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              {/* Connect command */}
              <div className="mb-4">
                <p className="text-sm font-medium text-theme-secondary mb-2">2. Connect to VPN:</p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-100 flex items-center justify-between">
                  <code>{cliConnectCommand}</code>
                  <button
                    onClick={() => copyCommand(cliConnectCommand, 'connect')}
                    className="ml-4 text-theme-muted hover:text-white transition-colors flex-shrink-0"
                    title="Copy to clipboard"
                  >
                    {copied === 'connect' ? (
                      <svg className="w-5 h-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              {/* CLI not installed? */}
              <div className="bg-theme-tertiary rounded-lg p-4 mt-4">
                <p className="text-sm font-medium text-theme-secondary mb-2">Don't have the CLI installed?</p>
                <div className="bg-gray-900 rounded-lg p-3 font-mono text-xs text-gray-100 flex items-center justify-between">
                  <code className="break-all">{cliInstallCommand}</code>
                  <button
                    onClick={() => copyCommand(cliInstallCommand, 'install')}
                    className="ml-4 text-theme-muted hover:text-white transition-colors flex-shrink-0"
                    title="Copy to clipboard"
                  >
                    {copied === 'install' ? (
                      <svg className="w-4 h-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    )}
                  </button>
                </div>
                <p className="text-xs text-theme-tertiary mt-2">
                  Or download from the <a href={`${serverUrl}/downloads/`} className="text-primary-600 hover:underline">downloads page</a>
                </p>
              </div>
            </div>

            {/* Manual Download - Secondary Action */}
            <div className="card border-theme">
              <button
                onClick={() => setShowManualDownload(!showManualDownload)}
                className="w-full flex items-center justify-between text-left"
              >
                <div>
                  <h2 className="text-lg font-semibold text-theme-primary">Manual Configuration</h2>
                  <p className="text-sm text-theme-tertiary">Download an OpenVPN config file for use with any OpenVPN client</p>
                </div>
                <svg
                  className={`w-5 h-5 text-theme-muted transition-transform ${showManualDownload ? 'rotate-180' : ''}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {showManualDownload && (
                <div className="mt-4 pt-4 border-t border-theme">
                  {selectedGateway ? (
                    <div className="space-y-4">
                      <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
                        <div className="flex items-start">
                          <svg className="w-5 h-5 text-amber-600 dark:text-amber-400 mr-2 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                          </svg>
                          <p className="text-sm text-amber-700 dark:text-amber-400">
                            Manual configs expire after 24 hours. For persistent access, use the CLI.
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center justify-between p-3 bg-theme-tertiary rounded-lg">
                        <div>
                          <p className="font-medium text-theme-primary">{selectedGateway.name}</p>
                          <p className="text-sm text-theme-tertiary">{selectedGateway.hostname || selectedGateway.publicIp}</p>
                        </div>
                        <button
                          onClick={handleConnect}
                          disabled={generating}
                          className="btn btn-secondary"
                        >
                          {generating ? (
                            <span className="flex items-center">
                              <svg className="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                              </svg>
                              Generating...
                            </span>
                          ) : (
                            <>
                              <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                              </svg>
                              Generate Config
                            </>
                          )}
                        </button>
                      </div>

                      {/* Platform-specific instructions */}
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4">
                        <div>
                          <h3 className="font-medium text-theme-primary mb-2">Windows</h3>
                          <ol className="text-sm text-theme-secondary space-y-1">
                            <li>1. Download OpenVPN GUI</li>
                            <li>2. Import the .ovpn file</li>
                            <li>3. Right-click tray icon &rarr; Connect</li>
                          </ol>
                        </div>
                        <div>
                          <h3 className="font-medium text-theme-primary mb-2">macOS</h3>
                          <ol className="text-sm text-theme-secondary space-y-1">
                            <li>1. Install Tunnelblick or OpenVPN Connect</li>
                            <li>2. Double-click the .ovpn file</li>
                            <li>3. Click Connect</li>
                          </ol>
                        </div>
                        <div>
                          <h3 className="font-medium text-theme-primary mb-2">Linux</h3>
                          <ol className="text-sm text-theme-secondary space-y-1">
                            <li>1. Install openvpn package</li>
                            <li>2. Run: sudo openvpn config.ovpn</li>
                            <li>3. Or import to NetworkManager</li>
                          </ol>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <p className="text-theme-tertiary text-center py-4">
                      Select a gateway first to generate a configuration.
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      ) : connectionType === 'mesh' ? (
        /* Mesh Hub Section */
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Mesh Hub Selection */}
          <div className="lg:col-span-1">
            <div className="card">
              <h2 className="text-lg font-semibold text-theme-primary mb-4">Select Mesh Hub</h2>
              <div className="space-y-2">
                {meshHubs.map((hub) => (
                  <button
                    key={hub.id}
                    onClick={() => setSelectedMeshHub(hub)}
                    className={`w-full text-left p-4 rounded-lg border-2 transition-colors ${
                      selectedMeshHub?.id === hub.id
                        ? 'border-primary-600 selected-highlight'
                        : 'border-theme hover:border-primary-400 hover-theme'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-medium text-theme-primary">{hub.name}</p>
                        <p className="text-sm text-theme-tertiary">{hub.description || 'Mesh Network'}</p>
                      </div>
                      <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-600 text-white">
                        Online
                      </span>
                    </div>
                    <div className="mt-2 text-xs text-theme-muted">
                      {hub.connectedspokes} spoke{hub.connectedspokes !== 1 ? 's' : ''} connected
                    </div>
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Mesh Connection Instructions */}
          <div className="lg:col-span-2 space-y-6">
            {/* CLI Connect - Primary Action */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-theme-primary">Connect with CLI</h2>
                <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-600 text-white">
                  Recommended
                </span>
              </div>

              <p className="text-theme-secondary mb-4">
                Use the GateKey CLI for the easiest connection experience. The CLI handles authentication,
                configuration, and connection automatically.
              </p>

              {/* Setup command (if not already configured) */}
              <div className="mb-4">
                <p className="text-sm font-medium text-theme-secondary mb-2">1. First time setup (run once):</p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-100 flex items-center justify-between">
                  <code className="break-all">{cliSetupCommand}</code>
                  <button
                    onClick={() => copyCommand(cliSetupCommand, 'mesh-setup')}
                    className="ml-4 text-theme-muted hover:text-white transition-colors flex-shrink-0"
                    title="Copy to clipboard"
                  >
                    {copied === 'mesh-setup' ? (
                      <svg className="w-5 h-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              {/* Connect command */}
              <div className="mb-4">
                <p className="text-sm font-medium text-theme-secondary mb-2">2. Connect to Mesh Network:</p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-100 flex items-center justify-between">
                  <code>{cliMeshConnectCommand}</code>
                  <button
                    onClick={() => copyCommand(cliMeshConnectCommand, 'mesh-connect')}
                    className="ml-4 text-theme-muted hover:text-white transition-colors flex-shrink-0"
                    title="Copy to clipboard"
                  >
                    {copied === 'mesh-connect' ? (
                      <svg className="w-5 h-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    )}
                  </button>
                </div>
              </div>

              {/* CLI not installed? */}
              <div className="bg-theme-tertiary rounded-lg p-4 mt-4">
                <p className="text-sm font-medium text-theme-secondary mb-2">Don't have the CLI installed?</p>
                <div className="bg-gray-900 rounded-lg p-3 font-mono text-xs text-gray-100 flex items-center justify-between">
                  <code className="break-all">{cliInstallCommand}</code>
                  <button
                    onClick={() => copyCommand(cliInstallCommand, 'mesh-install')}
                    className="ml-4 text-theme-muted hover:text-white transition-colors flex-shrink-0"
                    title="Copy to clipboard"
                  >
                    {copied === 'mesh-install' ? (
                      <svg className="w-4 h-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                    )}
                  </button>
                </div>
                <p className="text-xs text-theme-tertiary mt-2">
                  Or download from the <a href={`${serverUrl}/downloads/`} className="text-primary-600 hover:underline">downloads page</a>
                </p>
              </div>
            </div>

            {/* Manual Download - Secondary Action */}
            <div className="card border-theme">
              <button
                onClick={() => setShowManualDownload(!showManualDownload)}
                className="w-full flex items-center justify-between text-left"
              >
                <div>
                  <h2 className="text-lg font-semibold text-theme-primary">Manual Configuration</h2>
                  <p className="text-sm text-theme-tertiary">Download an OpenVPN config file for use with any OpenVPN client</p>
                </div>
                <svg
                  className={`w-5 h-5 text-theme-muted transition-transform ${showManualDownload ? 'rotate-180' : ''}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {showManualDownload && (
                <div className="mt-4 pt-4 border-t border-theme">
                  {selectedMeshHub ? (
                    <div className="space-y-4">
                      <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
                        <div className="flex items-start">
                          <svg className="w-5 h-5 text-amber-600 dark:text-amber-400 mr-2 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                          </svg>
                          <p className="text-sm text-amber-700 dark:text-amber-400">
                            Manual configs expire after 24 hours. For persistent access, use the CLI.
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center justify-between p-3 bg-theme-tertiary rounded-lg">
                        <div>
                          <p className="font-medium text-theme-primary">{selectedMeshHub.name}</p>
                          <p className="text-sm text-theme-tertiary">{selectedMeshHub.description || 'Mesh Network'}</p>
                        </div>
                        <button
                          onClick={handleMeshConnect}
                          disabled={generating}
                          className="btn btn-secondary"
                        >
                          {generating ? (
                            <span className="flex items-center">
                              <svg className="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
                                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                              </svg>
                              Generating...
                            </span>
                          ) : (
                            <>
                              <svg className="w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                              </svg>
                              Generate Config
                            </>
                          )}
                        </button>
                      </div>

                      <div className="info-box">
                        <p className="info-box-text">
                          <strong className="text-theme-primary">Note:</strong> Mesh configs provide access to all spoke networks you're authorized for.
                          Routes are included in the config and will be automatically applied when connected.
                        </p>
                      </div>

                      {/* Platform-specific instructions */}
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4">
                        <div>
                          <h3 className="font-medium text-theme-primary mb-2">Windows</h3>
                          <ol className="text-sm text-theme-secondary space-y-1">
                            <li>1. Download OpenVPN GUI</li>
                            <li>2. Import the .ovpn file</li>
                            <li>3. Right-click tray icon &rarr; Connect</li>
                          </ol>
                        </div>
                        <div>
                          <h3 className="font-medium text-theme-primary mb-2">macOS</h3>
                          <ol className="text-sm text-theme-secondary space-y-1">
                            <li>1. Install Tunnelblick or OpenVPN Connect</li>
                            <li>2. Double-click the .ovpn file</li>
                            <li>3. Click Connect</li>
                          </ol>
                        </div>
                        <div>
                          <h3 className="font-medium text-theme-primary mb-2">Linux</h3>
                          <ol className="text-sm text-theme-secondary space-y-1">
                            <li>1. Install openvpn package</li>
                            <li>2. Run: sudo openvpn config.ovpn</li>
                            <li>3. Or import to NetworkManager</li>
                          </ol>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <p className="text-theme-tertiary text-center py-4">
                      Select a mesh hub first to generate a configuration.
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      ) : null}

      {/* Mesh config modal */}
      {meshConfig && (
        <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
          <div className="bg-theme-card rounded-lg shadow-xl max-w-md border border-theme w-full mx-4 p-6">
            <div className="text-center">
              <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100 dark:bg-green-900/30 mb-4">
                <svg className="h-6 w-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-theme-primary mb-2">
                Mesh Configuration Ready
              </h3>
              <p className="text-theme-tertiary mb-4">
                Your mesh VPN configuration has been generated.
              </p>
              <div className="bg-theme-tertiary rounded-lg p-4 mb-4 text-left">
                <p className="text-sm text-theme-secondary">
                  <strong>Hub:</strong> {meshConfig.hubname}
                </p>
                <p className="text-sm text-theme-secondary">
                  <strong>File:</strong> mesh-{meshConfig.hubname}.ovpn
                </p>
              </div>
              <div className="flex space-x-3">
                <button
                  onClick={() => setMeshConfig(null)}
                  className="flex-1 btn btn-secondary"
                >
                  Cancel
                </button>
                <button
                  onClick={handleMeshDownload}
                  className="flex-1 btn btn-primary"
                >
                  <svg className="w-4 h-4 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                  </svg>
                  Download
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Generated config modal */}
      {generatedConfig && (
        <div className="fixed inset-0 flex items-center justify-center z-50" style={{ backgroundColor: 'rgba(0, 0, 0, 0.5)' }}>
          <div className="bg-theme-card rounded-lg shadow-xl max-w-md border border-theme w-full mx-4 p-6">
            <div className="text-center">
              <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100 dark:bg-green-900/30 mb-4">
                <svg className="h-6 w-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-theme-primary mb-2">
                Configuration Ready
              </h3>
              <p className="text-theme-tertiary mb-4">
                Your VPN configuration has been generated.
              </p>
              <div className="bg-theme-tertiary rounded-lg p-4 mb-4 text-left">
                <p className="text-sm text-theme-secondary">
                  <strong>File:</strong> {generatedConfig.fileName}
                </p>
                <p className="text-sm text-theme-secondary">
                  <strong>Gateway:</strong> {generatedConfig.gatewayName}
                </p>
                <p className="text-sm text-theme-secondary">
                  <strong>Expires:</strong> {new Date(generatedConfig.expiresAt).toLocaleString()}
                </p>
              </div>

              {isCliFlow ? (
                <div className="space-y-3">
                  <button
                    onClick={handleCliRedirect}
                    className="w-full btn btn-primary"
                  >
                    <svg className="w-5 h-5 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                    Send to CLI
                  </button>
                  <button
                    onClick={handleDownload}
                    className="w-full btn btn-secondary"
                  >
                    Download Manually
                  </button>
                  <button
                    onClick={() => setGeneratedConfig(null)}
                    className="w-full text-theme-tertiary hover:text-theme-secondary text-sm"
                  >
                    Cancel
                  </button>
                </div>
              ) : (
                <div className="flex space-x-3">
                  <button
                    onClick={() => setGeneratedConfig(null)}
                    className="flex-1 btn btn-secondary"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleDownload}
                    className="flex-1 btn btn-primary"
                  >
                    <svg className="w-4 h-4 mr-2 inline" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    Download
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
