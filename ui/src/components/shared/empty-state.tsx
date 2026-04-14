import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { cn } from "@/lib/utils"
import { Box, FolderGit2, Orbit, Plus, ShipWheel, Warehouse, type LucideIcon } from "lucide-react"
import * as React from "react"

interface EmptyStateProps {
  title: string
  description?: string | React.ReactNode
  icon?: LucideIcon
  border?: boolean
  actionText?: string
  onAction?: () => void
  actionDisabled?: boolean
  actionIcon?: LucideIcon
  className?: string
}

export function EmptyState({
  title,
  description,
  icon: Icon,
  border = true,
  actionText,
  onAction,
  actionDisabled,
  actionIcon: ActionIcon,
  className,
}: EmptyStateProps) {
  return (
    <Empty className={cn("bg-muted/20", border && "border border-dashed", className)}>
      <EmptyHeader>
        {Icon && (
          <EmptyMedia variant="icon">
            <Icon className="text-muted-foreground" />
          </EmptyMedia>
        )}
        <EmptyTitle>{title}</EmptyTitle>
        {description && <EmptyDescription>{description}</EmptyDescription>}
        {actionText && onAction && (
          <EmptyContent>
            <Button onClick={onAction} disabled={actionDisabled}>
              {ActionIcon && <ActionIcon className="h-4 w-4" />}
              {actionText}
            </Button>
          </EmptyContent>
        )}
      </EmptyHeader>
    </Empty>
  )
}

export function EmptyEnvironmentState({ onAction }: { onAction?: () => void }) {
  return (
    <EmptyState
      title="No environments yet"
      description="Create your first environment to start managing your deployments and applications."
      icon={Orbit}
      actionText="Create Environment"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}

export function EmptyApplicationState({
  onAction,
  environmentName,
  actionDisabled
}: {
  onAction?: () => void,
  environmentName?: string,
  actionDisabled?: boolean
}) {
  return (
    <EmptyState
      title="No applications yet"
      description={environmentName ? `Create your first application in ${environmentName}.` : "Select an environment to create applications."}
      icon={Box}
      actionText="Create Application"
      onAction={onAction}
      actionDisabled={actionDisabled}
      actionIcon={Plus}
    />
  )
}

export function EmptyClusterState({ onAction }: { onAction?: () => void }) {
  return (
    <EmptyState
      title="No clusters yet"
      description="Add your first Kubernetes cluster to start managing your infrastructure."
      icon={ShipWheel}
      actionText="Add Cluster"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}

export function EmptyCodeRepositoryState({ onAction }: { onAction?: () => void }) {
  return (
    <EmptyState
      title="No code repositories yet"
      description="Add your first repository to manage build settings and deployments."
      icon={FolderGit2}
      actionText="Add Repository"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}

export function EmptyRegistryState({ onAction }: { onAction?: () => void }) {
  return (
    <EmptyState
      title="No registries yet"
      description="Add your first registry to support builds and deployments."
      icon={Warehouse}
      actionText="Add Registry"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}
