import { type ActionMetadata } from "@/api/apps"
import { AppActionIcons } from "@/components/applications/app-action-icons"
import * as React from "react"

interface AppActionIconsWrapperProps {
  appId: string
  envId: string
  actions: ActionMetadata[]
  appGroups?: Array<{ id: string; name: string }>
  currentGroupId?: string
  onMoveToGroup?: (groupId: string) => void
  onRemoveFromGroup?: () => void
  onActionInteractionChange?: (appId: string, active: boolean) => void
}

export function AppActionIconsWrapper({
  appId,
  envId,
  actions,
  appGroups,
  currentGroupId,
  onMoveToGroup,
  onRemoveFromGroup,
  onActionInteractionChange,
}: AppActionIconsWrapperProps) {
  const handleInteractionChange = React.useCallback((active: boolean) => {
    onActionInteractionChange?.(appId, active)
  }, [appId, onActionInteractionChange])

  if (!actions || actions.length === 0) {
    return null
  }

  return (
    <AppActionIcons
      appId={appId}
      envId={envId}
      actions={actions}
      appGroups={appGroups}
      currentGroupId={currentGroupId}
      onMoveToGroup={onMoveToGroup}
      onRemoveFromGroup={onRemoveFromGroup}
      onInteractionChange={handleInteractionChange}
    />
  )
}
