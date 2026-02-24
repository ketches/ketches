import type { BuildStatus } from "@/api/builds"
import { buildStatusLabels } from "@/api/builds"
import { ColorBadge } from "@/components/shared/color-badge"
import { Ban, CheckCircle2, Clock, FolderGit2, Loader2, XCircle } from "lucide-react"

const statusConfig: Record<BuildStatus, { icon: React.ElementType; color: "blue" | "green" | "sky" | "purple" | "red" | "yellow" | "orange" | "gray" }> = {
  pending: { icon: Clock, color: "gray" },
  cloning: { icon: FolderGit2, color: "sky" },
  building: { icon: Loader2, color: "blue" },
  succeeded: { icon: CheckCircle2, color: "green" },
  failed: { icon: XCircle, color: "red" },
  cancelled: { icon: Ban, color: "gray" },
}

export function BuildStatusBadge({ status }: { status: BuildStatus }) {
  const config = statusConfig[status] || statusConfig.pending
  const Icon = config.icon
  const isAnimated = status === 'cloning' || status === 'building' || status === 'pending'

  return (
    <ColorBadge color={config.color} className="gap-1">
      <Icon className={`h-3 w-3 ${isAnimated ? 'animate-spin' : ''}`} />
      {buildStatusLabels[status] || status}
    </ColorBadge>
  )
}
