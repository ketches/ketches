import { useQuery } from "@tanstack/react-query"
import {
  Box,
  ChevronsUpDown,
  Orbit,
} from "lucide-react"
import * as React from "react"

import { envsApi } from "@/api/envs"
import { ApplicationList } from "@/components/applications/application-list"
import { CreateEnvironmentDialog } from "@/components/environment/create-environment-dialog"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyEnvironmentState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useProjectStore } from "@/stores/project"

export function ApplicationsPage() {
  const [createEnvDialogOpen, setCreateEnvDialogOpen] = React.useState(false)
  const { activeProjectId, activeEnvId, setActiveEnvId } = useProjectStore()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  const { data: envsResponse, isLoading } = useQuery({
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
      if (!currentEnvExists) {
        setActiveEnvId(safeEnvs[0].id)
      }
    } else {
      if (activeEnvId !== null) {
        setActiveEnvId(null)
      }
    }
  }, [safeEnvs, activeEnvId, setActiveEnvId])

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
                  <Orbit className="mr-2 h-4 w-4" />
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
        <ApplicationList envId={activeEnvId} envName={activeEnv?.name} />
      ) : null}

      <CreateEnvironmentDialog
        open={createEnvDialogOpen}
        onOpenChange={setCreateEnvDialogOpen}
        onSuccess={(env) => {
          setActiveEnvId(env.id)
        }}
      />
    </div>
  )
}

export default ApplicationsPage
