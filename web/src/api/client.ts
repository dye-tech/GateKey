import axios from 'axios'

export const api = axios.create({
  baseURL: '',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Helper to read a cookie value by name
function getCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'))
  return match ? decodeURIComponent(match[2]) : null
}

// Add request interceptor to include CSRF token for state-changing requests
api.interceptors.request.use(
  (config) => {
    const method = config.method?.toUpperCase()
    // For POST, PUT, DELETE, PATCH - include CSRF token
    if (method && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
      const csrfToken = getCookie('gatekey_csrf')
      console.debug('CSRF token for request:', csrfToken ? 'found' : 'NOT FOUND', 'cookies:', document.cookie)
      if (csrfToken) {
        config.headers['X-CSRF-Token'] = csrfToken
      }
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Add response interceptor for handling auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Redirect to login if unauthorized
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// API types
export interface Gateway {
  id: string
  name: string
  hostname: string
  publicIp: string
  vpnPort: number
  vpnProtocol: string
  isActive: boolean
  lastHeartbeat: string | null
  gatewayType?: GatewayType
  wgPublicKey?: string
}

export interface AuthProvider {
  type: string
  name: string
  displayName: string
  loginUrl: string
}

export interface GeneratedConfig {
  id: string
  fileName: string
  gatewayName: string
  expiresAt: string
  downloadUrl: string
  cliCallback: boolean
}

export interface LocalLoginResponse {
  user: {
    username: string
    email: string
    is_admin: boolean
  }
  token: string
}

// API functions
export async function getProviders(): Promise<AuthProvider[]> {
  const response = await api.get('/api/v1/auth/providers')
  // Map snake_case from API to camelCase for frontend
  return (response.data.providers || []).map((p: Record<string, string>) => ({
    type: p.type,
    name: p.name,
    displayName: p.display_name,
    loginUrl: p.login_url,
  }))
}

export async function localLogin(username: string, password: string): Promise<LocalLoginResponse> {
  const response = await api.post('/api/v1/auth/local/login', { username, password })
  return response.data
}

export async function getGateways(): Promise<Gateway[]> {
  const response = await api.get('/api/v1/gateways')
  return response.data.gateways || []
}

export async function generateConfig(gatewayId: string, cliCallbackUrl?: string): Promise<GeneratedConfig> {
  const response = await api.post('/api/v1/configs/generate', {
    gateway_id: gatewayId,
    cli_callback_url: cliCallbackUrl
  })
  return response.data
}

export async function downloadConfig(configId: string): Promise<Blob> {
  const response = await api.get(`/api/v1/configs/download/${configId}`, {
    responseType: 'blob',
  })
  return response.data
}

// Admin Gateway API
export type CryptoProfile = 'modern' | 'fips' | 'compatible'
export type GatewayType = 'openvpn' | 'wireguard'

export interface AdminGateway {
  id: string
  name: string
  hostname: string
  publicIp: string
  publicIpv6?: string
  vpnPort: number
  vpnProtocol: string
  cryptoProfile: CryptoProfile
  vpnSubnet: string
  vpnSubnetV6?: string
  tlsAuthEnabled: boolean
  fullTunnelMode: boolean
  pushDns: boolean
  dnsServers: string[]
  sessionEnabled: boolean
  isActive: boolean
  lastHeartbeat: string | null
  createdAt: string
  updatedAt: string
  // Gateway type (openvpn or wireguard)
  gatewayType: GatewayType
  // WireGuard-specific fields
  wgPublicKey?: string
  wgListenPort?: number
  // FIPS enforcement
  enforceFipsMode: boolean
}

export interface RegisterGatewayRequest {
  name: string
  hostname?: string
  public_ip?: string
  public_ip_v6?: string
  vpn_port?: number
  vpn_protocol?: string
  crypto_profile?: CryptoProfile
  vpn_subnet?: string
  vpn_subnet_v6?: string
  tls_auth_enabled?: boolean
  full_tunnel_mode?: boolean
  push_dns?: boolean
  dns_servers?: string[]
  session_enabled?: boolean
  // Gateway type (openvpn or wireguard)
  gateway_type?: GatewayType
  // WireGuard-specific fields
  wg_listen_port?: number
  // FIPS enforcement
  enforce_fips_mode?: boolean
}

export interface RegisterGatewayResponse {
  id: string
  name: string
  hostname: string
  vpnPort: number
  vpnProtocol: string
  token: string
  message: string
  gatewayType?: GatewayType
  wgPublicKey?: string
  wgListenPort?: number
}

export async function getAdminGateways(): Promise<AdminGateway[]> {
  const response = await api.get('/api/v1/admin/gateways')
  return response.data.gateways || []
}

export async function registerGateway(req: RegisterGatewayRequest): Promise<RegisterGatewayResponse> {
  const response = await api.post('/api/v1/admin/gateways', req)
  return response.data
}

export async function deleteGateway(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/gateways/${id}`)
}

export async function reprovisionGateway(id: string): Promise<{ message: string; configVersion: string }> {
  const response = await api.post(`/api/v1/admin/gateways/${id}/reprovision`)
  return response.data
}

export interface RotateTokenResponse {
  message: string
  token: string
  installScript: string
  gatewayName?: string
  hubName?: string
  spokeName?: string
}

export async function rotateGatewayToken(id: string): Promise<RotateTokenResponse> {
  const response = await api.post(`/api/v1/admin/gateways/${id}/rotate-token`)
  return response.data
}

export interface UpdateGatewayRequest {
  name: string
  hostname?: string
  public_ip?: string
  public_ip_v6?: string
  vpn_port?: number
  vpn_protocol?: string
  crypto_profile?: CryptoProfile
  vpn_subnet?: string
  vpn_subnet_v6?: string
  tls_auth_enabled?: boolean
  full_tunnel_mode?: boolean
  push_dns?: boolean
  dns_servers?: string[]
  session_enabled?: boolean
  enforce_fips_mode?: boolean
}

export async function updateGateway(id: string, req: UpdateGatewayRequest): Promise<void> {
  await api.put(`/api/v1/admin/gateways/${id}`, req)
}

// Gateway User/Group Assignment API
export interface GatewayUser {
  userId: string
  email: string
  name: string
  createdAt: string
}

export interface GatewayGroup {
  groupName: string
  createdAt: string
}

export async function getGatewayUsers(gatewayId: string): Promise<GatewayUser[]> {
  const response = await api.get(`/api/v1/admin/gateways/${gatewayId}/users`)
  return response.data.users || []
}

export async function assignUserToGateway(gatewayId: string, userId: string): Promise<void> {
  await api.post(`/api/v1/admin/gateways/${gatewayId}/users`, { user_id: userId })
}

export async function removeUserFromGateway(gatewayId: string, userId: string): Promise<void> {
  await api.delete(`/api/v1/admin/gateways/${gatewayId}/users/${userId}`)
}

export async function getGatewayGroups(gatewayId: string): Promise<GatewayGroup[]> {
  const response = await api.get(`/api/v1/admin/gateways/${gatewayId}/groups`)
  return response.data.groups || []
}

export async function assignGroupToGateway(gatewayId: string, groupName: string): Promise<void> {
  await api.post(`/api/v1/admin/gateways/${gatewayId}/groups`, { group_name: groupName })
}

export async function removeGroupFromGateway(gatewayId: string, groupName: string): Promise<void> {
  await api.delete(`/api/v1/admin/gateways/${gatewayId}/groups/${groupName}`)
}

// Network API
export interface Network {
  id: string
  name: string
  description: string
  cidr: string
  cidrV6?: string // IPv6 CIDR (optional)
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateNetworkRequest {
  name: string
  description?: string
  cidr?: string // IPv4 CIDR (optional if IPv6 provided)
  cidrV6?: string // IPv6 CIDR (optional if IPv4 provided)
  is_active?: boolean
}

export async function getNetworks(): Promise<Network[]> {
  const response = await api.get('/api/v1/admin/networks')
  return response.data.networks || []
}

export async function createNetwork(req: CreateNetworkRequest): Promise<Network> {
  const response = await api.post('/api/v1/admin/networks', req)
  return response.data
}

export async function updateNetwork(id: string, req: CreateNetworkRequest): Promise<void> {
  await api.put(`/api/v1/admin/networks/${id}`, req)
}

export async function deleteNetwork(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/networks/${id}`)
}

export async function getNetworkGateways(networkId: string): Promise<Gateway[]> {
  const response = await api.get(`/api/v1/admin/networks/${networkId}/gateways`)
  return response.data.gateways || []
}

export interface NetworkAccessRule {
  id: string
  name: string
  description: string
  ruleType: AccessRuleType
  value: string
  valueV6?: string // IPv6 IP or CIDR (optional)
  portRange?: string
  protocol?: string
  networkId?: string
  isActive: boolean
  users: string[]
  groups: string[]
}

export async function getNetworkAccessRules(networkId: string): Promise<NetworkAccessRule[]> {
  const response = await api.get(`/api/v1/admin/networks/${networkId}/access-rules`)
  return (response.data.access_rules || []).map((r: Record<string, unknown>) => ({
    id: r.id,
    name: r.name,
    description: r.description || '',
    ruleType: r.rule_type as AccessRuleType,
    value: r.value,
    portRange: r.port_range,
    protocol: r.protocol,
    networkId: r.network_id,
    isActive: r.is_active,
    users: r.users || [],
    groups: r.groups || [],
  }))
}

// Network Mesh Hub associations
export interface NetworkMeshHub {
  id: string
  name: string
  description: string
  gatewayType: GatewayType
  publicEndpoint: string
  status: MeshHubStatus
}

export async function getNetworkMeshHubs(networkId: string): Promise<NetworkMeshHub[]> {
  const response = await api.get(`/api/v1/admin/networks/${networkId}/mesh-hubs`)
  return (response.data.hubs || []).map((h: Record<string, unknown>) => ({
    id: h.id,
    name: h.name,
    description: h.description || '',
    gatewayType: (h.gatewayType as GatewayType) || 'openvpn',
    publicEndpoint: h.publicEndpoint,
    status: h.status as MeshHubStatus,
  }))
}

export async function getGatewayNetworks(gatewayId: string): Promise<Network[]> {
  const response = await api.get(`/api/v1/admin/gateways/${gatewayId}/networks`)
  return response.data.networks || []
}

export async function assignGatewayToNetwork(gatewayId: string, networkId: string): Promise<void> {
  await api.post(`/api/v1/admin/gateways/${gatewayId}/networks`, { network_id: networkId })
}

export async function removeGatewayFromNetwork(gatewayId: string, networkId: string): Promise<void> {
  await api.delete(`/api/v1/admin/gateways/${gatewayId}/networks/${networkId}`)
}

// Access Rule API
export type AccessRuleType = 'ip' | 'cidr' | 'hostname' | 'hostname_wildcard'

export interface AccessRule {
  id: string
  name: string
  description: string
  ruleType: AccessRuleType
  value: string
  valueV6?: string // IPv6 IP or CIDR (optional)
  portRange?: string
  protocol?: string
  networkId?: string
  isActive: boolean
  createdAt: string
  updatedAt: string
  users?: string[]
  groups?: string[]
}

export interface CreateAccessRuleRequest {
  name: string
  description?: string
  rule_type: AccessRuleType
  value?: string // IPv4 value (optional if IPv6 provided for IP/CIDR rules)
  value_v6?: string // IPv6 IP or CIDR (optional if IPv4 provided)
  port_range?: string
  protocol?: string
  network_id?: string
  is_active?: boolean
}

export async function getAccessRules(): Promise<AccessRule[]> {
  const response = await api.get('/api/v1/admin/access-rules')
  return response.data.accessRules || []
}

export async function getAccessRule(id: string): Promise<AccessRule> {
  const response = await api.get(`/api/v1/admin/access-rules/${id}`)
  return response.data
}

export async function createAccessRule(req: CreateAccessRuleRequest): Promise<AccessRule> {
  const response = await api.post('/api/v1/admin/access-rules', req)
  return response.data
}

export async function updateAccessRule(id: string, req: CreateAccessRuleRequest): Promise<void> {
  await api.put(`/api/v1/admin/access-rules/${id}`, req)
}

export async function deleteAccessRule(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/access-rules/${id}`)
}

export async function assignRuleToUser(ruleId: string, userId: string): Promise<void> {
  await api.post(`/api/v1/admin/access-rules/${ruleId}/users`, { user_id: userId })
}

export async function removeRuleFromUser(ruleId: string, userId: string): Promise<void> {
  await api.delete(`/api/v1/admin/access-rules/${ruleId}/users/${userId}`)
}

export async function assignRuleToGroup(ruleId: string, groupName: string): Promise<void> {
  await api.post(`/api/v1/admin/access-rules/${ruleId}/groups`, { group_name: groupName })
}

export async function removeRuleFromGroup(ruleId: string, groupName: string): Promise<void> {
  await api.delete(`/api/v1/admin/access-rules/${ruleId}/groups/${groupName}`)
}

// User Management API
export interface SSOUser {
  id: string
  externalId: string
  provider: string
  email: string
  name: string
  groups: string[]
  isAdmin: boolean
  isActive: boolean
  lastLoginAt: string | null
  createdAt: string
  updatedAt: string
}

export interface LocalUser {
  id: string
  username: string
  email: string
  isAdmin: boolean
  lastLoginAt: string | null
  createdAt: string
}

export interface Group {
  name: string
  memberCount: number
}

export interface GroupMember {
  id: string
  email: string
  name: string
  provider: string
}

export async function getUsers(): Promise<SSOUser[]> {
  const response = await api.get('/api/v1/admin/users')
  return (response.data.users || []).map((u: Record<string, unknown>) => ({
    id: u.id,
    externalId: u.external_id,
    provider: u.provider,
    email: u.email,
    name: u.name,
    groups: u.groups || [],
    isAdmin: u.is_admin,
    isActive: u.is_active,
    lastLoginAt: u.last_login_at,
    createdAt: u.created_at,
    updatedAt: u.updated_at,
  }))
}

export async function getUser(id: string): Promise<SSOUser> {
  const response = await api.get(`/api/v1/admin/users/${id}`)
  const u = response.data
  return {
    id: u.id,
    externalId: u.external_id,
    provider: u.provider,
    email: u.email,
    name: u.name,
    groups: u.groups || [],
    isAdmin: u.is_admin,
    isActive: u.is_active,
    lastLoginAt: u.last_login_at,
    createdAt: u.created_at,
    updatedAt: u.updated_at,
  }
}

export async function getUserAccessRules(userId: string): Promise<AccessRule[]> {
  const response = await api.get(`/api/v1/admin/users/${userId}/access-rules`)
  return response.data.access_rules || []
}

export interface UserGateway {
  id: string
  name: string
  hostname: string
  publicIp: string
  vpnPort: number
  vpnProtocol: string
  isActive: boolean
  lastHeartbeat: string | null
}

export async function getUserGateways(userId: string): Promise<UserGateway[]> {
  const response = await api.get(`/api/v1/admin/users/${userId}/gateways`)
  return (response.data.gateways || []).map((g: Record<string, unknown>) => ({
    id: g.id,
    name: g.name,
    hostname: g.hostname || '',
    publicIp: g.public_ip || '',
    vpnPort: g.vpn_port,
    vpnProtocol: g.vpn_protocol,
    isActive: g.is_active,
    lastHeartbeat: g.last_heartbeat,
  }))
}

export async function assignUserGateway(userId: string, gatewayId: string): Promise<void> {
  await api.post(`/api/v1/admin/users/${userId}/gateways`, { gateway_id: gatewayId })
}

export async function removeUserGateway(userId: string, gatewayId: string): Promise<void> {
  await api.delete(`/api/v1/admin/users/${userId}/gateways/${gatewayId}`)
}

export async function getLocalUsers(): Promise<LocalUser[]> {
  const response = await api.get('/api/v1/admin/local-users')
  return (response.data.users || []).map((u: Record<string, unknown>) => ({
    id: u.id,
    username: u.username,
    email: u.email,
    isAdmin: u.is_admin,
    lastLoginAt: u.last_login_at,
    createdAt: u.created_at,
  }))
}

export interface CreateLocalUserRequest {
  username: string
  password: string
  email: string
  is_admin?: boolean
}

export async function createLocalUser(req: CreateLocalUserRequest): Promise<void> {
  await api.post('/api/v1/admin/local-users', req)
}

export async function deleteLocalUser(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/local-users/${id}`)
}

// Group Management API
export async function getGroups(): Promise<Group[]> {
  const response = await api.get('/api/v1/admin/groups')
  return (response.data.groups || []).map((g: Record<string, unknown>) => ({
    name: g.name,
    memberCount: g.member_count || 0,
  }))
}

export async function getGroupMembers(groupName: string): Promise<GroupMember[]> {
  const response = await api.get(`/api/v1/admin/groups/${encodeURIComponent(groupName)}/members`)
  return response.data.members || []
}

export async function getGroupAccessRules(groupName: string): Promise<AccessRule[]> {
  const response = await api.get(`/api/v1/admin/groups/${encodeURIComponent(groupName)}/access-rules`)
  return response.data.access_rules || []
}

// Local Group Management API
export interface LocalGroup {
  id: string
  name: string
  description: string
  memberCount: number
  createdAt: string
  updatedAt: string
}

export interface LocalGroupMember {
  userId: string
  memberType: 'sso' | 'local'
  email: string
  name: string
  createdAt: string
}

export interface CreateLocalGroupRequest {
  name: string
  description?: string
}

export interface UpdateLocalGroupRequest {
  name?: string
  description?: string
}

export async function getLocalGroups(): Promise<LocalGroup[]> {
  const response = await api.get('/api/v1/admin/local-groups')
  return (response.data.groups || []).map((g: Record<string, unknown>) => ({
    id: g.id,
    name: g.name,
    description: g.description || '',
    memberCount: g.member_count || 0,
    createdAt: g.created_at,
    updatedAt: g.updated_at,
  }))
}

export async function getLocalGroup(id: string): Promise<LocalGroup> {
  const response = await api.get(`/api/v1/admin/local-groups/${id}`)
  const g = response.data
  return {
    id: g.id,
    name: g.name,
    description: g.description || '',
    memberCount: g.member_count || 0,
    createdAt: g.created_at,
    updatedAt: g.updated_at,
  }
}

export async function createLocalGroup(req: CreateLocalGroupRequest): Promise<LocalGroup> {
  const response = await api.post('/api/v1/admin/local-groups', req)
  const g = response.data
  return {
    id: g.id,
    name: g.name,
    description: g.description || '',
    memberCount: g.member_count || 0,
    createdAt: g.created_at,
    updatedAt: g.updated_at,
  }
}

export async function updateLocalGroup(id: string, req: UpdateLocalGroupRequest): Promise<void> {
  await api.put(`/api/v1/admin/local-groups/${id}`, req)
}

export async function deleteLocalGroup(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/local-groups/${id}`)
}

export async function getLocalGroupMembers(groupId: string): Promise<LocalGroupMember[]> {
  const response = await api.get(`/api/v1/admin/local-groups/${groupId}/members`)
  return (response.data.members || []).map((m: Record<string, unknown>) => ({
    userId: m.user_id,
    memberType: m.member_type as 'sso' | 'local',
    email: m.email || '',
    name: m.name || '',
    createdAt: m.created_at,
  }))
}

export async function addLocalGroupMember(groupId: string, userId: string, memberType: 'sso' | 'local'): Promise<void> {
  await api.post(`/api/v1/admin/local-groups/${groupId}/members`, { user_id: userId, member_type: memberType })
}

export async function removeLocalGroupMember(groupId: string, userId: string, memberType: 'sso' | 'local'): Promise<void> {
  await api.delete(`/api/v1/admin/local-groups/${groupId}/members/${userId}/${memberType}`)
}

// CA Management API
export interface CAInfo {
  serial_number: string
  subject: string
  issuer: string
  not_before: string
  not_after: string
  is_ca: boolean
  fingerprint: string
  certificate: string
}

export async function getCA(): Promise<CAInfo> {
  const response = await api.get('/api/v1/admin/settings/ca')
  return response.data
}

export async function rotateCA(): Promise<CAInfo> {
  const response = await api.post('/api/v1/admin/settings/ca/rotate')
  return response.data
}

export interface UpdateCARequest {
  certificate: string
  private_key: string
}

export async function updateCA(req: UpdateCARequest): Promise<CAInfo> {
  const response = await api.put('/api/v1/admin/settings/ca', req)
  return response.data
}

// CA List and Management
export interface CAListItem {
  id: string
  status: 'active' | 'pending' | 'retired' | 'revoked'
  serial_number: string
  not_before: string
  not_after: string
  fingerprint: string
  description: string
  created_at: string
}

export async function listCAs(): Promise<CAListItem[]> {
  const response = await api.get('/api/v1/admin/settings/ca/list')
  return response.data.cas || []
}

export async function prepareCARotation(description?: string): Promise<{ id: string; fingerprint: string }> {
  const response = await api.post('/api/v1/admin/settings/ca/prepare-rotation', { description })
  return response.data
}

export async function activateCA(id: string): Promise<void> {
  await api.post(`/api/v1/admin/settings/ca/activate/${id}`)
}

export async function revokeCA(id: string): Promise<void> {
  await api.post(`/api/v1/admin/settings/ca/revoke/${id}`)
}

// System Settings
export interface SystemSettings {
  allowed_crypto_profiles?: string
  session_duration_hours?: string
  secure_cookies?: string
  vpn_cert_validity_hours?: string
  require_fips?: string
  min_tls_version?: string
  allowed_ciphers?: string
}

export async function getSystemSettings(): Promise<SystemSettings> {
  const response = await api.get('/api/v1/admin/settings')
  return response.data.settings || {}
}

// Proxy Application API (Web Access)
export interface ProxyApplication {
  id: string
  name: string
  slug: string
  description: string
  internalUrl: string
  iconUrl?: string
  isActive: boolean
  preserveHostHeader: boolean
  stripPrefix: boolean
  injectHeaders: Record<string, string>
  allowedHeaders: string[]
  websocketEnabled: boolean
  timeoutSeconds: number
  skipTlsVerify: boolean
  createdAt: string
  updatedAt: string
}

export interface UserProxyApplication {
  id: string
  name: string
  slug: string
  description: string
  iconUrl?: string
  proxyUrl: string
  createdAt: string
}

export interface CreateProxyAppRequest {
  name: string
  slug: string
  description?: string
  internal_url: string
  icon_url?: string
  is_active?: boolean
  preserve_host_header?: boolean
  strip_prefix?: boolean
  inject_headers?: Record<string, string>
  allowed_headers?: string[]
  websocket_enabled?: boolean
  timeout_seconds?: number
  skip_tls_verify?: boolean
}

export interface UpdateProxyAppRequest {
  name?: string
  slug?: string
  description?: string
  internal_url?: string
  icon_url?: string
  is_active?: boolean
  preserve_host_header?: boolean
  strip_prefix?: boolean
  inject_headers?: Record<string, string>
  allowed_headers?: string[]
  websocket_enabled?: boolean
  timeout_seconds?: number
  skip_tls_verify?: boolean
}

export interface ProxyAccessLog {
  id: string
  proxyAppId: string
  userId: string
  userEmail: string
  requestMethod: string
  requestPath: string
  responseStatus: number
  responseTimeMs: number
  clientIp: string
  userAgent: string
  createdAt: string
}

// Admin Proxy App API
export async function getProxyApps(): Promise<ProxyApplication[]> {
  const response = await api.get('/api/v1/admin/proxy-apps')
  return (response.data.applications || []).map((app: Record<string, unknown>) => ({
    id: app.id,
    name: app.name,
    slug: app.slug,
    description: app.description || '',
    internalUrl: app.internal_url,
    iconUrl: app.icon_url,
    isActive: app.is_active,
    preserveHostHeader: app.preserve_host_header,
    stripPrefix: app.strip_prefix,
    injectHeaders: app.inject_headers || {},
    allowedHeaders: app.allowed_headers || ['*'],
    websocketEnabled: app.websocket_enabled,
    timeoutSeconds: app.timeout_seconds,
    skipTlsVerify: (app.skip_tls_verify as boolean) ?? false,
    createdAt: app.created_at,
    updatedAt: app.updated_at,
  }))
}

export async function getProxyApp(id: string): Promise<ProxyApplication> {
  const response = await api.get(`/api/v1/admin/proxy-apps/${id}`)
  const app = response.data
  return {
    id: app.id,
    name: app.name,
    slug: app.slug,
    description: app.description || '',
    internalUrl: app.internal_url,
    iconUrl: app.icon_url,
    isActive: app.is_active,
    preserveHostHeader: app.preserve_host_header,
    stripPrefix: app.strip_prefix,
    injectHeaders: app.inject_headers || {},
    allowedHeaders: app.allowed_headers || ['*'],
    websocketEnabled: app.websocket_enabled,
    timeoutSeconds: app.timeout_seconds,
    skipTlsVerify: app.skip_tls_verify ?? false,
    createdAt: app.created_at,
    updatedAt: app.updated_at,
  }
}

export async function createProxyApp(req: CreateProxyAppRequest): Promise<ProxyApplication> {
  const response = await api.post('/api/v1/admin/proxy-apps', req)
  const app = response.data
  return {
    id: app.id,
    name: app.name,
    slug: app.slug,
    description: app.description || '',
    internalUrl: app.internal_url,
    iconUrl: app.icon_url,
    isActive: app.is_active,
    preserveHostHeader: app.preserve_host_header,
    stripPrefix: app.strip_prefix,
    injectHeaders: app.inject_headers || {},
    allowedHeaders: app.allowed_headers || ['*'],
    websocketEnabled: app.websocket_enabled,
    timeoutSeconds: app.timeout_seconds,
    skipTlsVerify: app.skip_tls_verify ?? false,
    createdAt: app.created_at,
    updatedAt: app.updated_at,
  }
}

export async function updateProxyApp(id: string, req: UpdateProxyAppRequest): Promise<void> {
  await api.put(`/api/v1/admin/proxy-apps/${id}`, req)
}

export async function deleteProxyApp(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/proxy-apps/${id}`)
}

// Proxy App User Assignment
export async function getProxyAppUsers(appId: string): Promise<string[]> {
  const response = await api.get(`/api/v1/admin/proxy-apps/${appId}/users`)
  return response.data.users || []
}

export async function assignProxyAppToUser(appId: string, userId: string): Promise<void> {
  await api.post(`/api/v1/admin/proxy-apps/${appId}/users`, { user_id: userId })
}

export async function removeProxyAppFromUser(appId: string, userId: string): Promise<void> {
  await api.delete(`/api/v1/admin/proxy-apps/${appId}/users/${userId}`)
}

// Proxy App Group Assignment
export async function getProxyAppGroups(appId: string): Promise<string[]> {
  const response = await api.get(`/api/v1/admin/proxy-apps/${appId}/groups`)
  return response.data.groups || []
}

export async function assignProxyAppToGroup(appId: string, groupName: string): Promise<void> {
  await api.post(`/api/v1/admin/proxy-apps/${appId}/groups`, { group_name: groupName })
}

export async function removeProxyAppFromGroup(appId: string, groupName: string): Promise<void> {
  await api.delete(`/api/v1/admin/proxy-apps/${appId}/groups/${encodeURIComponent(groupName)}`)
}

// Proxy App Logs
export async function getProxyAppLogs(appId: string): Promise<ProxyAccessLog[]> {
  const response = await api.get(`/api/v1/admin/proxy-apps/${appId}/logs`)
  return (response.data.logs || []).map((log: Record<string, unknown>) => ({
    id: log.id,
    proxyAppId: log.proxy_app_id,
    userId: log.user_id,
    userEmail: log.user_email,
    requestMethod: log.request_method,
    requestPath: log.request_path,
    responseStatus: log.response_status,
    responseTimeMs: log.response_time_ms,
    clientIp: log.client_ip,
    userAgent: log.user_agent,
    createdAt: log.created_at,
  }))
}

// Cross-app proxy access log entry (with app_name)
export interface ProxyAccessLogEntry {
  id: string
  proxy_app_id: string
  app_name?: string
  user_id: string
  user_email: string
  request_method: string
  request_path: string
  response_status: number
  response_time_ms: number
  response_size_bytes?: number
  client_ip: string
  user_agent: string
  request_headers?: Record<string, string>
  response_headers?: Record<string, string>
  created_at: string
}

export async function getAllProxyLogs(params?: {
  app_id?: string
  user_email?: string
  method?: string
  min_status?: number
  max_status?: number
  limit?: number
  offset?: number
}): Promise<ProxyAccessLogEntry[]> {
  const query = new URLSearchParams()
  if (params?.app_id) query.set('app_id', params.app_id)
  if (params?.user_email) query.set('user_email', params.user_email)
  if (params?.method) query.set('method', params.method)
  if (params?.min_status) query.set('min_status', String(params.min_status))
  if (params?.max_status) query.set('max_status', String(params.max_status))
  if (params?.limit) query.set('limit', String(params.limit))
  if (params?.offset) query.set('offset', String(params.offset))
  const qs = query.toString()
  const response = await api.get(`/api/v1/admin/proxy-logs${qs ? '?' + qs : ''}`)
  return response.data.logs || []
}

// User Portal - Get apps user can access
export async function getUserProxyApps(): Promise<UserProxyApplication[]> {
  const response = await api.get('/api/v1/proxy-apps')
  return (response.data.applications || []).map((app: Record<string, unknown>) => ({
    id: app.id,
    name: app.name,
    slug: app.slug,
    description: app.description || '',
    iconUrl: app.icon_url,
    proxyUrl: app.proxy_url,
    createdAt: app.created_at,
  }))
}

// VPN Config Management
export interface VPNConfig {
  id: string
  gatewayId: string
  gatewayName: string
  fileName: string
  expiresAt: string
  createdAt: string
  isRevoked: boolean
  revokedAt: string | null
  downloaded: boolean
}

// Get current user's VPN configs
export async function getUserConfigs(): Promise<VPNConfig[]> {
  const response = await api.get('/api/v1/configs')
  return (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    gatewayId: cfg.gatewayId,
    gatewayName: cfg.gatewayName,
    fileName: cfg.fileName,
    expiresAt: cfg.expiresAt,
    createdAt: cfg.createdAt,
    isRevoked: cfg.isRevoked,
    revokedAt: cfg.revokedAt,
    downloaded: cfg.downloaded,
  }))
}

