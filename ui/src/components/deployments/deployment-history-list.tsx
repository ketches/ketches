import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, History, Package, RotateCcw } from "lucide-react"
import { toast } from "sonner"

import { deploymentHistoryApi, type DeploymentHistory } from "@/api/deployment-history"
import { EmptyState } from "@/components/shared/empty-state"
import { DataTable } from "@/components/data-table/data-table"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { formatDate } from "@/lib/utils"
import type { AxiosError } from "axios"
import * as React from "react"

interface DeploymentHistoryListProps {
  appId: string
}

export function DeploymentHistoryList({ appId }: DeploymentHistoryListProps) {
  const queryClient = useQueryClient()
  const [selectedHistory, setSelectedHistory] = React.useState<DeploymentHistory | null>(null)
  const [showRollbackDialog, setShowRollbackDialog] = React.useState(false)
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: response, isLoading } = useQuery({
    queryKey: ["deployment-history", appId, pagination.pageIndex + 1, pagination.pageSize],
    queryFn: () => deploymentHistoryApi.list(appId, pagination.pageIndex + 1, pagination.pageSize),
    refetchInterval: 10000,
  })

  const histories = response?.items ?? []
  const totalCount = response?.pagination.total || 0

  const rollbackMutation = useMutation({
    mutationFn: (historyId: string) => deploymentHistoryApi.rollback(appId, historyId),
    onSuccess: () => {
      toast.success("Rollback successful")
      queryClient.invalidateQueries({ queryKey: ["deployment-history", appId] })
      queryClient.invalidateQueries({ queryKey: ["app", appId] })
      setShowRollbackDialog(false)
      setSelectedHistory(null)
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || "Failed to rollback")
    },
  })

  const columns: ColumnDef<DeploymentHistory>[] = [
    {
      accessorKey: "created_at",
      header: "Time",
      cell: ({ row }) => (
        <span className="flex items-center gap-1 text-xs text-muted-foreground">
          <Clock className="h-3 w-3" />
          {formatDate(row.original.created_at)}
        </span>
      ),
    },
    {
      accessorKey: "image_after",
      header: "Image",
      cell: ({ row }) => (
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground line-through">
            {row.original.image_before}
          </span>
          <span className="text-xs font-mono">{row.original.image_after}</span>
        </div>
      ),
    },
    {
      id: "replicas",
      header: "Replicas",
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.replicas_before} → {row.original.replicas_after}
        </span>
      ),
    },
    {
      accessorKey: "deploy_type",
      header: "Type",
      cell: ({ row }) => (
        <span className="text-xs capitalize px-2 py-1 rounded-full bg-primary/10 text-primary">
          {row.original.deploy_type}
        </span>
      ),
    },
    {
      accessorKey: "reason",
      header: "Reason",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">{row.original.reason}</span>
      ),
    },
    {
      id: "actions",
      header: () => <span className="flex justify-end">Actions</span>,
      cell: ({ row }) => (
        <div className="flex justify-end">
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => {
              setSelectedHistory(row.original)
              setShowRollbackDialog(true)
            }}
            title="Rollback to this deployment"
          >
            <RotateCcw className="h-3 w-3" />
          </Button>
        </div>
      ),
    },
  ]

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <History className="h-4 w-4" />
            Deployment History
          </CardTitle>
          <CardDescription>Track all deployment changes and rollback when needed</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center p-8">
              <Package className="h-6 w-6 animate-spin" />
            </div>
          ) : (histories || []).length === 0 ? (
            <EmptyState
              title="No deployment history yet"
              description="Deployment history will appear here when you update the application"
              icon={History}
            />
          ) : (
            <DataTable
              columns={columns}
              data={histories}
              borderless
              manualPagination
              pagination={pagination}
              onPaginationChange={setPagination}
              totalCount={totalCount}
            />
          )}
        </CardContent>
      </Card>

      <Dialog open={showRollbackDialog} onOpenChange={setShowRollbackDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rollback Deployment</DialogTitle>
            <DialogDescription>
              Are you sure you want to rollback to this deployment? This will restore the previous image and
              configuration.
            </DialogDescription>
          </DialogHeader>
          {selectedHistory && (
            <div className="space-y-2 py-4">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Image:</span>
                <span className="font-mono text-xs">{selectedHistory.image_before}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Replicas:</span>
                <span>{selectedHistory.replicas_before}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">CPU Request:</span>
                <span>{selectedHistory.request_cpu_before}m</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Memory Request:</span>
                <span>{selectedHistory.request_memory_before}Mi</span>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRollbackDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => selectedHistory && rollbackMutation.mutate(selectedHistory.id)}
              disabled={rollbackMutation.isPending}
            >
              {rollbackMutation.isPending && <RotateCcw className="h-4 w-4 animate-spin" />}
              Rollback
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
