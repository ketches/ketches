import { appGroupsApi, type AppGroupWithApps } from '@/api/app-groups'
// import { appsApi } from '@/api/apps'
import { ApplicationList } from '@/components/applications/application-list'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { type PaginationState } from "@tanstack/react-table"
import { LayoutList, MoreVertical, Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Label } from '../ui/label'
import { EditAppGroupDialog } from './edit-app-group-dialog'
interface Props { envId: string }

function GroupAppList({ groupId, envId, currentGroupId }: { groupId: string; envId: string; currentGroupId: string }) {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [searchQuery, setSearchQuery] = useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)

  const { data: groupAppsResponse } = useQuery({
    queryKey: ['group-apps', groupId, debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => appGroupsApi.listGroupApps(groupId, {
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
      search: debouncedSearch,
    }),
    refetchInterval: 5000,
    placeholderData: (prev) => prev,
  })

  return (
    <ApplicationList
      envId={envId}
      hideToolbarActions={true}
      currentGroupId={currentGroupId}
      externalApps={groupAppsResponse?.items || []}
      externalPagination={pagination}
      onExternalPaginationChange={setPagination}
      externalTotalCount={groupAppsResponse?.pagination?.total || 0}
      externalSearchQuery={searchQuery}
      onExternalSearchChange={setSearchQuery}
    />
  )
}


export function AppGroupsView({ envId }: Props) {
  const queryClient = useQueryClient()
  const [editTarget, setEditTarget] = useState<AppGroupWithApps | null>(null)

  const { data: groupedApps = [] } = useQuery({
    queryKey: ['app-groups', envId],
    queryFn: () => appGroupsApi.list(envId),
    enabled: !!envId,
  })

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
        return (
          <div key={group.id}>
            <div className="group/header flex items-center gap-2 mb-3">
              <Label className="text-sm font-normal text-muted-foreground"><LayoutList className="h-4 w-4" />{group.name}</Label>
              <div className="opacity-0 group-hover/header:opacity-100 transition-opacity">
                <DropdownMenu>
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <DropdownMenuTrigger
                          render={
                            <Button variant="ghost" size="icon" className="h-6 w-6" />
                          }
                        />
                      }
                    >
                      <MoreVertical className="h-3 w-3" />
                    </TooltipTrigger>
                    <TooltipContent>Group actions</TooltipContent>
                  </Tooltip>
                  <DropdownMenuContent>
                    <DropdownMenuItem onClick={() => setEditTarget(group)}>
                      <Pencil />
                      Edit
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      variant="destructive"
                      onClick={() => deleteMutation.mutate(group.id)}
                    >
                      <Trash2 />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>

            {/* Full DataTable identical to All Apps, filtered to this group's app IDs */}
            <GroupAppList
              groupId={group.id}
              envId={envId}
              currentGroupId={group.id}
            />
          </div>
        )
      })}

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