// Revoke user's own config
export async function revokeConfig(configId: string): Promise<void> {
  await api.post(`/api/v1/configs/${configId}/revoke`)
}

// ==================== WireGuard Config Management ====================

export interface WireGuardConfig {
  id: string
  gatewayId: string
  gatewayName: string
  fileName: string
  clientPublicKey: string
  assignedIp: string
  expiresAt: string
  createdAt: string
  downloadedAt: string | null
  isRevoked: boolean
  revokedAt: string | null
  revokedReason: string
}

export interface GeneratedWireGuardConfig {
  id: string
  fileName: string
  gatewayName: string
  expiresAt: string
  downloadUrl: string
  clientPublicKey: string
  assignedIp: string
}

// Generate a new WireGuard config for a gateway
export async function generateWireGuardConfig(gatewayId: string): Promise<GeneratedWireGuardConfig> {
  const response = await api.post('/api/v1/wireguard/configs/generate', { gateway_id: gatewayId })
  return {
    id: response.data.id,
    fileName: response.data.file_name,
    gatewayName: response.data.gateway_name,
    expiresAt: response.data.expires_at,
    downloadUrl: response.data.download_url,
    clientPublicKey: response.data.client_public_key,
    assignedIp: response.data.assigned_ip,
  }
}

// Download a WireGuard config file
export async function downloadWireGuardConfig(configId: string): Promise<Blob> {
  const response = await api.get(`/api/v1/wireguard/configs/download/${configId}`, {
    responseType: 'blob',
  })
  return response.data
}

