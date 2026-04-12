import { useProjectRole } from "@/hooks/useProjectRole"
import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError, type AxiosError } from "axios"
import {
  Box,
  ChartLine,
  ChevronsUpDown,
  CircleAlert,
  Clock,
  Copy,
  GalleryVerticalEnd,
  Globe,
  Hammer,
  Info,
  Orbit,
  Pencil,
  RefreshCw,
  Settings2,
  ShieldCheck,
  ShipWheel,
  Telescope,
  Trash2,
  Wrench
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { appsApi } from "@/api/apps"
import { envsApi } from "@/api/envs"
import { ApplicationList } from "@/components/applications/application-list"
import { EditEnvironmentDialog } from "@/components/environment/edit-environment-dialog"
import { EnvCertificates } from "@/components/environment/env-certificates"
import { EnvSettingsTab } from "@/components/environment/env-settings-tab"
import { EnvDomains } from "@/components/environment/env-domains"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { EnvironmentResourceMetrics } from "@/components/monitoring/environment-resource-metrics"
import { MetricsTimeRangeSelector } from "@/components/monitoring/metrics-time-range-selector"
import { useTimeRange } from "@/components/monitoring/use-time-range"
import { EmptyState } from "@/components/shared/empty-state"
import { DetailHeroSkeleton, InfoCardSkeleton, PanelCardSkeleton, StatCardsSkeleton, TabsSkeleton } from "@/components/shared/page-skeletons"
import { StatCard } from "@/components/shared/stat-card"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

export function EnvironmentDetailPage() {
  const { envId } = useParams<{ envId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const activeTab = searchParams.get("tab") || "overview"
  const [editOpen, setEditOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const { activeProjectId, activeProjectName, activeEnvId, setActiveContextWithNames } = useProjectStore()
  const hasSyncedProjectFromEnvRef = React.useRef(false)
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")
  const { timeRange, setTimeRange, rangeSeconds, step } = useTimeRange()
  const { data: envsResponse } = useQuery({
    queryKey: ['envs', activeProjectId],
    queryFn: () => envsApi.list(activeProjectId!),
    enabled: !!activeProjectId,
  })
  const envs = envsResponse?.items ?? []
  const safeEnvs = Array.isArray(envs) ? envs : []

  const { data: env, isLoading: envLoading, error: envError } = useQuery({
    queryKey: ["env", envId],
    queryFn: () => envsApi.get(envId!),
    enabled: !!envId,
    retry: false,
  })

  React.useEffect(() => {
    hasSyncedProjectFromEnvRef.current = false
  }, [envId])

  React.useEffect(() => {
    if (!hasSyncedProjectFromEnvRef.current && env?.project_id) {
      if (activeProjectId !== env.project_id || activeEnvId !== env.id) {
        setActiveContextWithNames(env.project_id, env.project_name, env.id, env.name)
      }
      hasSyncedProjectFromEnvRef.current = true
    }
  }, [env?.project_id, env?.project_name, env?.id, env?.name, activeProjectId, activeEnvId, setActiveContextWithNames])

  const shouldLoadApps = !!envId && (activeTab === "overview" || activeTab === "applications")
  const { data: appsResponse } = useQuery({
    queryKey: ["apps", envId],
    queryFn: () => appsApi.list(envId!),
    enabled: shouldLoadApps,
  })
  const apps = appsResponse?.items ?? []

  const safeApps = Array.isArray(apps) ? apps : []

  const deleteMutation = useMutation<unknown, AxiosError<{ error: string }>, void>({
    mutationFn: () => envsApi.delete(envId!),
    onSuccess: () => {
      toast.success("Environment deleted", {
        description: "The environment has been successfully deleted",
      })
      queryClient.invalidateQueries({ queryKey: ["envs"] })
      navigate("/environments")
    },
    onError: (error) => {
      toast.error("Failed to delete environment", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  if (envLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <DetailHeroSkeleton actions={3} />
        <TabsSkeleton count={4} />
        <InfoCardSkeleton fields={4} />
        <StatCardsSkeleton count={3} columnsClassName="grid grid-cols-1 gap-4 md:grid-cols-3" />
        <PanelCardSkeleton
          titleWidth="w-24"
          actionWidth="w-32"
          contentHeight="h-80"
          className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent"
        />
      </div>
    )
  }

  if (envError || !env) {
    if (isAxiosError(envError) && envError.response?.status === 403) {
      return (
        <EmptyState
          title="No permission"
          description="You do not have permission to view this environment."
          icon={CircleAlert}
        />
      )
    }

    return (
      <NotFoundPage
        resourceType="Environment"
        backHref="/environments"
        backLabel="Back to Environments"
      />
    )
  }

  const breadcrumbs: BreadcrumbItem[] = isAdmin ? [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    {
      label: activeProjectName ?? "Project",
      icon: GalleryVerticalEnd,
      href: `/projects/${activeProjectId}`,
    },
  ] : [{ label: "Environments", icon: Orbit, href: "/environments" }]
  breadcrumbs.push(
    {
      label: env.name,
      icon: Orbit,
      dropdown: safeEnvs.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-fit">
            <DropdownMenuGroup>
              {safeEnvs.map(e => (
                <DropdownMenuItem
                  key={e.id}
                  onClick={() => {
                    setActiveContextWithNames(activeProjectId, activeProjectName, e.id, e.name)
                    navigate(`/environments/${e.id}`)
                  }}
                >
                  <Orbit className="h-4 w-4" />
                  {e.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    })

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary">
              <Orbit className="h-8 w-8" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight">{env.name}</h1>
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="font-mono">{env.slug}</span>
                {env.description && (
                  <>
                    <span>•</span>
                    <span>{env.description}</span>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {!isViewer && (
              <Button
                variant="outline"
                size="icon"
                onClick={() => setEditOpen(true)}
              >
                <Pencil />
              </Button>
            )}
            <Button
              variant="outline"
              onClick={() => queryClient.invalidateQueries({ queryKey: ["env", envId] })}
            >
              <RefreshCw />
              Refresh
            </Button>
            {!isViewer && (
              <Button
                variant="outline"
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 />
                Delete
              </Button>
            )}
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview">
            <Telescope />
            Overview
          </TabsTrigger>
          <TabsTrigger value="applications">
            <Box />
            Applications
          </TabsTrigger>
          <TabsTrigger value="certificates">
            <ShieldCheck />
            Certificates
          </TabsTrigger>
          <TabsTrigger value="domains">
            <Globe />
            Domains
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings2 />
            Settings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Info className="h-4 w-4" />
                Environment Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Slug</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono">{env.slug}</p>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                      onClick={() => {
                        navigator.clipboard.writeText(env.slug)
                        toast.success("Slug copied to clipboard")
                      }}
                    >
                      <Copy />
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Namespace</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono">{env.cluster_namespace}</p>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                      onClick={(e) => {
                        e.stopPropagation()
                        navigator.clipboard.writeText(env.cluster_namespace)
                        toast.success("Namespace copied to clipboard")
                      }}
                    >
                      <Copy />
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Created At</p>
                  <div className="flex items-center gap-1.5 text-sm">
                    <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                    <span>{formatDate(env.created_at)}</span>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Build Environment</p>
                  <div className="flex items-center gap-2">
                    {env.is_build_env ? (
                      <>
                        <Badge variant="default" className="gap-1"><Hammer className="h-3 w-3" />Build Env</Badge>
                        {!isViewer && (
                          <Button variant="outline" size="sm" className="h-6 text-xs" onClick={() => {
                            envsApi.unsetBuildEnv(env.id).then(() => {
                              queryClient.invalidateQueries({ queryKey: ['env', env.id] })
                              toast.success('Build environment unset')
                            }).catch(() => toast.error('Failed to unset build environment'))
                          }}>Unset</Button>
                        )}
                      </>
                    ) : (
                      !isViewer && (
                        <Button variant="outline" size="sm" className="h-6 text-xs" onClick={() => {
                          envsApi.setBuildEnv(env.id).then(() => {
                            queryClient.invalidateQueries({ queryKey: ['env', env.id] })
                            toast.success('Set as build environment')
                          }).catch(() => toast.error('Failed to set build environment'))
                        }}>
                          <Hammer />
                          Set as Build Env
                        </Button>
                      )
                    )}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <StatCard
              title="Applications"
              value={safeApps.length}
              icon={Box}
              color="blue"
              onClick={() => setSearchParams({ tab: "applications" }, { replace: true })}
              description={
                <>
                  {safeApps.filter(a => a.status === "running").length} running, {safeApps.filter(a => a.status === "undeployed").length} undeployed
                </>
              }
            />
            <StatCard
              title="Total Builds"
              value="404"
              icon={Wrench}
              color="blue"
              description="across repos"
            />
            <StatCard
              title="Cluster"
              value={env.cluster_name || "Unknown"}
              icon={ShipWheel}
              color="sky"
              description={
                env.cluster_connection_status === "connected" ? (
                  <span className="text-green-600">Connected</span>
                ) : (
                  <span className="text-red-600">Disconnected</span>
                )
              }
            />
          </div>

          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <ChartLine className="h-4 w-4" />Metrics</CardTitle>
              <CardAction>
                <MetricsTimeRangeSelector value={timeRange} onChange={setTimeRange} />
              </CardAction>
            </CardHeader>
            <CardContent>
              <EnvironmentResourceMetrics
                clusterId={env.cluster_id}
                projectId={env.project_id}
                prometheusAvailable={env.has_prometheus_integration}
                namespace={env.cluster_namespace}
                timeRange={timeRange}
                rangeSeconds={rangeSeconds}
                step={step}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="applications" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Box className="h-4 w-4" />Applications in this Environment</CardTitle>
              <CardDescription>
                Manage and monitor applications deployed to this environment
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ApplicationList envId={envId!} envName={env.name} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="certificates" className="space-y-4 mt-2">
          <EnvCertificates envId={envId!} isViewer={isViewer} />
        </TabsContent>

        <TabsContent value="domains" className="space-y-4 mt-2">
          <EnvDomains envId={envId!} isViewer={isViewer} />
        </TabsContent>

        <TabsContent value="settings" className="space-y-4 mt-2">
          <EnvSettingsTab envId={envId!} isViewer={isViewer} />
        </TabsContent>
      </Tabs>

      <EditEnvironmentDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        env={env}
        onSuccess={() => {
          queryClient.invalidateQueries({ queryKey: ["env", envId] })
        }}
      />

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the environment "{env.name}" from the platform.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMutation.mutate()}
              disabled={deleteMutation.isPending}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default EnvironmentDetailPage
