import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"
import type { LucideIcon } from "lucide-react"
import * as React from "react"

interface StatCardProps {
  title: string
  value: string | number
  icon?: LucideIcon
  description?: React.ReactNode
  onClick?: () => void
  className?: string
  color?: string
}

export function StatCard({
  title,
  value,
  icon: Icon,
  description,
  onClick,
  className,
  color,
}: StatCardProps) {
  return (
    <Card
      className={cn(
        onClick ? "cursor-pointer hover:bg-muted/50 transition-colors" : "",
        className
      )}
      onClick={onClick}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">
          {title}
        </CardTitle>
        <div className={`p-1.5 bg-${color || "blue"}-500/10 rounded-md text-${color || "blue"}-600 shrink-0`}>
          {Icon && <Icon className="h-4 w-4" />}
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {description && (
          <div className="text-xs text-muted-foreground mt-1">
            {description}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
