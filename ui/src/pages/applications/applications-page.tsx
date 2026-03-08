import { envsApi } from "@/api/envs"
import { AppGroupsView } from "@/components/applications/app-groups-view"
import { ApplicationList } from "@/components/applications/application-list"
import { CreateAppDialog } from "@/components/applications/create-app-dialog"
import { CreateAppGroupDialog } from "@/components/applications/create-app-group-dialog"
import { ImportAppsDialog } from "@/components/applications/import-apps-dialog"
import { CreateEnvironmentDialog } from "@/components/environment/create-environment-dialog"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyEnvironmentState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useProjectStore } from "@/stores/project"
import { useQuery } from "@tanstack/react-query"
import { Box, ChevronsUpDown, LayoutList, List, Loader2, Orbit, Plus, Star, Upload } from "lucide-react"
import * as React from "react"

export function ApplicationsPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const [createEnvDialogOpen, setCreateEnvDialogOpen] = React.useState(false)
  const [createAppDialogOpen, setCreateAppDialogOpen] = React.useState(false)
  const [createGroupDialogOpen, setCreateGroupDialogOpen] = React.useState(false)
  const [importDialogOpen, setImportDialogOpen] = React.useState(false)

  const { hasHydrated, activeProjectId: activeProjectIdFromStore, activeEnvId, setActiveEnvId } = useProjectStore()
  const activeProjectId = projectIdProp ?? activeProjectIdFromStore
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  const [activeTab, setActiveTab] = React.useState(() => {
    return localStorage.getItem('applications-active-tab') ?? 'all'
  })

  // Local env selection state used only in embedded mode (projectIdProp set)
  const [embeddedEnvId, setEmbeddedEnvId] = React.useState<string | null>(null)

  const { data: envsResponse, isLoading, isFetched, refetch: refetchEnvs } = useQuery({
    queryKey: ['envs', activeProjectId],
    queryFn: () => envsApi.list(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const envs = envsResponse?.items ?? []
  const safeEnvs = React.useMemo(() => (Array.isArray(envs) ? envs : []), [envs])
  const activeEnv = safeEnvs.find(e => e.id === activeEnvId) || safeEnvs[0]

  // In embedded mode, maintain local env selection defaulting to first env
  const embeddedEnv = safeEnvs.find(e => e.id === embeddedEnvId) || safeEnvs[0]
  const effectiveEnvId = projectIdProp ? (embeddedEnv?.id ?? null) : activeEnvId
  const effectiveEnv = projectIdProp ? embeddedEnv : activeEnv

  React.useEffect(() => {
    if (projectIdProp) {
      // In embedded mode, sync local state when envs load
      if (safeEnvs.length > 0 && !embeddedEnvId) {
        setEmbeddedEnvId(safeEnvs[0].id)
      }
      setActiveTab('all') // Default to "All" tab in embedded mode
    } else {
      if (!hasHydrated) return
      if (!isFetched) return
      // In standalone mode, sync global store
      if (safeEnvs.length > 0) {
        const currentEnvExists = safeEnvs.some(e => e.id === activeEnvId)
        if (!currentEnvExists) setActiveEnvId(safeEnvs[0].id)
      } else {
        if (activeEnvId !== null) setActiveEnvId(null)
      }
    }
  }, [safeEnvs, activeEnvId, setActiveEnvId, projectIdProp, embeddedEnvId, hasHydrated, isFetched])

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    localStorage.setItem('applications-active-tab', value)
  }

  const breadcrumbs = [
    { label: "Applications", icon: Box },
    {
      label: activeEnv?.name || "Select Environment",
      icon: Orbit,
      dropdown: safeEnvs.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="ghost" size="icon-sm">
                <ChevronsUpDown />
              </Button>
            }
          />
          <DropdownMenuContent align="start" className="w-fit">
            <DropdownMenuGroup>
              {safeEnvs.map(env => (
                <DropdownMenuItem key={env.id} onClick={() => setActiveEnvId(env.id)}>
                  <Orbit className="h-4 w-4" />
                  {env.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    }
  ]

  return (
    <div className="flex flex-col flex-1 gap-6">
      {!projectIdProp && <PageHeader items={breadcrumbs} />}
      {!projectIdProp && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Applications</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage your applications and deployments
            </p>
          </div>
        </div>
      )}

      {isLoading && envs.length === 0 ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : !isLoading && safeEnvs.length === 0 ? (
        <EmptyEnvironmentState onAction={!isViewer ? () => setCreateEnvDialogOpen(true) : undefined} />
      ) : effectiveEnvId ? (
        <div className="flex flex-col flex-1">
          <Tabs value={activeTab} onValueChange={handleTabChange} className="flex-1 flex flex-col w-auto h-7">
            {/* Tabs row: TabsList on the left (hidden in embedded mode), action buttons on the right */}
            <div className="flex items-center justify-between mb-2">
              {/* Environment selector (embedded mode) or All/Groups/Favorites tabs (standalone mode) */}
              {projectIdProp ? (
                <Combobox
                  value={embeddedEnv?.id ?? null}
                  onValueChange={(val: string | null) => val && setEmbeddedEnvId(val)}
                  itemToStringLabel={(v) => safeEnvs.find(e => e.id === v)?.name ?? v ?? ""}
                >
                  <ComboboxInput className="w-48" />
                  <ComboboxContent className="w-fit">
                    <ComboboxList>
                      {safeEnvs.map((env) => (
                        <ComboboxItem key={env.id} value={env.id}>
                          {env.name}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              ) : (
                <TabsList>
                  <TabsTrigger value="all"><List />All</TabsTrigger>
                  <TabsTrigger value="groups"><LayoutList />Groups</TabsTrigger>
                  <TabsTrigger value="favorites"><Star />Favorites</TabsTrigger>
                </TabsList>
              )}
              {!isViewer && (
                <div className="flex items-center gap-2">
                  {activeTab === 'groups' && !projectIdProp && (
                    <Button variant="outline" onClick={() => setCreateGroupDialogOpen(true)}>
                      <Plus />
                      Add Group
                    </Button>
                  )}
                  <Button onClick={() => setCreateAppDialogOpen(true)}>
                    <Plus />
                    Create Application
                  </Button>
                  <Button variant="outline" onClick={() => setImportDialogOpen(true)}>
                    <Upload />
                    Import
                  </Button>
                </div>
              )}
            </div>

            <div className="flex-1">
              {activeTab === 'all' && (
                <ApplicationList
                  envId={effectiveEnvId}
                  envName={effectiveEnv?.name}
                  hideToolbarActions={true}
                />
              )}
              {activeTab === 'groups' && (
                <AppGroupsView envId={effectiveEnvId} />
              )}
              {activeTab === 'favorites' && (
                <ApplicationList
                  envId={effectiveEnvId}
                  envName={effectiveEnv?.name}
                  favoritesOnly={true}
                  hideToolbarActions={true}
                />
              )}
            </div>
          </Tabs>
        </div>
      ) : null}

      <CreateEnvironmentDialog
        open={createEnvDialogOpen}
        onOpenChange={setCreateEnvDialogOpen}
        onSuccess={(env) => { setActiveEnvId(env.id) }}
      />

      <CreateAppDialog
        open={createAppDialogOpen}
        onOpenChange={setCreateAppDialogOpen}
      />

      <ImportAppsDialog
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        envId={effectiveEnvId!}
        onSuccess={() => refetchEnvs()}
      />

      <CreateAppGroupDialog
        envId={effectiveEnvId!}
        open={createGroupDialogOpen}
        onOpenChange={setCreateGroupDialogOpen}
      />
    </div>
  )
}
export default ApplicationsPage