// Get current user's WireGuard configs
export async function getUserWireGuardConfigs(): Promise<WireGuardConfig[]> {
  const response = await api.get('/api/v1/wireguard/configs')
  return (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    gatewayId: cfg.gateway_id,
    gatewayName: cfg.gateway_name,
    fileName: cfg.file_name,
    clientPublicKey: cfg.client_public_key,
    assignedIp: cfg.assigned_ip,
    expiresAt: cfg.expires_at,
    createdAt: cfg.created_at,
    downloadedAt: cfg.downloaded_at,
    isRevoked: cfg.revoked_at !== null,
    revokedAt: cfg.revoked_at,
    revokedReason: cfg.revocation_reason || '',
  }))
}

// Revoke user's own WireGuard config
export async function revokeWireGuardConfig(configId: string): Promise<void> {
  await api.post(`/api/v1/wireguard/configs/${configId}/revoke`)
}

// Admin: List all WireGuard configs
export interface AdminWireGuardConfig extends WireGuardConfig {
  userId: string
  userEmail: string
  userName: string
}

export async function adminListWireGuardConfigs(): Promise<{ configs: AdminWireGuardConfig[]; total: number }> {
  const response = await api.get('/api/v1/admin/wireguard/configs')
  const configs = (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    userId: cfg.user_id,
    userEmail: cfg.user_email,
    userName: cfg.user_name,
    gatewayId: cfg.gateway_id,
    gatewayName: cfg.gateway_name,
    fileName: cfg.file_name,
    clientPublicKey: cfg.client_public_key,
    assignedIp: cfg.assigned_ip,
    expiresAt: cfg.expires_at,
    createdAt: cfg.created_at,
    downloadedAt: cfg.downloaded_at,
    isRevoked: cfg.revoked_at !== null,
    revokedAt: cfg.revoked_at,
    revokedReason: cfg.revocation_reason || '',
  }))
  return { configs, total: response.data.total || configs.length }
}

