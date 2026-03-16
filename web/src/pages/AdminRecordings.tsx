import { useState, useEffect, useCallback } from 'react'
import {
  SessionRecording,
  RecordingSettings,
  getRecordings,
  streamRecording,
  deleteRecording,
  getRecordingSettings,
  updateRecordingSettings,
} from '../api/client'

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function statusBadge(isComplete: boolean) {
  if (isComplete) {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
        Complete
      </span>
    )
  }
  return (
    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
      Recording
    </span>
  )
}

export default function AdminRecordings() {
  const [activeTab, setActiveTab] = useState<'recordings' | 'settings'>('recordings')
  const [recordings, setRecordings] = useState<SessionRecording[]>([])
  const [settings, setSettings] = useState<RecordingSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [playingId, setPlayingId] = useState<string | null>(null)
  const [playbackContent, setPlaybackContent] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [settingsForm, setSettingsForm] = useState({
    enabled: 'false',
    storage_path: '/var/lib/gatekey/recordings',
    retention_days: '90',
  })
  const [savingSettings, setSavingSettings] = useState(false)
  const [settingsMsg, setSettingsMsg] = useState('')

  const loadRecordings = useCallback(async () => {
    try {
      setLoading(true)
      const data = await getRecordings()
      setRecordings(data)
      setError('')
    } catch {
      setError('Failed to load recordings')
    } finally {
      setLoading(false)
    }
  }, [])

  const loadSettings = useCallback(async () => {
    try {
      const data = await getRecordingSettings()
      setSettings(data)
      setSettingsForm({
        enabled: data.enabled,
        storage_path: data.storage_path,
        retention_days: String(data.retention_days),
      })
    } catch {
      setError('Failed to load settings')
    }
  }, [])

  useEffect(() => {
    if (activeTab === 'recordings') {
      loadRecordings()
    } else {
      loadSettings()
    }
  }, [activeTab, loadRecordings, loadSettings])

  const handlePlay = async (id: string) => {
    try {
      const content = await streamRecording(id)
      setPlaybackContent(content)
      setPlayingId(id)
    } catch {
      setError('Failed to load recording')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteRecording(id)
      setDeleteConfirm(null)
      loadRecordings()
    } catch {
      setError('Failed to delete recording')
    }
  }

  const handleSaveSettings = async () => {
    try {
      setSavingSettings(true)
      await updateRecordingSettings(settingsForm)
      setSettingsMsg('Settings saved')
      setTimeout(() => setSettingsMsg(''), 3000)
      loadSettings()
    } catch {
      setSettingsMsg('Failed to save settings')
    } finally {
      setSavingSettings(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-theme-primary">Session Recordings</h1>
      </div>

      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-sm text-red-800 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-theme">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('recordings')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'recordings'
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : 'border-transparent text-theme-secondary hover:text-theme-primary hover:border-gray-300'
            }`}
          >
            Recordings
          </button>
          <button
            onClick={() => setActiveTab('settings')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'settings'
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : 'border-transparent text-theme-secondary hover:text-theme-primary hover:border-gray-300'
            }`}
          >
            Settings
          </button>
        </nav>
      </div>

      {activeTab === 'recordings' && (
        <>
          {loading ? (
            <div className="flex justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : recordings.length === 0 ? (
            <div className="text-center py-12 text-theme-secondary">
              <p>No recordings found.</p>
              <p className="text-sm mt-1">Recordings are created when an admin connects to an agent with recording enabled.</p>
            </div>
          ) : (
            <div className="bg-theme-primary shadow rounded-lg overflow-hidden">
              <table className="min-w-full divide-y divide-theme">
                <thead className="bg-theme-secondary">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">User</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Target</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Duration</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Size</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Status</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Date</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-theme-secondary uppercase tracking-wider">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-theme">
                  {recordings.map((rec) => (
                    <tr key={rec.id} className="hover:bg-theme-tertiary">
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-primary">{rec.user_email}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-secondary">{rec.target_node_name || '-'}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-secondary">{formatDuration(rec.duration_seconds)}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-secondary">{formatFileSize(rec.file_size_bytes)}</td>
                      <td className="px-6 py-4 whitespace-nowrap">{statusBadge(rec.is_complete)}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-theme-secondary">
                        {new Date(rec.created_at).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm space-x-2">
                        {rec.is_complete && (
                          <button
                            onClick={() => handlePlay(rec.id)}
                            className="text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300 font-medium"
                          >
                            Play
                          </button>
                        )}
                        {deleteConfirm === rec.id ? (
                          <>
                            <button
                              onClick={() => handleDelete(rec.id)}
                              className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 font-medium"
                            >
                              Confirm
                            </button>
                            <button
                              onClick={() => setDeleteConfirm(null)}
                              className="text-theme-secondary hover:text-theme-primary font-medium"
                            >
                              Cancel
                            </button>
                          </>
                        ) : (
                          <button
                            onClick={() => setDeleteConfirm(rec.id)}
                            className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 font-medium"
                          >
                            Delete
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {activeTab === 'settings' && settings && (
        <div className="bg-theme-primary shadow rounded-lg p-6 max-w-2xl">
          <h2 className="text-lg font-semibold text-theme-primary mb-4">Recording Settings</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium text-theme-primary">Recording Enabled</label>
              <button
                onClick={() => setSettingsForm(prev => ({ ...prev, enabled: prev.enabled === 'true' ? 'false' : 'true' }))}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  settingsForm.enabled === 'true' ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    settingsForm.enabled === 'true' ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>
            <div>
              <label className="block text-sm font-medium text-theme-primary mb-1">Storage Path</label>
              <input
                type="text"
                value={settingsForm.storage_path}
                onChange={(e) => setSettingsForm(prev => ({ ...prev, storage_path: e.target.value }))}
                className="w-full px-3 py-2 border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-theme-primary mb-1">Retention (days)</label>
              <input
                type="number"
                value={settingsForm.retention_days}
                onChange={(e) => setSettingsForm(prev => ({ ...prev, retention_days: e.target.value }))}
                className="w-full px-3 py-2 border border-theme rounded-md bg-theme-primary text-theme-primary focus:outline-none focus:ring-2 focus:ring-primary-500"
                min="0"
              />
              <p className="text-xs text-theme-secondary mt-1">Set to 0 to keep recordings forever.</p>
            </div>
            <div className="flex items-center space-x-3">
              <button
                onClick={handleSaveSettings}
                disabled={savingSettings}
                className="px-4 py-2 bg-primary-600 text-white rounded-md hover:bg-primary-700 disabled:opacity-50"
              >
                {savingSettings ? 'Saving...' : 'Save Settings'}
              </button>
              {settingsMsg && <span className="text-sm text-theme-secondary">{settingsMsg}</span>}
            </div>
          </div>
        </div>
      )}

      {/* Playback Modal */}
      {playingId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-theme-primary rounded-lg shadow-xl w-full max-w-4xl max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-theme">
              <h3 className="text-lg font-semibold text-theme-primary">Session Playback</h3>
              <button
                onClick={() => { setPlayingId(null); setPlaybackContent('') }}
                className="text-theme-secondary hover:text-theme-primary"
              >
                <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex-1 overflow-auto p-4 bg-gray-900">
              <pre className="text-green-400 text-sm font-mono whitespace-pre-wrap break-all">
                {playbackContent}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
