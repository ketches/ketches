import { CollabPriority, DefectSeverity, DefectStatus, RequirementStatus, TaskStatus } from "@/api/collaboration"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

// Entity type for status color mapping
export type EntityType = "task" | "requirement" | "defect" | "sprint"

// Task status color map
const taskColorMap: Record<string, string> = {
  todo: 'bg-slate-100 text-slate-700 border border-slate-200',
  in_progress: 'bg-blue-100 text-blue-700 border border-blue-200',
  review: 'bg-amber-100 text-amber-700 border border-amber-200',
  done: 'bg-green-100 text-green-700 border border-green-200',
  cancelled: 'bg-red-100 text-red-700 border border-red-200',
}

// Requirement status color map
const requirementColorMap: Record<string, string> = {
  triage: 'bg-purple-100 text-purple-700 border border-purple-200',
  confirmed: 'bg-indigo-100 text-indigo-700 border border-indigo-200',
  in_progress: 'bg-blue-100 text-blue-700 border border-blue-200',
  done: 'bg-green-100 text-green-700 border border-green-200',
  closed: 'bg-slate-100 text-slate-700 border border-slate-200',
}

// Defect status color map
const defectColorMap: Record<string, string> = {
  new: 'bg-red-100 text-red-700 border border-red-200',
  processing: 'bg-orange-100 text-orange-700 border border-orange-200',
  pending_verify: 'bg-amber-100 text-amber-700 border border-amber-200',
  closed: 'bg-green-100 text-green-700 border border-green-200',
  rejected: 'bg-slate-100 text-slate-700 border border-slate-200',
}

// Sprint status color map
const sprintColorMap: Record<string, string> = {
  planned: 'bg-slate-100 text-slate-700 border border-slate-200',
  active: 'bg-green-100 text-green-700 border border-green-200',
  closed: 'bg-gray-100 text-gray-500 border border-gray-200',
}

interface StatusBadgeProps {
  status: string
  entityType?: EntityType
}

export function StatusBadge({ status, entityType }: StatusBadgeProps) {
  // If entityType is provided, use entity-specific color map
  if (entityType) {
    const colorMaps = {
      task: taskColorMap,
      requirement: requirementColorMap,
      defect: defectColorMap,
      sprint: sprintColorMap,
    }
    const colorMap = colorMaps[entityType]
    const statusColorClass = colorMap[status] || 'bg-gray-100 text-gray-700 border border-gray-200'

    return (
      <span className={cn('inline-block uppercase text-[10px] px-2 py-1 rounded opacity-80 hover:opacity-100', statusColorClass)}>
        {status.replace(/_/g, ' ')}
      </span>
    )
  }

  // Fallback to original generic Badge variant logic for backward compatibility
  let variant: "default" | "secondary" | "destructive" | "outline" = "outline"

  switch (status) {
    case RequirementStatus.DONE:
    case TaskStatus.DONE:
    case DefectStatus.CLOSED:
      variant = "default"
      break
    case RequirementStatus.IN_PROGRESS:
    case TaskStatus.IN_PROGRESS:
    case DefectStatus.PROCESSING:
      variant = "secondary"
      break
    case RequirementStatus.TRIAGE:
    case TaskStatus.TODO:
    case DefectStatus.NEW:
      variant = "outline"
      break
    case DefectStatus.REJECTED:
    case TaskStatus.CANCELLED:
      variant = "destructive"
      break
  }

  return (
    <Badge variant={variant} className="uppercase text-[10px]">
      {status.replace(/_/g, " ")}
    </Badge>
  )
}

export function PriorityBadge({ priority }: { priority: string }) {
  let color = "text-muted-foreground"

  switch (priority) {
    case CollabPriority.P0:
      color = "text-red-500 font-bold"
      break
    case CollabPriority.P1:
      color = "text-orange-500 font-medium"
      break
    case CollabPriority.P2:
      color = "text-blue-500"
      break
    case CollabPriority.P3:
      color = "text-green-500"
      break
  }

  return (
    <span className={`text-xs uppercase ${color}`}>
      {priority}
    </span>
  )
}

export function SeverityBadge({ severity }: { severity: string }) {
  let color = "text-muted-foreground"

  switch (severity) {
    case DefectSeverity.CRITICAL:
      color = "text-destructive font-bold"
      break
    case DefectSeverity.HIGH:
      color = "text-orange-600 font-semibold"
      break
    case DefectSeverity.MEDIUM:
      color = "text-yellow-600"
      break
    case DefectSeverity.LOW:
      color = "text-green-600"
      break
  }

  return (
    <span className={`text-xs uppercase ${color}`}>
      {severity}
    </span>
  )
}

// Due date badge with color-coded urgency
export function DueDateBadge({ dueDate }: { dueDate?: string }) {
  if (!dueDate) return <span className="text-xs text-muted-foreground">—</span>

  const now = new Date()
  const due = new Date(dueDate)
  const diffMs = due.getTime() - now.getTime()
  const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))

  let colorClass = "text-muted-foreground"
  if (diffDays < 0) {
    colorClass = "text-red-600 font-medium" // Overdue
  } else if (diffDays <= 3) {
    colorClass = "text-orange-600 font-medium" // Approaching deadline
  }

  const formatted = due.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })

  return (
    <span className={cn('text-xs', colorClass)}>
      {formatted}
    </span>
  )
}