// Admin: Revoke any WireGuard config
export async function adminRevokeWireGuardConfig(configId: string, reason?: string): Promise<void> {
  await api.post(`/api/v1/admin/wireguard/configs/${configId}/revoke`, { reason })
}

// Admin: Revoke all WireGuard configs for a user
export async function adminRevokeUserWireGuardConfigs(userId: string, reason?: string): Promise<{ revokedCount: number }> {
  const response = await api.post(`/api/v1/admin/users/${userId}/revoke-wireguard-configs`, { reason })
  return { revokedCount: response.data.revoked_count || 0 }
}

// Admin: Revoke any config
export async function adminRevokeConfig(configId: string, reason?: string): Promise<void> {
  await api.post(`/api/v1/admin/configs/${configId}/revoke`, { reason })
}

// Admin: Revoke all configs for a user
export async function adminRevokeUserConfigs(userId: string, reason?: string): Promise<{ revokedCount: number }> {
  const response = await api.post(`/api/v1/admin/users/${userId}/revoke-configs`, { reason })
  return { revokedCount: response.data.revokedCount || 0 }
}

// Login Logs and Monitoring
export interface LoginLog {
  id: string
  userId: string
  userEmail: string
  userName: string
  provider: string
  providerName: string
  ipAddress: string
  userAgent: string
  country: string
  countryCode: string
  city: string
  success: boolean
  failureReason: string
  sessionId: string
  createdAt: string
}

export interface LoginLogStats {
  totalLogins: number
  successfulLogins: number
  failedLogins: number
  uniqueUsers: number
  uniqueIps: number
  loginsByProvider: Record<string, number>
  loginsByCountry: Record<string, number>
  recentFailures: LoginLog[]
}

export interface LoginLogFilter {
  userEmail?: string
  userId?: string
  ipAddress?: string
  provider?: string
  success?: boolean
  startTime?: string
  endTime?: string
  limit?: number
  offset?: number
}

// Get login logs with filtering
export async function getLoginLogs(filter?: LoginLogFilter): Promise<{ logs: LoginLog[]; total: number }> {
  const params = new URLSearchParams()
  if (filter?.userEmail) params.append('user_email', filter.userEmail)
  if (filter?.userId) params.append('user_id', filter.userId)
  if (filter?.ipAddress) params.append('ip_address', filter.ipAddress)
  if (filter?.provider) params.append('provider', filter.provider)
  if (filter?.success !== undefined) params.append('success', String(filter.success))
  if (filter?.startTime) params.append('start_time', filter.startTime)
  if (filter?.endTime) params.append('end_time', filter.endTime)
  if (filter?.limit) params.append('limit', String(filter.limit))
  if (filter?.offset) params.append('offset', String(filter.offset))

  const response = await api.get(`/api/v1/admin/login-logs?${params.toString()}`)
  return {
    logs: (response.data.logs || []).map((log: Record<string, unknown>) => ({
      id: log.id,
      userId: log.user_id,
      userEmail: log.user_email,
      userName: log.user_name || '',
      provider: log.provider,
      providerName: log.provider_name || '',
      ipAddress: log.ip_address,
      userAgent: log.user_agent || '',
      country: log.country || '',
      countryCode: log.country_code || '',
      city: log.city || '',
      success: log.success,
      failureReason: log.failure_reason || '',
      sessionId: log.session_id || '',
      createdAt: log.created_at,
    })),
    total: response.data.total || 0,
  }
}

// Get login statistics
export async function getLoginLogStats(days?: number): Promise<LoginLogStats> {
  const params = days ? `?days=${days}` : ''
  const response = await api.get(`/api/v1/admin/login-logs/stats${params}`)
  return {
    totalLogins: response.data.total_logins || 0,
    successfulLogins: response.data.successful_logins || 0,
    failedLogins: response.data.failed_logins || 0,
    uniqueUsers: response.data.unique_users || 0,
    uniqueIps: response.data.unique_ips || 0,
    loginsByProvider: response.data.logins_by_provider || {},
    loginsByCountry: response.data.logins_by_country || {},
    recentFailures: (response.data.recent_failures || []).map((log: Record<string, unknown>) => ({
      id: log.id,
      userId: log.user_id,
      userEmail: log.user_email,
      userName: log.user_name || '',
      provider: log.provider,
      providerName: log.provider_name || '',
      ipAddress: log.ip_address,
      userAgent: log.user_agent || '',
      country: log.country || '',
      countryCode: log.country_code || '',
      city: log.city || '',
      success: log.success,
      failureReason: log.failure_reason || '',
      sessionId: log.session_id || '',
      createdAt: log.created_at,
    })),
  }
}

// Purge old login logs
export async function purgeLoginLogs(days: number): Promise<{ deletedCount: number }> {
  const response = await api.delete(`/api/v1/admin/login-logs?days=${days}`)
  return { deletedCount: response.data.deleted_count || 0 }
}

// Get login log retention setting
export async function getLoginLogRetention(): Promise<{ days: number }> {
  const response = await api.get('/api/v1/admin/login-logs/retention')
  return { days: response.data.days || 30 }
}

// Set login log retention setting
export async function setLoginLogRetention(days: number): Promise<void> {
  await api.put('/api/v1/admin/login-logs/retention', { days })
}

// ==================== Mesh Networking ====================

export type MeshHubStatus = 'pending' | 'online' | 'offline' | 'error'
export type MeshSpokeStatus = 'pending' | 'connected' | 'disconnected' | 'error'

export interface MeshHub {
  id: string
  name: string
  description: string
  gatewayType: GatewayType
  publicEndpoint: string
  publicEndpointV6?: string
  vpnPort: number
  vpnProtocol: string
  vpnSubnet: string
  vpnSubnetV6?: string
  cryptoProfile: CryptoProfile
  tlsAuthEnabled: boolean
  wgPublicKey?: string
  wgListenPort?: number
  fullTunnelMode: boolean
  pushDns: boolean
  dnsServers: string[]
  localNetworks: string[]
  enforceFipsMode: boolean
  sessionEnabled: boolean
  status: MeshHubStatus
  statusMessage: string
  connectedSpokes: number
  connectedClients: number
  lastHeartbeat: string | null
  createdAt: string
  updatedAt: string
}

export interface MeshHubWithToken extends MeshHub {
  apiToken: string
  controlPlaneUrl: string
}

export interface CreateMeshHubRequest {
  name: string
  description?: string
  gatewayType?: GatewayType  // 'openvpn' (default) or 'wireguard'
  publicEndpoint?: string  // IPv4 public endpoint (optional if IPv6 provided)
  publicEndpointV6?: string  // IPv6 public endpoint (optional if IPv4 provided)
  vpnPort?: number
  vpnProtocol?: string
  vpnSubnet?: string  // IPv4 VPN subnet (optional if IPv6 provided)
  vpnSubnetV6?: string  // IPv6 VPN subnet (optional if IPv4 provided)
  cryptoProfile?: CryptoProfile
  tlsAuthEnabled?: boolean
  wgListenPort?: number  // WireGuard listen port (default: 51820)
  fullTunnelMode?: boolean
  pushDns?: boolean
  dnsServers?: string[]
  localNetworks?: string[]
  enforceFipsMode?: boolean
  sessionEnabled?: boolean
}

export interface MeshSpoke {
  id: string
  hubId: string
  name: string
  description: string
  gatewayType: GatewayType
  localNetworks: string[]
  wgPublicKey?: string
  fullTunnelMode: boolean
  pushDns: boolean
  dnsServers: string[]
  enforceFipsMode: boolean
  sessionEnabled: boolean
  tunnelIp: string
  tunnelIpV6?: string
  status: MeshSpokeStatus
  statusMessage: string
  bytesSent: number
  bytesReceived: number
  remoteIp: string
  lastSeen: string | null
  hasClientCert: boolean
  createdAt: string
  updatedAt: string
}

