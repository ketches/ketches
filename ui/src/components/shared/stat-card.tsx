import * as React from "react"
import type { LucideIcon } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

interface StatCardProps {
  title: string
  value: string | number
  icon?: LucideIcon
  description?: React.ReactNode
  onClick?: () => void
  className?: string
}

export function StatCard({
  title,
  value,
  icon: Icon,
  description,
  onClick,
  className,
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
        {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
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
