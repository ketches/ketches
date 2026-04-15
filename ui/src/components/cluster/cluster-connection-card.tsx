import type { Cluster } from "@/api/clusters"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { formatDate } from "@/lib/utils"
import { Info, Pencil } from "lucide-react"
import * as React from "react"
import { EditClusterKubeConfigDialog } from "./edit-cluster-kube-config-dialog"

interface ClusterConnectionCardProps {
  cluster: Cluster
}

export function ClusterConnectionCard({ cluster }: ClusterConnectionCardProps) {
  const [dialogOpen, setDialogOpen] = React.useState(false)

  return (
    <>
      <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent group/card">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Info className="h-4 w-4" />
            Connection
          </CardTitle>
          <CardDescription>
            Manage kubeconfig-derived connection details and the cluster gateway host.
          </CardDescription>
          <CardAction className="opacity-0 transition-opacity group-hover/card:opacity-100 group-focus-within/card:opacity-100">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label="Edit connection settings"
              onClick={() => setDialogOpen(true)}
            >
              <Pencil />
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-[11px] font-medium text-muted-foreground">API Server</p>
            <p className="text-sm font-mono">{cluster.api_server || "Unavailable"}</p>
          </div>
          <div>
            <p className="text-[11px] font-medium text-muted-foreground">Gateway Host</p>
            <p className="text-sm font-mono">{cluster.gateway_host || "Not configured"}</p>
          </div>
          <div>
            <p className="text-[11px] font-medium text-muted-foreground">KubeConfig</p>
            <p className="text-sm font-medium">{cluster.has_kube_config ? "Configured" : "Not configured"}</p>
          </div>
          <div>
            <p className="text-[11px] font-medium text-muted-foreground">Last Checked</p>
            <p className="text-sm">{cluster.last_checked_at ? formatDate(cluster.last_checked_at) : "Never"}</p>
          </div>
        </CardContent>
      </Card>

      <EditClusterKubeConfigDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        cluster={cluster}
      />
    </>
  )
}