export interface MeshSpokeWithToken extends MeshSpoke {
  token: string
}

export interface CreateMeshSpokeRequest {
  name: string
  description?: string
  localNetworks: string[]
  sessionEnabled?: boolean
  enforceFipsMode?: boolean
}

// Mesh Hub Management
export async function getMeshHubs(): Promise<MeshHub[]> {
  const response = await api.get('/api/v1/admin/mesh/hubs')
  return (response.data.hubs || []).map((hub: Record<string, unknown>) => ({
    id: hub.id,
    name: hub.name,
    description: hub.description || '',
    gatewayType: (hub.gatewayType as GatewayType) || 'openvpn',
    publicEndpoint: hub.publicEndpoint,
    vpnPort: hub.vpnPort,
    vpnProtocol: hub.vpnProtocol,
    vpnSubnet: hub.vpnSubnet,
    vpnSubnetV6: hub.vpnSubnetV6 as string | undefined,
    cryptoProfile: hub.cryptoProfile,
    tlsAuthEnabled: hub.tlsAuthEnabled,
    wgPublicKey: hub.wgPublicKey as string | undefined,
    wgListenPort: hub.wgListenPort as number | undefined,
    fullTunnelMode: hub.fullTunnelMode || false,
    pushDns: hub.pushDns || false,
    dnsServers: (hub.dnsServers as string[]) || [],
    localNetworks: (hub.localNetworks as string[]) || [],
    enforceFipsMode: hub.enforceFipsMode || false,
    sessionEnabled: hub.sessionEnabled ?? true,
    status: hub.status,
    statusMessage: hub.statusMessage || '',
    connectedSpokes: hub.connectedSpokes || 0,
    connectedClients: hub.connectedClients || 0,
    lastHeartbeat: hub.lastHeartbeat,
    createdAt: hub.createdAt,
    updatedAt: hub.updatedAt,
  }))
}

export async function getMeshHub(id: string): Promise<MeshHub> {
  const response = await api.get(`/api/v1/admin/mesh/hubs/${id}`)
  const hub = response.data.hub
  return {
    id: hub.id,
    name: hub.name,
    description: hub.description || '',
    gatewayType: hub.gatewayType || 'openvpn',
    publicEndpoint: hub.publicEndpoint,
    vpnPort: hub.vpnPort,
    vpnProtocol: hub.vpnProtocol,
    vpnSubnet: hub.vpnSubnet,
    vpnSubnetV6: hub.vpnSubnetV6,
    cryptoProfile: hub.cryptoProfile,
    tlsAuthEnabled: hub.tlsAuthEnabled,
    wgPublicKey: hub.wgPublicKey,
    wgListenPort: hub.wgListenPort,
    fullTunnelMode: hub.fullTunnelMode || false,
    pushDns: hub.pushDns || false,
    dnsServers: hub.dnsServers || [],
    localNetworks: hub.localNetworks || [],
    enforceFipsMode: hub.enforceFipsMode || false,
    sessionEnabled: hub.sessionEnabled ?? true,
    status: hub.status,
    statusMessage: hub.statusMessage || '',
    connectedSpokes: hub.connectedSpokes || 0,
    connectedClients: hub.connectedClients || 0,
    lastHeartbeat: hub.lastHeartbeat,
    createdAt: hub.createdAt,
    updatedAt: hub.updatedAt,
  }
}

export async function createMeshHub(req: CreateMeshHubRequest): Promise<MeshHubWithToken> {
  const response = await api.post('/api/v1/admin/mesh/hubs', req)
  const hub = response.data.hub
  return {
    id: hub.id,
    name: hub.name,
    description: hub.description || '',
    gatewayType: hub.gatewayType || 'openvpn',
    publicEndpoint: hub.publicEndpoint,
    vpnPort: hub.vpnPort,
    vpnProtocol: hub.vpnProtocol,
    vpnSubnet: hub.vpnSubnet,
    vpnSubnetV6: hub.vpnSubnetV6,
    cryptoProfile: hub.cryptoProfile,
    tlsAuthEnabled: hub.tlsAuthEnabled,
    wgPublicKey: hub.wgPublicKey,
    wgListenPort: hub.wgListenPort,
    fullTunnelMode: hub.fullTunnelMode || false,
    pushDns: hub.pushDns || false,
    dnsServers: hub.dnsServers || [],
    localNetworks: hub.localNetworks || [],
    enforceFipsMode: hub.enforceFipsMode || false,
    sessionEnabled: hub.sessionEnabled ?? true,
    apiToken: hub.apiToken,
    controlPlaneUrl: hub.controlPlaneUrl,
    status: hub.status,
    statusMessage: '',
    connectedSpokes: 0,
    connectedClients: 0,
    lastHeartbeat: null,
    createdAt: hub.createdAt || new Date().toISOString(),
    updatedAt: hub.updatedAt || new Date().toISOString(),
  }
}

export async function updateMeshHub(id: string, req: Partial<CreateMeshHubRequest>): Promise<void> {
  await api.put(`/api/v1/admin/mesh/hubs/${id}`, req)
}

export async function deleteMeshHub(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/hubs/${id}`)
}

export async function rotateMeshHubToken(id: string): Promise<RotateTokenResponse> {
  const response = await api.post(`/api/v1/admin/mesh/hubs/${id}/rotate-token`)
  return response.data
}

export async function provisionMeshHub(id: string): Promise<{ configVersion: string }> {
  const response = await api.post(`/api/v1/admin/mesh/hubs/${id}/provision`)
  return { configVersion: response.data.configVersion }
}

export async function getMeshHubInstallScript(id: string): Promise<string> {
  const response = await api.get(`/api/v1/admin/mesh/hubs/${id}/install-script`)
  return response.data
}

// Mesh Hub Access Control
export async function getMeshHubUsers(hubId: string): Promise<string[]> {
  const response = await api.get(`/api/v1/admin/mesh/hubs/${hubId}/users`)
  return response.data.users || []
}

export async function assignMeshHubUser(hubId: string, userId: string): Promise<void> {
  await api.post(`/api/v1/admin/mesh/hubs/${hubId}/users`, { userId })
}

export async function removeMeshHubUser(hubId: string, userId: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/hubs/${hubId}/users/${userId}`)
}

export async function getMeshHubGroups(hubId: string): Promise<string[]> {
  const response = await api.get(`/api/v1/admin/mesh/hubs/${hubId}/groups`)
  return response.data.groups || []
}

export async function assignMeshHubGroup(hubId: string, groupName: string): Promise<void> {
  await api.post(`/api/v1/admin/mesh/hubs/${hubId}/groups`, { groupName })
}

export async function removeMeshHubGroup(hubId: string, groupName: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/hubs/${hubId}/groups/${encodeURIComponent(groupName)}`)
}

// Mesh Hub Network Access (Zero-Trust)
export interface MeshHubNetwork {
  id: string
  name: string
  description: string
  cidr: string
  isActive: boolean
}

export async function getMeshHubNetworks(hubId: string): Promise<MeshHubNetwork[]> {
  const response = await api.get(`/api/v1/admin/mesh/hubs/${hubId}/networks`)
  return response.data.networks || []
}

export async function assignMeshHubNetwork(hubId: string, networkId: string): Promise<void> {
  await api.post(`/api/v1/admin/mesh/hubs/${hubId}/networks`, { networkId })
}

export async function removeMeshHubNetwork(hubId: string, networkId: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/hubs/${hubId}/networks/${networkId}`)
}

// Mesh Spoke Management
export async function getMeshSpokes(hubId: string): Promise<MeshSpoke[]> {
  const response = await api.get(`/api/v1/admin/mesh/hubs/${hubId}/spokes`)
  return (response.data.spokes || []).map((spoke: Record<string, unknown>) => ({
    id: spoke.id,
    hubId: spoke.hubId,
    name: spoke.name,
    description: spoke.description || '',
    gatewayType: (spoke.gatewayType as GatewayType) || 'openvpn',
    localNetworks: (spoke.localNetworks as string[]) || [],
    wgPublicKey: spoke.wgPublicKey as string | undefined,
    fullTunnelMode: spoke.fullTunnelMode || false,
    pushDns: spoke.pushDns || false,
    dnsServers: (spoke.dnsServers as string[]) || [],
    enforceFipsMode: spoke.enforceFipsMode || false,
    sessionEnabled: spoke.sessionEnabled ?? true,
    tunnelIp: spoke.tunnelIp || '',
    tunnelIpV6: spoke.tunnelIpV6 as string | undefined,
    status: spoke.status,
    statusMessage: spoke.statusMessage || '',
    bytesSent: spoke.bytesSent || 0,
    bytesReceived: spoke.bytesReceived || 0,
    remoteIp: spoke.remoteIp || '',
    lastSeen: spoke.lastSeen,
    hasClientCert: spoke.hasClientCert || false,
    createdAt: spoke.createdAt,
    updatedAt: spoke.updatedAt,
  }))
}

export async function getMeshSpoke(id: string): Promise<MeshSpoke> {
  const response = await api.get(`/api/v1/admin/mesh/spokes/${id}`)
  const spoke = response.data.spoke
  return {
    id: spoke.id,
    hubId: spoke.hubId,
    name: spoke.name,
    description: spoke.description || '',
    gatewayType: spoke.gatewayType || 'openvpn',
    localNetworks: spoke.localNetworks || [],
    wgPublicKey: spoke.wgPublicKey,
    fullTunnelMode: spoke.fullTunnelMode || false,
    pushDns: spoke.pushDns || false,
    dnsServers: spoke.dnsServers || [],
    enforceFipsMode: spoke.enforceFipsMode || false,
    sessionEnabled: spoke.sessionEnabled ?? true,
    tunnelIp: spoke.tunnelIp || '',
    tunnelIpV6: spoke.tunnelIpV6,
    status: spoke.status,
    statusMessage: spoke.statusMessage || '',
    bytesSent: spoke.bytesSent || 0,
    bytesReceived: spoke.bytesReceived || 0,
    remoteIp: spoke.remoteIp || '',
    lastSeen: spoke.lastSeen,
    hasClientCert: spoke.hasClientCert || false,
    createdAt: spoke.createdAt,
    updatedAt: spoke.updatedAt,
  }
}

