import { useState, useEffect } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import { getGateways, generateConfig, Gateway, GeneratedConfig } from '../api/client'
import { useAuth } from '../contexts/AuthContext'

export default function Connect() {
  const { user } = useAuth()
  const [searchParams] = useSearchParams()
  const [gateways, setGateways] = useState<Gateway[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedGateway, setSelectedGateway] = useState<Gateway | null>(null)
  const [generating, setGenerating] = useState(false)
  const [generatedConfig, setGeneratedConfig] = useState<GeneratedConfig | null>(null)
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
    loadGateways()
  }, [])

  async function loadGateways() {
    try {
      const data = await getGateways()
      setGateways(data)
      // Auto-select first online gateway if available
      const firstOnline = data.find(g => g.isActive)
      if (firstOnline) {
        setSelectedGateway(firstOnline)
      }
    } catch (err) {
      setError('Failed to load gateways')
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

  const cliSetupCommand = `gatekey config init --server ${serverUrl}`
  const cliInstallCommand = `curl -sSL ${serverUrl}/scripts/install-client.sh | bash`

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="card">
        <h1 className="text-2xl font-bold text-gray-900">Connect to VPN</h1>
        <p className="text-gray-500 mt-1">
          Select a gateway and connect using the GateKey CLI.
        </p>
        {isCliFlow && (
          <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="flex items-center">
              <svg className="w-5 h-5 text-blue-600 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span className="text-sm text-blue-700 font-medium">CLI Mode</span>
            </div>
            <p className="text-sm text-blue-600 mt-1">
              Select a gateway and the configuration will be automatically sent to your CLI client.
            </p>
          </div>
        )}
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
          {error}
        </div>
      )}

      {/* Loading state */}
      {loading ? (
        <div className="card flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      ) : gateways.length === 0 ? (
        <div className="card text-center py-12">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No gateways available</h3>
          {isAdmin ? (
            <>
              <p className="mt-2 text-gray-500">
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
            <p className="mt-2 text-gray-500">
              Contact your administrator to set up VPN gateways.
            </p>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Gateway Selection */}
          <div className="lg:col-span-1">
            <div className="card">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Select Gateway</h2>
              <div className="space-y-2">
                {gateways.map((gateway) => (
                  <button
                    key={gateway.id}
                    onClick={() => setSelectedGateway(gateway)}
                    disabled={!gateway.isActive}
                    className={`w-full text-left p-4 rounded-lg border-2 transition-colors ${
                      selectedGateway?.id === gateway.id
                        ? 'border-primary-500 bg-primary-50'
                        : gateway.isActive
                        ? 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                        : 'border-gray-100 bg-gray-50 opacity-60 cursor-not-allowed'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-medium text-gray-900">{gateway.name}</p>
                        <p className="text-sm text-gray-500">{gateway.hostname || gateway.publicIp}</p>
                      </div>
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                        gateway.isActive
                          ? 'bg-green-100 text-green-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}>
                        {gateway.isActive ? 'Online' : 'Offline'}
                      </span>
                    </div>
                    <div className="mt-2 text-xs text-gray-400">
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
                <h2 className="text-lg font-semibold text-gray-900">Connect with CLI</h2>
                <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800">
                  Recommended
                </span>
              </div>

              <p className="text-gray-600 mb-4">
                Use the GateKey CLI for the easiest connection experience. The CLI handles authentication,
                configuration, and connection automatically.
              </p>

              {/* Setup command (if not already configured) */}
              <div className="mb-4">
                <p className="text-sm font-medium text-gray-700 mb-2">1. First time setup (run once):</p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-100 flex items-center justify-between">
                  <code className="break-all">{cliSetupCommand}</code>
                  <button
                    onClick={() => copyCommand(cliSetupCommand, 'setup')}
                    className="ml-4 text-gray-400 hover:text-white transition-colors flex-shrink-0"
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
                <p className="text-sm font-medium text-gray-700 mb-2">2. Connect to VPN:</p>
                <div className="bg-gray-900 rounded-lg p-4 font-mono text-sm text-gray-100 flex items-center justify-between">
                  <code>{cliConnectCommand}</code>
                  <button
                    onClick={() => copyCommand(cliConnectCommand, 'connect')}
                    className="ml-4 text-gray-400 hover:text-white transition-colors flex-shrink-0"
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
              <div className="bg-gray-50 rounded-lg p-4 mt-4">
                <p className="text-sm font-medium text-gray-700 mb-2">Don't have the CLI installed?</p>
                <div className="bg-gray-900 rounded-lg p-3 font-mono text-xs text-gray-100 flex items-center justify-between">
                  <code className="break-all">{cliInstallCommand}</code>
                  <button
                    onClick={() => copyCommand(cliInstallCommand, 'install')}
                    className="ml-4 text-gray-400 hover:text-white transition-colors flex-shrink-0"
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
                <p className="text-xs text-gray-500 mt-2">
                  Or download from the <a href={`${serverUrl}/downloads/`} className="text-primary-600 hover:underline">downloads page</a>
                </p>
              </div>
            </div>

            {/* Manual Download - Secondary Action */}
            <div className="card border-gray-200">
              <button
                onClick={() => setShowManualDownload(!showManualDownload)}
                className="w-full flex items-center justify-between text-left"
              >
                <div>
                  <h2 className="text-lg font-semibold text-gray-900">Manual Configuration</h2>
                  <p className="text-sm text-gray-500">Download an OpenVPN config file for use with any OpenVPN client</p>
                </div>
                <svg
                  className={`w-5 h-5 text-gray-400 transition-transform ${showManualDownload ? 'rotate-180' : ''}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {showManualDownload && (
                <div className="mt-4 pt-4 border-t border-gray-200">
                  {selectedGateway ? (
                    <div className="space-y-4">
                      <div className="bg-amber-50 border border-amber-200 rounded-lg p-3">
                        <div className="flex items-start">
                          <svg className="w-5 h-5 text-amber-600 mr-2 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                          </svg>
                          <p className="text-sm text-amber-700">
                            Manual configs expire after 24 hours. For persistent access, use the CLI.
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                        <div>
                          <p className="font-medium text-gray-900">{selectedGateway.name}</p>
                          <p className="text-sm text-gray-500">{selectedGateway.hostname || selectedGateway.publicIp}</p>
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
                          <h3 className="font-medium text-gray-900 mb-2">Windows</h3>
                          <ol className="text-sm text-gray-600 space-y-1">
                            <li>1. Download OpenVPN GUI</li>
                            <li>2. Import the .ovpn file</li>
                            <li>3. Right-click tray icon &rarr; Connect</li>
                          </ol>
                        </div>
                        <div>
                          <h3 className="font-medium text-gray-900 mb-2">macOS</h3>
                          <ol className="text-sm text-gray-600 space-y-1">
                            <li>1. Install Tunnelblick or OpenVPN Connect</li>
                            <li>2. Double-click the .ovpn file</li>
                            <li>3. Click Connect</li>
                          </ol>
                        </div>
                        <div>
                          <h3 className="font-medium text-gray-900 mb-2">Linux</h3>
                          <ol className="text-sm text-gray-600 space-y-1">
                            <li>1. Install openvpn package</li>
                            <li>2. Run: sudo openvpn config.ovpn</li>
                            <li>3. Or import to NetworkManager</li>
                          </ol>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <p className="text-gray-500 text-center py-4">
                      Select a gateway first to generate a configuration.
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Generated config modal */}
      {generatedConfig && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <div className="text-center">
              <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-green-100 mb-4">
                <svg className="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Configuration Ready
              </h3>
              <p className="text-gray-500 mb-4">
                Your VPN configuration has been generated.
              </p>
              <div className="bg-gray-50 rounded-lg p-4 mb-4 text-left">
                <p className="text-sm text-gray-600">
                  <strong>File:</strong> {generatedConfig.fileName}
                </p>
                <p className="text-sm text-gray-600">
                  <strong>Gateway:</strong> {generatedConfig.gatewayName}
                </p>
                <p className="text-sm text-gray-600">
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
                    className="w-full text-gray-500 hover:text-gray-700 text-sm"
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
