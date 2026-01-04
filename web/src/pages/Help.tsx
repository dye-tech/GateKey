import { useState, useEffect } from 'react'

interface Section {
  id: string
  title: string
  icon: React.ReactNode
}

const sections: Section[] = [
  {
    id: 'downloads',
    title: 'Downloads',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
      </svg>
    ),
  },
  {
    id: 'install',
    title: 'Install Client',
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
    id: 'admin-configs',
    title: 'Config Management',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
  },
  {
    id: 'api-keys',
    title: 'API Keys',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    ),
  },
  {
    id: 'admin-cli',
    title: 'Admin CLI',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
    ),
  },
  {
    id: 'diagnostics',
    title: 'Diagnostics',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065zM15 12a3 3 0 11-6 0 3 3 0 016 0z" />
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
          <h2 className="text-sm font-semibold text-theme-primary uppercase tracking-wider mb-4">
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
                    : 'text-theme-secondary hover:bg-theme-secondary hover:text-theme-primary'
                }`}
              >
                <span className={`mr-3 ${activeSection === section.id ? 'text-primary-600' : 'text-theme-muted'}`}>
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
        {/* Downloads Section */}
        <section id="downloads" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Download GateKey CLIs</h1>
            <p className="text-theme-secondary mb-6">
              Download the GateKey command-line tools for your platform.
            </p>

            {/* GateKey Client */}
            <div className="mb-8">
              <h3 className="text-lg font-medium text-theme-primary mb-3">GateKey Client (gatekey)</h3>
              <p className="text-theme-secondary mb-4">
                The VPN client for connecting to gateways. Required for end users.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <a
                  href={`${baseUrl}/bin/gatekey-linux-amd64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  <svg className="h-8 w-8 mr-3" viewBox="0 0 256 256" fill="none">
                    <ellipse cx="128" cy="156" rx="72" ry="84" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="176" rx="48" ry="60" fill="#FFFFFF"/>
                    <circle cx="128" cy="72" r="48" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="80" rx="32" ry="28" fill="#FFFFFF"/>
                    <circle cx="116" cy="70" r="4" fill="#1a1a1a"/>
                    <circle cx="140" cy="70" r="4" fill="#1a1a1a"/>
                    <path d="M128 76 L116 92 L140 92 Z" fill="#F4A103"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">Linux (x64)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-linux-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-linux-arm64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  <svg className="h-8 w-8 mr-3" viewBox="0 0 256 256" fill="none">
                    <ellipse cx="128" cy="156" rx="72" ry="84" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="176" rx="48" ry="60" fill="#FFFFFF"/>
                    <circle cx="128" cy="72" r="48" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="80" rx="32" ry="28" fill="#FFFFFF"/>
                    <circle cx="116" cy="70" r="4" fill="#1a1a1a"/>
                    <circle cx="140" cy="70" r="4" fill="#1a1a1a"/>
                    <path d="M128 76 L116 92 L140 92 Z" fill="#F4A103"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">Linux (ARM64)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-linux-arm64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-darwin-amd64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">macOS (Intel)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-darwin-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-darwin-arm64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">macOS (Apple Silicon)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-darwin-arm64</p>
                  </div>
                </a>
              </div>
            </div>

            {/* GateKey Admin CLI */}
            <div className="mb-8">
              <h3 className="text-lg font-medium text-theme-primary mb-3">GateKey Admin CLI (gatekey-admin)</h3>
              <p className="text-theme-secondary mb-4">
                Administrative CLI for managing gateways, networks, users, API keys, and more. For administrators only.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <a
                  href={`${baseUrl}/bin/gatekey-admin-linux-amd64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey-admin"
                >
                  <svg className="h-8 w-8 mr-3" viewBox="0 0 256 256" fill="none">
                    <ellipse cx="128" cy="156" rx="72" ry="84" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="176" rx="48" ry="60" fill="#FFFFFF"/>
                    <circle cx="128" cy="72" r="48" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="80" rx="32" ry="28" fill="#FFFFFF"/>
                    <circle cx="116" cy="70" r="4" fill="#1a1a1a"/>
                    <circle cx="140" cy="70" r="4" fill="#1a1a1a"/>
                    <path d="M128 76 L116 92 L140 92 Z" fill="#F4A103"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">Linux (x64)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-admin-linux-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-admin-linux-arm64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey-admin"
                >
                  <svg className="h-8 w-8 mr-3" viewBox="0 0 256 256" fill="none">
                    <ellipse cx="128" cy="156" rx="72" ry="84" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="176" rx="48" ry="60" fill="#FFFFFF"/>
                    <circle cx="128" cy="72" r="48" fill="#1a1a1a"/>
                    <ellipse cx="128" cy="80" rx="32" ry="28" fill="#FFFFFF"/>
                    <circle cx="116" cy="70" r="4" fill="#1a1a1a"/>
                    <circle cx="140" cy="70" r="4" fill="#1a1a1a"/>
                    <path d="M128 76 L116 92 L140 92 Z" fill="#F4A103"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">Linux (ARM64)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-admin-linux-arm64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-admin-darwin-amd64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey-admin"
                >
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">macOS (Intel)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-admin-darwin-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-admin-darwin-arm64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey-admin"
                >
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">macOS (Apple Silicon)</p>
                    <p className="text-xs text-theme-tertiary">gatekey-admin-darwin-arm64</p>
                  </div>
                </a>
              </div>
            </div>

            {/* GateKey Android App */}
            <div className="mb-8">
              <h3 className="text-lg font-medium text-theme-primary mb-3">GateKey Android App</h3>
              <p className="text-theme-secondary mb-4">
                Mobile VPN client for Android devices. Connect to gateways on the go.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                <a
                  href="/mobile/gatekey-android.apk"
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-colors"
                  download="gatekey-android.apk"
                >
                  <svg className="h-8 w-8 mr-3" viewBox="0 0 24 24" fill="#3DDC84">
                    <path d="M17.6 11.4c0-.4-.3-.7-.7-.7h-.2c-.4 0-.7.3-.7.7v4.1c0 .4.3.7.7.7h.2c.4 0 .7-.3.7-.7v-4.1zm-9.9 0c0-.4-.3-.7-.7-.7h-.2c-.4 0-.7.3-.7.7v4.1c0 .4.3.7.7.7H7c.4 0 .7-.3.7-.7v-4.1zM14.7 4l.9-1.6c0-.1 0-.2-.1-.2-.1 0-.2 0-.2.1l-.9 1.7c-.8-.4-1.6-.5-2.4-.5s-1.7.2-2.4.5L8.7 2.3c0-.1-.1-.1-.2-.1-.1 0-.1.1-.1.2L9.3 4C7.9 4.7 6.9 6 6.6 7.5h10.8c-.3-1.5-1.3-2.8-2.7-3.5zm-4.4 2c-.3 0-.5-.2-.5-.5s.2-.5.5-.5.5.2.5.5-.2.5-.5.5zm3.4 0c-.3 0-.5-.2-.5-.5s.2-.5.5-.5.5.2.5.5-.2.5-.5.5zM6.6 8.1v7.2c0 .5.4.9.9.9h.5v2.2c0 .4.3.7.7.7h.2c.4 0 .7-.3.7-.7v-2.2h2.8v2.2c0 .4.3.7.7.7h.2c.4 0 .7-.3.7-.7v-2.2h.5c.5 0 .9-.4.9-.9V8.1H6.6z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">Download for Android</p>
                    <p className="text-xs text-theme-tertiary">gatekey-android.apk (2.3 MB)</p>
                  </div>
                </a>
              </div>

              {/* Android Installation Instructions */}
              <div className="info-box">
                <h4 className="info-box-title mb-3">Installation Instructions</h4>
                <ol className="list-decimal list-inside space-y-2 info-box-text">
                  <li>Download the APK file to your Android device</li>
                  <li>Open your device <strong>Settings</strong> → <strong>Security</strong></li>
                  <li>Enable <strong>Install from unknown sources</strong> (or allow your browser to install apps)</li>
                  <li>Open the downloaded APK file to install</li>
                  <li>Launch GateKey and sign in with your credentials</li>
                  <li>Select a gateway and tap <strong>Connect</strong></li>
                </ol>
                <p className="mt-3 text-xs text-theme-muted">
                  <strong>Note:</strong> Requires Android 8.0 (Oreo) or later. The app uses the system VPN APIs and will request VPN permissions on first connect.
                </p>
              </div>
            </div>

            {/* Quick Install */}
            <div className="bg-theme-tertiary rounded-lg p-4">
              <h3 className="text-lg font-medium text-theme-primary mb-2">Quick Install (Linux/macOS)</h3>
              <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto mb-4">
                <code className="text-green-400 text-sm">
                  curl -sSL {baseUrl}/scripts/install-client.sh | bash
                </code>
              </div>
              <p className="text-sm text-theme-secondary">
                Or download the binary manually, make it executable with <code className="bg-theme-secondary px-1 rounded">chmod +x</code>, and move it to your PATH.
              </p>
            </div>
          </div>
        </section>

        {/* Install Section */}
        <section id="install" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Install GateKey CLI</h1>
            <p className="text-theme-secondary mb-6">
              Download and install the GateKey CLI client to connect to VPN gateways from your terminal.
            </p>

            {/* Quick Install Script */}
            <div className="mb-6">
              <h3 className="text-lg font-medium text-theme-primary mb-2">Quick Install (Linux/macOS)</h3>
              <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
                <code className="text-green-400 text-sm">
                  curl -sSL {baseUrl}/scripts/install-client.sh | bash
                </code>
              </div>
            </div>

            {/* Download Binaries */}
            <div className="mb-6">
              <h3 className="text-lg font-medium text-theme-primary mb-3">Download Binaries</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <a
                  href={`${baseUrl}/bin/gatekey-linux-amd64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
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
                    <p className="font-medium text-theme-primary">Linux (x64)</p>
                    <p className="text-sm text-theme-tertiary">gatekey-linux-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-darwin-amd64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  {/* Apple Logo */}
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">macOS (Intel)</p>
                    <p className="text-sm text-theme-tertiary">gatekey-darwin-amd64</p>
                  </div>
                </a>
                <a
                  href={`${baseUrl}/bin/gatekey-darwin-arm64`}
                  className="flex items-center p-4 border border-theme rounded-lg hover:border-primary-500 hover:bg-primary-50 transition-colors"
                  download="gatekey"
                >
                  {/* Apple Logo */}
                  <svg className="h-8 w-8 mr-3" fill="#555555" viewBox="0 0 24 24">
                    <path d="M18.71 19.5C17.88 20.74 17 21.95 15.66 21.97C14.32 22 13.89 21.18 12.37 21.18C10.84 21.18 10.37 21.95 9.1 22C7.79 22.05 6.8 20.68 5.96 19.47C4.25 17 2.94 12.45 4.7 9.39C5.57 7.87 7.13 6.91 8.82 6.88C10.1 6.86 11.32 7.75 12.11 7.75C12.89 7.75 14.37 6.68 15.92 6.84C16.57 6.87 18.39 7.1 19.56 8.82C19.47 8.88 17.39 10.1 17.41 12.63C17.44 15.65 20.06 16.66 20.09 16.67C20.06 16.74 19.67 18.11 18.71 19.5M13 3.5C13.73 2.67 14.94 2.04 15.94 2C16.07 3.17 15.6 4.35 14.9 5.19C14.21 6.04 13.07 6.7 11.95 6.61C11.8 5.46 12.36 4.26 13 3.5Z"/>
                  </svg>
                  <div>
                    <p className="font-medium text-theme-primary">macOS (Apple Silicon)</p>
                    <p className="text-sm text-theme-tertiary">gatekey-darwin-arm64</p>
                  </div>
                </a>
              </div>
            </div>

            {/* Manual Install */}
            <div className="bg-theme-tertiary rounded-lg p-4">
              <h3 className="text-lg font-medium text-theme-primary mb-2">Manual Installation</h3>
              <ol className="list-decimal list-inside space-y-2 text-sm text-theme-secondary">
                <li>Download the binary for your platform</li>
                <li>Make it executable: <code className="bg-theme-secondary px-1 rounded">chmod +x gatekey-*</code></li>
                <li>Move to PATH: <code className="bg-theme-secondary px-1 rounded">sudo mv gatekey-* /usr/local/bin/gatekey</code></li>
                <li>Verify: <code className="bg-theme-secondary px-1 rounded">gatekey version</code></li>
              </ol>
            </div>
          </div>
        </section>

        {/* Configure Section */}
        <section id="configure" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Configure GateKey CLI</h1>
            <p className="text-theme-secondary mb-6">
              Set up the GateKey CLI to connect to your organization's VPN.
            </p>

            {/* Quick Start Steps */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 bg-theme-tertiary rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">1</span>
                  <span className="font-medium text-theme-primary">Initialize Config</span>
                </div>
                <code className="text-xs bg-theme-secondary px-2 py-1 rounded block">gatekey config init --server {baseUrl}</code>
              </div>
              <div className="p-4 bg-theme-tertiary rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">2</span>
                  <span className="font-medium text-theme-primary">Login</span>
                </div>
                <code className="text-xs bg-theme-secondary px-2 py-1 rounded block">gatekey login</code>
              </div>
              <div className="p-4 bg-theme-tertiary rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">3</span>
                  <span className="font-medium text-theme-primary">Connect</span>
                </div>
                <code className="text-xs bg-theme-secondary px-2 py-1 rounded block">gatekey connect</code>
              </div>
              <div className="p-4 bg-theme-tertiary rounded-lg">
                <div className="flex items-center space-x-3 mb-2">
                  <span className="flex-shrink-0 w-6 h-6 bg-primary-600 text-white rounded-full flex items-center justify-center text-sm font-medium">4</span>
                  <span className="font-medium text-theme-primary">Disconnect</span>
                </div>
                <code className="text-xs bg-theme-secondary px-2 py-1 rounded block">gatekey disconnect</code>
              </div>
            </div>

            {/* Multi-Gateway Support */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Multi-Gateway Support</h3>
            <p className="text-theme-secondary mb-4">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Configuration File</h3>
            <p className="text-theme-secondary mb-4">
              The CLI stores its configuration in <code className="bg-theme-secondary px-1 rounded">~/.gatekey/config.yaml</code>. You can manually edit this file or use the CLI commands.
            </p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
              <pre className="text-green-400 text-sm">{`# View current configuration
gatekey config show

# Set server URL
gatekey config set server ${baseUrl}

# Reset configuration
gatekey config reset`}</pre>
            </div>

            {/* CLI Command Reference */}
            <h3 className="text-lg font-medium text-theme-primary mb-3 mt-6">CLI Command Reference</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Command</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey login</td>
                    <td className="px-4 py-3">Authenticate with SSO (opens browser) or API key (--api-key)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey logout</td>
                    <td className="px-4 py-3">Clear saved credentials and session</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey connect [gateway]</td>
                    <td className="px-4 py-3">Connect to VPN gateway. Use --mesh &lt;hub&gt; for mesh hubs</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey disconnect [gateway]</td>
                    <td className="px-4 py-3">Disconnect from VPN. Use --all to disconnect all</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey status</td>
                    <td className="px-4 py-3">Show current connection status and session info</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey list</td>
                    <td className="px-4 py-3">List available gateways and mesh hubs</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey config init</td>
                    <td className="px-4 py-3">Initialize config with --server URL</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey config show</td>
                    <td className="px-4 py-3">Display current configuration</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey config set &lt;key&gt; &lt;value&gt;</td>
                    <td className="px-4 py-3">Update configuration setting</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey version</td>
                    <td className="px-4 py-3">Show version information</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">gatekey fips-check</td>
                    <td className="px-4 py-3">Verify FIPS 140-2 compliant OpenSSL library</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Common Flags */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Common Flags</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`--server URL     Override server URL from config
--config PATH    Use alternate config file
--api-key KEY    Use API key for authentication (login command)
--no-browser     Print login URL instead of opening browser
--all            Disconnect from all gateways
--mesh HUB       Connect to mesh hub instead of gateway
-o, --output     Output format: table, json, yaml`}</pre>
            </div>
          </div>
        </section>

        {/* Gateways Section */}
        <section id="gateways" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Gateways</h1>
            <p className="text-theme-secondary mb-6">
              Gateways are VPN entry points that users connect to. Each gateway runs the OpenVPN server and GateKey gateway agent.
            </p>

            {/* Creating a Gateway */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Creating a Gateway</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Gateway Settings</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Setting</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Tunnel Modes</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Split Tunnel (Default)</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Only routes traffic for permitted networks</li>
                  <li>• Internet traffic uses normal connection</li>
                  <li>• Routes pushed based on user's access rules</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Full Tunnel</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Routes ALL traffic through VPN</li>
                  <li>• Uses redirect-gateway directive</li>
                  <li>• Useful for traffic inspection</li>
                </ul>
              </div>
            </div>

            {/* Installing Gateway Agent */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Installing Gateway Agent</h3>
            <p className="text-theme-secondary mb-4">
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
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Networks</h1>
            <p className="text-theme-secondary mb-6">
              Networks define the CIDR blocks that are reachable through your VPN. Networks are assigned to gateways to advertise routes.
            </p>

            {/* Creating Networks */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Creating a Network</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Assigning Networks to Gateways</h3>
            <p className="text-theme-secondary mb-4">
              Networks must be assigned to gateways to be advertised to VPN clients:
            </p>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary mb-6">
              <li>Go to <strong>Gateways</strong> and click <strong>Access</strong> on the gateway</li>
              <li>Select the <strong>Networks</strong> tab</li>
              <li>Add the networks this gateway should serve</li>
            </ol>

            {/* How Routes Work */}
            <div className="info-box">
              <div className="flex">
                <svg className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div className="ml-3">
                  <h3 className="info-box-title">How Route Pushing Works</h3>
                  <p className="mt-1 info-box-text">
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
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Access Rules</h1>
            <p className="text-theme-secondary mb-6">
              Access rules define what specific resources users can reach. GateKey uses a Zero Trust model where all traffic is blocked by default.
            </p>

            {/* Default Deny */}
            <div className="info-box mb-6">
              <div className="flex">
                <svg className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div className="ml-3">
                  <h3 className="info-box-title">Default Deny Policy</h3>
                  <p className="mt-1 info-box-text">
                    All traffic is <strong>blocked by default</strong>. Users can only access resources that are explicitly permitted through access rules.
                  </p>
                </div>
              </div>
            </div>

            {/* Rule Types */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Rule Types</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Example</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">IP Address</td>
                    <td className="px-4 py-3"><code className="bg-theme-secondary px-1 rounded">192.168.1.100</code></td>
                    <td className="px-4 py-3">Single IP address</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">CIDR Range</td>
                    <td className="px-4 py-3"><code className="bg-theme-secondary px-1 rounded">10.0.0.0/24</code></td>
                    <td className="px-4 py-3">Network range (also pushed as route)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Hostname</td>
                    <td className="px-4 py-3"><code className="bg-theme-secondary px-1 rounded">api.internal.com</code></td>
                    <td className="px-4 py-3">Exact hostname match</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Wildcard</td>
                    <td className="px-4 py-3"><code className="bg-theme-secondary px-1 rounded">*.internal.com</code></td>
                    <td className="px-4 py-3">Pattern matching for hostnames</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Creating Rules */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Creating Access Rules</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Assigning Rules to Users/Groups</h3>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary">
              <li>On the <strong>Access Rules</strong> page, click <strong>Assign</strong> on a rule</li>
              <li>Add users or groups that should have this access</li>
              <li>Rules take effect immediately (within 10 seconds)</li>
            </ol>
          </div>
        </section>

        {/* Proxy Apps Section */}
        <section id="proxy-apps" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Proxy Applications</h1>
            <p className="text-theme-secondary mb-6">
              Proxy Apps provide clientless access to internal web applications. Users can access internal apps through their browser without installing a VPN client.
            </p>

            {/* How It Works */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">How It Works</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6">
              <ol className="list-decimal list-inside space-y-2 text-theme-secondary">
                <li>User authenticates via SSO</li>
                <li>User accesses <code className="bg-theme-secondary px-1 rounded">{baseUrl}/proxy/app-slug/</code></li>
                <li>GateKey verifies user has permission to access the app</li>
                <li>Traffic is proxied to the internal application</li>
                <li>User headers (email, groups) are injected for the backend</li>
              </ol>
            </div>

            {/* Creating Proxy Apps */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Creating a Proxy Application</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Proxy Settings</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Setting</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
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
            <h1 className="text-2xl font-bold text-theme-primary mb-2">OIDC Providers</h1>
            <p className="text-theme-secondary mb-6">
              Configure OpenID Connect (OIDC) providers to enable Single Sign-On (SSO) authentication with your identity provider.
            </p>

            {/* Supported Providers */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Supported Providers</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
              <div className="p-3 border border-theme rounded-lg text-center">
                <p className="font-medium text-theme-primary">Okta</p>
              </div>
              <div className="p-3 border border-theme rounded-lg text-center">
                <p className="font-medium text-theme-primary">Azure AD</p>
              </div>
              <div className="p-3 border border-theme rounded-lg text-center">
                <p className="font-medium text-theme-primary">Google</p>
              </div>
              <div className="p-3 border border-theme rounded-lg text-center">
                <p className="font-medium text-theme-primary">Keycloak</p>
              </div>
            </div>

            {/* Configuration */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Configuring OIDC</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Callback URL</h3>
            <p className="text-theme-secondary mb-2">Configure this callback URL in your identity provider:</p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
              <code className="text-green-400 text-sm">{baseUrl}/api/v1/auth/oidc/callback</code>
            </div>
          </div>
        </section>

        {/* SAML Providers Section */}
        <section id="saml-providers" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">SAML Providers</h1>
            <p className="text-theme-secondary mb-6">
              Configure SAML 2.0 providers for enterprise Single Sign-On integration.
            </p>

            {/* Configuration */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Configuring SAML</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Service Provider Metadata</h3>
            <p className="text-theme-secondary mb-2">Configure these values in your identity provider:</p>
            <div className="bg-theme-tertiary rounded-lg p-4 space-y-3">
              <div>
                <p className="text-sm font-medium text-theme-secondary">Entity ID (Audience):</p>
                <code className="text-sm text-theme-secondary">{baseUrl}</code>
              </div>
              <div>
                <p className="text-sm font-medium text-theme-secondary">ACS URL:</p>
                <code className="text-sm text-theme-secondary">{baseUrl}/api/v1/auth/saml/acs</code>
              </div>
            </div>
          </div>
        </section>

        {/* Monitoring Section */}
        <section id="monitoring" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Login Monitoring</h1>
            <p className="text-theme-secondary mb-6">
              Monitor user authentication events, track login activity, and configure log retention policies.
            </p>

            {/* Overview */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Overview</h3>
            <p className="text-theme-secondary mb-4">
              The Monitoring page provides visibility into all authentication events across your GateKey deployment.
              Track successful and failed logins, identify suspicious activity, and maintain compliance with audit requirements.
            </p>

            {/* Features */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Features</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Login Logs</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• View all authentication events</li>
                  <li>• Filter by user, IP, provider, status</li>
                  <li>• See IP address and location data</li>
                  <li>• Track failed login attempts with reasons</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Statistics</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Total logins (successful/failed)</li>
                  <li>• Unique users and IP addresses</li>
                  <li>• Logins by provider (OIDC/SAML/Local)</li>
                  <li>• Logins by country/location</li>
                </ul>
              </div>
            </div>

            {/* Data Captured */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Data Captured</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Field</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Log Retention</h3>
            <p className="text-theme-secondary mb-4">
              Configure how long login logs are retained in the database. Logs older than the retention period
              are automatically deleted by a background job that runs every 6 hours.
            </p>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6">
              <ul className="space-y-2 text-theme-secondary text-sm">
                <li><strong>Default:</strong> 30 days</li>
                <li><strong>Minimum:</strong> 1 day</li>
                <li><strong>Forever:</strong> Set to 0 to keep logs indefinitely</li>
              </ul>
            </div>

            {/* Manual Purge */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Manual Purge</h3>
            <p className="text-theme-secondary mb-4">
              You can manually purge logs older than a specified number of days from the Settings tab.
              This is useful for immediately clearing old data or complying with data retention policies.
            </p>
            <div className="info-box">
              <div className="flex">
                <svg className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div className="ml-3">
                  <h3 className="info-box-title">Warning</h3>
                  <p className="mt-1 info-box-text">
                    Purging logs is irreversible. Once deleted, login history cannot be recovered.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Admin Config Management Section */}
        <section id="admin-configs" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Admin Config Management</h1>
            <p className="text-theme-secondary mb-6">
              Administrators can view and manage all VPN configurations across all users from a centralized dashboard.
            </p>

            {/* Overview */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Overview</h3>
            <p className="text-theme-secondary mb-4">
              The Admin All Configs page provides administrators with a complete view of every VPN configuration
              generated across the organization. This includes both standard gateway configs and mesh VPN configs,
              with full visibility into who generated each config and when.
            </p>

            {/* Features */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Features</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Gateway Configs</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• View all gateway VPN configurations</li>
                  <li>• See user email and name for each config</li>
                  <li>• Filter by user, status (active/revoked/expired)</li>
                  <li>• Revoke any active configuration</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Mesh Configs</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• View all mesh hub VPN configurations</li>
                  <li>• See user ownership for each config</li>
                  <li>• Track expiration and download status</li>
                  <li>• Revoke mesh configs as needed</li>
                </ul>
              </div>
            </div>

            {/* Information Displayed */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Information Displayed</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Field</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium">User</td>
                    <td className="px-4 py-3">Email and display name of the user who generated the config</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Gateway/Hub</td>
                    <td className="px-4 py-3">The VPN gateway or mesh hub the config connects to</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">File Name</td>
                    <td className="px-4 py-3">The generated .ovpn configuration file name</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Created</td>
                    <td className="px-4 py-3">When the configuration was generated</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Expires</td>
                    <td className="px-4 py-3">When the configuration will expire</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium">Status</td>
                    <td className="px-4 py-3">Active, Revoked, or Expired</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Accessing the Page */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Accessing Config Management</h3>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary mb-6">
              <li>Navigate to <strong>Administration</strong> in the sidebar</li>
              <li>Click <strong>All Configs</strong></li>
              <li>Use the tabs to switch between Gateway and Mesh configs</li>
              <li>Use filters to find specific configs by user or status</li>
            </ol>

            {/* Revoking Configs */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Revoking Configurations</h3>
            <p className="text-theme-secondary mb-4">
              Administrators can revoke any active VPN configuration. When a config is revoked:
            </p>
            <ul className="list-disc list-inside space-y-2 text-theme-secondary mb-6">
              <li>The user will no longer be able to connect using that configuration</li>
              <li>The certificate associated with the config is marked as invalid</li>
              <li>A reason for revocation can be recorded for audit purposes</li>
              <li>The config remains visible with a "Revoked" status</li>
            </ul>

            <div className="info-box">
              <div className="flex">
                <svg className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div className="ml-3">
                  <h3 className="info-box-title">Tip</h3>
                  <p className="mt-1 info-box-text">
                    You can also view and revoke configs for a specific user from the Users page by clicking on a user and going to the "VPN Configs" tab.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* API Keys Section */}
        <section id="api-keys" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">API Keys</h1>
            <p className="text-theme-secondary mb-6">
              API keys provide programmatic access to GateKey without browser-based SSO. They're ideal for automation, CI/CD pipelines, headless servers, and CLI usage.
            </p>

            {/* API Key Format */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">API Key Format</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6">
              <code className="text-sm text-theme-secondary font-mono">gk_&lt;base64-encoded-random-bytes&gt;</code>
              <p className="text-sm text-theme-secondary mt-2">
                Example: <code className="bg-theme-secondary px-1 rounded">gk_dGhpcyBpcyBhIHNhbXBsZSBhcGkga2V5...</code>
              </p>
            </div>

            {/* Security Notice */}
            <div className="info-box mb-6">
              <div className="flex">
                <svg className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <div className="ml-3">
                  <h3 className="info-box-title">Security Notice</h3>
                  <p className="mt-1 info-box-text">
                    The full API key is only shown <strong>once</strong> at creation time. Store it securely - it cannot be retrieved later.
                  </p>
                </div>
              </div>
            </div>

            {/* Creating API Keys */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Creating API Keys</h3>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary mb-6">
              <li>Navigate to your <strong>User Profile</strong> (click your name in the top right)</li>
              <li>Go to the <strong>API Keys</strong> tab</li>
              <li>Click <strong>Create API Key</strong></li>
              <li>Enter a descriptive name (e.g., "CI/CD Pipeline", "Laptop CLI")</li>
              <li>Optionally set an expiration time</li>
              <li>Click <strong>Create</strong></li>
              <li><strong>Copy and save the API key immediately</strong></li>
            </ol>

            {/* Using API Keys */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Using API Keys</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`# GateKey Client
gatekey login --api-key gk_your_api_key_here
gatekey connect

# Admin CLI
gatekey-admin login --api-key gk_your_api_key_here
gatekey-admin gateway list

# Direct API Access
curl -H "Authorization: Bearer gk_your_api_key_here" \\
  ${baseUrl}/api/v1/gateways`}</pre>
            </div>

            {/* Scopes */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Available Scopes</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Scope</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">*</td>
                    <td className="px-4 py-3">Full access (default for user-created keys)</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">read:gateways</td>
                    <td className="px-4 py-3">List and view gateways</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">write:gateways</td>
                    <td className="px-4 py-3">Create, update, delete gateways</td>
                  </tr>
                  <tr>
                    <td className="px-4 py-3 font-medium font-mono">vpn:connect</td>
                    <td className="px-4 py-3">Generate VPN configurations and connect</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* Best Practices */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Best Practices</h3>
            <ul className="list-disc list-inside space-y-2 text-theme-secondary">
              <li>Use scopes to limit what each key can do (principle of least privilege)</li>
              <li>Rotate keys periodically (every 90 days recommended)</li>
              <li>Never commit API keys to version control</li>
              <li>Use secrets management systems (Vault, AWS Secrets Manager)</li>
              <li>Monitor "Last Used" timestamps and revoke unused keys</li>
            </ul>
          </div>
        </section>

        {/* Admin CLI Section */}
        <section id="admin-cli" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Admin CLI (gatekey-admin)</h1>
            <p className="text-theme-secondary mb-6">
              The Admin CLI provides command-line administration for GateKey deployments, enabling management of gateways, networks, access rules, users, API keys, and mesh networking.
            </p>

            {/* Quick Start */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Quick Start</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`# 1. Download and install (see Downloads section above)
chmod +x gatekey-admin && sudo mv gatekey-admin /usr/local/bin/

# 2. Initialize with your server URL
gatekey-admin config init --server ${baseUrl}

# 3. Authenticate (opens browser for SSO)
gatekey-admin login

# Or authenticate with API key
gatekey-admin login --api-key gk_your_api_key_here

# 4. Start managing
gatekey-admin gateway list
gatekey-admin user list`}</pre>
            </div>

            {/* Commands Overview */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Available Commands</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Gateway Management</h4>
                <ul className="text-sm text-theme-secondary space-y-1 font-mono">
                  <li>gateway list</li>
                  <li>gateway create</li>
                  <li>gateway update &lt;id&gt;</li>
                  <li>gateway delete &lt;id&gt;</li>
                  <li>gateway reprovision &lt;id&gt;</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Network & Access</h4>
                <ul className="text-sm text-theme-secondary space-y-1 font-mono">
                  <li>network list|create|delete</li>
                  <li>access-rule list|create|delete</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">User Management</h4>
                <ul className="text-sm text-theme-secondary space-y-1 font-mono">
                  <li>user list</li>
                  <li>user get &lt;id&gt;</li>
                  <li>user revoke-configs &lt;id&gt;</li>
                  <li>local-user list|create|delete</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">API Key Management</h4>
                <ul className="text-sm text-theme-secondary space-y-1 font-mono">
                  <li>api-key list [--user EMAIL]</li>
                  <li>api-key create NAME</li>
                  <li>api-key revoke &lt;id&gt;</li>
                  <li>api-key revoke-all --user EMAIL</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Mesh Networking</h4>
                <ul className="text-sm text-theme-secondary space-y-1 font-mono">
                  <li>mesh hub list|create|delete</li>
                  <li>mesh hub provision &lt;id&gt;</li>
                  <li>mesh spoke list|create|delete</li>
                  <li>mesh spoke provision &lt;id&gt;</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Certificates & Audit</h4>
                <ul className="text-sm text-theme-secondary space-y-1 font-mono">
                  <li>ca show|rotate|list</li>
                  <li>audit list [--action X]</li>
                  <li>connection list|disconnect</li>
                </ul>
              </div>
            </div>

            {/* Output Formats */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Output Formats</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`# Table format (default)
gatekey-admin gateway list

# JSON format
gatekey-admin gateway list -o json

# YAML format
gatekey-admin gateway list -o yaml`}</pre>
            </div>

            {/* Config File */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Configuration</h3>
            <p className="text-theme-secondary mb-4">
              Configuration is stored in <code className="bg-theme-secondary px-1 rounded">~/.gatekey-admin/config.yaml</code>:
            </p>
            <div className="bg-theme-tertiary rounded-lg p-4 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`server_url: ${baseUrl}
api_key: gk_your_api_key_here
output: table`}</pre>
            </div>
          </div>
        </section>

        {/* Diagnostics Section */}
        <section id="diagnostics" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Diagnostics</h1>
            <p className="text-theme-secondary mb-6">
              Network troubleshooting tools and remote session management for debugging connectivity issues across your VPN infrastructure.
            </p>

            {/* Network Tools */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Network Tools</h3>
            <p className="text-theme-secondary mb-4">
              Run network diagnostics from the control plane or any connected hub, gateway, or spoke. Access via <strong>Administration → Diagnostics → Network Tools</strong>.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Available Tools</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• <strong>ping</strong> - Test connectivity to a host</li>
                  <li>• <strong>nslookup</strong> - DNS resolution lookup</li>
                  <li>• <strong>traceroute</strong> - Trace network path</li>
                  <li>• <strong>nc (netcat)</strong> - TCP port connectivity test</li>
                  <li>• <strong>nmap</strong> - Port scanning</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Execution Locations</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• <strong>Control Plane</strong> - Run from the central server</li>
                  <li>• <strong>Gateways</strong> - Run from VPN gateways</li>
                  <li>• <strong>Mesh Hubs</strong> - Run from hub servers</li>
                  <li>• <strong>Mesh Spokes</strong> - Run from spoke clients</li>
                </ul>
              </div>
            </div>

            {/* Remote Sessions */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Remote Sessions</h3>
            <p className="text-theme-secondary mb-4">
              Connect to and execute shell commands on remote hubs, gateways, and spokes. Access via <strong>Administration → Diagnostics → Remote Sessions</strong>.
            </p>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6">
              <h4 className="font-medium text-theme-primary mb-2">How It Works</h4>
              <ul className="text-sm text-theme-secondary space-y-2">
                <li>• Agents connect <strong>outbound</strong> to the control plane (no inbound firewall rules needed)</li>
                <li>• Enable remote sessions by setting <code className="bg-theme-secondary px-1 rounded">session_enabled: true</code> in agent config</li>
                <li>• Connected agents appear in the Remote Sessions list</li>
                <li>• Click "Connect" to open an interactive terminal</li>
              </ul>
            </div>

            {/* CLI Commands */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">CLI Commands</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`# Network Troubleshooting
gatekey-admin troubleshoot ping 8.8.8.8
gatekey-admin troubleshoot nslookup google.com
gatekey-admin troubleshoot traceroute 10.0.0.1
gatekey-admin troubleshoot nc api.internal.com 443
gatekey-admin troubleshoot nmap 10.0.0.1 --ports 22,80,443

# Remote Sessions
gatekey-admin session list              # List connected agents
gatekey-admin session exec hub-1 "ip addr"  # Execute single command
gatekey-admin session connect hub-1     # Interactive shell`}</pre>
            </div>
          </div>
        </section>

        {/* Mesh Networking Section */}
        <section id="mesh" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Mesh Networking</h1>
            <p className="text-theme-secondary mb-6">
              Connect remote sites and networks using a hub-and-spoke VPN mesh. Mesh networking allows gateways behind NAT to connect to a central hub, enabling site-to-site connectivity without inbound firewall rules.
            </p>

            {/* Architecture Overview */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Architecture Overview</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`                 ┌─────────────────┐
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
│   Spoke A   │  │   Spoke B   │  │   Spoke C   │
│  10.0.0.0/8 │  │ 192.168.0/24│  │ 172.16.0/16 │
└─────────────┘  └─────────────┘  └─────────────┘
  Home Lab         AWS VPC         Office Network`}</pre>
            </div>

            {/* Key Features */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Key Features</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Spoke-Initiated Connections</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Spokes connect TO the hub (outbound)</li>
                  <li>• Works behind NAT/firewalls</li>
                  <li>• No inbound ports required on spokes</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Standalone Hub</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Hub runs separately from control plane</li>
                  <li>• Deploy on any public server or cloud VM</li>
                  <li>• Syncs configuration from control plane</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Zero-Trust Access</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Fine-grained user/group access control</li>
                  <li>• Per-spoke network permissions</li>
                  <li>• Routes pushed based on access rules</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Dynamic Routing</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Spokes advertise local networks</li>
                  <li>• Hub aggregates and distributes routes</li>
                  <li>• Automatic route updates</li>
                </ul>
              </div>
            </div>

            {/* Setting Up a Hub */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Setting Up a Mesh Hub</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
              <li>Navigate to <strong>Administration → Mesh</strong></li>
              <li>Click <strong>Add Hub</strong> and configure:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Display name for the hub (e.g., "primary-hub")</li>
                  <li><strong>Public Endpoint:</strong> Hostname or IP where spokes will connect</li>
                  <li><strong>VPN Port:</strong> OpenVPN port (default: 1194)</li>
                  <li><strong>VPN Protocol:</strong> UDP (recommended) or TCP</li>
                  <li><strong>VPN Subnet:</strong> Tunnel IP range (e.g., 172.30.0.0/16)</li>
                  <li><strong>Crypto Profile:</strong> FIPS, Modern, or Compatible</li>
                  <li><strong>TLS-Auth:</strong> Enable for additional HMAC security layer (recommended)</li>
                  <li><strong>Full Tunnel Mode:</strong> Route all client traffic through hub</li>
                  <li><strong>Push DNS:</strong> Push DNS servers to connected clients</li>
                  <li><strong>DNS Servers:</strong> Custom DNS servers (defaults to 1.1.1.1, 8.8.8.8)</li>
                  <li><strong>Local Networks:</strong> Networks directly reachable from the hub</li>
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

            {/* Setting Up Spokes */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Adding Mesh Spokes</h3>
            <ol className="list-decimal list-inside space-y-3 text-theme-secondary mb-6">
              <li>In the <strong>Mesh</strong> page, switch to the <strong>Spokes</strong> tab</li>
              <li>Select the hub this spoke will connect to</li>
              <li>Click <strong>Add Spoke</strong> and configure:
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1 text-sm">
                  <li><strong>Name:</strong> Identifier for this spoke (e.g., "home-lab")</li>
                  <li><strong>Description:</strong> Optional description</li>
                  <li><strong>Local Networks:</strong> CIDR blocks behind this spoke (e.g., 10.0.0.0/8)</li>
                </ul>
              </li>
              <li><strong>Save the Spoke Token</strong> - shown only once</li>
              <li>Click <strong>Install Script</strong> for the deployment script</li>
              <li>Run on your spoke server:
                <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto mt-2">
                  <code className="text-green-400 text-sm">sudo bash install-mesh-spoke.sh</code>
                </div>
              </li>
            </ol>

            {/* How It Works */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">How It Works</h3>
            <div className="info-box mb-6">
              <ol className="list-decimal list-inside space-y-2 text-gray-700 dark:text-gray-300 text-sm">
                <li><strong>Spoke Connects:</strong> Mesh spoke initiates OpenVPN connection to hub</li>
                <li><strong>Authentication:</strong> Spoke authenticates using its provisioned certificate</li>
                <li><strong>Route Advertisement:</strong> Spoke tells hub about its local networks via <code className="bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 px-1 rounded">iroute</code></li>
                <li><strong>Hub Aggregates:</strong> Hub collects routes from all spokes</li>
                <li><strong>Traffic Routing:</strong> Traffic between sites flows through the hub</li>
                <li><strong>Control Plane Sync:</strong> Hub periodically syncs access rules and config</li>
              </ol>
            </div>

            {/* Binaries */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Mesh Binaries</h3>
            <div className="overflow-x-auto mb-6">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Binary</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">Access Control</h3>
            <p className="text-theme-secondary mb-4">
              Mesh networking supports fine-grained access control at both the hub and spoke level, allowing you to control which users can connect and which networks they can access.
            </p>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Hub Access Control</h4>
                <p className="text-sm text-theme-secondary mb-2">
                  Control who can connect to the mesh network as a VPN client.
                </p>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Assign users directly to hub</li>
                  <li>• Assign groups for team access</li>
                  <li>• Users without access cannot generate mesh configs</li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Spoke Access Control</h4>
                <p className="text-sm text-theme-secondary mb-2">
                  Control who can route traffic to networks behind each spoke.
                </p>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Per-spoke user/group assignments</li>
                  <li>• Limit access to specific CIDR ranges</li>
                  <li>• Fine-grained network segmentation</li>
                </ul>
              </div>
            </div>

            {/* Managing Hub Access */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Managing Hub Access</h3>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary mb-6">
              <li>Navigate to <strong>Administration → Mesh</strong></li>
              <li>On the <strong>Hubs</strong> tab, click the actions menu on a hub</li>
              <li>Select <strong>Manage Access</strong></li>
              <li>Add users or groups that should be able to connect to this mesh network</li>
              <li>Users can then generate VPN configs from the Connect page</li>
            </ol>

            {/* Managing Spoke Access */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Managing Spoke Access</h3>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary mb-6">
              <li>Navigate to <strong>Administration → Mesh → Spokes</strong></li>
              <li>Select the hub and find the spoke you want to configure</li>
              <li>Click the actions menu and select <strong>Manage Access</strong></li>
              <li>The modal shows the networks accessible via this spoke</li>
              <li>Add users or groups that should have access to these networks</li>
            </ol>

            <div className="info-box mb-6">
              <div className="flex">
                <svg className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div className="ml-3">
                  <h3 className="info-box-title">Example: Network Segmentation</h3>
                  <p className="mt-1 info-box-text">
                    A spoke advertises networks <code className="bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 px-1 rounded">10.0.0.0/24</code> (prod) and <code className="bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 px-1 rounded">10.0.1.0/24</code> (dev).
                    You can assign the "Developers" group to the spoke so they can access both networks, while the "QA" group only
                    gets access via a different spoke that only advertises the dev network.
                  </p>
                </div>
              </div>
            </div>

            {/* Client Connectivity */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Connecting as a Client</h3>
            <p className="text-theme-secondary mb-4">
              Users with hub access can download VPN configs to connect to the mesh network:
            </p>
            <ol className="list-decimal list-inside space-y-2 text-theme-secondary mb-6">
              <li>Navigate to the <strong>Connect</strong> page</li>
              <li>Switch to the <strong>Mesh Networks</strong> tab</li>
              <li>Find the mesh hub you have access to</li>
              <li>Click <strong>Download Config</strong></li>
              <li>Import the <code className="bg-theme-secondary px-1 rounded">.ovpn</code> file into your OpenVPN client</li>
            </ol>

            {/* CLI Commands */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Using the GateKey CLI</h3>
            <p className="text-theme-secondary mb-4">
              The GateKey CLI provides commands for connecting to mesh networks:
            </p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto mb-6">
              <pre className="text-green-400 text-sm">{`# List available mesh hubs
gatekey mesh list

# Connect to a mesh hub
gatekey connect --mesh primary-hub

# Check connection status
gatekey status

# Disconnect
gatekey disconnect primary-hub`}</pre>
            </div>

            {/* Troubleshooting */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">Troubleshooting</h3>
            <div className="space-y-4">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Spoke Won't Connect</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Verify hub endpoint is reachable from spoke</li>
                  <li>• Check firewall allows outbound UDP/1194 (or configured port)</li>
                  <li>• Verify spoke token is correct</li>
                  <li>• Check logs: <code className="bg-theme-secondary px-1 rounded">journalctl -u gatekey-mesh-gateway -f</code></li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Hub Shows Offline</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Verify hub can reach control plane</li>
                  <li>• Check hub service: <code className="bg-theme-secondary px-1 rounded">systemctl status gatekey-hub</code></li>
                  <li>• Verify API token is correct in hub config</li>
                  <li>• Check logs: <code className="bg-theme-secondary px-1 rounded">journalctl -u gatekey-hub -f</code></li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Routes Not Working</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Verify spoke's local networks are correctly configured</li>
                  <li>• Check IP forwarding is enabled on hub and spokes</li>
                  <li>• Verify firewall rules allow forwarded traffic</li>
                  <li>• Check OpenVPN routing: <code className="bg-theme-secondary px-1 rounded">ip route show</code></li>
                </ul>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Client Can't Access Spoke Networks</h4>
                <ul className="text-sm text-theme-secondary space-y-1">
                  <li>• Verify user has hub access (Manage Access → Users/Groups)</li>
                  <li>• Check network is assigned to hub (Manage Access → Networks)</li>
                  <li>• Verify user has access rules for the target network</li>
                  <li>• Regenerate VPN config after making access changes</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* General Settings Section */}
        <section id="general" className="scroll-mt-24">
          <div className="card">
            <h1 className="text-2xl font-bold text-theme-primary mb-2">General Settings</h1>
            <p className="text-theme-secondary mb-6">
              Configure global settings for your GateKey deployment.
            </p>

            {/* Settings Table */}
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Setting</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Description</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
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
            <h3 className="text-lg font-medium text-theme-primary mt-6 mb-3">Crypto Profiles</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-tertiary">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Profile</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Ciphers</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-theme-tertiary uppercase">Use Case</th>
                  </tr>
                </thead>
                <tbody className="bg-theme-card divide-y divide-theme text-sm">
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
            <h1 className="text-2xl font-bold text-theme-primary mb-2">Certificate Authority (CA)</h1>
            <p className="text-theme-secondary mb-6">
              GateKey includes an embedded Certificate Authority for issuing VPN certificates. Manage your CA from the Settings page.
            </p>

            {/* CA Overview */}
            <h3 className="text-lg font-medium text-theme-primary mb-3">How the CA Works</h3>
            <div className="bg-theme-tertiary rounded-lg p-4 mb-6">
              <ul className="space-y-2 text-theme-secondary">
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
            <h3 className="text-lg font-medium text-theme-primary mb-3">CA Operations</h3>
            <div className="space-y-4">
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Download CA Certificate</h4>
                <p className="text-sm text-theme-secondary">
                  Download the CA public certificate. This can be used to verify certificates or for manual client configuration.
                </p>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Rotate CA</h4>
                <p className="text-sm text-theme-secondary">
                  Generate a new CA certificate and key. <strong className="text-red-600">Warning:</strong> This will invalidate all existing certificates. All gateways will need to reprovision and all active VPN connections will be terminated.
                </p>
              </div>
              <div className="p-4 border border-theme rounded-lg">
                <h4 className="font-medium text-theme-primary mb-2">Import CA</h4>
                <p className="text-sm text-theme-secondary">
                  Import an existing CA certificate and private key. Useful for migrating from another PKI system or using an enterprise CA.
                </p>
              </div>
            </div>

            {/* Best Practices */}
            <h3 className="text-lg font-medium text-theme-primary mt-6 mb-3">Best Practices</h3>
            <div className="info-box">
              <ul className="space-y-2 info-box-text">
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
            <h2 className="text-xl font-semibold text-theme-primary mb-4">Permission Flow</h2>
            <div className="bg-theme-tertiary rounded-lg p-6 font-mono text-sm overflow-x-auto">
              <pre className="text-theme-secondary">{`User requests VPN connection
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
