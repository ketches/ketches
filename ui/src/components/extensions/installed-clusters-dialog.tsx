import { useQuery } from "@tanstack/react-query"
import { Blocks, ExternalLink, ShipWheel } from "lucide-react"
import { useNavigate } from "react-router-dom"

import { clustersApi, type Extension, type InstalledCluster } from "@/api/clusters"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface InstalledClustersDialogProps {
  extension: Extension | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function InstalledClustersDialog({
  extension,
  open,
  onOpenChange,
}: InstalledClustersDialogProps) {
  const navigate = useNavigate()

  const { data: clusters = [], isLoading } = useQuery({
    queryKey: ["extension-installed-clusters", extension?.id],
    queryFn: () => clustersApi.getExtensionInstalledClusters(extension!.id),
    enabled: !!extension && open,
  })

  const safeCluster: InstalledCluster[] = Array.isArray(clusters) ? clusters : []

  const handleNavigateToCluster = (clusterId: string) => {
    onOpenChange(false)
    navigate(`/clusters/${clusterId}?tab=extensions`)
  }

  const statusColor = (status: string) => {
    if (status === "deployed") return "green" as const
    if (status === "failed") return "red" as const
    if (status === "pending") return "orange" as const
    return "blue" as const
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Installed Clusters</DialogTitle>
          <DialogDescription>
            Clusters where{" "}
            <span className="font-medium">
              {extension?.display_name || extension?.name}
            </span>{" "}
            is installed
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto min-h-40 space-y-2">
          {isLoading ? (
            <div className="flex items-center justify-center py-10">
              <span className="text-sm text-muted-foreground animate-pulse">
                Loading clusters...
              </span>
            </div>
          ) : safeCluster.length === 0 ? (
            <EmptyState
              title="Not installed anywhere"
              description="This extension has not been installed on any cluster yet."
              icon={Blocks}
            />
          ) : (
            safeCluster.map((c) => (
              <div
                key={c.cluster_id}
                className="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <div className="p-1.5 bg-sky-500/10 rounded-md text-sky-600 shrink-0">
                    <ShipWheel className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium truncate">{c.cluster_name}</span>
                      <ColorBadge
                        color={statusColor(c.status)}
                      >
                        {c.status?.toUpperCase() || "UNKNOWN"}
                      </ColorBadge>
                    </div>
                    <p className="text-xs text-muted-foreground font-mono truncate">
                      {c.release_name} · {c.namespace}
                      {c.version ? ` · ${c.version}` : ""}
                    </p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleNavigateToCluster(c.cluster_id)}
                >
                  <ExternalLink className="h-3.5 w-3.5 mr-1" />
                  View
                </Button>
              </div>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
