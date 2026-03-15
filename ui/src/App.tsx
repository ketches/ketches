import { AuthRoute } from "@/components/auth/auth-route"
import { BottomPanel } from "@/components/applications/bottom-panel"
import { ProtectedRoute } from "@/components/auth/protected-route"
import { AppHeader } from "@/components/layout/app-header"
import { AppSidebar } from "@/components/sidebar/app-sidebar"
import { ThemeProvider } from "@/components/theme-provider/theme-provider"
import {
  SidebarInset,
  SidebarProvider,
} from "@/components/ui/sidebar"
import { Toaster } from "@/components/ui/sonner"
import { BottomPanelProvider } from "@/contexts/bottom-panel-context"
import { BreadcrumbProvider } from "@/contexts/breadcrumb-context"
import { ActivitiesPage } from "@/pages/activities/activities-page"
import { ApplicationDetailPage } from "@/pages/applications/application-detail-page"
import { ApplicationsPage } from "@/pages/applications/applications-page"
import { LoginPage } from "@/pages/auth/login-page"
import { SignupPage } from "@/pages/auth/signup-page"
import { ClusterDetailPage } from "@/pages/clusters/cluster-detail-page"
import { ClusterNodeDetailPage } from "@/pages/clusters/cluster-node-detail-page"
import { ClustersPage } from "@/pages/clusters/clusters-page"
import { CodeRepositoriesPage } from "@/pages/code-repositories/code-repositories-page"
import { CodeRepositoryDetailPage } from "@/pages/code-repositories/code-repository-detail-page"
import { CollaborationsPage } from "@/pages/collaborations/collaborations-page"
import { ContainerRegistriesPage } from "@/pages/container-registries/container-registries-page"
import { DashboardPage } from "@/pages/dashboard/dashboard-page"
import { EnvironmentDetailPage } from "@/pages/environments/environment-detail-page"
import { EnvironmentsPage } from "@/pages/environments/environments-page"
import { ExtensionsPage } from "@/pages/extensions/extensions-page"
import { MembersPage } from "@/pages/members/members-page"
import { PlatformSettingsPage } from "@/pages/platform-settings/platform-settings-page"
import { PluginsPage } from "@/pages/plugins/plugins-page"
import { ProjectDetailPage } from "@/pages/projects/project-detail-page"
import { ProjectsPage } from "@/pages/projects/projects-page"
import { RecycleBinPage } from "@/pages/recycle-bin/recycle-bin-page"
import { UsersPage } from "@/pages/users/users-page"
import { useAuthStore } from "@/stores/auth"
import * as React from "react"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"

function AdminRoute({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((state) => state.user)
  if (user?.role !== 'admin') {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

const SIDEBAR_COOKIE_NAME = "sidebar_state"
const SIDEBAR_COOKIE_MAX_AGE = 60 * 60 * 24 * 7

function getSidebarCookie(): boolean | undefined {
  if (typeof document === "undefined") return undefined
  const match = document.cookie.match(new RegExp("(^| )" + SIDEBAR_COOKIE_NAME + "=([^;]+)"))
  if (match) return match[2] === "true"
  return undefined
}

function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = React.useState(true)

  React.useEffect(() => {
    const stored = getSidebarCookie()
    if (stored !== undefined) {
      setSidebarOpen(stored)
    }
  }, [])

  const handleSidebarChange = (open: boolean) => {
    setSidebarOpen(open)
    document.cookie = `${SIDEBAR_COOKIE_NAME}=${open}; path=/; max-age=${SIDEBAR_COOKIE_MAX_AGE}`
  }

  return (
    <SidebarProvider open={sidebarOpen} onOpenChange={handleSidebarChange}>
      <BottomPanelProvider>
        <AppSidebar />
        <SidebarInset>
          <AppHeader />
          <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
            {children}
          </div>
        </SidebarInset>
        <BottomPanel />
      </BottomPanelProvider>
    </SidebarProvider>
  )
}

export function App() {
  return (
    <ThemeProvider defaultTheme="system" storageKey="vite-ui-theme">
      <Toaster position="top-center" richColors />
      <BreadcrumbProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<AuthRoute><LoginPage /></AuthRoute>} />
            <Route path="/signup" element={<AuthRoute><SignupPage /></AuthRoute>} />

            <Route path="/" element={
              <ProtectedRoute>
                <DashboardLayout><DashboardPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/projects" element={
              <ProtectedRoute>
                <DashboardLayout><ProjectsPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/projects/:projectId" element={
              <ProtectedRoute>
                <DashboardLayout><ProjectDetailPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/collaborations" element={
              <ProtectedRoute>
                <DashboardLayout><CollaborationsPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/activities" element={
              <ProtectedRoute>
                <DashboardLayout><ActivitiesPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/environments" element={
              <ProtectedRoute>
                <DashboardLayout><EnvironmentsPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/environments/:envId" element={
              <ProtectedRoute>
                <DashboardLayout><EnvironmentDetailPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/applications" element={
              <ProtectedRoute>
                <DashboardLayout><ApplicationsPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/applications/:appId" element={
              <ProtectedRoute>
                <DashboardLayout><ApplicationDetailPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/code-repositories" element={
              <ProtectedRoute>
                <DashboardLayout><CodeRepositoriesPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/code-repositories/:repoId" element={
              <ProtectedRoute>
                <DashboardLayout><CodeRepositoryDetailPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/container-registries" element={
              <ProtectedRoute>
                <DashboardLayout><ContainerRegistriesPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/clusters" element={
              <ProtectedRoute>
                <DashboardLayout><ClustersPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/clusters/:clusterId" element={
              <ProtectedRoute>
                <DashboardLayout><ClusterDetailPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/clusters/:clusterId/nodes/:nodeName" element={
              <ProtectedRoute>
                <DashboardLayout><ClusterNodeDetailPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/users" element={
              <ProtectedRoute>
                <AdminRoute>
                  <DashboardLayout><UsersPage /></DashboardLayout>
                </AdminRoute>
              </ProtectedRoute>
            } />
            <Route path="/extensions" element={
              <ProtectedRoute>
                <AdminRoute>
                  <DashboardLayout><ExtensionsPage /></DashboardLayout>
                </AdminRoute>
              </ProtectedRoute>
            } />
            <Route path="/platform-settings" element={
              <ProtectedRoute>
                <AdminRoute>
                  <DashboardLayout><PlatformSettingsPage /></DashboardLayout>
                </AdminRoute>
              </ProtectedRoute>
            } />
            <Route path="/plugins" element={
              <ProtectedRoute>
                <DashboardLayout><PluginsPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/members" element={
              <ProtectedRoute>
                <DashboardLayout><MembersPage /></DashboardLayout>
              </ProtectedRoute>
            } />
            <Route path="/recycle-bin" element={
              <ProtectedRoute>
                <DashboardLayout><RecycleBinPage /></DashboardLayout>
              </ProtectedRoute>
            } />
          </Routes>
        </BrowserRouter>
      </BreadcrumbProvider>
    </ThemeProvider>
  )
}

export default App
