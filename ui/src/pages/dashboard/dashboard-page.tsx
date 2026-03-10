import { useQuery } from "@tanstack/react-query"
import {
  Activity,
  ArrowRight,
  Blocks,
  Box,
  ChartLine,
  FolderGit,
  GalleryVerticalEnd,
  LayoutDashboard,
  Loader2,
  Orbit,
  Puzzle,
  ShipWheel,
  Users,
  Zap
} from "lucide-react"
import { useNavigate } from "react-router-dom"

import { clustersApi } from "@/api/clusters"
import { dashboardApi } from "@/api/dashboard"
import { envsApi, type Env } from "@/api/envs"
import { projectsApi } from "@/api/projects"
import { PageHeader } from "@/components/layout/page-header"
import { EnvironmentResourceMetrics } from "@/components/monitoring/environment-resource-metrics"
import { MetricsTimeRangeSelector } from "@/components/monitoring/metrics-time-range-selector"
import { useTimeRange } from "@/components/monitoring/use-time-range"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyEnvironmentState } from "@/components/shared/empty-state"
import { StatCard } from "@/components/shared/stat-card"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

function EnvironmentMetricsPanel({ env }: { env: Env }) {
  const { timeRange, setTimeRange, rangeSeconds, step } = useTimeRange()
  return (
    <div className="p-4 border rounded-lg">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h4 className="font-medium">{env.name}</h4>
          <p className="text-xs text-muted-foreground font-mono">{env.cluster_namespace}</p>
        </div>
        <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
      </div>
      <EnvironmentResourceMetrics
        clusterId={env.cluster_id || ""}
        namespace={env.cluster_namespace || ""}
        timeRange={timeRange}
        rangeSeconds={rangeSeconds}
        step={step}
      />
    </div>
  )
}

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

  const connectedClusters = clusters.filter((c) => c.connection_status === "connected").length

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={[{ label: "Dashboard", icon: LayoutDashboard }]} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Admin Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-1">Platform-wide overview and statistics</p>
        </div>
      </div>

      {isLoading ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
            <StatCard
              title="Clusters"
              value={stats?.cluster_count || 0}
              icon={ShipWheel}
              description={`${connectedClusters} connected`}
              onClick={() => navigate("/clusters")}
              color="sky"
            />
            <StatCard
              title="Projects"
              value={stats?.project_count || 0}
              icon={GalleryVerticalEnd}
              description="Active projects"
              color="indigo"
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
              color="green"
            />
            <StatCard
              title="Users"
              value={stats?.user_count || 0}
              icon={Users}
              description="Registered users"
              onClick={() => navigate("/users")}
              color="red"
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
                        className="flex items-center justify-between cursor-pointer bg-muted hover:bg-sky-500/10 p-2 px-4 rounded-md -mx-2"
                        onClick={() => navigate(`/clusters/${cluster.id}`)}
                      >
                        <div className="flex items-center gap-3">
                          <ShipWheel className="h-4 w-4 text-muted-foreground" />
                          <div>
                            <p className="text-sm font-medium">{cluster.name}</p>
                            <p className="text-xs text-muted-foreground font-mono">{cluster.slug}</p>
                          </div>
                        </div>
                        <ColorBadge color={cluster.connection_status === "connected" ? "green" : cluster.connection_status === "disconnected" ? "red" : "gray"}>
                          {cluster.connection_status || "unknown"}
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
                  <Zap className="h-4 w-4" />
                  Quick Actions
                </CardTitle>
                <CardDescription>Common administrative tasks</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid gap-2">
                  <Button variant="outline" className="justify-between" onClick={() => navigate("/clusters")}>
                    <div className="flex items-center gap-2">
                      <ShipWheel className="text-sky-600" />
                      Manage Clusters
                    </div>
                    <ArrowRight />
                  </Button>
                  <Button variant="outline" className="justify-between" onClick={() => navigate("/extensions")}>
                    <div className="flex items-center gap-2">
                      <Blocks className="text-purple-600" />
                      Manage Extensions
                    </div>
                    <ArrowRight />
                  </Button>
                  <Button variant="outline" className="justify-between" onClick={() => navigate("/users")}>
                    <div className="flex items-center gap-2">
                      <Users className="text-red-600" />
                      Manage Users
                    </div>
                    <ArrowRight />
                  </Button>
                  <Button variant="outline" className="justify-between" onClick={() => navigate("/projects")}>
                    <div className="flex items-center gap-2">
                      <GalleryVerticalEnd className="text-indigo-600" />
                      Manage Projects
                    </div>
                    <ArrowRight />
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          <Card className="border-destructive">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-4 w-4" />
                Activities
              </CardTitle>
              <CardDescription>Recent platform-wide events and actions</CardDescription>
            </CardHeader>
            <CardContent>

            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

export function UserDashboard({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const navigate = useNavigate()
  const activeProjectIdFromStore = useProjectStore((state) => state.activeProjectId)
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore

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
          <GalleryVerticalEnd className="h-12 w-12 text-muted-foreground mb-4" />
          <h2 className="text-lg font-medium">No Project Selected</h2>
          <p className="text-sm text-muted-foreground">Select a project from the sidebar to view its dashboard</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      {!projectIdProp && <PageHeader items={[{ label: "Dashboard", icon: LayoutDashboard }]} />}

      {!projectIdProp && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">{project?.name || (statsLoading ? "Loading..." : "No Project Selected")}</h1>
            <p className="text-sm text-muted-foreground mt-1">Overview of your project resources</p>
          </div>
        </div>
      )}

      {statsLoading ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <>
          <div className="grid gap-4 md:grid-cols-5">
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
              color="green"
            />
            <StatCard
              title="Code Repositories"
              value={stats?.code_repository_count || 0}
              icon={FolderGit}
              description="Code repositories"
              onClick={() => navigate("/code-repositories")}
              color="indigo"
            />
            <StatCard
              title="Plugins"
              value={stats?.plugin_count || 0}
              icon={Puzzle}
              description="Plugins"
              onClick={() => navigate("/plugins")}
              color="amber"
            />
            <StatCard
              title="Members"
              value={stats?.project_member_count || 0}
              icon={Users}
              description="Project members"
              onClick={() => navigate("/members")}
              color="red"
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
                <EmptyEnvironmentState />
              ) : (
                <div className="space-y-6">
                  {environments.map((env) => (
                    <EnvironmentMetricsPanel key={env.id} env={env} />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

export function DashboardPage() {
  const user = useAuthStore((state) => state.user)
  const isAdmin = user?.role === "admin"

  return isAdmin ? <AdminDashboard /> : <UserDashboard />
}

export default DashboardPage
