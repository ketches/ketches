import { useQuery } from "@tanstack/react-query"
import {
  Activity,
  AlertCircle,
  Box,
  Boxes,
  ChartLine,
  FolderKanban,
  GalleryVerticalEnd,
  LayoutDashboard,
  Loader2,
  Orbit,
  ShipWheel,
  Users
} from "lucide-react"
import { useNavigate } from "react-router-dom"

import { clustersApi } from "@/api/clusters"
import { dashboardApi } from "@/api/dashboard"
import { envsApi } from "@/api/envs"
import { projectsApi } from "@/api/projects"
import { PageHeader } from "@/components/layout/page-header"
import { EnvironmentResourceMetrics } from "@/components/monitoring/environment-resource-metrics"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyEnvironmentState } from "@/components/shared/empty-state"
import { StatCard } from "@/components/shared/stat-card"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

function AdminDashboard() {
  const navigate = useNavigate()

  const { data: stats, isLoading } = useQuery({
    queryKey: ["dashboard-stats-admin"],
    queryFn: () => dashboardApi.getStats(),
  })

  const { data: clusters = [] } = useQuery({
    queryKey: ["clusters-simple"],
    queryFn: () => clustersApi.listSimple(),
  })

  const connectedClusters = clusters.filter((c) => c.status === "connected").length
  const disconnectedClusters = clusters.filter((c) => c.status === "disconnected").length

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={[{ label: "Dashboard", icon: LayoutDashboard }]} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Admin Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-1">Platform-wide overview and statistics</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <StatCard
          title="Clusters"
          value={stats?.cluster_count || 0}
          icon={ShipWheel}
          description={`${connectedClusters} connected`}
          onClick={() => navigate("/clusters")}
        />
        <StatCard
          title="Projects"
          value={stats?.project_count || 0}
          icon={GalleryVerticalEnd}
          description="Active projects"
        />
        <StatCard
          title="Applications"
          value={stats?.application_count || 0}
          icon={Box}
          description="Total applications"
        />
        <StatCard
          title="Environments"
          value={stats?.environment_count || 0}
          icon={Orbit}
          description="Across all projects"
        />
        <StatCard
          title="Users"
          value={stats?.user_count || 0}
          icon={Users}
          description="Registered users"
          onClick={() => navigate("/users")}
        />
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Cluster Health
            </CardTitle>
            <CardDescription>Connection status of all clusters</CardDescription>
          </CardHeader>
          <CardContent>
            {clusters.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <ShipWheel className="h-12 w-12 text-muted-foreground mb-2" />
                <p className="text-sm text-muted-foreground">No clusters configured</p>
                <Button variant="outline" className="mt-4" onClick={() => navigate("/clusters")}>
                  Add Cluster
                </Button>
              </div>
            ) : (
              <div className="space-y-4">
                {clusters.slice(0, 5).map((cluster) => (
                  <div
                    key={cluster.id}
                    className="flex items-center justify-between cursor-pointer hover:bg-muted/50 p-2 rounded-md -mx-2"
                    onClick={() => navigate(`/clusters/${cluster.id}`)}
                  >
                    <div className="flex items-center gap-3">
                      <ShipWheel className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="text-sm font-medium">{cluster.name}</p>
                        <p className="text-xs text-muted-foreground font-mono">{cluster.slug}</p>
                      </div>
                    </div>
                    <ColorBadge color={cluster.status === "connected" ? "green" : cluster.status === "disconnected" ? "red" : "gray"}>
                      {cluster.status || "unknown"}
                    </ColorBadge>
                  </div>
                ))}
                {clusters.length > 5 && (
                  <Button variant="ghost" className="w-full" onClick={() => navigate("/clusters")}>
                    View all {clusters.length} clusters
                  </Button>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Boxes className="h-4 w-4" />
              Quick Actions
            </CardTitle>
            <CardDescription>Common administrative tasks</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-2">
              <Button variant="outline" className="justify-start" onClick={() => navigate("/clusters")}>
                <ShipWheel />
                Manage Clusters
              </Button>
              <Button variant="outline" className="justify-start" onClick={() => navigate("/users")}>
                <Users />
                Manage Users
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      {disconnectedClusters > 0 && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-4 w-4" />
              Cluster Alerts
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">
              {disconnectedClusters} cluster(s) are disconnected. Check the cluster configuration and connectivity.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function UserDashboard() {
  const navigate = useNavigate()
  const activeProjectId = useProjectStore((state) => state.activeProjectId)

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ["dashboard-stats-user", activeProjectId],
    queryFn: () => dashboardApi.getStats(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const { data: project } = useQuery({
    queryKey: ["project", activeProjectId],
    queryFn: () => projectsApi.get(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const { data: environments = [] } = useQuery({
    queryKey: ["environments-simple", activeProjectId],
    queryFn: () => envsApi.listSimpleByProject(activeProjectId!),
    enabled: !!activeProjectId,
  })

  if (!activeProjectId) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <PageHeader items={[{ label: "Dashboard", icon: LayoutDashboard }]} />
        <div className="flex flex-col flex-1 items-center justify-center">
          <FolderKanban className="h-12 w-12 text-muted-foreground mb-4" />
          <h2 className="text-lg font-medium">No Project Selected</h2>
          <p className="text-sm text-muted-foreground">Select a project from the sidebar to view its dashboard</p>
        </div>
      </div>
    )
  }

  if (statsLoading) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={[{ label: "Dashboard", icon: LayoutDashboard }]} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{project?.name || "Project Dashboard"}</h1>
          <p className="text-sm text-muted-foreground mt-1">Overview of your project resources</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          title="Applications"
          value={stats?.application_count || 0}
          icon={Box}
          description="Deployed applications"
          onClick={() => navigate("/applications")}
        />
        <StatCard
          title="Environments"
          value={stats?.environment_count || 0}
          icon={Orbit}
          description="Active environments"
          onClick={() => navigate("/environments")}
        />
        <StatCard
          title="Members"
          value={stats?.member_count || 0}
          icon={Users}
          description="Project members"
          onClick={() => navigate("/members")}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ChartLine className="h-4 w-4" />
            Environment Resource Usage
          </CardTitle>
          <CardDescription>
            Real-time resource consumption across your environments
          </CardDescription>
        </CardHeader>
        <CardContent>
          {environments.length === 0 ? (
            // <Empty>
            //   <EmptyHeader>
            //     <EmptyMedia variant="icon">
            //       <Box />
            //     </EmptyMedia>
            //     <EmptyTitle>No Environments</EmptyTitle>
            //   </EmptyHeader>
            //   <EmptyContent>
            //     <p className="text-sm text-muted-foreground">No environments found in this project.</p>
            //   </EmptyContent>
            // </Empty>
            <EmptyEnvironmentState />
          ) : (
            <div className="space-y-6">
              {environments.map((env) => (
                <div
                  key={env.id}
                  className="p-4 border rounded-lg cursor-pointer hover:bg-muted/50 transition-colors"
                  onClick={() => navigate(`/environments/${env.id}`)}
                >
                  <div className="flex items-center justify-between mb-4">
                    <div>
                      <h4 className="font-medium">{env.name}</h4>
                      <p className="text-xs text-muted-foreground font-mono">{env.metadata?.cluster_namespace}</p>
                    </div>
                    <ColorBadge color={env.status === "ready" ? "green" : "gray"}>{env.status}</ColorBadge>
                  </div>

                  <EnvironmentResourceMetrics
                    clusterId={env.metadata?.cluster_id || ""}
                    namespace={env.metadata?.cluster_namespace || ""}
                  />
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export function DashboardPage() {
  const user = useAuthStore((state) => state.user)
  const isAdmin = user?.role === "admin"

  return isAdmin ? <AdminDashboard /> : <UserDashboard />
}

export default DashboardPage

