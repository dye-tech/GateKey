import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { api, getCA, rotateCA, updateCA, CAInfo } from '../api/client'

interface OIDCProvider {
  name: string
  display_name: string
  issuer: string
  client_id: string
  client_secret?: string
  redirect_url: string
  scopes: string[]
  admin_group?: string
  enabled: boolean
}

interface SAMLProvider {
  name: string
  display_name: string
  idp_metadata_url: string
  entity_id: string
  acs_url: string
  admin_group?: string
  enabled: boolean
}

type TabType = 'oidc' | 'saml' | 'general' | 'ca'

const tabTitles: Record<TabType, string> = {
  oidc: 'OIDC Providers',
  saml: 'SAML Providers',
  general: 'General Settings',
  ca: 'Certificate Authority',
}

export default function AdminSettings() {
  const { tab } = useParams<{ tab: string }>()
  const activeTab = (tab as TabType) || 'oidc'

  const [oidcProviders, setOidcProviders] = useState<OIDCProvider[]>([])
  const [samlProviders, setSamlProviders] = useState<SAMLProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // OIDC form state
  const [editingOidc, setEditingOidc] = useState<OIDCProvider | null>(null)
  const [showOidcForm, setShowOidcForm] = useState(false)

  // SAML form state
  const [editingSaml, setEditingSaml] = useState<SAMLProvider | null>(null)
  const [showSamlForm, setShowSamlForm] = useState(false)

  useEffect(() => {
    loadSettings()
  }, [])

  async function loadSettings() {
    try {
      setLoading(true)
      const [oidcRes, samlRes] = await Promise.all([
        api.get('/api/v1/admin/settings/oidc'),
        api.get('/api/v1/admin/settings/saml')
      ])
      setOidcProviders(oidcRes.data.providers || [])
      setSamlProviders(samlRes.data.providers || [])
    } catch (err) {
      setError('Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  async function saveOidcProvider(provider: OIDCProvider) {
    try {
      setSaving(true)
      setError(null)

      if (editingOidc && oidcProviders.some(p => p.name === editingOidc.name)) {
        await api.put(`/api/v1/admin/settings/oidc/${editingOidc.name}`, provider)
      } else {
        await api.post('/api/v1/admin/settings/oidc', provider)
      }

      setSuccess('OIDC provider saved successfully')
      setShowOidcForm(false)
      setEditingOidc(null)
      await loadSettings()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to save OIDC provider')
    } finally {
      setSaving(false)
    }
  }

  async function deleteOidcProvider(name: string) {
    if (!confirm(`Are you sure you want to delete the OIDC provider "${name}"?`)) return

    try {
      await api.delete(`/api/v1/admin/settings/oidc/${name}`)
      setSuccess('OIDC provider deleted')
      await loadSettings()
    } catch (err) {
      setError('Failed to delete OIDC provider')
    }
  }

  async function saveSamlProvider(provider: SAMLProvider) {
    try {
      setSaving(true)
      setError(null)

      if (editingSaml && samlProviders.some(p => p.name === editingSaml.name)) {
        await api.put(`/api/v1/admin/settings/saml/${editingSaml.name}`, provider)
      } else {
        await api.post('/api/v1/admin/settings/saml', provider)
      }

      setSuccess('SAML provider saved successfully')
      setShowSamlForm(false)
      setEditingSaml(null)
      await loadSettings()
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to save SAML provider')
    } finally {
      setSaving(false)
    }
  }

  async function deleteSamlProvider(name: string) {
    if (!confirm(`Are you sure you want to delete the SAML provider "${name}"?`)) return

    try {
      await api.delete(`/api/v1/admin/settings/saml/${name}`)
      setSuccess('SAML provider deleted')
      await loadSettings()
    } catch (err) {
      setError('Failed to delete SAML provider')
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">{tabTitles[activeTab]}</h1>
        <p className="text-gray-500 mt-1">Configure authentication providers and system settings</p>
      </div>

      {/* Alerts */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 flex justify-between items-center">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-500 hover:text-red-700">&times;</button>
        </div>
      )}
      {success && (
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg text-green-700 flex justify-between items-center">
          <span>{success}</span>
          <button onClick={() => setSuccess(null)} className="text-green-500 hover:text-green-700">&times;</button>
        </div>
      )}

      {/* Tab Content */}
      <div className="bg-white rounded-lg shadow p-6">
        {activeTab === 'oidc' && (
          <OIDCTab
            providers={oidcProviders}
            showForm={showOidcForm}
            editing={editingOidc}
            saving={saving}
            onAdd={() => { setEditingOidc(null); setShowOidcForm(true); }}
            onEdit={(p) => { setEditingOidc(p); setShowOidcForm(true); }}
            onDelete={deleteOidcProvider}
            onSave={saveOidcProvider}
            onCancel={() => { setShowOidcForm(false); setEditingOidc(null); }}
          />
        )}
        {activeTab === 'saml' && (
          <SAMLTab
            providers={samlProviders}
            showForm={showSamlForm}
            editing={editingSaml}
            saving={saving}
            onAdd={() => { setEditingSaml(null); setShowSamlForm(true); }}
            onEdit={(p) => { setEditingSaml(p); setShowSamlForm(true); }}
            onDelete={deleteSamlProvider}
            onSave={saveSamlProvider}
            onCancel={() => { setShowSamlForm(false); setEditingSaml(null); }}
          />
        )}
        {activeTab === 'general' && <GeneralTab />}
        {activeTab === 'ca' && <CATab />}
      </div>
    </div>
  )
}

// OIDC Tab Component
interface OIDCTabProps {
  providers: OIDCProvider[]
  showForm: boolean
  editing: OIDCProvider | null
  saving: boolean
  onAdd: () => void
  onEdit: (provider: OIDCProvider) => void
  onDelete: (name: string) => void
  onSave: (provider: OIDCProvider) => void
  onCancel: () => void
}

function OIDCTab({ providers, showForm, editing, saving, onAdd, onEdit, onDelete, onSave, onCancel }: OIDCTabProps) {
  const [form, setForm] = useState<OIDCProvider>({
    name: '',
    display_name: '',
    issuer: '',
    client_id: '',
    client_secret: '',
    redirect_url: '',
    scopes: ['openid', 'profile', 'email'],
    admin_group: '',
    enabled: true
  })

  useEffect(() => {
    if (editing) {
      setForm({ ...editing, client_secret: '' })
    } else {
      setForm({
        name: '',
        display_name: '',
        issuer: '',
        client_id: '',
        client_secret: '',
        redirect_url: window.location.origin + '/api/v1/auth/oidc/callback',
        scopes: ['openid', 'profile', 'email'],
        admin_group: '',
        enabled: true
      })
    }
  }, [editing, showForm])

  if (showForm) {
    return (
      <div className="space-y-6">
        <h3 className="text-lg font-medium">{editing ? 'Edit OIDC Provider' : 'Add OIDC Provider'}</h3>
        <form onSubmit={(e) => { e.preventDefault(); onSave(form); }} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Provider Name (unique ID)</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '') })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                placeholder="keycloak"
                required
                disabled={!!editing}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Display Name</label>
              <input
                type="text"
                value={form.display_name}
                onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                placeholder="Sign in with Keycloak"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Issuer URL</label>
            <input
              type="url"
              value={form.issuer}
              onChange={(e) => setForm({ ...form, issuer: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder="https://keycloak.example.com/realms/myrealm"
              required
            />
            <p className="text-xs text-gray-500 mt-1">The OIDC issuer URL (usually ends with /realms/name for Keycloak)</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Client ID</label>
              <input
                type="text"
                value={form.client_id}
                onChange={(e) => setForm({ ...form, client_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                placeholder="gatekey"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Client Secret</label>
              <input
                type="password"
                value={form.client_secret}
                onChange={(e) => setForm({ ...form, client_secret: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                placeholder={editing ? '(unchanged if empty)' : 'Enter client secret'}
                required={!editing}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Redirect URL</label>
            <input
              type="url"
              value={form.redirect_url}
              onChange={(e) => setForm({ ...form, redirect_url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder={window.location.origin + '/api/v1/auth/oidc/callback'}
              required
            />
            <p className="text-xs text-gray-500 mt-1">Configure this URL in your OIDC provider as an allowed redirect URI</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Scopes</label>
            <input
              type="text"
              value={form.scopes.join(' ')}
              onChange={(e) => setForm({ ...form, scopes: e.target.value.split(' ').filter(s => s) })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder="openid profile email"
            />
            <p className="text-xs text-gray-500 mt-1">Space-separated list of OIDC scopes</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Admin Group (optional)</label>
            <input
              type="text"
              value={form.admin_group || ''}
              onChange={(e) => setForm({ ...form, admin_group: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder="gatekey-admins"
            />
            <p className="text-xs text-gray-500 mt-1">Users in this group will be granted admin access. Leave empty to disable.</p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="oidc-enabled"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="oidc-enabled" className="ml-2 text-sm text-gray-700">Enable this provider</label>
          </div>

          <div className="flex justify-end space-x-3 pt-4 border-t">
            <button type="button" onClick={onCancel} className="btn btn-secondary">Cancel</button>
            <button type="submit" disabled={saving} className="btn btn-primary">
              {saving ? 'Saving...' : 'Save Provider'}
            </button>
          </div>
        </form>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-medium">OIDC Providers</h3>
          <p className="text-sm text-gray-500">Configure OpenID Connect providers for single sign-on</p>
        </div>
        <button onClick={onAdd} className="btn btn-primary">
          + Add OIDC Provider
        </button>
      </div>

      {providers.length === 0 ? (
        <div className="text-center py-8 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          <p className="mt-2 text-gray-500">No OIDC providers configured</p>
          <button onClick={onAdd} className="mt-2 text-primary-600 hover:text-primary-700">
            Add your first OIDC provider
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {providers.map((provider) => (
            <div key={provider.name} className="border rounded-lg p-4 flex justify-between items-center">
              <div>
                <div className="flex items-center space-x-2">
                  <h4 className="font-medium">{provider.display_name}</h4>
                  <span className={`px-2 py-0.5 text-xs rounded-full ${provider.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}`}>
                    {provider.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
                <p className="text-sm text-gray-500 mt-1">{provider.issuer}</p>
                <p className="text-xs text-gray-400">Client ID: {provider.client_id}</p>
                {provider.admin_group && (
                  <p className="text-xs text-orange-600 mt-1">Admin group: {provider.admin_group}</p>
                )}
              </div>
              <div className="flex space-x-2">
                <button onClick={() => onEdit(provider)} className="btn btn-secondary text-sm inline-flex items-center">
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                  Edit
                </button>
                <button onClick={() => onDelete(provider.name)} className="btn text-sm text-red-600 bg-red-50 border border-red-200 hover:bg-red-100 inline-flex items-center">
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// SAML Tab Component
interface SAMLTabProps {
  providers: SAMLProvider[]
  showForm: boolean
  editing: SAMLProvider | null
  saving: boolean
  onAdd: () => void
  onEdit: (provider: SAMLProvider) => void
  onDelete: (name: string) => void
  onSave: (provider: SAMLProvider) => void
  onCancel: () => void
}

function SAMLTab({ providers, showForm, editing, saving, onAdd, onEdit, onDelete, onSave, onCancel }: SAMLTabProps) {
  const [form, setForm] = useState<SAMLProvider>({
    name: '',
    display_name: '',
    idp_metadata_url: '',
    entity_id: '',
    acs_url: '',
    admin_group: '',
    enabled: true
  })

  useEffect(() => {
    if (editing) {
      setForm(editing)
    } else {
      setForm({
        name: '',
        display_name: '',
        idp_metadata_url: '',
        entity_id: window.location.origin,
        acs_url: window.location.origin + '/api/v1/auth/saml/acs',
        admin_group: '',
        enabled: true
      })
    }
  }, [editing, showForm])

  if (showForm) {
    return (
      <div className="space-y-6">
        <h3 className="text-lg font-medium">{editing ? 'Edit SAML Provider' : 'Add SAML Provider'}</h3>
        <form onSubmit={(e) => { e.preventDefault(); onSave(form); }} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Provider Name (unique ID)</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '') })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                placeholder="okta"
                required
                disabled={!!editing}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Display Name</label>
              <input
                type="text"
                value={form.display_name}
                onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                placeholder="Sign in with Okta"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">IdP Metadata URL</label>
            <input
              type="url"
              value={form.idp_metadata_url}
              onChange={(e) => setForm({ ...form, idp_metadata_url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder="https://okta.example.com/app/xxx/sso/saml/metadata"
              required
            />
            <p className="text-xs text-gray-500 mt-1">URL to the IdP's SAML metadata XML</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Entity ID (SP)</label>
            <input
              type="text"
              value={form.entity_id}
              onChange={(e) => setForm({ ...form, entity_id: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder={window.location.origin}
              required
            />
            <p className="text-xs text-gray-500 mt-1">The Service Provider entity ID (usually your application's URL)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">ACS URL</label>
            <input
              type="url"
              value={form.acs_url}
              onChange={(e) => setForm({ ...form, acs_url: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder={window.location.origin + '/api/v1/auth/saml/acs'}
              required
            />
            <p className="text-xs text-gray-500 mt-1">Assertion Consumer Service URL - configure this in your IdP</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Admin Group (optional)</label>
            <input
              type="text"
              value={form.admin_group || ''}
              onChange={(e) => setForm({ ...form, admin_group: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              placeholder="gatekey-admins"
            />
            <p className="text-xs text-gray-500 mt-1">Users in this group will be granted admin access. Leave empty to disable.</p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="saml-enabled"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label htmlFor="saml-enabled" className="ml-2 text-sm text-gray-700">Enable this provider</label>
          </div>

          <div className="flex justify-end space-x-3 pt-4 border-t">
            <button type="button" onClick={onCancel} className="btn btn-secondary">Cancel</button>
            <button type="submit" disabled={saving} className="btn btn-primary">
              {saving ? 'Saving...' : 'Save Provider'}
            </button>
          </div>
        </form>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-medium">SAML Providers</h3>
          <p className="text-sm text-gray-500">Configure SAML 2.0 identity providers for single sign-on</p>
        </div>
        <button onClick={onAdd} className="btn btn-primary">
          + Add SAML Provider
        </button>
      </div>

      {providers.length === 0 ? (
        <div className="text-center py-8 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
          </svg>
          <p className="mt-2 text-gray-500">No SAML providers configured</p>
          <button onClick={onAdd} className="mt-2 text-primary-600 hover:text-primary-700">
            Add your first SAML provider
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {providers.map((provider) => (
            <div key={provider.name} className="border rounded-lg p-4 flex justify-between items-center">
              <div>
                <div className="flex items-center space-x-2">
                  <h4 className="font-medium">{provider.display_name}</h4>
                  <span className={`px-2 py-0.5 text-xs rounded-full ${provider.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}`}>
                    {provider.enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
                <p className="text-sm text-gray-500 mt-1">{provider.idp_metadata_url}</p>
                <p className="text-xs text-gray-400">Entity ID: {provider.entity_id}</p>
                {provider.admin_group && (
                  <p className="text-xs text-orange-600 mt-1">Admin group: {provider.admin_group}</p>
                )}
              </div>
              <div className="flex space-x-2">
                <button onClick={() => onEdit(provider)} className="btn btn-secondary text-sm inline-flex items-center">
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                  Edit
                </button>
                <button onClick={() => onDelete(provider.name)} className="btn text-sm text-red-600 bg-red-50 border border-red-200 hover:bg-red-100 inline-flex items-center">
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// CA Tab Component
function CATab() {
  const [caInfo, setCaInfo] = useState<CAInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [rotating, setRotating] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [showUpload, setShowUpload] = useState(false)
  const [certPem, setCertPem] = useState('')
  const [keyPem, setKeyPem] = useState('')

  useEffect(() => {
    loadCA()
  }, [])

  async function loadCA() {
    try {
      setLoading(true)
      setError(null)
      const data = await getCA()
      setCaInfo(data)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to load CA information')
    } finally {
      setLoading(false)
    }
  }

  async function handleRotate() {
    if (!confirm('Are you sure you want to rotate the CA? This will invalidate all existing certificates. Gateways will need to be re-provisioned and users will need new VPN configs.')) {
      return
    }

    try {
      setRotating(true)
      setError(null)
      const data = await rotateCA()
      setCaInfo(data)
      setSuccess('CA rotated successfully. Please re-provision all gateways.')
      setTimeout(() => setSuccess(null), 10000)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to rotate CA')
    } finally {
      setRotating(false)
    }
  }

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault()
    if (!confirm('Are you sure you want to replace the CA? This will invalidate all existing certificates. Gateways will need to be re-provisioned and users will need new VPN configs.')) {
      return
    }

    try {
      setUploading(true)
      setError(null)
      const data = await updateCA({ certificate: certPem, private_key: keyPem })
      setCaInfo(data)
      setSuccess('CA updated successfully. Please re-provision all gateways.')
      setShowUpload(false)
      setCertPem('')
      setKeyPem('')
      setTimeout(() => setSuccess(null), 10000)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error || 'Failed to update CA')
    } finally {
      setUploading(false)
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  function downloadCertificate() {
    if (!caInfo) return
    const blob = new Blob([caInfo.certificate], { type: 'application/x-pem-file' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'gatekey-ca.crt'
    a.click()
    URL.revokeObjectURL(url)
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Certificate Authority</h3>
        <p className="text-sm text-gray-500">Manage the CA used to sign VPN certificates</p>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm flex justify-between items-center">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-500 hover:text-red-700">&times;</button>
        </div>
      )}
      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm flex justify-between items-center">
          <span>{success}</span>
          <button onClick={() => setSuccess(null)} className="text-green-500 hover:text-green-700">&times;</button>
        </div>
      )}

      {caInfo ? (
        <div className="space-y-6">
          {/* Current CA Info */}
          <div className="border rounded-lg p-4">
            <h4 className="font-medium mb-4">Current CA Certificate</h4>
            <div className="grid gap-3 text-sm">
              <div className="flex">
                <span className="text-gray-500 w-32">Subject:</span>
                <span className="font-mono">{caInfo.subject}</span>
              </div>
              <div className="flex">
                <span className="text-gray-500 w-32">Serial Number:</span>
                <span className="font-mono uppercase">{caInfo.serial_number}</span>
              </div>
              <div className="flex">
                <span className="text-gray-500 w-32">Valid From:</span>
                <span>{formatDate(caInfo.not_before)}</span>
              </div>
              <div className="flex">
                <span className="text-gray-500 w-32">Valid Until:</span>
                <span>{formatDate(caInfo.not_after)}</span>
              </div>
              <div className="flex">
                <span className="text-gray-500 w-32">Fingerprint:</span>
                <span className="font-mono text-xs break-all">{caInfo.fingerprint}</span>
              </div>
            </div>
            <div className="mt-4 pt-4 border-t">
              <button onClick={downloadCertificate} className="btn btn-secondary text-sm inline-flex items-center">
                <svg className="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                Download CA Certificate
              </button>
            </div>
          </div>

          {/* Actions */}
          <div className="border rounded-lg p-4">
            <h4 className="font-medium mb-4">CA Management</h4>
            <div className="space-y-4">
              <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div>
                  <p className="font-medium">Generate New CA</p>
                  <p className="text-sm text-gray-500">Create a new self-signed CA certificate</p>
                </div>
                <button
                  onClick={handleRotate}
                  disabled={rotating}
                  className="btn btn-primary inline-flex items-center"
                >
                  <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  {rotating ? 'Rotating...' : 'Rotate CA'}
                </button>
              </div>
              <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div>
                  <p className="font-medium">Import Custom CA</p>
                  <p className="text-sm text-gray-500">Use your own CA certificate and private key</p>
                </div>
                <button
                  onClick={() => setShowUpload(!showUpload)}
                  className="btn btn-secondary inline-flex items-center"
                >
                  <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                  </svg>
                  {showUpload ? 'Cancel' : 'Import CA'}
                </button>
              </div>
            </div>
          </div>

          {/* Upload Form */}
          {showUpload && (
            <div className="border rounded-lg p-4 border-primary-200 bg-primary-50">
              <h4 className="font-medium mb-4">Import Custom CA</h4>
              <form onSubmit={handleUpload} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">CA Certificate (PEM)</label>
                  <textarea
                    value={certPem}
                    onChange={(e) => setCertPem(e.target.value)}
                    rows={8}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-xs focus:ring-2 focus:ring-primary-500"
                    placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">CA Private Key (PEM)</label>
                  <textarea
                    value={keyPem}
                    onChange={(e) => setKeyPem(e.target.value)}
                    rows={8}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-xs focus:ring-2 focus:ring-primary-500"
                    placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">Supported formats: PKCS#8, RSA, EC private key</p>
                </div>
                <div className="flex justify-end space-x-3 pt-4 border-t">
                  <button type="button" onClick={() => { setShowUpload(false); setCertPem(''); setKeyPem(''); }} className="btn btn-secondary">
                    Cancel
                  </button>
                  <button type="submit" disabled={uploading} className="btn btn-primary">
                    {uploading ? 'Uploading...' : 'Import CA'}
                  </button>
                </div>
              </form>
            </div>
          )}

          {/* Warning */}
          <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
            <p className="text-yellow-800 text-sm">
              <strong>Warning:</strong> Changing the CA will invalidate all existing VPN certificates.
              After rotating or importing a new CA, you must re-provision all gateways and users will need to download new VPN configurations.
            </p>
          </div>
        </div>
      ) : (
        <div className="text-center py-8 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
          <p className="text-gray-500">CA not initialized</p>
        </div>
      )}
    </div>
  )
}

// General Settings Tab
function GeneralTab() {
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    loadSettings()
  }, [])

  async function loadSettings() {
    try {
      setLoading(true)
      const res = await api.get('/api/v1/admin/settings')
      setSettings(res.data.settings || {})
    } catch (err) {
      setError('Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  async function saveSettings() {
    try {
      setSaving(true)
      setError(null)
      await api.put('/api/v1/admin/settings', settings)
      setSuccess('Settings saved successfully')
      setTimeout(() => setSuccess(null), 3000)
    } catch (err) {
      setError('Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">General Settings</h3>
        <p className="text-sm text-gray-500">Configure general system settings</p>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {error}
        </div>
      )}
      {success && (
        <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
          {success}
        </div>
      )}

      <div className="grid gap-6">
        <div className="border rounded-lg p-4">
          <h4 className="font-medium mb-4">Session Settings</h4>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Session Duration (hours)</label>
              <input
                type="number"
                value={settings.session_duration_hours || '12'}
                onChange={(e) => setSettings({ ...settings, session_duration_hours: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                min="1"
                max="168"
              />
              <p className="text-xs text-gray-500 mt-1">How long user sessions remain valid</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">VPN Certificate Validity (hours)</label>
              <input
                type="number"
                value={settings.vpn_cert_validity_hours || '24'}
                onChange={(e) => setSettings({ ...settings, vpn_cert_validity_hours: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
                min="1"
                max="168"
              />
              <p className="text-xs text-gray-500 mt-1">How long VPN certificates remain valid</p>
            </div>
          </div>
          <div className="mt-4">
            <div className="flex items-center">
              <input
                type="checkbox"
                id="secure-cookies"
                checked={settings.secure_cookies === 'true'}
                onChange={(e) => setSettings({ ...settings, secure_cookies: e.target.checked ? 'true' : 'false' })}
                className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
              />
              <label htmlFor="secure-cookies" className="ml-2 text-sm text-gray-700">
                Secure Cookies (HTTPS only)
              </label>
            </div>
            <p className="text-xs text-gray-500 mt-1 ml-6">Enable only if using HTTPS</p>
          </div>
        </div>

        <div className="border rounded-lg p-4">
          <h4 className="font-medium mb-4">Security Settings</h4>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    id="require-fips"
                    checked={settings.require_fips === 'true'}
                    onChange={(e) => setSettings({ ...settings, require_fips: e.target.checked ? 'true' : 'false' })}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                  />
                  <label htmlFor="require-fips" className="ml-2 font-medium text-gray-700">
                    Require FIPS 140-3 Compliance
                  </label>
                </div>
                <p className="text-sm text-gray-500 mt-1 ml-6">
                  When enabled, clients must pass FIPS compliance checks before connecting.
                  This enforces the use of FIPS 140-3 validated cryptographic modules.
                </p>
              </div>
            </div>
            {settings.require_fips === 'true' && (
              <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p className="text-sm text-yellow-800">
                  <strong>Note:</strong> Enabling FIPS requirement will block connections from clients
                  that don't have FIPS mode enabled. Make sure your users' systems are FIPS-compliant
                  before enabling this setting.
                </p>
              </div>
            )}
          </div>
          <div className="mt-4 pt-4 border-t">
            <button
              onClick={saveSettings}
              disabled={saving}
              className="btn btn-primary inline-flex items-center"
            >
              <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
              </svg>
              {saving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </div>

        <div className="border rounded-lg p-4">
          <h4 className="font-medium mb-4">OpenVPN Encryption Settings</h4>
          <p className="text-sm text-gray-500 mb-4">Control which encryption profiles are allowed for VPN gateways</p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Allowed Crypto Profiles</label>
              <p className="text-xs text-gray-500 mb-3">Select which encryption profiles gateways can use. At least one must be selected.</p>
              <div className="space-y-2">
                {[
                  { id: 'modern', name: 'Modern', desc: 'AES-256-GCM, CHACHA20-POLY1305, TLS 1.2+ (Recommended)' },
                  { id: 'fips', name: 'FIPS 140-3', desc: 'AES-256-GCM only, SHA-384, TLS 1.2+ (Government compliance)' },
                  { id: 'compatible', name: 'Compatible', desc: 'AES-256-CBC fallback, TLS 1.0+ (Legacy support)' },
                ].map((profile) => {
                  const allowedProfiles = (settings.allowed_crypto_profiles || 'modern,fips,compatible').split(',')
                  const isChecked = allowedProfiles.includes(profile.id)
                  return (
                    <label key={profile.id} className="flex items-start p-3 bg-gray-50 rounded-lg cursor-pointer hover:bg-gray-100">
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={(e) => {
                          let profiles = allowedProfiles.filter(p => p.trim() !== '')
                          if (e.target.checked) {
                            if (!profiles.includes(profile.id)) {
                              profiles.push(profile.id)
                            }
                          } else {
                            profiles = profiles.filter(p => p !== profile.id)
                          }
                          // Ensure at least one profile is selected
                          if (profiles.length === 0) profiles = ['modern']
                          setSettings({ ...settings, allowed_crypto_profiles: profiles.join(',') })
                        }}
                        className="h-4 w-4 mt-0.5 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                      />
                      <div className="ml-3">
                        <span className="font-medium text-gray-900">{profile.name}</span>
                        <p className="text-xs text-gray-500">{profile.desc}</p>
                      </div>
                    </label>
                  )
                })}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Minimum TLS Version</label>
              <select
                value={settings.min_tls_version || '1.2'}
                onChange={(e) => setSettings({ ...settings, min_tls_version: e.target.value })}
                className="w-full max-w-xs px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500"
              >
                <option value="1.0">TLS 1.0 (Legacy)</option>
                <option value="1.1">TLS 1.1</option>
                <option value="1.2">TLS 1.2 (Recommended)</option>
                <option value="1.3">TLS 1.3 (Most Secure)</option>
              </select>
              <p className="text-xs text-gray-500 mt-1">Minimum TLS version for VPN connections</p>
            </div>

            {(settings.allowed_crypto_profiles && !settings.allowed_crypto_profiles.includes('modern')) && (
              <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p className="text-sm text-yellow-800">
                  <strong>Warning:</strong> The "Modern" profile is recommended for best security. Disabling it may reduce overall security.
                </p>
              </div>
            )}

            {settings.allowed_crypto_profiles === 'compatible' && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                <p className="text-sm text-red-800">
                  <strong>Security Risk:</strong> Only allowing the "Compatible" profile means VPN connections may use weaker encryption (AES-CBC). Consider enabling "Modern" or "FIPS" profiles for better security.
                </p>
              </div>
            )}
          </div>

          <div className="mt-4 pt-4 border-t">
            <button
              onClick={saveSettings}
              disabled={saving}
              className="btn btn-primary inline-flex items-center"
            >
              <svg className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
              </svg>
              {saving ? 'Saving...' : 'Save Encryption Settings'}
            </button>
          </div>
        </div>

        <div className="border rounded-lg p-4">
          <h4 className="font-medium mb-2">Service Provider Metadata</h4>
          <p className="text-sm text-gray-500 mb-3">Use these values when configuring your identity provider</p>
          <div className="space-y-2 font-mono text-sm bg-gray-50 p-3 rounded">
            <div>
              <span className="text-gray-500">Entity ID:</span>
              <span className="ml-2">{window.location.origin}</span>
            </div>
            <div>
              <span className="text-gray-500">OIDC Callback:</span>
              <span className="ml-2">{window.location.origin}/api/v1/auth/oidc/callback</span>
            </div>
            <div>
              <span className="text-gray-500">SAML ACS:</span>
              <span className="ml-2">{window.location.origin}/api/v1/auth/saml/acs</span>
            </div>
            <div>
              <span className="text-gray-500">SAML Metadata:</span>
              <span className="ml-2">{window.location.origin}/api/v1/auth/saml/metadata</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
