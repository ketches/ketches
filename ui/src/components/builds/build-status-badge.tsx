import type { BuildStatus } from "@/api/builds"
import { buildStatusLabels } from "@/api/builds"
import { ColorBadge } from "@/components/shared/color-badge"
import { Ban, CheckCircle2, Clock, FolderGit2, Loader2, XCircle } from "lucide-react"

const statusBuildSetting: Record<BuildStatus, { icon: React.ElementType; color: "blue" | "green" | "sky" | "purple" | "red" | "yellow" | "orange" | "gray" }> = {
  pending: { icon: Clock, color: "gray" },
  cloning: { icon: FolderGit2, color: "sky" },
  building: { icon: Loader2, color: "blue" },
  succeeded: { icon: CheckCircle2, color: "green" },
  deployed: { icon: CheckCircle2, color: "green" },
  failed: { icon: XCircle, color: "red" },
  cancelled: { icon: Ban, color: "gray" },
  unknown: { icon: Ban, color: "gray" },
}

export function BuildStatusBadge({ status, className }: { status: BuildStatus, className?: string }) {
  const setting = statusBuildSetting[status] || statusBuildSetting.pending
  const Icon = setting.icon
  const isAnimated = status === 'cloning' || status === 'building' || status === 'pending'

  return (
    <ColorBadge color={setting.color} className={`gap-1 ${className || ''}`}>
      <Icon className={`h-3 w-3 ${isAnimated ? 'animate-spin' : ''}`} />
      {buildStatusLabels[status] || status}
    </ColorBadge>
  )
}
