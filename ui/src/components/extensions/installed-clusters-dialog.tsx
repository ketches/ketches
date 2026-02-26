import { useQueries, useQuery } from "@tanstack/react-query"
import { ExternalLink, ServerCrash } from "lucide-react"
import { useNavigate } from "react-router-dom"

import { clustersApi, type InstalledExtension } from "@/api/clusters"
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
  open: boolean
  onOpenChange: (open: boolean) => void
  // The extension catalog item name used to match installed extensions by name
  extensionName: string
  extensionDisplayName?: string
}

interface ClusterWithInstall {
  clusterId: string
  clusterName: string
  extension: InstalledExtension
}

// Fetches all clusters, then queries each cluster's extensions to find matches.
// Uses useQueries (not hooks-in-a-loop) to safely run parallel queries.
export function InstalledClustersDialog({
  open,
  onOpenChange,
  extensionName,
  extensionDisplayName,
}: InstalledClustersDialogProps) {
  const navigate = useNavigate()

  const { data: clusters = [], isLoading: clustersLoading } = useQuery({
    queryKey: ["clusters-simple"],
    queryFn: () => clustersApi.listSimple(),
    enabled: open,
  })

  // Use useQueries to safely run one query per cluster in parallel
  const extensionQueries = useQueries({
    queries: clusters.map((cluster) => ({
      queryKey: ["extension-installed", cluster.id, extensionName],
      queryFn: async (): Promise<InstalledExtension | null> => {
        try {
          return await clustersApi.getExtension(cluster.id, extensionName)
        } catch {
          return null
        }
      },
      enabled: open && Boolean(cluster.id) && Boolean(extensionName),
      staleTime: 30 * 1000,
    })),
  })

  const isLoading =
    clustersLoading || extensionQueries.some((q) => q.isLoading)

  const installedList: ClusterWithInstall[] = extensionQueries
    .map((q, i) => {
      const cluster = clusters[i]
      if (!cluster || !q.data) return null
      return {
        clusterId: cluster.id,
        clusterName: cluster.name,
        extension: q.data,
      }
    })
    .filter((x): x is ClusterWithInstall => x !== null)

  const handleNavigate = (clusterId: string) => {
    onOpenChange(false)
    navigate(`/clusters/${clusterId}?tab=extensions`)
  }

  const getStatusColor = (status: string) => {
    if (status === "deployed") return "green"
    if (status === "failed") return "red"
    return "blue"
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Installed Clusters</DialogTitle>
          <DialogDescription>
            Clusters where{" "}
            <span className="font-medium">
              {extensionDisplayName || extensionName}
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
          ) : installedList.length === 0 ? (
            <EmptyState
              title="Not installed anywhere"
              description="This extension has not been installed on any cluster yet."
              icon={ServerCrash}
            />
          ) : (
            installedList.map((item) => (
              <div
                key={item.clusterId}
                className="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 transition-colors"
              >
                <div className="flex flex-col gap-1 min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium truncate">
                      {item.clusterName}
                    </span>
                    <ColorBadge
                      color={getStatusColor(item.extension.status)}
                      className="text-[10px]"
                    >
                      {item.extension.status?.toUpperCase() || "UNKNOWN"}
                    </ColorBadge>
                  </div>
                  <span className="text-xs text-muted-foreground font-mono truncate">
                    {item.extension.chart_version
                      ? `v${item.extension.chart_version}`
                      : ""}
                    {item.extension.release_namespace
                      ? ` · ${item.extension.release_namespace}`
                      : ""}
                  </span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleNavigate(item.clusterId)}
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
