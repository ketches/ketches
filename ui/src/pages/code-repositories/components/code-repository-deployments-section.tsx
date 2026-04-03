import { type ColumnDef } from "@tanstack/react-table"
import { Clock, Copy, ExternalLink, GitBranch, History, Lock, Globe, GlobeLock, Rocket } from "lucide-react"
import { toast } from "sonner"
import * as React from "react"

import type { BuildDeployment } from "@/api/builds"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { formatDate } from "@/lib/utils"

function DeploymentErrorPopover({ errorMessage }: { errorMessage: string }) {
  const [open, setOpen] = React.useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={<button type="button" className="inline-flex items-center" />}
      >
        <BuildStatusBadge status="failed" />
      </PopoverTrigger>
      <PopoverContent side="top" align="start" className="w-md max-w-[calc(100vw-2rem)] gap-2">
        <p className="text-xs font-medium text-destructive">Deployment failed</p>
        <p className="wrap-break-word whitespace-pre-wrap text-xs text-muted-foreground">{errorMessage}</p>
      </PopoverContent>
    </Popover>
  )
}

interface CodeRepositoryDeploymentsSectionProps {
  deployments: BuildDeployment[]
  isLoading: boolean
  onOpenApplication: (appId: string) => void
}

export function CodeRepositoryDeploymentsSection({
  deployments,
  isLoading,
  onOpenApplication,
}: CodeRepositoryDeploymentsSectionProps) {
  const columns: ColumnDef<BuildDeployment>[] = [
    {
      accessorKey: "build_number",
      header: "Build #",
      cell: ({ row }) => <span className="font-medium">{row.original.build_number}</span>,
    },
    {
      accessorKey: "git_ref",
      header: "Ref",
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <GitBranch className="h-3 w-3" />
          {row.original.git_ref}
        </span>
      ),
    },
    {
      accessorKey: "image_full_name",
      header: "Image",
      cell: ({ row }) => (
        <div className="flex items-center gap-1">
          <span className="block truncate font-mono text-xs" title={row.original.image_full_name}>
            {row.original.image_full_name}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 transition-opacity group-hover/row:opacity-100"
            onClick={(event) => {
              event.stopPropagation()
              navigator.clipboard.writeText(row.original.image_full_name)
              toast.success("Image address copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      id: "environment",
      header: "Environment",
      cell: ({ row }) => row.original.env_name ?? "-",
    },
    {
      id: "application",
      header: "Application",
      cell: ({ row }) => (
        <Button variant="link" className="h-auto p-0 text-xs" onClick={() => onOpenApplication(row.original.app_id)}>
          <ExternalLink />
          {row.original.app_name}
        </Button>
      ),
    },
    {
      accessorKey: "status",
      header: "Deploy Status",
      cell: ({ row }) => {
        if (row.original.status === "failed" && row.original.error_message) {
          return <DeploymentErrorPopover errorMessage={row.original.error_message} />
        }

        return <BuildStatusBadge status={row.original.status} className={row.original.status === "failed" ? "cursor-pointer" : ""} />
      },
    },
    {
      accessorKey: "created_at",
      header: "Deployed At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.created_at)}</span>
        </div>
      ),
    },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <Rocket className="h-4 w-4" />
          Deployment History
        </CardTitle>
        <CardDescription>
          Track when and where builds from this repository were deployed.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {deployments.length === 0 ? (
          <EmptyState
            title="No deployment history"
            description="Deploy a successful build to an environment to see it here."
            icon={History}
          />
        ) : (
          <DataTable
            columns={columns}
            data={deployments}
            isLoading={isLoading}
          />
        )}
      </CardContent>
    </Card>
  )
}