export async function createMeshSpoke(hubId: string, req: CreateMeshSpokeRequest): Promise<MeshSpokeWithToken> {
  const response = await api.post(`/api/v1/admin/mesh/hubs/${hubId}/spokes`, req)
  const spoke = response.data.spoke
  return {
    id: spoke.id,
    hubId: spoke.hubId,
    name: spoke.name,
    description: spoke.description || '',
    gatewayType: spoke.gatewayType || 'openvpn',
    localNetworks: spoke.localNetworks || [],
    wgPublicKey: spoke.wgPublicKey,
    fullTunnelMode: spoke.fullTunnelMode || false,
    pushDns: spoke.pushDns || false,
    dnsServers: spoke.dnsServers || [],
    enforceFipsMode: spoke.enforceFipsMode || false,
    sessionEnabled: spoke.sessionEnabled ?? true,
    tunnelIp: '',
    tunnelIpV6: spoke.tunnelIpV6,
    token: spoke.token,
    status: spoke.status,
    statusMessage: '',
    bytesSent: 0,
    bytesReceived: 0,
    remoteIp: '',
    lastSeen: null,
    hasClientCert: false,
    createdAt: spoke.createdAt || new Date().toISOString(),
    updatedAt: spoke.updatedAt || new Date().toISOString(),
  }
}

export async function updateMeshSpoke(id: string, req: Partial<CreateMeshSpokeRequest>): Promise<void> {
  await api.put(`/api/v1/admin/mesh/spokes/${id}`, req)
}

export async function deleteMeshSpoke(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/spokes/${id}`)
}

export async function rotateMeshSpokeToken(id: string): Promise<RotateTokenResponse> {
  const response = await api.post(`/api/v1/admin/mesh/spokes/${id}/rotate-token`)
  return response.data
}

export async function provisionMeshSpoke(id: string): Promise<{ tunnelIp: string }> {
  const response = await api.post(`/api/v1/admin/mesh/spokes/${id}/provision`)
  return { tunnelIp: response.data.tunnelIp }
}

export async function getMeshSpokeInstallScript(id: string): Promise<string> {
  const response = await api.get(`/api/v1/admin/mesh/spokes/${id}/install-script`)
  return response.data
}

// Mesh Spoke Access Control
export async function getMeshSpokeUsers(spokeId: string): Promise<string[]> {
  const response = await api.get(`/api/v1/admin/mesh/spokes/${spokeId}/users`)
  return response.data.users || []
}

export async function assignMeshSpokeUser(spokeId: string, userId: string): Promise<void> {
  await api.post(`/api/v1/admin/mesh/spokes/${spokeId}/users`, { userId })
}

export async function removeMeshSpokeUser(spokeId: string, userId: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/spokes/${spokeId}/users/${userId}`)
}

export async function getMeshSpokeGroups(spokeId: string): Promise<string[]> {
  const response = await api.get(`/api/v1/admin/mesh/spokes/${spokeId}/groups`)
  return response.data.groups || []
}

export async function assignMeshSpokeGroup(spokeId: string, groupName: string): Promise<void> {
  await api.post(`/api/v1/admin/mesh/spokes/${spokeId}/groups`, { groupName })
}

export async function removeMeshSpokeGroup(spokeId: string, groupName: string): Promise<void> {
  await api.delete(`/api/v1/admin/mesh/spokes/${spokeId}/groups/${encodeURIComponent(groupName)}`)
}

// ==================== User Mesh Hub Access ====================

export interface UserMeshHub {
  id: string
  name: string
  description: string
  hubType: 'openvpn' | 'wireguard'
  status: string
  connectedspokes: number
}

export interface MeshClientConfig {
  hubname: string
  config: string
}

export async function getUserMeshHubs(): Promise<UserMeshHub[]> {
  const response = await api.get('/api/v1/mesh/hubs')
  return response.data.hubs || []
}

export interface MeshClientConfigWithId {
  id: string
  hubname: string
  config: string
  expiresAt: string
}

export async function generateMeshClientConfig(hubId: string): Promise<MeshClientConfigWithId> {
  const response = await api.post('/api/v1/mesh/generate-config', { hubid: hubId })
  return {
    id: response.data.id,
    hubname: response.data.hubname,
    config: response.data.config,
    expiresAt: response.data.expiresAt,
  }
}

export async function generateWireGuardMeshClientConfig(hubId: string): Promise<MeshClientConfigWithId> {
  const response = await api.post('/api/v1/mesh/wireguard/generate-config', { hubid: hubId })
  return {
    id: response.data.id,
    hubname: response.data.hubname,
    config: response.data.config,
    expiresAt: response.data.expiresAt,
  }
}

// ==================== User Mesh Config Management ====================

export interface MeshVPNConfig {
  id: string
  hubId: string
  hubName: string
  fileName: string
  expiresAt: string
  createdAt: string
  isRevoked: boolean
  revokedAt: string | null
  downloaded: boolean
}

// Get current user's mesh VPN configs
export async function getUserMeshConfigs(): Promise<MeshVPNConfig[]> {
  const response = await api.get('/api/v1/mesh-configs')
  return (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    hubId: cfg.hubId,
    hubName: cfg.hubName,
    fileName: cfg.fileName,
    expiresAt: cfg.expiresAt,
    createdAt: cfg.createdAt,
    isRevoked: cfg.isRevoked,
    revokedAt: cfg.revokedAt,
    downloaded: cfg.downloaded,
  }))
}

// Revoke user's own mesh config
export async function revokeMeshConfig(configId: string): Promise<void> {
  await api.post(`/api/v1/mesh-configs/${configId}/revoke`)
}

// Download mesh config file
export async function downloadMeshConfig(configId: string): Promise<Blob> {
  const response = await api.get(`/api/v1/mesh-configs/${configId}/download`, {
    responseType: 'blob',
  })
  return response.data
}

// Admin VPN Config with user info
export interface AdminVPNConfig extends VPNConfig {
  userId: string
  userEmail: string
  userName: string
  serialNumber?: string
  fingerprint?: string
  revokedReason?: string
}

// Admin Mesh VPN Config with user info
export interface AdminMeshVPNConfig extends MeshVPNConfig {
  userId: string
  userEmail: string
  userName: string
  serialNumber?: string
  fingerprint?: string
  revokedReason?: string
}

// Admin: List all gateway configs with user info
export async function adminListAllConfigs(): Promise<{ configs: AdminVPNConfig[]; total: number }> {
  const response = await api.get('/api/v1/admin/configs')
  const configs = (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    userId: cfg.userId,
    userEmail: cfg.userEmail,
    userName: cfg.userName,
    gatewayId: cfg.gatewayId,
    gatewayName: cfg.gatewayName,
    fileName: cfg.fileName,
    serialNumber: cfg.serialNumber,
    fingerprint: cfg.fingerprint,
    expiresAt: cfg.expiresAt,
    createdAt: cfg.createdAt,
    isRevoked: cfg.isRevoked,
    revokedAt: cfg.revokedAt,
    revokedReason: cfg.revokedReason,
    downloaded: cfg.downloaded,
  }))
  return { configs, total: response.data.total || configs.length }
}

// Admin: List all mesh configs with user info
export async function adminListMeshConfigs(): Promise<{ configs: AdminMeshVPNConfig[]; total: number }> {
  const response = await api.get('/api/v1/admin/mesh-configs')
  const configs = (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    userId: cfg.userId,
    userEmail: cfg.userEmail,
    userName: cfg.userName,
    hubId: cfg.hubId,
    hubName: cfg.hubName,
    fileName: cfg.fileName,
    serialNumber: cfg.serialNumber,
    fingerprint: cfg.fingerprint,
    expiresAt: cfg.expiresAt,
    createdAt: cfg.createdAt,
    isRevoked: cfg.isRevoked,
    revokedAt: cfg.revokedAt,
    revokedReason: cfg.revokedReason,
    downloaded: cfg.downloaded,
  }))
  return { configs, total: response.data.total || configs.length }
}

// Admin: Revoke any mesh config
export async function adminRevokeMeshConfig(configId: string, reason?: string): Promise<void> {
  await api.post(`/api/v1/admin/mesh-configs/${configId}/revoke`, { reason })
}

// Admin: Revoke all mesh configs for a user
export async function adminRevokeUserMeshConfigs(userId: string, reason?: string): Promise<{ revokedCount: number }> {
  const response = await api.post(`/api/v1/admin/users/${userId}/revoke-mesh-configs`, { reason })
  return { revokedCount: response.data.revokedCount || 0 }
}

// Admin: Get all gateway configs for a specific user
export async function getAdminUserConfigs(userId: string): Promise<VPNConfig[]> {
  const response = await api.get(`/api/v1/admin/users/${userId}/configs`)
  return (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    gatewayId: cfg.gatewayId,
    gatewayName: cfg.gatewayName,
    fileName: cfg.fileName,
    expiresAt: cfg.expiresAt,
    createdAt: cfg.createdAt,
    isRevoked: cfg.isRevoked,
    revokedAt: cfg.revokedAt,
    downloaded: cfg.downloaded,
  }))
}

// Admin: Get all mesh configs for a specific user
export async function getAdminUserMeshConfigs(userId: string): Promise<MeshVPNConfig[]> {
  const response = await api.get(`/api/v1/admin/users/${userId}/mesh-configs`)
  return (response.data.configs || []).map((cfg: Record<string, unknown>) => ({
    id: cfg.id,
    hubId: cfg.hubId,
    hubName: cfg.hubName,
    fileName: cfg.fileName,
    expiresAt: cfg.expiresAt,
    createdAt: cfg.createdAt,
    isRevoked: cfg.isRevoked,
    revokedAt: cfg.revokedAt,
    downloaded: cfg.downloaded,
  }))
}

// Admin: Delete all API keys for a user
export async function adminDeleteUserAPIKeys(userId: string): Promise<{ deletedCount: number }> {
  const response = await api.delete(`/api/v1/admin/users/${userId}/api-keys/all`)
  return { deletedCount: response.data.deletedCount || 0 }
}

// ==================== API Keys ====================

export interface APIKey {
  id: string
  name: string
  description: string
  keyPrefix: string
  scopes: string[]
  isAdminProvisioned: boolean
  provisionedBy?: string
  expiresAt: string | null
  lastUsedAt: string | null
  lastUsedIp: string | null
  isRevoked: boolean
  revokedAt: string | null
  createdAt: string
}

export interface CreateAPIKeyRequest {
  name: string
  description?: string
  scopes?: string[]
  expires_in?: string // e.g., "30d", "90d", "1y", "never"
}

export interface CreateAPIKeyResponse extends APIKey {
  rawKey: string // Only returned on creation
}

