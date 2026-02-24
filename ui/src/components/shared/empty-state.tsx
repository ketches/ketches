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
import { Box, FileText, FolderGit2, Orbit, Plus, ShipWheel, Warehouse, type LucideIcon } from "lucide-react"
import * as React from "react"

interface EmptyStateProps {
  title: string
  description?: string | React.ReactNode
  icon?: LucideIcon
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
  actionText,
  onAction,
  actionDisabled,
  actionIcon: ActionIcon,
  className,
}: EmptyStateProps) {
  return (
    <Empty className={cn("border border-dashed bg-muted/10", className)}>
      <EmptyHeader>
        {Icon && (
          <EmptyMedia variant="icon">
            <Icon />
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
      title="No environments found"
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
      title="No applications found"
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
      title="No code repositories"
      description="Add a Git repository to manage build configs and deploy to environments."
      icon={FolderGit2}
      actionText="Add Repository"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}

export function EmptyTemplateState({ onAction }: { onAction?: () => void }) {
  return (
    <EmptyState
      title="No templates yet"
      description="Create your first template to define reusable configurations for your resources."
      icon={FileText}
      actionText="Create Template"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}

export function EmptyRegistryState({ onAction }: { onAction?: () => void }) {
  return (
    <EmptyState
      title="No container registries"
      description="Add a container registry to push and pull images for builds and deployments."
      icon={Warehouse}
      actionText="Add Registry"
      onAction={onAction}
      actionIcon={Plus}
    />
  )
}
