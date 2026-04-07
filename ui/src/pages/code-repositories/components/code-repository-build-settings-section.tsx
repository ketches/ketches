import { type ColumnDef } from "@tanstack/react-table"
import { Hammer, Pencil, Play, Plus, Trash2 } from "lucide-react"

import type { BuildSetting } from "@/api/code-repositories"
import { DataTable } from "@/components/data-table/data-table"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

interface CodeRepositoryBuildSettingsSectionProps {
  buildSettings: BuildSetting[]
  isLoading: boolean
  isViewer: boolean
  onCreate: () => void
  onEdit: (setting: BuildSetting) => void
  onBuild: (setting: BuildSetting) => void
  onDelete: (setting: BuildSetting) => void
}

export function CodeRepositoryBuildSettingsSection({
  buildSettings,
  isLoading,
  isViewer,
  onCreate,
  onEdit,
  onBuild,
  onDelete,
}: CodeRepositoryBuildSettingsSectionProps) {
  const columns: ColumnDef<BuildSetting>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      accessorKey: "git_ref",
      header: "Ref",
    },
    {
      accessorKey: "dockerfile_path",
      header: "Dockerfile",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.dockerfile_path}</span>,
    },
    {
      accessorKey: "build_context",
      header: "Context",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.build_context}</span>,
    },
    {
      accessorKey: "image_name",
      header: "Image Name",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.image_name}</span>,
    },
    {
      id: "registry",
      header: "Registry",
      cell: ({ row }) => row.original.registry?.name ?? row.original.registry_id,
    },
  ]

  if (!isViewer) {
    columns.push({
      id: "actions",
      header: () => <span className="flex justify-end">Actions</span>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={<Button variant="ghost" size="icon-sm" onClick={() => onEdit(row.original)} />}
            >
              <Pencil />
            </TooltipTrigger>
            <TooltipContent>Edit build setting</TooltipContent>
          </Tooltip>
          <Button variant="outline" size="sm" onClick={() => onBuild(row.original)}>
            <Play />
            Build
          </Button>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                  onClick={() => onDelete(row.original)}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>Delete build setting</TooltipContent>
          </Tooltip>
        </div>
      ),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm">
          <Hammer className="h-4 w-4" />
          Build Settings
        </CardTitle>
        <CardDescription>
          One repo can have multiple build settings (for example frontend and backend).
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <DataTable
          columns={columns}
          data={buildSettings}
          sourceDataCount={buildSettings.length}
          isLoading={isLoading}
          sourceEmptyContent={(
            <EmptyState
              title="No build settings"
              description="Add a build setting to start building images from this repository."
              icon={Hammer}
              actionText={!isViewer ? "Create build setting" : undefined}
              onAction={!isViewer ? onCreate : undefined}
              actionIcon={!isViewer ? Plus : undefined}
            />
          )}
          useStandaloneEmptyState
          rightToolbar={!isViewer ? () => (
            <Button onClick={onCreate}>
              <Plus />
              Create
            </Button>
          ) : undefined}
        />
      </CardContent>
    </Card>
  )
}
