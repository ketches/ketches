import { Activity } from "lucide-react"

import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"

export function ActivityPage() {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader items={[{ label: "Activity", icon: Activity }]} />
      
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Activity</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Track your recent actions and project events.
          </p>
        </div>
      </div>

      <div className="flex flex-1 items-center justify-center min-h-[400px]">
        <EmptyState
          title="No activity yet"
          description="Your project activity and recent actions will appear here."
          icon={Activity}
        />
      </div>
    </div>
  )
}
