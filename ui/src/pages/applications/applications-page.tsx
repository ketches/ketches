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
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useProjectStore } from "@/stores/project"
import { useQuery } from "@tanstack/react-query"
import { Box, ChevronsUpDown, List, ListTree, Orbit, Plus, Star, Upload } from "lucide-react"
import * as React from "react"

export function ApplicationsPage() {
  const [createEnvDialogOpen, setCreateEnvDialogOpen] = React.useState(false)
  const [createAppDialogOpen, setCreateAppDialogOpen] = React.useState(false)
  const [createGroupDialogOpen, setCreateGroupDialogOpen] = React.useState(false)
  const [importDialogOpen, setImportDialogOpen] = React.useState(false)

  const { activeProjectId, activeEnvId, setActiveEnvId } = useProjectStore()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  const [activeTab, setActiveTab] = React.useState(() => {
    return localStorage.getItem('applications-active-tab') ?? 'all'
  })

  const { data: envsResponse, isLoading, refetch: refetchEnvs } = useQuery({
    queryKey: ['envs', activeProjectId],
    queryFn: () => envsApi.list(activeProjectId!),
    enabled: !!activeProjectId,
  })

  const envs = envsResponse?.items ?? []
  const safeEnvs = Array.isArray(envs) ? envs : []
  const activeEnv = safeEnvs.find(e => e.id === activeEnvId) || safeEnvs[0]

  React.useEffect(() => {
    if (safeEnvs.length > 0) {
      const currentEnvExists = safeEnvs.some(e => e.id === activeEnvId)
      if (!currentEnvExists) setActiveEnvId(safeEnvs[0].id)
    } else {
      if (activeEnvId !== null) setActiveEnvId(null)
    }
  }, [safeEnvs, activeEnvId, setActiveEnvId])

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
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-40">
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
      <PageHeader items={breadcrumbs} />
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Applications</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Manage your applications and deployments
          </p>
        </div>
      </div>

      {!isLoading && safeEnvs.length === 0 ? (
        <EmptyEnvironmentState onAction={!isViewer ? () => setCreateEnvDialogOpen(true) : undefined} />
      ) : activeEnvId ? (
        <div className="flex flex-col flex-1">
          <Tabs value={activeTab} onValueChange={handleTabChange} className="flex-1 flex flex-col w-auto h-7">
            {/* Tabs row: TabsList on the left, action buttons on the right */}
            <div className="flex items-center justify-between mb-2">
              <TabsList>
                <TabsTrigger value="all"><List />All</TabsTrigger>
                <TabsTrigger value="groups"><ListTree />Groups</TabsTrigger>
                <TabsTrigger value="favorites"><Star />Favorites</TabsTrigger>
              </TabsList>
              {!isViewer && (
                <div className="flex items-center gap-2">
                  {activeTab === 'groups' && (
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
              <TabsContent value="all" className="mt-0 h-full">
                <ApplicationList
                  envId={activeEnvId}
                  envName={activeEnv?.name}
                  hideToolbarActions={true}
                />
              </TabsContent>
              <TabsContent value="groups" className="mt-0 h-full">
                <AppGroupsView projectId={activeProjectId!} envId={activeEnvId} />
              </TabsContent>
              <TabsContent value="favorites" className="mt-0 h-full">
                <ApplicationList
                  envId={activeEnvId}
                  envName={activeEnv?.name}
                  favoritesOnly={true}
                  hideToolbarActions={true}
                />
              </TabsContent>
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
        envId={activeEnvId!}
        onSuccess={() => refetchEnvs()}
      />

      <CreateAppGroupDialog
        projectId={activeProjectId!}
        open={createGroupDialogOpen}
        onOpenChange={setCreateGroupDialogOpen}
      />
    </div>
  )
}
export default ApplicationsPage
