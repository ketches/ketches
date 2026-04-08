import { cn } from "@/lib/utils"

import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

export function BreadcrumbSkeleton() {
  return <Skeleton className="h-4 w-40" />
}

export function PageHeadingSkeleton({
  titleWidth = "w-56",
  descriptionWidth = "w-72",
}: {
  titleWidth?: string
  descriptionWidth?: string
}) {
  return (
    <div className="space-y-2">
      <Skeleton className={cn("h-8", titleWidth)} />
      <Skeleton className={cn("h-4", descriptionWidth)} />
    </div>
  )
}

export function DetailHeroSkeleton({
  showBadge = false,
  actions = 2,
}: {
  showBadge?: boolean
  actions?: number
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-between items-start gap-4">
        <div className="flex items-center gap-4">
          <Skeleton className="h-14 w-14 rounded-lg" />
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Skeleton className="h-8 w-56" />
              {showBadge && <Skeleton className="h-6 w-20 rounded-full" />}
            </div>
            <Skeleton className="h-4 w-72" />
          </div>
        </div>
        <div className="flex flex-wrap items-center justify-end gap-2">
          {Array.from({ length: actions }).map((_, index) => (
            <Skeleton key={index} className="h-9 w-24" />
          ))}
        </div>
      </div>
    </div>
  )
}

export function TabsSkeleton({ count = 4 }: { count?: number }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {Array.from({ length: count }).map((_, index) => (
        <Skeleton
          key={index}
          className={cn(
            "h-9 rounded-md",
            index % 3 === 0 ? "w-24" : index % 3 === 1 ? "w-28" : "w-20"
          )}
        />
      ))}
    </div>
  )
}

export function InfoCardSkeleton({
  fields = 6,
  columnsClassName = "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3",
  titleWidth = "w-40",
}: {
  fields?: number
  columnsClassName?: string
  titleWidth?: string
}) {
  return (
    <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
      <CardHeader>
        <CardTitle>
          <Skeleton className={cn("h-5", titleWidth)} />
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className={columnsClassName}>
          {Array.from({ length: fields }).map((_, index) => (
            <div key={index} className="space-y-2">
              <Skeleton className="h-3 w-24" />
              <Skeleton className={cn("h-4", index % 3 === 0 ? "w-28" : index % 3 === 1 ? "w-40" : "w-52")} />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export function StatCardsSkeleton({
  count = 4,
  columnsClassName = "grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4",
}: {
  count?: number
  columnsClassName?: string
}) {
  return (
    <div className={columnsClassName}>
      {Array.from({ length: count }).map((_, index) => (
        <Card key={index}>
          <CardHeader className="space-y-3">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-7 w-20" />
            <Skeleton className="h-3 w-28" />
          </CardHeader>
        </Card>
      ))}
    </div>
  )
}

export function PanelCardSkeleton({
  titleWidth = "w-36",
  descriptionWidth,
  actionWidth,
  contentHeight = "h-72",
  className,
}: {
  titleWidth?: string
  descriptionWidth?: string
  actionWidth?: string
  contentHeight?: string
  className?: string
}) {
  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>
          <Skeleton className={cn("h-5", titleWidth)} />
        </CardTitle>
        {descriptionWidth && (
          <CardDescription>
            <Skeleton className={cn("h-4", descriptionWidth)} />
          </CardDescription>
        )}
        {actionWidth && (
          <CardAction>
            <Skeleton className={cn("h-9", actionWidth)} />
          </CardAction>
        )}
      </CardHeader>
      <CardContent>
        <Skeleton className={cn("w-full rounded-xl bg-muted/10", contentHeight)} />
      </CardContent>
    </Card>
  )
}
