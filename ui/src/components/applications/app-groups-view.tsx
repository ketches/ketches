import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { appGroupsApi, type AppGroupWithApps } from '@/api/app-groups'
import { appsApi } from '@/api/apps'
import { ApplicationList } from '@/components/applications/application-list'
import { EditAppGroupDialog } from './edit-app-group-dialog'

interface Props { projectId: string; envId: string }

export function AppGroupsView({ projectId, envId }: Props) {
  const queryClient = useQueryClient()
  const [editTarget, setEditTarget] = useState<AppGroupWithApps | null>(null)

  const { data: groupedApps = [] } = useQuery({
    queryKey: ['grouped-apps', projectId, envId],
    queryFn: () => appGroupsApi.listGrouped(projectId, envId),
    enabled: !!projectId && !!envId,
  })

  // Fetch all apps to compute ungrouped set
  const { data: allAppsPage } = useQuery({
    queryKey: ['apps', envId, { page: 1, pageSize: 1000 }],
    queryFn: () => appsApi.list(envId, { page: 1, pageSize: 1000 }),
    enabled: !!envId,
  })
  const allApps: any[] = (allAppsPage as any)?.items ?? []

  // Compute ungrouped app IDs
  const groupedAppIds = new Set(groupedApps.flatMap(g => g.apps.map(a => a.id)))
  const ungroupedIds = new Set<string>(
    allApps.filter(a => !groupedAppIds.has(a.id)).map(a => a.id as string)
  )

  const deleteMutation = useMutation({
    mutationFn: (groupId: string) => appGroupsApi.delete(groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grouped-apps', projectId, envId] })
      queryClient.invalidateQueries({ queryKey: ['app-groups', projectId] })
      toast.success('Group deleted')
    },
    onError: () => toast.error('Failed to delete group'),
  })

  return (
    <div className="space-y-8">
      {groupedApps.map(group => {
        const allowedIds = new Set(group.apps.map(a => a.id))
        return (
          <div key={group.id}>
            {/* Group header with hover-visible actions */}
            <div className="group/header flex items-center gap-2 mb-3">
              <h3 className="text-sm font-semibold">{group.name}</h3>
              {group.description && (
                <span className="text-xs text-muted-foreground">{group.description}</span>
              )}
              <div className="opacity-0 group-hover/header:opacity-100 transition-opacity">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-6 w-6">
                      <MoreHorizontal className="h-3 w-3" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    <DropdownMenuItem onClick={() => setEditTarget(group)}>
                      <Pencil className="h-4 w-4 mr-2" />
                      Edit
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      className="text-destructive focus:text-destructive"
                      onClick={() => deleteMutation.mutate(group.id)}
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>

            {/* Full DataTable identical to All Apps, filtered to this group's app IDs */}
            <ApplicationList
              envId={envId}
              hideToolbarActions={true}
              allowedAppIds={allowedIds}
              currentGroupId={group.id}
            />
          </div>
        )
      })}

      {/* Ungrouped section — only shown when there are ungrouped apps */}
      {ungroupedIds.size > 0 && (
        <div>
          <h3 className="text-sm font-semibold mb-3 text-muted-foreground">Ungrouped</h3>
          <ApplicationList
            envId={envId}
            hideToolbarActions={true}
            allowedAppIds={ungroupedIds}
          />
        </div>
      )}

      {editTarget && (
        <EditAppGroupDialog
          group={editTarget}
          open={!!editTarget}
          onOpenChange={open => !open && setEditTarget(null)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['grouped-apps', projectId, envId] })
            setEditTarget(null)
          }}
        />
      )}
    </div>
  )
}
