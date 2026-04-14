import { BottomPanel } from "@/components/applications/bottom-panel"
import { AuthRoute } from "@/components/auth/auth-route"
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
import { LoginPage } from "@/pages/auth/login-page"
import { SignupPage } from "@/pages/auth/signup-page"
import { useAuthStore } from "@/stores/auth"
import * as React from "react"
import { BrowserRouter, Navigate, Outlet, Route, Routes } from "react-router-dom"

const DashboardPage = React.lazy(() => import("@/pages/dashboard/dashboard-page").then((module) => ({ default: module.DashboardPage })))
const ProjectsPage = React.lazy(() => import("@/pages/projects/projects-page").then((module) => ({ default: module.ProjectsPage })))
const ProjectDetailPage = React.lazy(() => import("@/pages/projects/project-detail-page").then((module) => ({ default: module.ProjectDetailPage })))
const BuilderPage = React.lazy(() => import("@/pages/builder-sessions/builder-page").then((module) => ({ default: module.BuilderPage })))
const BuilderSessionPage = React.lazy(() => import("@/pages/builder-sessions/builder-session-page").then((module) => ({ default: module.BuilderSessionPage })))
const CollaborationsPage = React.lazy(() => import("@/pages/collaborations/collaborations-page").then((module) => ({ default: module.CollaborationsPage })))
const ActivitiesPage = React.lazy(() => import("@/pages/activities/activities-page").then((module) => ({ default: module.ActivitiesPage })))
const AccountPage = React.lazy(() => import("@/pages/account/account-page").then((module) => ({ default: module.AccountPage })))
const EnvironmentsPage = React.lazy(() => import("@/pages/environments/environments-page").then((module) => ({ default: module.EnvironmentsPage })))
const EnvironmentDetailPage = React.lazy(() => import("@/pages/environments/environment-detail-page").then((module) => ({ default: module.EnvironmentDetailPage })))
const ApplicationsPage = React.lazy(() => import("@/pages/applications/applications-page").then((module) => ({ default: module.ApplicationsPage })))
const ApplicationDetailPage = React.lazy(() => import("@/pages/applications/application-detail-page").then((module) => ({ default: module.ApplicationDetailPage })))
const CodeRepositoriesPage = React.lazy(() => import("@/pages/code-repositories/code-repositories-page").then((module) => ({ default: module.CodeRepositoriesPage })))
const CodeRepositoryDetailPage = React.lazy(() => import("@/pages/code-repositories/code-repository-detail-page").then((module) => ({ default: module.CodeRepositoryDetailPage })))
const ContainerRegistriesPage = React.lazy(() => import("@/pages/container-registries/container-registries-page").then((module) => ({ default: module.ContainerRegistriesPage })))
const ClustersPage = React.lazy(() => import("@/pages/clusters/clusters-page").then((module) => ({ default: module.ClustersPage })))
const ClusterDetailPage = React.lazy(() => import("@/pages/clusters/cluster-detail-page").then((module) => ({ default: module.ClusterDetailPage })))
const ClusterNodeDetailPage = React.lazy(() => import("@/pages/clusters/cluster-node-detail-page").then((module) => ({ default: module.ClusterNodeDetailPage })))
const UsersPage = React.lazy(() => import("@/pages/users/users-page").then((module) => ({ default: module.UsersPage })))
const ExtensionsPage = React.lazy(() => import("@/pages/extensions/extensions-page").then((module) => ({ default: module.ExtensionsPage })))
const PlatformSettingsPage = React.lazy(() => import("@/pages/platform-settings/platform-settings-page").then((module) => ({ default: module.PlatformSettingsPage })))
const PluginsPage = React.lazy(() => import("@/pages/plugins/plugins-page").then((module) => ({ default: module.PluginsPage })))
const MembersPage = React.lazy(() => import("@/pages/members/members-page").then((module) => ({ default: module.MembersPage })))
const RecycleBinPage = React.lazy(() => import("@/pages/recycle-bin/recycle-bin-page").then((module) => ({ default: module.RecycleBinPage })))

function AdminRoute({ children }: { children?: React.ReactNode }) {
  const user = useAuthStore((state) => state.user)
  if (user?.role !== 'admin') {
    return <Navigate to="/" replace />
  }
  return <>{children ?? <Outlet />}</>
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
          <div className="flex h-full min-h-0 flex-1 flex-col gap-4 overflow-hidden p-4 pt-0">
            {children}
          </div>
        </SidebarInset>
        <BottomPanel />
      </BottomPanelProvider>
    </SidebarProvider>
  )
}

function ProtectedPageFallback() {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-muted-foreground">
      Loading page...
    </div>
  )
}

function ProtectedLayoutRoute() {
  return (
    <ProtectedRoute>
      <DashboardLayout>
        <React.Suspense fallback={<ProtectedPageFallback />}>
          <Outlet />
        </React.Suspense>
      </DashboardLayout>
    </ProtectedRoute>
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

            <Route element={<ProtectedLayoutRoute />}>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/projects" element={<ProjectsPage />} />
              <Route path="/projects/:projectId" element={<ProjectDetailPage />} />
              <Route path="/builder-sessions" element={<BuilderPage />} />
              <Route path="/builder-sessions/:sessionId" element={<BuilderSessionPage />} />
              <Route path="/collaborations" element={<CollaborationsPage />} />
              <Route path="/activities" element={<ActivitiesPage />} />
              <Route path="/account" element={<AccountPage />} />
              <Route path="/environments" element={<EnvironmentsPage />} />
              <Route path="/environments/:envId" element={<EnvironmentDetailPage />} />
              <Route path="/applications" element={<ApplicationsPage />} />
              <Route path="/applications/:appId" element={<ApplicationDetailPage />} />
              <Route path="/code-repositories" element={<CodeRepositoriesPage />} />
              <Route path="/code-repositories/:repoId" element={<CodeRepositoryDetailPage />} />
              <Route path="/container-registries" element={<ContainerRegistriesPage />} />
              <Route path="/clusters" element={<ClustersPage />} />
              <Route path="/clusters/:clusterId" element={<ClusterDetailPage />} />
              <Route path="/clusters/:clusterId/nodes/:nodeName" element={<ClusterNodeDetailPage />} />
              <Route path="/plugins" element={<PluginsPage />} />
              <Route path="/members" element={<MembersPage />} />
              <Route path="/recycle-bin" element={<RecycleBinPage />} />

              <Route element={<AdminRoute />}>
                <Route path="/users" element={<UsersPage />} />
                <Route path="/extensions" element={<ExtensionsPage />} />
                <Route path="/platform-settings" element={<PlatformSettingsPage />} />
              </Route>
            </Route>
          </Routes>
        </BrowserRouter>
      </BreadcrumbProvider>
    </ThemeProvider>
  )
}

export default App