// User API Keys - self-service
export async function getUserAPIKeys(): Promise<APIKey[]> {
  const response = await api.get('/api/v1/api-keys')
  return (response.data.api_keys || []).map((key: Record<string, unknown>) => ({
    id: key.id,
    name: key.name,
    description: key.description || '',
    keyPrefix: key.key_prefix,
    scopes: key.scopes || [],
    isAdminProvisioned: key.is_admin_provisioned,
    provisionedBy: key.provisioned_by,
    expiresAt: key.expires_at,
    lastUsedAt: key.last_used_at,
    lastUsedIp: key.last_used_ip,
    isRevoked: key.is_revoked,
    revokedAt: key.revoked_at,
    createdAt: key.created_at,
  }))
}

export async function createUserAPIKey(req: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
  const response = await api.post('/api/v1/api-keys', req)
  const key = response.data
  return {
    id: key.id,
    name: key.name,
    description: key.description || '',
    keyPrefix: key.key_prefix,
    scopes: key.scopes || [],
    isAdminProvisioned: key.is_admin_provisioned,
    provisionedBy: key.provisioned_by,
    expiresAt: key.expires_at,
    lastUsedAt: key.last_used_at,
    lastUsedIp: key.last_used_ip,
    isRevoked: key.is_revoked,
    revokedAt: key.revoked_at,
    createdAt: key.created_at,
    rawKey: key.raw_key,
  }
}

export async function revokeUserAPIKey(keyId: string): Promise<void> {
  await api.delete(`/api/v1/api-keys/${keyId}`)
}

// Admin API Keys - manage all users' keys
export async function getAdminAPIKeys(): Promise<AdminAPIKey[]> {
  const response = await api.get('/api/v1/admin/api-keys')
  return (response.data.api_keys || []).map((key: Record<string, unknown>) => ({
    id: key.id,
    name: key.name,
    description: key.description || '',
    keyPrefix: key.key_prefix,
    scopes: key.scopes || [],
    isAdminProvisioned: key.is_admin_provisioned,
    provisionedBy: key.provisioned_by,
    expiresAt: key.expires_at,
    lastUsedAt: key.last_used_at,
    lastUsedIp: key.last_used_ip,
    isRevoked: key.is_revoked,
    revokedAt: key.revoked_at,
    createdAt: key.created_at,
    userId: key.user_id,
    userEmail: key.user_email,
    userName: key.user_name || '',
  }))
}

export interface AdminAPIKey extends APIKey {
  userId: string
  userEmail: string
  userName: string
}

export async function getAdminUserAPIKeys(userId: string): Promise<APIKey[]> {
  const response = await api.get(`/api/v1/admin/users/${userId}/api-keys`)
  return (response.data.api_keys || []).map((key: Record<string, unknown>) => ({
    id: key.id,
    name: key.name,
    description: key.description || '',
    keyPrefix: key.key_prefix,
    scopes: key.scopes || [],
    isAdminProvisioned: key.is_admin_provisioned,
    provisionedBy: key.provisioned_by,
    expiresAt: key.expires_at,
    lastUsedAt: key.last_used_at,
    lastUsedIp: key.last_used_ip,
    isRevoked: key.is_revoked,
    revokedAt: key.revoked_at,
    createdAt: key.created_at,
  }))
}

export async function createAdminUserAPIKey(userId: string, req: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
  const response = await api.post(`/api/v1/admin/users/${userId}/api-keys`, req)
  const key = response.data
  return {
    id: key.id,
    name: key.name,
    description: key.description || '',
    keyPrefix: key.key_prefix,
    scopes: key.scopes || [],
    isAdminProvisioned: key.is_admin_provisioned,
    provisionedBy: key.provisioned_by,
    expiresAt: key.expires_at,
    lastUsedAt: key.last_used_at,
    lastUsedIp: key.last_used_ip,
    isRevoked: key.is_revoked,
    revokedAt: key.revoked_at,
    createdAt: key.created_at,
    rawKey: key.raw_key,
  }
}

export async function revokeAdminAPIKey(keyId: string): Promise<void> {
  await api.delete(`/api/v1/admin/api-keys/${keyId}`)
}

export async function revokeAdminUserAPIKeys(userId: string): Promise<{ revokedCount: number }> {
  const response = await api.delete(`/api/v1/admin/users/${userId}/api-keys`)
  return { revokedCount: response.data.revoked_count || 0 }
}

// Topology API
export interface TopologyGateway {
  id: string
  name: string
  hostname: string
  publicIp: string
  publicIpV6?: string
  vpnPort: number
  vpnProtocol: string
  isActive: boolean
  lastHeartbeat: string | null
  clientCount: number
}

export interface TopologyMeshHub {
  id: string
  name: string
  publicEndpoint: string
  publicEndpointV6?: string
  publicIp: string
  publicIpV6?: string
  vpnPort: number
  vpnSubnet: string
  vpnSubnetV6?: string
  serverTunnelIp: string
  serverTunnelIpV6?: string
  localNetworks: string[]
  status: string
  lastHeartbeat: string | null
  connectedSpokes: number
  connectedUsers: number
  gatewayType: string // 'openvpn' or 'wireguard'
}

export interface TopologyMeshSpoke {
  id: string
  hubId: string
  name: string
  localNetworks: string[]
  tunnelIp: string
  tunnelIpV6?: string
  status: string
  lastSeen: string | null
  remoteIp: string
  remoteIpV6?: string
  gatewayType: string // 'openvpn' or 'wireguard'
}

export interface TopologyConnection {
  id: string
  source: string
  target: string
  type: string
  status: string
}

export interface TopologyResponse {
  gateways: TopologyGateway[]
  meshHubs: TopologyMeshHub[]
  meshSpokes: TopologyMeshSpoke[]
  connections: TopologyConnection[]
}

export async function getTopology(): Promise<TopologyResponse> {
  const response = await api.get('/api/v1/admin/topology')
  return response.data
}

// Active Sessions API
export interface ActiveSession {
  id: string
  userId: string
  userEmail: string
  userName: string
  gatewayId: string
  gatewayName: string
  nodeType: string
  clientIp: string
  vpnAddress: string
  connectedAt: string
  bytesSent: number
  bytesRecv: number
  lastSeenAt?: string
}

export async function getActiveSessions(): Promise<{ sessions: ActiveSession[]; total: number }> {
  const response = await api.get('/api/v1/admin/sessions/active')
  return response.data
}

// Session Disconnect API
export interface DisconnectResponse {
  success: boolean
  message: string
  sessionId?: string
  userEmail?: string
  gatewayId?: string
  nodeType?: string
  vpnAddress?: string
  disconnects?: number
}

export async function disconnectSession(sessionId: string, reason?: string): Promise<DisconnectResponse> {
  const response = await api.post(`/api/v1/admin/sessions/${sessionId}/disconnect`, {
    reason: reason || 'Disconnected by administrator'
  })
  return response.data
}

export async function disconnectUser(userId: string, reason?: string): Promise<DisconnectResponse> {
  const response = await api.post(`/api/v1/admin/users/${userId}/disconnect`, {
    reason: reason || 'Disconnected by administrator'
  })
  return response.data
}

// User Status & Management API
export interface UserStatus {
  userId: string
  userEmail: string
  userType: string
  isActive: boolean
  activeSessions: number
  sessions: Array<{
    id: string
    gateway_id: string
    gateway_name: string
    client_ip: string
    vpn_address: string
    connected_at: string
    node_type: string
  }>
}

export async function getUserStatus(userId: string): Promise<UserStatus> {
  const response = await api.get(`/api/v1/admin/users/${userId}/status`)
  return response.data
}

export async function getUserActiveSessions(userId: string): Promise<{ sessions: Array<Record<string, unknown>>; count: number }> {
  const response = await api.get(`/api/v1/admin/users/${userId}/sessions`)
  return response.data
}

export interface DisableUserResponse {
  success: boolean
  message: string
  userEmail: string
  userType: string
  sessionsDisconnected: number
}

export async function disableUser(userId: string, reason?: string, disconnectActive = true): Promise<DisableUserResponse> {
  const response = await api.post(`/api/v1/admin/users/${userId}/disable`, {
    reason: reason || 'Disabled by administrator',
    disconnect_active: disconnectActive
  })
  return response.data
}

export async function enableUser(userId: string): Promise<{ success: boolean; message: string; userEmail: string; userType: string }> {
  const response = await api.post(`/api/v1/admin/users/${userId}/enable`, {})
  return response.data
}

// Network Tools API
export interface NetworkToolInfo {
  name: string
  description: string
  options: string[]
  required?: string[]
}

export interface NetworkToolLocation {
  id: string
  name: string
  type: string
}

export interface NetworkToolsInfoResponse {
  tools: NetworkToolInfo[]
  locations: NetworkToolLocation[]
}

export async function getNetworkToolsInfo(): Promise<NetworkToolsInfoResponse> {
  const response = await api.get('/api/v1/admin/network-tools')
  return {
    tools: response.data.tools || [],
    locations: (response.data.locations || []).map((loc: Record<string, string>) => ({
      id: loc.id,
      name: loc.name,
      type: loc.type,
    })),
  }
}

export interface NetworkToolRequest {
  tool: string
  target: string
  port?: number
  ports?: string
  location?: string
  options?: Record<string, string>
}

export interface NetworkToolResult {
  tool: string
  target: string
  status: string
  output: string
  error?: string
  duration: string
  location: string
  startedAt: string
}

export async function executeNetworkTool(req: NetworkToolRequest): Promise<NetworkToolResult> {
  const response = await api.post('/api/v1/admin/network-tools/execute', req)
  return response.data
}

// ==================== Remote Sessions ====================

export interface RemoteSessionAgent {
  id: string
  nodeType: string // hub, gateway, spoke
  nodeId: string
  nodeName: string
  connectedAt: string
}

export async function getRemoteSessionAgents(): Promise<RemoteSessionAgent[]> {
  const response = await api.get('/api/v1/admin/remote-session/agents')
  return (response.data.agents || []).map((agent: Record<string, unknown>) => ({
    id: agent.agentId as string,
    nodeType: agent.nodeType as string,
    nodeId: agent.nodeId as string,
    nodeName: agent.nodeName as string,
    connectedAt: agent.connected as string,
  }))
}

