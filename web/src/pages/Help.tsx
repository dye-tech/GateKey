import { useState, useEffect } from 'react'

interface Section {
  id: string
  title: string
  icon: React.ReactNode
}

const sections: Section[] = [
  {
    id: 'install',
    title: 'Install',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
      </svg>
    ),
  },
  {
    id: 'configure',
    title: 'Configure',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    ),
  },
  {
    id: 'gateways',
    title: 'Gateways',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
      </svg>
    ),
  },
  {
    id: 'networks',
    title: 'Networks',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
      </svg>
    ),
  },
  {
    id: 'access-rules',
    title: 'Access Rules',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
    ),
  },
  {
    id: 'proxy-apps',
    title: 'Proxy Apps',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
  {
    id: 'oidc-providers',
    title: 'OIDC Providers',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    ),
  },
  {
    id: 'saml-providers',
    title: 'SAML Providers',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
    ),
  },
  {
    id: 'monitoring',
    title: 'Monitoring',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    ),
  },
  {
    id: 'mesh',
    title: 'Mesh Networking',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2zM12 7v10M7 12h10" />
      </svg>
    ),
  },
  {
    id: 'general',
    title: 'General Settings',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
      </svg>
    ),
  },
  {
    id: 'certificate-ca',
    title: 'Certificate CA',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
      </svg>
    ),
  },
]

