import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, useCallback } from 'react'
import { notificationsApi } from '@/api/notifications'

const UNREAD_COUNT_KEY = ['notifications', 'unread-count']
const NOTIFICATIONS_LIST_KEY = ['notifications', 'list']

export function useNotifications() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const queryClient = useQueryClient()

  const { data: unreadData } = useQuery({
    queryKey: UNREAD_COUNT_KEY,
    queryFn: notificationsApi.getUnreadCount,
    refetchInterval: 30000,
  })

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_KEY })
    queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_LIST_KEY })
  }, [queryClient])

  return {
    unreadCount: unreadData?.count ?? 0,
    dialogOpen,
    setDialogOpen,
    invalidate,
  }
}

export { NOTIFICATIONS_LIST_KEY, UNREAD_COUNT_KEY }
