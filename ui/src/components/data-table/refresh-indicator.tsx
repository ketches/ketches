import { cn } from "@/lib/utils"
import { Loader2, RefreshCw } from "lucide-react"

interface RefreshIndicatorProps {
  className?: string
}

export function RefreshIndicator({ className }: RefreshIndicatorProps) {
  return (
    <div className={cn("flex items-center gap-2 rounded-md border bg-background/95 px-3 py-1.5 text-xs text-muted-foreground shadow-sm", className)}>
      <Loader2 className="h-4 w-4 animate-spin" />
      <span>Refreshing...</span>
    </div>
  )
}

interface RefreshButtonIconProps {
  spinning?: boolean
  className?: string
}

export function RefreshButtonIcon({ spinning = false, className }: RefreshButtonIconProps) {
  return <RefreshCw className={cn(spinning && "animate-spin", className)} />
}