export default function Help() {
  const baseUrl = window.location.origin
  const [activeSection, setActiveSection] = useState('install')

  useEffect(() => {
    const handleScroll = () => {
      const sectionElements = sections.map(s => document.getElementById(s.id))
      const scrollPosition = window.scrollY + 150

      for (let i = sectionElements.length - 1; i >= 0; i--) {
        const section = sectionElements[i]
        if (section && section.offsetTop <= scrollPosition) {
          setActiveSection(sections[i].id)
          break
        }
      }
    }

    window.addEventListener('scroll', handleScroll)
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const scrollToSection = (id: string) => {
    const element = document.getElementById(id)
    if (element) {
      const offset = 100
      const elementPosition = element.getBoundingClientRect().top + window.pageYOffset
      window.scrollTo({ top: elementPosition - offset, behavior: 'smooth' })
    }
  }

  return (
    <div className="flex gap-8">
      {/* Sidebar Navigation */}
      <div className="hidden lg:block w-64 flex-shrink-0">
        <div className="sticky top-24">
          <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider mb-4">
            Documentation
          </h2>
          <nav className="space-y-1">
            {sections.map((section) => (
              <button
                key={section.id}
                onClick={() => scrollToSection(section.id)}
                className={`w-full flex items-center px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
                  activeSection === section.id
                    ? 'bg-primary-50 text-primary-700'
                    : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                }`}
              >
                <span className={`mr-3 ${activeSection === section.id ? 'text-primary-600' : 'text-gray-400'}`}>
                  {section.icon}
                </span>
                {section.title}
              </button>
            ))}
          </nav>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 max-w-4xl space-y-12">
        {/* Install Section */}
        <section id="install" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Install GateKey CLI</h1>
            <p className="text-gray-600 mb-6">
              Download and install the GateKey CLI client to connect to VPN gateways from your terminal.
            </p>

            {/* Quick Install Script */}
            <div className="mb-6">
              <h3 className="text-lg font-medium text-gray-900 mb-2">Quick Install (Linux/macOS)</h3>
              <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
                <code className="text-green-400 text-sm">
                  curl -sSL {baseUrl}/scripts/install-client.sh | bash
                </code>
              </div>
            </div>

            {/* Download Binaries */}
            <div className="mb-6">
              <h3 className="text-lg font-medium text-gray-900 mb-3">Download Binaries</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <a
                  href={`${baseUrl}/bin/gatekey-linux-amd64`}
                  className="flex items-center p-4 border border-gray-200 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  {/* Tux Linux Penguin */}
                  <svg className="h-8 w-8 mr-3" viewBox="0 0 256 256" fill="none">
                    {/* Body */}
                    <ellipse cx="128" cy="156" rx="72" ry="84" fill="#1a1a1a"/>
                    {/* Belly */}
                    <ellipse cx="128" cy="176" rx="48" ry="60" fill="#FFFFFF"/>
                    {/* Head */}
                    <circle cx="128" cy="72" r="48" fill="#1a1a1a"/>
                    {/* Face/mask */}
                    <ellipse cx="128" cy="80" rx="32" ry="28" fill="#FFFFFF"/>
                    {/* Eyes */}
                    <ellipse cx="114" cy="68" rx="8" ry="12" fill="#FFFFFF"/>
                    <ellipse cx="142" cy="68" rx="8" ry="12" fill="#FFFFFF"/>
                    <circle cx="116" cy="70" r="4" fill="#1a1a1a"/>
                    <circle cx="140" cy="70" r="4" fill="#1a1a1a"/>
                    {/* Beak */}
                    <path d="M128 76 L116 92 L140 92 Z" fill="#F4A103"/>
                    <path d="M128 84 L118 92 L138 92 Z" fill="#E08A00"/>
                    {/* Feet */}
                    <ellipse cx="96" cy="236" rx="24" ry="12" fill="#F4A103"/>
                    <ellipse cx="160" cy="236" rx="24" ry="12" fill="#F4A103"/>
                    {/* Wings/flippers */}
                    <ellipse cx="56" cy="156" rx="16" ry="48" fill="#1a1a1a" transform="rotate(-15 56 156)"/>
                    <ellipse cx="200" cy="156" rx="16" ry="48" fill="#1a1a1a" transform="rotate(15 200 156)"/>
                  </svg>
                  <div>
                    <p className="font-medium text-gray-900">Linux (x64)</p>
                    <p className="text-sm text-gray-500">gatekey-linux-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-darwin-amd64`}
                  className="flex items-center p-4 border border-gray-200 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  {/* Apple Logo */}
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-gray-900">macOS (Intel)</p>
                    <p className="text-sm text-gray-500">gatekey-darwin-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-darwin-arm64`}
                  className="flex items-center p-4 border border-gray-200 rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  {/* Apple Logo */}
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-gray-900">macOS (Apple Silicon)</p>
                    <p className="text-sm text-gray-500">gatekey-darwin-arm64</p>
                  </div>
                </a>
              </div>
            </div>

            {/* Manual Install */}
            <div className="bg-gray-50 rounded-lg p-4">
              <h3 className="text-lg font-medium text-gray-900 mb-2">Manual Installation</h3>
              <ol className="list-decimal list-inside space-y-2 text-sm text-gray-600">
                <li>Download the binary for your platform</li>
                <li>Make it executable: <code className="bg-gray-200 px-1 rounded">chmod +x gatekey-*</code></li>
                <li>Move to PATH: <code className="bg-gray-200 px-1 rounded">sudo mv gatekey-* /usr/local/bin/gatekey</code></li>
                <li>Verify: <code className="bg-gray-200 px-1 rounded">gatekey version</code></li>
              </ol>
            </div>
          </div>
        </section>

        {/* Configure Section */}
        <section id="configure" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Configure GateKey CLI</h1>
            <p className="text-gray-600 mb-6">
              Set up the GateKey CLI to connect to your organization's VPN.
            </p>

            {/* Quick Start Steps */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">1</span>
                  <span className="font-medium text-gray-900">Initialize Config</span>
                </div>
                <code className="text-xs bg-gray-200 px-2 py-1 rounded block">gatekey config init --server {baseUrl}</code>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">2</span>
                  <span className="font-medium text-gray-900">Login</span>
                </div>
                <code className="text-xs bg-gray-200 px-2 py-1 rounded block">gatekey login</code>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">3</span>
                  <span className="font-medium text-gray-900">Connect</span>
                </div>
                <code className="text-xs bg-gray-200 px-2 py-1 rounded block">gatekey connect</code>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">4</span>
                  <span className="font-medium text-gray-900">Disconnect</span>
                </div>
                <code className="text-xs bg-gray-200 px-2 py-1 rounded block">gatekey disconnect</code>
              </div>
            </div>

            {/* Multi-Gateway Support */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Multi-Gateway Support</h3>
            <p className="text-gray-600 mb-4">
              The GateKey client supports connecting to multiple gateways simultaneously. Each gateway gets its own tun interface.
            </p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto mb-4">
              <pre className="text-green-400 text-sm">{`# Connect to multiple gateways
gatekey connect gateway1      # Connect to first gateway (tun0)
gatekey connect gateway2      # Connect to second gateway (tun1)

# Check status
gatekey status                # Shows all active connections

# Disconnect
gatekey disconnect gateway1   # Disconnect from specific gateway
gatekey disconnect --all      # Disconnect from all gateways`}</pre>
            </div>

            {/* Configuration File */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Configuration File</h3>
            <p className="text-gray-600 mb-4">
              The CLI stores its configuration in <code className="bg-gray-100 px-1 rounded">~/.gatekey/config.yaml</code>. You can manually edit this file or use the CLI commands.
            </p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
              <pre className="text-green-400 text-sm">{`# View current configuration
gatekey config show

# Set server URL
gatekey config set server ${baseUrl}

# Reset configuration
gatekey config reset`}</pre>
            </div>
          </div>
        </section>

        {/* Gateways Section */}
        <section id="gateways" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Gateways</h1>
            <p className="text-gray-600 mb-6">
              Gateways are VPN entry points that users connect to. Each gateway runs the OpenVPN server and GateKey gateway agent.
            </p>

            {/* Creating a Gateway */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Creating a Gateway</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Gateways</strong></li>
              <li>Click <strong>Add Gateway</strong></li>
              <li>Enter the gateway details:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Display name for the gateway</li>
                  <li><strong>Hostname:</strong> Public hostname or IP address</li>
                  <li><strong>Port:</strong> OpenVPN port (default: 1194)</li>
                  <li><strong>Protocol:</strong> UDP (recommended) or TCP</li>
                </ul>
              </li>
              <li>Click <strong>Save</strong> to create the gateway</li>
            </ol>

            {/* Gateway Settings */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Gateway Settings</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Setting</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">VPN Subnet</td>
                    <td className="px-4 py-3">CIDR for VPN client IPs (default: 172.31.255.0/24)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Crypto Profile</td>
                    <td className="px-4 py-3">Encryption settings: Modern (default), FIPS, or Compatible</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Full Tunnel Mode</td>
                    <td className="px-4 py-3">Route all traffic through VPN (0.0.0.0/0)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Push DNS</td>
                    <td className="px-4 py-3">Push DNS server settings to clients</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">DNS Servers</td>
                    <td className="px-4 py-3">Custom DNS servers to push (e.g., 1.1.1.1, 8.8.8.8)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">TLS-Auth</td>
                    <td className="px-4 py-3">Enable HMAC signature layer for additional security</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Tunnel Modes */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Tunnel Modes</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Split Tunnel (Default)</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Only routes traffic for permitted networks</li>
                  <li>• Internet traffic uses normal connection</li>
                  <li>• Routes pushed based on user's access rules</li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Full Tunnel</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Routes ALL traffic through VPN</li>
                  <li>• Uses redirect-gateway directive</li>
                  <li>• Useful for traffic inspection</li>
                </ul>
              </div>
            </div>

            {/* Installing Gateway Agent */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Installing Gateway Agent</h3>
            <p className="text-gray-600 mb-4">
              After creating a gateway, install the gateway agent on your VPN server:
            </p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
              <pre className="text-green-400 text-sm">{`# Download the install script from the gateway details page
# Run on your VPN server:
curl -sSL ${baseUrl}/scripts/install-gateway.sh | \\
  GATEWAY_TOKEN=<token> bash

# The agent will:
# 1. Install OpenVPN and gatekey-gateway
# 2. Configure the firewall (nftables)
# 3. Register with the control plane
# 4. Start the VPN service`}</pre>
            </div>
          </div>
        </section>

        {/* Networks Section */}
        <section id="networks" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Networks</h1>
            <p className="text-gray-600 mb-6">
              Networks define the CIDR blocks that are reachable through your VPN. Networks are assigned to gateways to advertise routes.
            </p>

            {/* Creating Networks */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Creating a Network</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Networks</strong></li>
              <li>Click <strong>Add Network</strong></li>
              <li>Enter the network details:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Descriptive name (e.g., "Production Servers")</li>
                  <li><strong>CIDR:</strong> Network range (e.g., 10.0.0.0/8, 192.168.1.0/24)</li>
                  <li><strong>Description:</strong> Optional description</li>
                </ul>
              </li>
              <li>Click <strong>Save</strong></li>
            </ol>

            {/* Assigning to Gateways */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Assigning Networks to Gateways</h3>
            <p className="text-gray-600 mb-4">
              Networks must be assigned to gateways to be advertised to VPN clients:
            </p>
            <ol className="list-decimal list-inside space-y-2 text-gray-600 mb-6">
              <li>Go to <strong>Gateways</strong> and click <strong>Access</strong> on the gateway</li>
              <li>Select the <strong>Networks</strong> tab</li>
              <li>Add the networks this gateway should serve</li>
            </ol>

            {/* How Routes Work */}
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <div className="flex">
                <svg className="h-5 w-5 text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-blue-800">How Route Pushing Works</h3>
                  <p className="mt-1 text-sm text-blue-700">
                    When a user connects, GateKey pushes routes based on their access rules. Only CIDR-type access rules
                    are pushed as routes. This ensures users only receive routes to networks they're permitted to access.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Access Rules Section */}
        <section id="access-rules" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Access Rules</h1>
            <p className="text-gray-600 mb-6">
              Access rules define what specific resources users can reach. GateKey uses a Zero Trust model where all traffic is blocked by default.
            </p>

            {/* Default Deny */}
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
              <div className="flex">
                <svg className="h-5 w-5 text-red-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800">Default Deny Policy</h3>
                  <p className="mt-1 text-sm text-red-700">
                    All traffic is <strong>blocked by default</strong>. Users can only access resources that are explicitly permitted through access rules.
                  </p>
                </div>
              </div>
            </div>

            {/* Rule Types */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Rule Types</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Example</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">IP Address</td>
                    <td className="px-4 py-3"><code className="bg-gray-100 px-1 rounded">192.168.1.100</code></td>
                    <td className="px-4 py-3">Single IP address</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">CIDR Range</td>
                    <td className="px-4 py-3"><code className="bg-gray-100 px-1 rounded">10.0.0.0/24</code></td>
                    <td className="px-4 py-3">Network range (also pushed as route)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Hostname</td>
                    <td className="px-4 py-3"><code className="bg-gray-100 px-1 rounded">api.internal.com</code></td>
                    <td className="px-4 py-3">Exact hostname match</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Wildcard</td>
                    <td className="px-4 py-3"><code className="bg-gray-100 px-1 rounded">*.internal.com</code></td>
                    <td className="px-4 py-3">Pattern matching for hostnames</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Creating Rules */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Creating Access Rules</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Access Rules</strong></li>
              <li>Click <strong>Add Rule</strong></li>
              <li>Configure the rule:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Descriptive name for the rule</li>
                  <li><strong>Type:</strong> IP, CIDR, Hostname, or Wildcard</li>
                  <li><strong>Value:</strong> The target (IP, network, or hostname)</li>
                  <li><strong>Ports:</strong> Optional port restriction (e.g., 443, 80-8080)</li>
                  <li><strong>Protocol:</strong> TCP, UDP, or Any</li>
                </ul>
              </li>
              <li>Click <strong>Save</strong></li>
            </ol>

            {/* Assigning Rules */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Assigning Rules to Users/Groups</h3>
            <ol className="list-decimal list-inside space-y-2 text-gray-600">
              <li>On the <strong>Access Rules</strong> page, click <strong>Assign</strong> on a rule</li>
              <li>Add users or groups that should have this access</li>
              <li>Rules take effect immediately (within 10 seconds)</li>
            </ol>
          </div>
        </section>

        {/* Proxy Apps Section */}
        <section id="proxy-apps" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Proxy Applications</h1>
            <p className="text-gray-600 mb-6">
              Proxy Apps provide clientless access to internal web applications. Users can access internal apps through their browser without installing a VPN client.
            </p>

            {/* How It Works */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">How It Works</h3>
            <div className="bg-gray-50 rounded-lg p-4 mb-6">
              <ol className="list-decimal list-inside space-y-2 text-gray-600">
                <li>User authenticates via SSO</li>
                <li>User accesses <code className="bg-gray-200 px-1 rounded">{baseUrl}/proxy/app-slug/</code></li>
                <li>GateKey verifies user has permission to access the app</li>
                <li>Traffic is proxied to the internal application</li>
                <li>User headers (email, groups) are injected for the backend</li>
              </ol>
            </div>

            {/* Creating Proxy Apps */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Creating a Proxy Application</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Proxy Apps</strong></li>
              <li>Click <strong>Add Application</strong></li>
              <li>Configure the application:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Display name for the app</li>
                  <li><strong>Slug:</strong> URL path identifier (e.g., "grafana")</li>
                  <li><strong>Internal URL:</strong> Backend service URL (e.g., http://grafana:3000)</li>
                  <li><strong>Description:</strong> Optional description</li>
                </ul>
              </li>
              <li>Assign users, groups, or link to access rules</li>
              <li>Click <strong>Save</strong></li>
            </ol>

            {/* Proxy Settings */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Proxy Settings</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Setting</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">Preserve Host Header</td>
                    <td className="px-4 py-3">Forward original Host header to backend</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Strip Prefix</td>
                    <td className="px-4 py-3">Remove /proxy/slug from request path</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">WebSocket Enabled</td>
                    <td className="px-4 py-3">Support WebSocket connections</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Timeout</td>
                    <td className="px-4 py-3">Request timeout in seconds</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        {/* OIDC Providers Section */}
        <section id="oidc-providers" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">OIDC Providers</h1>
            <p className="text-gray-600 mb-6">
              Configure OpenID Connect (OIDC) providers to enable Single Sign-On (SSO) authentication with your identity provider.
            </p>

            {/* Supported Providers */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Supported Providers</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
              <div className="p-3 border border-gray-200 rounded-lg text-center">
                <p className="font-medium text-gray-900">Okta</p>
              </div>
              <div className="p-3 border border-gray-200 rounded-lg text-center">
                <p className="font-medium text-gray-900">Azure AD</p>
              </div>
              <div className="p-3 border border-gray-200 rounded-lg text-center">
                <p className="font-medium text-gray-900">Google</p>
              </div>
              <div className="p-3 border border-gray-200 rounded-lg text-center">
                <p className="font-medium text-gray-900">Keycloak</p>
              </div>
            </div>

            {/* Configuration */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Configuring OIDC</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Settings → OIDC</strong></li>
              <li>Click <strong>Add Provider</strong></li>
              <li>Enter the provider details:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Display name (e.g., "Okta")</li>
                  <li><strong>Issuer URL:</strong> OIDC issuer URL</li>
                  <li><strong>Client ID:</strong> OAuth client ID from your IdP</li>
                  <li><strong>Client Secret:</strong> OAuth client secret</li>
                  <li><strong>Scopes:</strong> Usually "openid profile email groups"</li>
                </ul>
              </li>
              <li>Click <strong>Save</strong></li>
            </ol>

            {/* Callback URL */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Callback URL</h3>
            <p className="text-gray-600 mb-2">Configure this callback URL in your identity provider:</p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
              <code className="text-green-400 text-sm">{baseUrl}/api/v1/auth/oidc/callback</code>
            </div>
          </div>
        </section>

        {/* SAML Providers Section */}
        <section id="saml-providers" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">SAML Providers</h1>
            <p className="text-gray-600 mb-6">
              Configure SAML 2.0 providers for enterprise Single Sign-On integration.
            </p>

            {/* Configuration */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Configuring SAML</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Settings → SAML</strong></li>
              <li>Click <strong>Add Provider</strong></li>
              <li>Enter the provider details:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Display name</li>
                  <li><strong>Entity ID:</strong> IdP entity ID</li>
                  <li><strong>SSO URL:</strong> IdP Single Sign-On URL</li>
                  <li><strong>Certificate:</strong> IdP signing certificate (X.509)</li>
                </ul>
              </li>
              <li>Click <strong>Save</strong></li>
            </ol>

            {/* SP Metadata */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Service Provider Metadata</h3>
            <p className="text-gray-600 mb-2">Configure these values in your identity provider:</p>
            <div className="bg-gray-50 rounded-lg p-4 space-y-3">
              <div>
                <p className="text-sm font-medium text-gray-700">Entity ID (Audience):</p>
                <code className="text-sm text-gray-600">{baseUrl}</code>
              </div>
              <div>
                <p className="text-sm font-medium text-gray-700">ACS URL:</p>
                <code className="text-sm text-gray-600">{baseUrl}/api/v1/auth/saml/acs</code>
              </div>
            </div>
          </div>
        </section>

        {/* Monitoring Section */}
        <section id="monitoring" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Login Monitoring</h1>
            <p className="text-gray-600 mb-6">
              Monitor user authentication events, track login activity, and configure log retention policies.
            </p>

            {/* Overview */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Overview</h3>
            <p className="text-gray-600 mb-4">
              The Monitoring page provides visibility into all authentication events across your GateKey deployment.
              Track successful and failed logins, identify suspicious activity, and maintain compliance with audit requirements.
            </p>

            {/* Features */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Features</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Login Logs</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• View all authentication events</li>
                  <li>• Filter by user, IP, provider, status</li>
                  <li>• See IP address and location data</li>
                  <li>• Track failed login attempts with reasons</li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Statistics</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Total logins (successful/failed)</li>
                  <li>• Unique users and IP addresses</li>
                  <li>• Logins by provider (OIDC/SAML/Local)</li>
                  <li>• Logins by country/location</li>
                </ul>
              </div>
            </div>

            {/* Data Captured */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Data Captured</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Field</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">Timestamp</td>
                    <td className="px-4 py-3">Date and time of the login attempt</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">User Email</td>
                    <td className="px-4 py-3">Email address of the user</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Provider</td>
                    <td className="px-4 py-3">Authentication provider (OIDC, SAML, Local)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">IP Address</td>
                    <td className="px-4 py-3">Client's public IP address</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Location</td>
                    <td className="px-4 py-3">Country and city (when available)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Status</td>
                    <td className="px-4 py-3">Success or failure with reason</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">User Agent</td>
                    <td className="px-4 py-3">Browser/client information</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Log Retention */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Log Retention</h3>
            <p className="text-gray-600 mb-4">
              Configure how long login logs are retained in the database. Logs older than the retention period
              are automatically deleted by a background job that runs every 6 hours.
            </p>
            <div className="bg-gray-50 rounded-lg p-4 mb-6">
              <ul className="space-y-2 text-gray-600 text-sm">
                <li><strong>Default:</strong> 30 days</li>
                <li><strong>Minimum:</strong> 1 day</li>
                <li><strong>Forever:</strong> Set to 0 to keep logs indefinitely</li>
              </ul>
            </div>

            {/* Manual Purge */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Manual Purge</h3>
            <p className="text-gray-600 mb-4">
              You can manually purge logs older than a specified number of days from the Settings tab.
              This is useful for immediately clearing old data or complying with data retention policies.
            </p>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <div className="flex">
                <svg className="h-5 w-5 text-yellow-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-yellow-800">Warning</h3>
                  <p className="mt-1 text-sm text-yellow-700">
                    Purging logs is irreversible. Once deleted, login history cannot be recovered.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Mesh Networking Section */}
        <section id="mesh" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Mesh Networking</h1>
            <p className="text-gray-600 mb-6">
              Connect remote sites and networks using a hub-and-spoke VPN mesh. Mesh networking allows gateways behind NAT to connect to a central hub, enabling site-to-site connectivity without inbound firewall rules.
            </p>

            {/* Architecture Overview */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Architecture Overview</h3>
            <div className="bg-gray-50 rounded-lg p-4 mb-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-700">{`                 ┌─────────────────┐
                 │  Control Plane  │
                 │   (GateKey UI)  │
                 └────────┬────────┘
                          │ API / Config Sync
                          ▼
                 ┌─────────────────┐
                 │   Mesh Hub      │◄── OpenVPN Server
                 │  (gatekey-hub)  │    Runs on public endpoint
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Gateway A  │  │  Gateway B  │  │  Gateway C  │
│  10.0.0.0/8 │  │ 192.168.0/24│  │ 172.16.0/16 │
└─────────────┘  └─────────────┘  └─────────────┘
  Home Lab         AWS VPC         Office Network`}</pre>
            </div>

            {/* Key Features */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Key Features</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Gateway-Initiated Connections</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Gateways connect TO the hub (outbound)</li>
                  <li>• Works behind NAT/firewalls</li>
                  <li>• No inbound ports required on gateways</li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Standalone Hub</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Hub runs separately from control plane</li>
                  <li>• Deploy on any public server or cloud VM</li>
                  <li>• Syncs configuration from control plane</li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">FIPS Compliant</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Uses OpenVPN with AES-256-GCM</li>
                  <li>• Configurable crypto profiles</li>
                  <li>• Optional TLS-Auth for added security</li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Dynamic Routing</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Gateways advertise local networks</li>
                  <li>• Hub aggregates and distributes routes</li>
                  <li>• Automatic route updates</li>
                </ul>
              </div>
            </div>

            {/* Setting Up a Hub */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Setting Up a Mesh Hub</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Mesh</strong></li>
              <li>Click <strong>Add Hub</strong> and configure:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Display name for the hub</li>
                  <li><strong>Public Endpoint:</strong> Hostname or IP where gateways will connect</li>
                  <li><strong>VPN Port:</strong> OpenVPN port (default: 1194)</li>
                  <li><strong>VPN Subnet:</strong> Tunnel IP range (e.g., 172.30.0.0/16)</li>
                  <li><strong>Crypto Profile:</strong> FIPS, Modern, or Compatible</li>
                </ul>
              </li>
              <li><strong>Save the API Token</strong> - shown only once at creation</li>
              <li>Click <strong>Install Script</strong> to get the deployment script</li>
              <li>Run the install script on your hub server:
                <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto mt-2">
                  <code className="text-green-400 text-sm">sudo bash install-hub.sh</code>
                </div>
              </li>
            </ol>

            {/* Setting Up Gateways */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Adding Mesh Gateways</h3>
            <ol className="list-decimal list-inside space-y-3 text-gray-600 mb-6">
              <li>In the <strong>Mesh</strong> page, switch to the <strong>Gateways</strong> tab</li>
              <li>Select the hub this gateway will connect to</li>
              <li>Click <strong>Add Gateway</strong> and configure:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Identifier for this gateway</li>
                  <li><strong>Local Networks:</strong> CIDR blocks behind this gateway (e.g., 10.0.0.0/8)</li>
                </ul>
              </li>
              <li><strong>Save the Gateway Token</strong> - shown only once</li>
              <li>Click <strong>Install Script</strong> for the deployment script</li>
              <li>Run on your gateway server:
                <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto mt-2">
                  <code className="text-green-400 text-sm">sudo bash install-gateway.sh</code>
                </div>
              </li>
            </ol>

            {/* How It Works */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">How It Works</h3>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
              <ol className="list-decimal list-inside space-y-2 text-blue-800 text-sm">
                <li><strong>Gateway Connects:</strong> Mesh gateway initiates OpenVPN connection to hub</li>
                <li><strong>Authentication:</strong> Gateway authenticates using its provisioned certificate</li>
                <li><strong>Route Advertisement:</strong> Gateway tells hub about its local networks via <code className="bg-blue-100 px-1 rounded">iroute</code></li>
                <li><strong>Hub Aggregates:</strong> Hub collects routes from all gateways</li>
                <li><strong>Traffic Routing:</strong> Traffic between sites flows through the hub</li>
                <li><strong>Control Plane Sync:</strong> Hub periodically syncs access rules and config</li>
              </ol>
            </div>

            {/* Binaries */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Mesh Binaries</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Binary</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey-hub</td>
                    <td className="px-4 py-3">Runs on the central hub server. Manages OpenVPN server and syncs with control plane.</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey-mesh-gateway</td>
                    <td className="px-4 py-3">Runs on remote sites. Connects to hub and advertises local networks.</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Access Control */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Access Control</h3>
            <p className="text-gray-600 mb-4">
              Mesh networking supports fine-grained access control at both the hub and spoke level, allowing you to control which users can connect and which networks they can access.
            </p>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Hub Access Control</h4>
                <p className="text-sm text-gray-600 mb-2">
                  Control who can connect to the mesh network as a VPN client.
                </p>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Assign users directly to hub</li>
                  <li>• Assign groups for team access</li>
                  <li>• Users without access cannot generate mesh configs</li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Spoke Access Control</h4>
                <p className="text-sm text-gray-600 mb-2">
                  Control who can route traffic to networks behind each spoke.
                </p>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Per-spoke user/group assignments</li>
                  <li>• Limit access to specific CIDR ranges</li>
                  <li>• Fine-grained network segmentation</li>
                </ul>
              </div>
            </div>

            {/* Managing Hub Access */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Managing Hub Access</h3>
            <ol className="list-decimal list-inside space-y-2 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Mesh</strong></li>
              <li>On the <strong>Hubs</strong> tab, click the actions menu on a hub</li>
              <li>Select <strong>Manage Access</strong></li>
              <li>Add users or groups that should be able to connect to this mesh network</li>
              <li>Users can then generate VPN configs from the Connect page</li>
            </ol>

            {/* Managing Spoke Access */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Managing Spoke Access</h3>
            <ol className="list-decimal list-inside space-y-2 text-gray-600 mb-6">
              <li>Navigate to <strong>Administration → Mesh → Spokes</strong></li>
              <li>Select the hub and find the spoke you want to configure</li>
              <li>Click the actions menu and select <strong>Manage Access</strong></li>
              <li>The modal shows the networks accessible via this spoke</li>
              <li>Add users or groups that should have access to these networks</li>
            </ol>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
              <div className="flex">
                <svg className="h-5 w-5 text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-blue-800">Example: Network Segmentation</h3>
                  <p className="mt-1 text-sm text-blue-700">
                    A spoke advertises networks <code className="bg-blue-100 px-1 rounded">10.0.0.0/24</code> (prod) and <code className="bg-blue-100 px-1 rounded">10.0.1.0/24</code> (dev).
                    You can assign the "Developers" group to the spoke so they can access both networks, while the "QA" group only
                    gets access via a different spoke that only advertises the dev network.
                  </p>
                </div>
              </div>
            </div>

            {/* Client Connectivity */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Connecting as a Client</h3>
            <p className="text-gray-600 mb-4">
              Users with hub access can download VPN configs to connect to the mesh network:
            </p>
            <ol className="list-decimal list-inside space-y-2 text-gray-600 mb-6">
              <li>Navigate to the <strong>Connect</strong> page</li>
              <li>Switch to the <strong>Mesh Networks</strong> tab</li>
              <li>Find the mesh hub you have access to</li>
              <li>Click <strong>Download Config</strong></li>
              <li>Import the <code className="bg-gray-100 px-1 rounded">.ovpn</code> file into your OpenVPN client</li>
            </ol>

            {/* Troubleshooting */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">Troubleshooting</h3>
            <div className="space-y-4">
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Gateway Won't Connect</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Verify hub endpoint is reachable from gateway</li>
                  <li>• Check firewall allows outbound UDP/1194 (or configured port)</li>
                  <li>• Verify gateway token is correct</li>
                  <li>• Check logs: <code className="bg-gray-100 px-1 rounded">journalctl -u gatekey-mesh-gateway -f</code></li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Hub Shows Offline</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Verify hub can reach control plane</li>
                  <li>• Check hub service: <code className="bg-gray-100 px-1 rounded">systemctl status gatekey-hub</code></li>
                  <li>• Verify API token is correct in hub config</li>
                  <li>• Check logs: <code className="bg-gray-100 px-1 rounded">journalctl -u gatekey-hub -f</code></li>
                </ul>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Routes Not Working</h4>
                <ul className="text-sm text-gray-600 space-y-1">
                  <li>• Verify gateway's local networks are correctly configured</li>
                  <li>• Check IP forwarding is enabled on hub and gateways</li>
                  <li>• Verify firewall rules allow forwarded traffic</li>
                  <li>• Check OpenVPN routing: <code className="bg-gray-100 px-1 rounded">ip route show</code></li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* General Settings Section */}
        <section id="general" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">General Settings</h1>
            <p className="text-gray-600 mb-6">
              Configure global settings for your GateKey deployment.
            </p>

            {/* Settings Table */}
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Setting</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">Session Timeout</td>
                    <td className="px-4 py-3">How long user sessions remain valid</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Config Validity</td>
                    <td className="px-4 py-3">Expiration time for generated VPN configs (default: 24 hours)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Default Crypto Profile</td>
                    <td className="px-4 py-3">Default encryption profile for new gateways</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Audit Logging</td>
                    <td className="px-4 py-3">Enable/disable detailed audit logging</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Crypto Profiles */}
            <h3 className="text-lg font-medium text-gray-900 mt-6 mb-3">Crypto Profiles</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Profile</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Ciphers</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Use Case</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200 text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">Modern</td>
                    <td className="px-4 py-3">AES-256-GCM, CHACHA20-POLY1305</td>
                    <td className="px-4 py-3">Default, best performance</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">FIPS</td>
                    <td className="px-4 py-3">AES-256-GCM, AES-128-GCM</td>
                    <td className="px-4 py-3">FIPS 140-3 compliance</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Compatible</td>
                    <td className="px-4 py-3">AES-256-GCM, AES-128-GCM, AES-256-CBC, AES-128-CBC</td>
                    <td className="px-4 py-3">Legacy client support</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        {/* Certificate CA Section */}
        <section id="certificate-ca" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-gray-900 mb-2">Certificate Authority (CA)</h1>
            <p className="text-gray-600 mb-6">
              GateKey includes an embedded Certificate Authority for issuing VPN certificates. Manage your CA from the Settings page.
            </p>

            {/* CA Overview */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">How the CA Works</h3>
            <div className="bg-gray-50 rounded-lg p-4 mb-6">
              <ul className="space-y-2 text-gray-600">
                <li className="flex items-start">
                  <svg className="h-5 w-5 text-green-500 mr-2 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  GateKey generates a root CA on first startup
                </li>
                <li className="flex items-start">
                  <svg className="h-5 w-5 text-green-500 mr-2 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  Client certificates are issued on-demand when generating configs
                </li>
                <li className="flex items-start">
                  <svg className="h-5 w-5 text-green-500 mr-2 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  Certificates are short-lived (default: 24 hours) for security
                </li>
                <li className="flex items-start">
                  <svg className="h-5 w-5 text-green-500 mr-2 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  Gateways receive their server certificates during provisioning
                </li>
              </ul>
            </div>

            {/* CA Operations */}
            <h3 className="text-lg font-medium text-gray-900 mb-3">CA Operations</h3>
            <div className="space-y-4">
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Download CA Certificate</h4>
                <p className="text-sm text-gray-600">
                  Download the CA public certificate. This can be used to verify certificates or for manual client configuration.
                </p>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Rotate CA</h4>
                <p className="text-sm text-gray-600">
                  Generate a new CA certificate and key. <strong className="text-red-600">Warning:</strong> This will invalidate all existing certificates. All gateways will need to reprovision and all active VPN connections will be terminated.
                </p>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <h4 className="font-medium text-gray-900 mb-2">Import CA</h4>
                <p className="text-sm text-gray-600">
                  Import an existing CA certificate and private key. Useful for migrating from another PKI system or using an enterprise CA.
                </p>
              </div>
            </div>

            {/* Best Practices */}
            <h3 className="text-lg font-medium text-gray-900 mt-6 mb-3">Best Practices</h3>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <ul className="space-y-2 text-sm text-yellow-800">
                <li>• Keep certificate validity short (24 hours recommended) to limit exposure</li>
                <li>• Rotate the CA periodically (annually) as part of security hygiene</li>
                <li>• Back up the CA certificate and key securely</li>
                <li>• Monitor audit logs for unusual certificate generation patterns</li>
              </ul>
            </div>
          </div>
        </section>

        {/* Permission Flow Diagram */}
        <section className="scroll-mt-24">
          <div className="card">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Permission Flow</h2>
            <div className="bg-gray-50 rounded-lg p-6 font-mono text-sm overflow-x-auto">
              <pre className="text-gray-700">{`User requests VPN connection
        │
        ▼
┌───────────────────────────────┐
│ Is user assigned to gateway?  │──NO──► Connection Denied
└───────────────┬───────────────┘
                │ YES
                ▼
┌───────────────────────────────┐
│ Generate short-lived config   │
│ (certificate valid 24 hours)  │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│ Gateway verifies on connect:  │
│ - Certificate valid?          │──NO──► Connection Rejected
│ - User still has access?      │
└───────────────┬───────────────┘
                │ YES
                ▼
┌───────────────────────────────┐
│ Apply firewall rules:         │
│ - Default: DENY ALL           │
│ - Allow: user's access rules  │
└───────────────┬───────────────┘
                │
                ▼
┌───────────────────────────────┐
│ User can only reach resources │
│ permitted by their rules      │
└───────────────────────────────┘`}</pre>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
