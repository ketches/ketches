import { Bell } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useNotifications } from '@/hooks/use-notifications'
import { NotificationDialog } from '@/components/notifications/notification-dialog'

export function NotificationBell() {
  const { unreadCount, dialogOpen, setDialogOpen } = useNotifications()

  return (
    <>
      <Button
        variant="ghost"
        size="icon-sm"
        onClick={() => setDialogOpen(true)}
        className="relative"
      >
        <Bell className="size-4" />
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-destructive px-1 text-[0.5rem] font-medium text-white">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </Button>
      <NotificationDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </>
  )
}
