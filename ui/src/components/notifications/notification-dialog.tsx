import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { formatDistanceToNow } from 'date-fns'
import { Bell, Check, Eye, X } from 'lucide-react'
import { useState } from 'react'

import { notificationsApi, type Notification, type NotificationAction } from '@/api/notifications'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { NOTIFICATIONS_LIST_KEY, UNREAD_COUNT_KEY } from '@/hooks/use-notifications'
import { ColorBadge } from '../shared/color-badge'
import { EmptyState } from '../shared/empty-state'

interface NotificationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NotificationDialog({ open, onOpenChange }: NotificationDialogProps) {
  const [page, setPage] = useState(1)
  const pageSize = 10
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: [...NOTIFICATIONS_LIST_KEY, page],
    queryFn: () => notificationsApi.list({ page, page_size: pageSize }),
    enabled: open,
  })

  const actionMutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: NotificationAction }) =>
      notificationsApi.handleAction(id, action),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_LIST_KEY })
      queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_KEY })
    },
  })

  const markAllReadMutation = useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_LIST_KEY })
      queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_KEY })
    },
  })

  const items = data?.items ?? []
  const pagination = data?.pagination
  const totalPages = pagination?.total_pages ?? 1

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[80vh] overflow-y-auto flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between pr-8">
            <DialogTitle className="flex items-center gap-2">
              <Bell className="size-4" />
              Notifications
              {pagination && pagination.total > 0 && (
                <Badge variant="secondary">{pagination.total}</Badge>
              )}
            </DialogTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => markAllReadMutation.mutate()}
              disabled={markAllReadMutation.isPending}
            >
              Mark all read
            </Button>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto -mx-4 px-4 min-h-0">
          {isLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground text-xs">
              Loading...
            </div>
          ) : items.length === 0 ? (
            <EmptyState
              title=""
              description="No notifications."
              icon={Bell}
              border={false}
            />
          ) : (
            <div className="flex flex-col gap-1">
              {items.map((notif) => (
                <NotificationItem
                  key={notif.id}
                  notification={notif}
                  onAction={(action) =>
                    actionMutation.mutate({ id: notif.id, action })
                  }
                  isActing={actionMutation.isPending}
                />
              ))}
            </div>
          )}
        </div>

        {totalPages > 1 && (
          <div className="flex items-center justify-between pt-2 border-t text-xs text-muted-foreground">
            <span>
              Page {page} of {totalPages}
            </span>
            <div className="flex gap-1">
              <Button
                variant="outline"
                size="xs"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                size="xs"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function NotificationItem({
  notification,
  onAction,
  isActing,
}: {
  notification: Notification
  onAction: (action: NotificationAction) => void
  isActing: boolean
}) {
  const isPending = notification.status === 'pending'
  const isInvitation = notification.category === 'invitation'

  const statusBadge = () => {
    switch (notification.status) {
      case 'accepted':
        return <ColorBadge color="green">Accepted</ColorBadge>
      case 'refused':
        return <ColorBadge color="red">Refused</ColorBadge>
      case 'read':
      case 'dismissed':
        return null
      default:
        return null
    }
  }

  return (
    <div
      className={`rounded-md border p-3 text-xs transition-colors ${isPending ? 'bg-accent/50 border-primary/20' : 'bg-background'
        }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-0.5">
            <span className="font-medium truncate">{notification.title}</span>
            {statusBadge()}
          </div>
          <p className="text-muted-foreground mb-1">{notification.message}</p>
          <div className="flex items-center gap-2 text-muted-foreground">
            {notification.sender_name && (
              <span>From {notification.sender_name}</span>
            )}
            {notification.project_name && (
              <>
                <span>&middot;</span>
                <span>{notification.project_name}</span>
              </>
            )}
            <span>&middot;</span>
            <span className='text-[10px]'>
              {formatDistanceToNow(new Date(notification.created_at), {
                addSuffix: true,
              })}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-1 shrink-0">
          {isInvitation && isPending && (
            <>
              <Button
                size="sm"
                onClick={() => onAction('accept')}
                disabled={isActing}
                className="bg-green-50 hover:bg-green-100 text-green-700 dark:bg-green-950 dark:hover:bg-green-900 dark:text-green-300 border-green-200 dark:border-green-800"
              >
                <Check />
                Accept
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onAction('refuse')}
                disabled={isActing}
              >
                <X />
                Refuse
              </Button>
            </>
          )}
          {!isInvitation && isPending && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onAction('read')}
              disabled={isActing}
            >
              <Eye />
              Read
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