// Get WebSocket URL for remote session
export function getRemoteSessionWebSocketUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws/admin/session`
}

// ==================== Geo-Fencing ====================

export interface GeoFenceSettings {
  enabled: boolean
  enforceMode: 'enforce' | 'audit'
}

export interface GeoFenceRule {
  id: string
  name: string
  description: string
  ipRange?: string    // IPv4 CIDR ranges (optional if IPv6 provided)
  ipv6Range?: string  // IPv6 CIDR ranges (optional if IPv4 provided)
  isActive: boolean
  createdAt: string
  updatedAt: string
  isGlobal?: boolean
  userCount?: number
  groupCount?: number
}

export async function getGeoFenceSettings(): Promise<GeoFenceSettings> {
  const response = await api.get('/api/v1/admin/geo-fence/settings')
  return response.data
}

export async function updateGeoFenceSettings(settings: GeoFenceSettings): Promise<GeoFenceSettings> {
  const response = await api.put('/api/v1/admin/geo-fence/settings', settings)
  return response.data
}

export async function getGeoFenceRules(): Promise<GeoFenceRule[]> {
  const response = await api.get('/api/v1/admin/geo-fence/rules')
  return response.data || []
}

export async function createGeoFenceRule(data: { name: string; description: string; ipRange?: string; ipv6Range?: string }): Promise<GeoFenceRule> {
  const response = await api.post('/api/v1/admin/geo-fence/rules', data)
  return response.data
}

export async function getGeoFenceRule(id: string): Promise<GeoFenceRule> {
  const response = await api.get(`/api/v1/admin/geo-fence/rules/${id}`)
  return response.data
}

export async function updateGeoFenceRule(id: string, data: { name: string; description: string; ipRange?: string; ipv6Range?: string; isActive: boolean }): Promise<GeoFenceRule> {
  const response = await api.put(`/api/v1/admin/geo-fence/rules/${id}`, data)
  return response.data
}

export async function deleteGeoFenceRule(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/geo-fence/rules/${id}`)
}

// Global geo-fence rules
export async function getGlobalGeoRules(): Promise<GeoFenceRule[]> {
  const response = await api.get('/api/v1/admin/geo-fence/global')
  return response.data || []
}

export async function addGlobalGeoRule(ruleId: string): Promise<void> {
  await api.post('/api/v1/admin/geo-fence/global', { ruleId })
}

export async function removeGlobalGeoRule(ruleId: string): Promise<void> {
  await api.delete(`/api/v1/admin/geo-fence/global/${ruleId}`)
}

// User geo-fence rules
export async function getUserGeoRules(userId: string): Promise<GeoFenceRule[]> {
  const response = await api.get(`/api/v1/admin/geo-fence/users/${userId}/rules`)
  return response.data || []
}

export async function addUserGeoRule(userId: string, ruleId: string): Promise<void> {
  await api.post(`/api/v1/admin/geo-fence/users/${userId}/rules`, { ruleId })
}

export async function removeUserGeoRule(userId: string, ruleId: string): Promise<void> {
  await api.delete(`/api/v1/admin/geo-fence/users/${userId}/rules/${ruleId}`)
}

// Group geo-fence rules
export async function getGroupGeoRules(groupName: string): Promise<GeoFenceRule[]> {
  const response = await api.get(`/api/v1/admin/geo-fence/groups/${encodeURIComponent(groupName)}/rules`)
  return response.data || []
}

export async function addGroupGeoRule(groupName: string, ruleId: string): Promise<void> {
  await api.post(`/api/v1/admin/geo-fence/groups/${encodeURIComponent(groupName)}/rules`, { ruleId })
}

export async function removeGroupGeoRule(groupName: string, ruleId: string): Promise<void> {
  await api.delete(`/api/v1/admin/geo-fence/groups/${encodeURIComponent(groupName)}/rules/${ruleId}`)
}

// ==================== JIT Access ====================

export interface JITResource {
  id: string
  name: string
  description: string
  type: string
}

export interface JITRequest {
  id: string
  requester_id?: string
  requester_email?: string
  resource_type: string
  resource_id: string
  resource_name: string
  justification: string
  duration_minutes: number
  status: string
  approver_email?: string
  approval_note?: string
  expires_at: string
  created_at: string
  decided_at?: string
}

export interface JITGrant {
  id: string
  request_id: string
  resource_type: string
  resource_id: string
  is_active: boolean
  granted_at: string
  expires_at: string
}

export interface JITStats {
  pending_requests: number
  active_grants: number
}

// User endpoints
export async function getJITResources(): Promise<JITResource[]> {
  const response = await api.get('/api/v1/jit/resources')
  return response.data.resources || []
}

export async function createJITRequest(req: { resource_type: string; resource_id: string; justification: string; duration_minutes: number }): Promise<JITRequest> {
  const response = await api.post('/api/v1/jit/requests', req)
  return response.data.request
}

export async function getMyJITRequests(): Promise<JITRequest[]> {
  const response = await api.get('/api/v1/jit/requests')
  return response.data.requests || []
}

export async function cancelJITRequest(id: string): Promise<void> {
  await api.post(`/api/v1/jit/requests/${id}/cancel`)
}

export async function getMyJITGrants(): Promise<JITGrant[]> {
  const response = await api.get('/api/v1/jit/grants')
  return response.data.grants || []
}

// Admin endpoints
export async function getAdminJITRequests(status?: string): Promise<JITRequest[]> {
  const params = status ? `?status=${status}` : ''
  const response = await api.get(`/api/v1/admin/jit/requests${params}`)
  return response.data.requests || []
}

export async function approveJITRequest(id: string, note?: string): Promise<JITGrant> {
  const response = await api.post(`/api/v1/admin/jit/requests/${id}/approve`, { note: note || '' })
  return response.data.grant
}

export async function denyJITRequest(id: string, note?: string): Promise<void> {
  await api.post(`/api/v1/admin/jit/requests/${id}/deny`, { note: note || '' })
}

export async function revokeJITGrant(id: string, reason?: string): Promise<void> {
  await api.post(`/api/v1/admin/jit/grants/${id}/revoke`, { reason: reason || '' })
}

export async function getJITStats(): Promise<JITStats> {
  const response = await api.get('/api/v1/admin/jit/stats')
  return response.data
}

export async function getSettings(): Promise<Record<string, string>> {
  const response = await api.get('/api/v1/admin/settings')
  const settings = response.data.settings || []
  const result: Record<string, string> = {}
  for (const s of settings) {
    if (s.key && s.value !== undefined) {
      result[s.key] = s.value
    }
  }
  return result
}

export async function updateSettings(settings: Record<string, string>): Promise<void> {
  await api.put('/api/v1/admin/settings', settings)
}

// JIT Policies
export interface JITPolicy {
  id: string
  name: string
  description: string
  resource_type: string
  resource_id: string | null
  max_duration_minutes: number
  default_duration_minutes: number
  request_expiry_minutes: number
  auto_approve: boolean
  require_justification: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface JITDetailedStats {
  pending_requests: number
  active_grants: number
  total_requests: number
  total_approved: number
  total_denied: number
  total_expired: number
  approval_rate: number
  avg_duration_minutes: number
  requests_by_type: Record<string, number>
  top_requesters: Array<{ email: string; count: number }>
}

export async function getJITPolicies(): Promise<JITPolicy[]> {
  const response = await api.get('/api/v1/admin/jit/policies')
  return response.data.policies || []
}

export async function createJITPolicy(policy: Partial<JITPolicy>): Promise<JITPolicy> {
  const response = await api.post('/api/v1/admin/jit/policies', policy)
  return response.data.policy
}

export async function updateJITPolicy(id: string, updates: Partial<JITPolicy>): Promise<void> {
  await api.put(`/api/v1/admin/jit/policies/${id}`, updates)
}

export async function deleteJITPolicy(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/jit/policies/${id}`)
}

export async function getJITDetailedStats(): Promise<JITDetailedStats> {
  const response = await api.get('/api/v1/admin/jit/stats/detailed')
  return response.data
}

// ==================== Session Recordings ====================

export interface SessionRecording {
  id: string
  session_type: string
  user_email: string
  target_node_name: string | null
  file_size_bytes: number
  duration_seconds: number
  is_complete: boolean
  created_at: string
  completed_at: string | null
}

export interface RecordingSettings {
  enabled: string
  storage_path: string
  retention_days: number
}

export async function getRecordings(limit = 50, offset = 0): Promise<SessionRecording[]> {
  const response = await api.get(`/api/v1/admin/recordings?limit=${limit}&offset=${offset}`)
  return response.data.recordings || []
}

export async function getRecording(id: string): Promise<SessionRecording> {
  const response = await api.get(`/api/v1/admin/recordings/${id}`)
  return response.data
}

export async function streamRecording(id: string): Promise<string> {
  const response = await api.get(`/api/v1/admin/recordings/${id}/stream`, { responseType: 'text' })
  return response.data
}

export async function deleteRecording(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/recordings/${id}`)
}

export async function getRecordingSettings(): Promise<RecordingSettings> {
  const response = await api.get('/api/v1/admin/recordings/settings')
  return response.data
}

export async function updateRecordingSettings(settings: Partial<Record<string, string>>): Promise<void> {
  await api.put('/api/v1/admin/recordings/settings', settings)
}

// ==================== Network Flow Logs ====================

export interface NetworkFlowLog {
  id: string
  gateway_name: string
  user_email: string
  source_ip: string
  dest_ip: string
  dest_port: number
  protocol: string
  bytes_sent: number
  bytes_received: number
  flow_start: string
  flow_end?: string
  created_at: string
}

export interface FlowStats {
  total_flows: number
  total_bytes: number
  unique_users: number
  unique_destinations: number
  flows_today: number
}

export interface TopDestination {
  dest_ip: string
  flow_count: number
  total_bytes: number
}

export async function getFlowLogs(params?: {
  gateway_id?: string; user_email?: string; dest_ip?: string;
  protocol?: string; limit?: number; offset?: number
}): Promise<NetworkFlowLog[]> {
  const query = new URLSearchParams()
  if (params) {
    Object.entries(params).forEach(([k, v]) => { if (v !== undefined) query.set(k, String(v)) })
  }
  const qs = query.toString()
  const response = await api.get(`/api/v1/admin/flow-logs${qs ? '?' + qs : ''}`)
  return response.data.flows || []
}

export async function getFlowStats(): Promise<FlowStats> {
  const response = await api.get('/api/v1/admin/flow-logs/stats')
  return response.data
}

export async function getTopDestinations(limit = 20): Promise<TopDestination[]> {
  const response = await api.get(`/api/v1/admin/flow-logs/top-destinations?limit=${limit}`)
  return response.data.destinations || []
}

export async function getUserFlowActivity(userId: string, limit = 100): Promise<NetworkFlowLog[]> {
  const response = await api.get(`/api/v1/admin/flow-logs/user/${userId}?limit=${limit}`)
  return response.data.flows || []
}
