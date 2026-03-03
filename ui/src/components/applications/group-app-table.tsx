import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { appGroupsApi } from '@/api/app-groups'
import { ColorBadge } from '@/components/shared/color-badge'
import { getAppStatusColor } from '@/lib/app-status'

interface AppRow { id: string; slug: string; name: string; status: string }
interface Props {
  apps: AppRow[]
  groupId: string | null
  projectId: string
  envId: string
}

export function GroupAppTable({ apps, groupId, projectId, envId }: Props) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const removeMutation = useMutation({
    mutationFn: ({ groupId, appId }: { groupId: string; appId: string }) =>
      appGroupsApi.removeApp(groupId, appId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grouped-apps', projectId, envId] })
      toast.success('Removed from group')
    },
    onError: () => toast.error('Failed to remove from group'),
  })

  if (apps.length === 0) {
    return <p className="text-xs text-muted-foreground py-2 px-1">No apps in this group.</p>
  }

  return (
    <div className="border rounded-md overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40">
            <th className="text-left px-3 py-2 font-medium text-xs text-muted-foreground">Name</th>
            <th className="text-left px-3 py-2 font-medium text-xs text-muted-foreground">Status</th>
            {groupId && <th className="px-3 py-2" />}
          </tr>
        </thead>
        <tbody>
          {apps.map(app => (
            <tr key={app.id} className="border-b last:border-0 hover:bg-muted/20 transition-colors">
              <td className="px-3 py-2">
                <span
                  className="font-medium cursor-pointer hover:text-primary transition-colors"
                  onClick={() => navigate(`/applications/${app.id}`)}
                >
                  {app.name}
                </span>
                <span className="ml-2 text-xs text-muted-foreground font-mono">{app.slug}</span>
              </td>
              <td className="px-3 py-2">
                <ColorBadge color={getAppStatusColor(app.status)}>
                  {app.status.toUpperCase()}
                </ColorBadge>
              </td>
              {groupId && (
                <td className="px-3 py-2 text-right">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => removeMutation.mutate({ groupId: groupId!, appId: app.id })}
                    title="Remove from group"
                  >
                    <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                  </Button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
