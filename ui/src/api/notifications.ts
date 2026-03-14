import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

export interface Notification {
  id: string
  sender_id: string
  sender_name: string
  category: 'invitation' | 'assignment' | 'info'
  event_type: string
  title: string
  message: string
  status: 'pending' | 'accepted' | 'refused' | 'read' | 'dismissed'
  read_at?: string
  resource_type: string
  resource_id: string
  project_id: string
  project_name: string
  action_data: string
  created_at: string
}

export type NotificationAction = 'accept' | 'refuse' | 'read' | 'dismiss'

export const notificationsApi = {
  list: async (params?: PaginationParams) => {
    return client.get('/v1/notifications', { params }) as Promise<{
      items: Notification[]
      pagination: PaginationResponse
    }>
  },

  getUnreadCount: async () => {
    return client.get('/v1/notifications/unread-count') as Promise<{ count: number }>
  },

  handleAction: async (id: string, action: NotificationAction) => {
    return client.post(`/v1/notifications/${id}/action`, { action })
  },

  markAllRead: async () => {
    return client.post('/v1/notifications/read-all')
  },
}
