import { type ActionMetadata } from "@/api/apps"
import { AppActionIcons } from "@/components/applications/app-action-icons"

interface AppActionIconsWrapperProps {
  appId: string
  envId: string
  actions: ActionMetadata[]
  appGroups?: Array<{ id: string; name: string }>
  currentGroupId?: string
  onMoveToGroup?: (groupId: string) => void
  onRemoveFromGroup?: () => void
}

export function AppActionIconsWrapper({ appId, envId, actions, appGroups, currentGroupId, onMoveToGroup, onRemoveFromGroup }: AppActionIconsWrapperProps) {
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
    />
  )
}
