import { Badge } from "@/components/ui/badge"
import { RequirementStatus, TaskStatus, CollabPriority, DefectStatus, DefectSeverity } from "@/api/collaboration"


export function StatusBadge({ status }: { status: string }) {
  let variant: "default" | "secondary" | "destructive" | "outline" = "outline"
  
  switch (status) {
    case RequirementStatus.DONE:
    case TaskStatus.DONE:
    case DefectStatus.CLOSED:
      variant = "default" // Green-ish usually, or primary
      break
    case RequirementStatus.IN_PROGRESS:
    case TaskStatus.IN_PROGRESS:
    case DefectStatus.PROCESSING:
      variant = "secondary" // Blue-ish
      break
    case RequirementStatus.TRIAGE:
    case TaskStatus.TODO:
    case DefectStatus.NEW:
      variant = "outline" // Gray
      break
    case DefectStatus.REJECTED:
    case TaskStatus.CANCELLED:
      variant = "destructive" // Red
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
