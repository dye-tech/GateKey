import { Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Connect from './pages/Connect'
import WebAccess from './pages/WebAccess'
import MyConfigs from './pages/MyConfigs'
import APIKeys from './pages/APIKeys'
import AdminSettings from './pages/AdminSettings'
import AdminUsers from './pages/AdminUsers'
import AdminGateways from './pages/AdminGateways'
import AdminNetworks from './pages/AdminNetworks'
import AdminAccessRules from './pages/AdminAccessRules'
import AdminProxyApps from './pages/AdminProxyApps'
import AdminMonitoring from './pages/AdminMonitoring'
import AdminMesh from './pages/AdminMesh'
import AdminConfigs from './pages/AdminConfigs'
import AdminTopology from './pages/AdminTopology'
import AdminNetworkTools from './pages/AdminNetworkTools'
import AdminRemoteSessions from './pages/AdminRemoteSessions'
import Help from './pages/Help'
import Layout from './components/Layout'
import { AuthProvider, useAuth } from './contexts/AuthContext'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (!user.isAdmin) {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/" element={
        <ProtectedRoute>
          <Layout>
            <Dashboard />
          </Layout>
        </ProtectedRoute>
      } />
      <Route path="/connect" element={
        <ProtectedRoute>
          <Layout>
            <Connect />
          </Layout>
        </ProtectedRoute>
      } />
      <Route path="/web-access" element={
        <ProtectedRoute>
          <Layout>
            <WebAccess />
          </Layout>
        </ProtectedRoute>
      } />
      <Route path="/my-configs" element={
        <ProtectedRoute>
          <Layout>
            <MyConfigs />
          </Layout>
        </ProtectedRoute>
      } />
      <Route path="/api-keys" element={
        <ProtectedRoute>
          <Layout>
            <APIKeys />
          </Layout>
        </ProtectedRoute>
      } />
      {/* Redirect old /configs route */}
      <Route path="/configs" element={<Navigate to="/connect" replace />} />
      <Route path="/admin/settings" element={<Navigate to="/admin/settings/oidc" replace />} />
      <Route path="/admin/settings/:tab" element={
        <AdminRoute>
          <Layout>
            <AdminSettings />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/users" element={
        <AdminRoute>
          <Layout>
            <AdminUsers />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/gateways" element={
        <AdminRoute>
          <Layout>
            <AdminGateways />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/networks" element={
        <AdminRoute>
          <Layout>
            <AdminNetworks />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/access-rules" element={
        <AdminRoute>
          <Layout>
            <AdminAccessRules />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/proxy-apps" element={
        <AdminRoute>
          <Layout>
            <AdminProxyApps />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/monitoring" element={
        <AdminRoute>
          <Layout>
            <AdminMonitoring />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/mesh" element={
        <AdminRoute>
          <Layout>
            <AdminMesh />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/topology" element={
        <AdminRoute>
          <Layout>
            <AdminTopology />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/network-tools" element={
        <AdminRoute>
          <Layout>
            <AdminNetworkTools />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/remote-sessions" element={
        <AdminRoute>
          <Layout>
            <AdminRemoteSessions />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/admin/configs" element={
        <AdminRoute>
          <Layout>
            <AdminConfigs />
          </Layout>
        </AdminRoute>
      } />
      <Route path="/help" element={
        <ProtectedRoute>
          <Layout>
            <Help />
          </Layout>
        </ProtectedRoute>
      } />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

function App() {
  return (
    <AuthProvider>
      <AppRoutes />
    </AuthProvider>
  )
}

export default App
