import axios from 'axios'

export const api = axios.create({
  baseURL: '',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

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

export interface AdminGateway {
  id: string
  name: string
  hostname: string
  publicIp: string
  vpnPort: number
  vpnProtocol: string
  cryptoProfile: CryptoProfile
  vpnSubnet: string
  tlsAuthEnabled: boolean
  fullTunnelMode: boolean
  pushDns: boolean
  dnsServers: string[]
  isActive: boolean
  lastHeartbeat: string | null
  createdAt: string
  updatedAt: string
}

export interface RegisterGatewayRequest {
  name: string
  hostname?: string
  public_ip?: string
  vpn_port?: number
  vpn_protocol?: string
  crypto_profile?: CryptoProfile
  vpn_subnet?: string
  tls_auth_enabled?: boolean
  full_tunnel_mode?: boolean
  push_dns?: boolean
  dns_servers?: string[]
}

export interface RegisterGatewayResponse {
  id: string
  name: string
  hostname: string
  vpnPort: number
  vpnProtocol: string
  token: string
  message: string
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

export interface UpdateGatewayRequest {
  name: string
  hostname?: string
  public_ip?: string
  vpn_port?: number
  vpn_protocol?: string
  crypto_profile?: CryptoProfile
  vpn_subnet?: string
  tls_auth_enabled?: boolean
  full_tunnel_mode?: boolean
  push_dns?: boolean
  dns_servers?: string[]
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
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface CreateNetworkRequest {
  name: string
  description?: string
  cidr: string
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
  value: string
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
  publicEndpoint: string
  vpnPort: number
  vpnProtocol: string
  vpnSubnet: string
  cryptoProfile: CryptoProfile
  tlsAuthEnabled: boolean
  fullTunnelMode: boolean
  pushDns: boolean
  dnsServers: string[]
  localNetworks: string[]
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
  publicEndpoint: string
  vpnPort?: number
  vpnProtocol?: string
  vpnSubnet?: string
  cryptoProfile?: CryptoProfile
  tlsAuthEnabled?: boolean
  fullTunnelMode?: boolean
  pushDns?: boolean
  dnsServers?: string[]
  localNetworks?: string[]
}

export interface MeshSpoke {
  id: string
  hubId: string
  name: string
  description: string
  localNetworks: string[]
  fullTunnelMode: boolean
  pushDns: boolean
  dnsServers: string[]
  tunnelIp: string
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
}

// Mesh Hub Management
export async function getMeshHubs(): Promise<MeshHub[]> {
  const response = await api.get('/api/v1/admin/mesh/hubs')
  return (response.data.hubs || []).map((hub: Record<string, unknown>) => ({
    id: hub.id,
    name: hub.name,
    description: hub.description || '',
    publicEndpoint: hub.publicEndpoint,
    vpnPort: hub.vpnPort,
    vpnProtocol: hub.vpnProtocol,
    vpnSubnet: hub.vpnSubnet,
    cryptoProfile: hub.cryptoProfile,
    tlsAuthEnabled: hub.tlsAuthEnabled,
    fullTunnelMode: hub.fullTunnelMode || false,
    pushDns: hub.pushDns || false,
    dnsServers: (hub.dnsServers as string[]) || [],
    localNetworks: (hub.localNetworks as string[]) || [],
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
    publicEndpoint: hub.publicEndpoint,
    vpnPort: hub.vpnPort,
    vpnProtocol: hub.vpnProtocol,
    vpnSubnet: hub.vpnSubnet,
    cryptoProfile: hub.cryptoProfile,
    tlsAuthEnabled: hub.tlsAuthEnabled,
    fullTunnelMode: hub.fullTunnelMode || false,
    pushDns: hub.pushDns || false,
    dnsServers: hub.dnsServers || [],
    localNetworks: hub.localNetworks || [],
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
    publicEndpoint: hub.publicEndpoint,
    vpnPort: hub.vpnPort,
    vpnProtocol: hub.vpnProtocol,
    vpnSubnet: hub.vpnSubnet,
    cryptoProfile: hub.cryptoProfile,
    tlsAuthEnabled: hub.tlsAuthEnabled,
    fullTunnelMode: hub.fullTunnelMode || false,
    pushDns: hub.pushDns || false,
    dnsServers: hub.dnsServers || [],
    localNetworks: hub.localNetworks || [],
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
    localNetworks: (spoke.localNetworks as string[]) || [],
    fullTunnelMode: spoke.fullTunnelMode || false,
    pushDns: spoke.pushDns || false,
    dnsServers: (spoke.dnsServers as string[]) || [],
    tunnelIp: spoke.tunnelIp || '',
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
    localNetworks: spoke.localNetworks || [],
    fullTunnelMode: spoke.fullTunnelMode || false,
    pushDns: spoke.pushDns || false,
    dnsServers: spoke.dnsServers || [],
    tunnelIp: spoke.tunnelIp || '',
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
    localNetworks: spoke.localNetworks || [],
    fullTunnelMode: spoke.fullTunnelMode || false,
    pushDns: spoke.pushDns || false,
    dnsServers: spoke.dnsServers || [],
    tunnelIp: '',
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

export async function generateMeshClientConfig(hubId: string): Promise<MeshClientConfig> {
  const response = await api.post('/api/v1/mesh/generate-config', { hubid: hubId })
  return {
    hubname: response.data.hubname,
    config: response.data.config,
  }
}
