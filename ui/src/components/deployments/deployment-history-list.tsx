import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Clock, History, Package, RotateCcw } from "lucide-react"
import { toast } from "sonner"

import { deploymentHistoryApi, type DeploymentHistory } from "@/api/deployment-history"
import { EmptyState } from "@/components/shared/empty-state"
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
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
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
  const [currentPage, setCurrentPage] = React.useState(1)
  const itemsPerPage = 10

  const { data: response, isLoading } = useQuery({
    queryKey: ["deployment-history", appId, currentPage, itemsPerPage],
    queryFn: () => deploymentHistoryApi.list(appId, currentPage, itemsPerPage),
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
            <div className="border-y border-x-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Time</TableHead>
                    <TableHead>Image</TableHead>
                    <TableHead>Replicas</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Reason</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(histories || []).map((history: DeploymentHistory) => (
                    <TableRow key={history.id}>
                      <TableCell>
                        <span className="flex items-center gap-1 text-xs text-muted-foreground">
                          <Clock className="h-3 w-3" />
                          {formatDate(history.created_at)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col gap-0.5">
                          <span className="text-xs text-muted-foreground line-through">
                            {history.image_before}
                          </span>
                          <span className="text-xs font-mono">{history.image_after}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm">
                          {history.replicas_before} → {history.replicas_after}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs capitalize px-2 py-1 rounded-full bg-primary/10 text-primary">
                          {history.deploy_type}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">{history.reason}</span>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          onClick={() => {
                            setSelectedHistory(history)
                            setShowRollbackDialog(true)
                          }}
                          title="Rollback to this deployment"
                        >
                          <RotateCcw className="h-3 w-3" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
          {totalCount > itemsPerPage && (
            <div className="pt-4">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                      className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                    />
                  </PaginationItem>
                  {Array.from({ length: Math.ceil(totalCount / itemsPerPage) }, (_, i) => i + 1).map((page) => (
                    <PaginationItem key={page}>
                      <PaginationLink
                        onClick={() => setCurrentPage(page)}
                        isActive={currentPage === page}
                        className="cursor-pointer"
                      >
                        {page}
                      </PaginationLink>
                    </PaginationItem>
                  ))}
                  <PaginationItem>
                    <PaginationNext
                      onClick={() => setCurrentPage(Math.min(Math.ceil(totalCount / itemsPerPage), currentPage + 1))}
                      className={currentPage >= Math.ceil(totalCount / itemsPerPage) ? "pointer-events-none opacity-50" : "cursor-pointer"}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
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
              {rollbackMutation.isPending && <RotateCcw className="mr-2 h-4 w-4 animate-spin" />}
              Rollback
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
