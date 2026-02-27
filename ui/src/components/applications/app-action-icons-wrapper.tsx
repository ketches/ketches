import { useQuery } from "@tanstack/react-query"

import { appsApi } from "@/api/apps"
import { AppActionIcons } from "@/components/applications/app-action-icons"

interface AppActionIconsWrapperProps {
  appId: string
  envId: string
}

export function AppActionIconsWrapper({ appId, envId }: AppActionIconsWrapperProps) {
  const { data: availableActions } = useQuery({
    queryKey: ['app-actions', appId],
    queryFn: () => appsApi.getAvailableActions(appId),
    staleTime: 30000,
  })

  if (!availableActions || !availableActions.actions || availableActions.actions.length === 0) {
    return null
  }

  return <AppActionIcons appId={appId} envId={envId} actions={availableActions.actions} />
}
