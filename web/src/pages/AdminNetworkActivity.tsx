import { useState, useEffect, useCallback } from 'react'
import {
  NetworkFlowLog,
  FlowStats,
  TopDestination,
  getFlowLogs,
  getFlowStats,
  getTopDestinations,
  getAdminGateways,
  AdminGateway,
} from '../api/client'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

type TabType = 'flows' | 'destinations' | 'statistics'

export default function AdminNetworkActivity() {
  const [activeTab, setActiveTab] = useState<TabType>('flows')
  const [stats, setStats] = useState<FlowStats | null>(null)
  const [flows, setFlows] = useState<NetworkFlowLog[]>([])
  const [destinations, setDestinations] = useState<TopDestination[]>([])
  const [gateways, setGateways] = useState<AdminGateway[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [offset, setOffset] = useState(0)
  const [filters, setFilters] = useState({
    gateway_id: '',
    user_email: '',
    dest_ip: '',
    protocol: '',
  })
  const limit = 50

  const loadStats = useCallback(async () => {
    try {
      const data = await getFlowStats()
      setStats(data)
    } catch {
      // stats are non-critical
    }
  }, [])

  const loadFlows = useCallback(async () => {
    try {
      setLoading(true)
      const params: Record<string, string | number> = { limit, offset }
      if (filters.gateway_id) params.gateway_id = filters.gateway_id
      if (filters.user_email) params.user_email = filters.user_email
      if (filters.dest_ip) params.dest_ip = filters.dest_ip
      if (filters.protocol) params.protocol = filters.protocol
      const data = await getFlowLogs(params)
      setFlows(data)
      setError('')
    } catch {
      setError('Failed to load flow logs')
    } finally {
      setLoading(false)
    }
  }, [filters, offset])

  const loadDestinations = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getTopDestinations(50)
      setDestinations(data)
      setError('')
    } catch {
      setError('Failed to load top destinations')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadStats()
    getAdminGateways().then(setGateways).catch(() => {})
  }, [loadStats])

  useEffect(() => {
    if (activeTab === 'flows') {
      loadFlows()
    } else if (activeTab === 'destinations') {
      loadDestinations()
    } else {
      loadStats()
      setLoading(false)
    }
  }, [activeTab, loadFlows, loadDestinations, loadStats])

  const handleFilterChange = (key: string, value: string) => {
    setOffset(0)
    setFilters(prev => ({ ...prev, [key]: value }))
  }

  const maxDestFlowCount = destinations.length > 0 ? Math.max(...destinations.map(d => d.flow_count)) : 1

  const tabs: { key: TabType; label: string }[] = [
    { key: 'flows', label: 'Flow Logs' },
    { key: 'destinations', label: 'Top Destinations' },
    { key: 'statistics', label: 'Statistics' },
  ]

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Network Activity</h1>

      {/* Stats Row */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          {[
            { label: 'Total Flows', value: stats.total_flows.toLocaleString() },
            { label: 'Total Bytes', value: formatBytes(stats.total_bytes) },
            { label: 'Unique Users', value: stats.unique_users.toLocaleString() },
            { label: 'Unique Destinations', value: stats.unique_destinations.toLocaleString() },
            { label: 'Flows Today', value: stats.flows_today.toLocaleString() },
          ].map((s) => (
            <div key={s.label} className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
              <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{s.label}</dt>
              <dd className="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{s.value}</dd>
            </div>
          ))}
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.key
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Flow Logs Tab */}
      {activeTab === 'flows' && (
        <div className="space-y-4">
          {/* Filter Bar */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Gateway</label>
                <select
                  value={filters.gateway_id}
                  onChange={(e) => handleFilterChange('gateway_id', e.target.value)}
                  className="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm text-sm"
                >
                  <option value="">All Gateways</option>
                  {gateways.map((g) => (
                    <option key={g.id} value={g.id}>{g.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">User Email</label>
                <input
                  type="text"
                  placeholder="Filter by email"
                  value={filters.user_email}
                  onChange={(e) => handleFilterChange('user_email', e.target.value)}
                  className="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Dest IP</label>
                <input
                  type="text"
                  placeholder="Filter by destination IP"
                  value={filters.dest_ip}
                  onChange={(e) => handleFilterChange('dest_ip', e.target.value)}
                  className="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Protocol</label>
                <select
                  value={filters.protocol}
                  onChange={(e) => handleFilterChange('protocol', e.target.value)}
                  className="w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm text-sm"
                >
                  <option value="">Any</option>
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                </select>
              </div>
            </div>
          </div>

          {/* Table */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
            {loading ? (
              <div className="p-8 text-center text-gray-500 dark:text-gray-400">Loading...</div>
            ) : flows.length === 0 ? (
              <div className="p-8 text-center text-gray-500 dark:text-gray-400">No flow logs found</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead className="bg-gray-50 dark:bg-gray-900">
                    <tr>
                      {['Time', 'Gateway', 'User', 'Source IP', 'Destination', 'Protocol', 'Sent', 'Received'].map((h) => (
                        <th key={h} className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {flows.map((f) => (
                      <tr key={f.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-300 whitespace-nowrap">{formatTime(f.flow_start)}</td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-300">{f.gateway_name}</td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-300">{f.user_email}</td>
                        <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 font-mono">{f.source_ip}</td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-300 font-mono">{f.dest_ip}{f.dest_port ? `:${f.dest_port}` : ''}</td>
                        <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 uppercase">{f.protocol}</td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-300 whitespace-nowrap">{formatBytes(f.bytes_sent)}</td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-300 whitespace-nowrap">{formatBytes(f.bytes_received)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {/* Pagination */}
            <div className="px-4 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-between items-center">
              <button
                onClick={() => setOffset(Math.max(0, offset - limit))}
                disabled={offset === 0}
                className="px-3 py-1 text-sm rounded border border-gray-300 dark:border-gray-600 disabled:opacity-50 text-gray-700 dark:text-gray-300"
              >
                Previous
              </button>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                Showing {offset + 1} - {offset + flows.length}
              </span>
              <button
                onClick={() => setOffset(offset + limit)}
                disabled={flows.length < limit}
                className="px-3 py-1 text-sm rounded border border-gray-300 dark:border-gray-600 disabled:opacity-50 text-gray-700 dark:text-gray-300"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Top Destinations Tab */}
      {activeTab === 'destinations' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
          {loading ? (
            <div className="p-8 text-center text-gray-500 dark:text-gray-400">Loading...</div>
          ) : destinations.length === 0 ? (
            <div className="p-8 text-center text-gray-500 dark:text-gray-400">No destination data available</div>
          ) : (
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-900">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Destination IP</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Flow Count</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Total Bytes</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-1/3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {destinations.map((d) => (
                  <tr key={d.dest_ip} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-6 py-4 text-sm font-mono text-gray-900 dark:text-gray-300">{d.dest_ip}</td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-300">{d.flow_count.toLocaleString()}</td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-300">{formatBytes(d.total_bytes)}</td>
                    <td className="px-6 py-4">
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5">
                        <div
                          className="bg-blue-600 h-2.5 rounded-full"
                          style={{ width: `${(d.flow_count / maxDestFlowCount) * 100}%` }}
                        />
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Statistics Tab */}
      {activeTab === 'statistics' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          {stats ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {[
                { label: 'Total Flows', value: stats.total_flows.toLocaleString(), desc: 'Total number of network flow records' },
                { label: 'Total Data Transferred', value: formatBytes(stats.total_bytes), desc: 'Combined sent and received bytes' },
                { label: 'Unique Users', value: stats.unique_users.toLocaleString(), desc: 'Distinct users with flow data' },
                { label: 'Unique Destinations', value: stats.unique_destinations.toLocaleString(), desc: 'Distinct destination IPs accessed' },
                { label: 'Flows Today', value: stats.flows_today.toLocaleString(), desc: 'Flow records created today' },
              ].map((s) => (
                <div key={s.label} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{s.label}</dt>
                  <dd className="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">{s.value}</dd>
                  <p className="mt-1 text-xs text-gray-400 dark:text-gray-500">{s.desc}</p>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-gray-500 dark:text-gray-400">Loading statistics...</div>
          )}
        </div>
      )}
    </div>
  )
}
