import { appGroupsApi, type AppGroupWithApps } from '@/api/app-groups'
// import { appsApi } from '@/api/apps'
import { ApplicationList } from '@/components/applications/application-list'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { LayoutList, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Label } from '../ui/label'
import { EditAppGroupDialog } from './edit-app-group-dialog'

interface Props { envId: string }

export function AppGroupsView({ envId }: Props) {
  const queryClient = useQueryClient()
  const [editTarget, setEditTarget] = useState<AppGroupWithApps | null>(null)

  const { data: groupedApps = [] } = useQuery({
    queryKey: ['app-groups', envId],
    queryFn: () => appGroupsApi.list(envId),
    enabled: !!envId,
  })

  // Fetch all apps to compute ungrouped set (using smaller page size to avoid freezing)
  // Fetch all apps to compute ungrouped set (using server-side search/pagination matching)
  // const { data: allAppsPage } = useQuery({
  //   queryKey: ['apps', envId, '', 0, 200, false, undefined],
  //   queryFn: () => appsApi.list(envId, { search: '', page: 1, pageSize: 200 }),
  //   enabled: !!envId,
  // })
  // const allApps: any[] = (allAppsPage as any)?.items ?? []

  // Compute ungrouped app IDs
  // const groupedAppIds = new Set(groupedApps.flatMap(g => g.apps.map(a => a.id)))
  // const ungroupedIds = new Set<string>(
  //   allApps.filter(a => !groupedAppIds.has(a.id)).map(a => a.id as string)
  // )

  const deleteMutation = useMutation({
    mutationFn: (groupId: string) => appGroupsApi.delete(groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-groups', envId] })
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
              <Label className="text-sm font-medium text-muted-foreground"><LayoutList className="h-4 w-4" />{group.name}</Label>
              <div className="opacity-0 group-hover/header:opacity-100 transition-opacity">
                <DropdownMenu>
                  <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="h-6 w-6"><MoreHorizontal className="h-3 w-3" /></Button>} />
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
      {/* {ungroupedIds.size > 0 && (
        <div>
          <Label className="text-sm font-medium mb-3 text-muted-foreground">Ungrouped</Label>
          <ApplicationList
            envId={envId}
            hideToolbarActions={true}
            allowedAppIds={ungroupedIds}
          />
        </div>
      )} */}

      {editTarget && (
        <EditAppGroupDialog
          group={editTarget}
          open={!!editTarget}
          onOpenChange={open => !open && setEditTarget(null)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['app-groups', envId] })
            setEditTarget(null)
          }}
        />
      )}
    </div>
  )
}
