import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import {
  CircleCheck,
  CircleSlash,
  Clock,
  Cpu,
  Edit,
  ExternalLink,
  Key,
  Loader2,
  MemoryStick,
  Plus,
  Puzzle,
  ScanSearch,
  Trash2
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { pluginsApi, type AppPlugin } from "@/api/plugins"
import { DataTable } from "@/components/data-table/data-table"
import { EditAppPluginEnvDialog } from "@/components/plugins/edit-app-plugin-env-dialog"
import { InstallPluginDialog } from "@/components/plugins/install-plugin-dialog"
import { PluginResourcePopover } from "@/components/plugins/plugin-resource-popover"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Link } from "react-router-dom"

interface AppPluginsProps {
  appId: string
  projectId: string
  readOnly?: boolean
}

export function AppPlugins({ appId, projectId, readOnly = false }: AppPluginsProps) {
  const [installOpen, setInstallOpen] = React.useState(false)
  const [pluginToUninstall, setPluginToUninstall] = React.useState<string | null>(null)
  const [editEnvPlugin, setEditEnvPlugin] = React.useState<AppPlugin | null>(null)
  const queryClient = useQueryClient()

  const { data: appPlugins = [], isLoading } = useQuery({
    queryKey: ["app-plugins", appId],
    queryFn: () => pluginsApi.listAppPlugins(appId),
  })

  const toggleMutation = useMutation({
    mutationFn: ({ pluginId, enabled }: { pluginId: string; enabled: boolean }) =>
      pluginsApi.togglePlugin(appId, pluginId, enabled),
    onSuccess: () => {
      toast.success("Plugin status updated")
      queryClient.invalidateQueries({ queryKey: ["app-plugins", appId] })
    },
    onError: (error: any) => {
      toast.error("Failed to update plugin status", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const uninstallMutation = useMutation({
    mutationFn: (pluginId: string) => pluginsApi.uninstallPlugin(appId, pluginId),
    onSuccess: () => {
      toast.success("Plugin uninstalled")
      queryClient.invalidateQueries({ queryKey: ["app-plugins", appId] })
      setPluginToUninstall(null)
    },
    onError: (error: any) => {
      toast.error("Failed to uninstall plugin", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const columns: ColumnDef<AppPlugin>[] = [
    {
      accessorKey: "plugin.name",
      header: "Plugin",
      cell: ({ row }) => {
        const ap = row.original
        return (
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <Link to={`/plugins`} target="_blank" rel="noopener noreferrer" className="font-medium text-sm flex items-center whitespace-nowrap">
                <ExternalLink className="h-3.5 w-3.5 mr-2" />
                {ap.plugin.name}
              </Link>
              <Badge
                variant={ap.plugin.plugin_type === "init" ? "default" : "secondary"}
                className="text-[9px] px-1.5 py-0 uppercase"
              >
                {ap.plugin.plugin_type}
              </Badge>
            </div>
            <span className="text-xs text-muted-foreground line-clamp-1">
              {ap.plugin.description}
            </span>
          </div>
        )
      },
    },
    {
      accessorKey: "enabled",
      header: "Status",
      cell: ({ row }) => {
        const enabled = row.original.enabled
        return (
          <ColorBadge color={enabled ? "green" : "red"} className="text-[10px]">
            {enabled ? "Enabled" : "Disabled"}
          </ColorBadge>
        )
      },
    },
    {
      id: "resources",
      header: "Resources",
      cell: ({ row }) => {
        const ap = row.original
        return (
          <div className="flex flex-col gap-1 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <Cpu className="h-3 w-3" />
              <span>
                {ap.request_cpu || 0} / {ap.limit_cpu || 0} m
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <MemoryStick className="h-3 w-3" />
              <span>
                {ap.request_memory || 0} / {ap.limit_memory || 0} Mi
              </span>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "created_at",
      header: "Installed",
      cell: ({ row }) => {
        return (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            {new Date(row.original.created_at).toLocaleDateString()}
          </div>
        )
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const ap = row.original
        if (readOnly) return null

        return (
          <div className="flex justify-end gap-1">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => setEditEnvPlugin(ap)}
              className="text-muted-foreground hover:text-foreground"
              title="Edit Environment Variables"
            >
              <Key />
              <span className="sr-only">Edit Env</span>
            </Button>

            <PluginResourcePopover appId={appId} appPlugin={ap}>
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-muted-foreground hover:text-foreground"
                title="Edit Resources"
              >
                <Edit />
                <span className="sr-only">Edit Resources</span>
              </Button>
            </PluginResourcePopover>

            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() =>
                toggleMutation.mutate({
                  pluginId: ap.plugin_id,
                  enabled: !ap.enabled,
                })
              }
              disabled={toggleMutation.isPending}
              className="text-muted-foreground hover:text-foreground"
              title={ap.enabled ? "Disable" : "Enable"}
            >
              {ap.enabled ? (
                <CircleSlash />
              ) : (
                <CircleCheck />
              )}
              <span className="sr-only">{ap.enabled ? "Disable" : "Enable"}</span>
            </Button>

            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => setPluginToUninstall(ap.plugin_id)}
              disabled={uninstallMutation.isPending}
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              title="Uninstall"
            >
              <Trash2 />
              <span className="sr-only">Uninstall</span>
            </Button>
          </div>
        )
      },
    },
  ]

  if (isLoading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div className="space-y-1">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-4 w-48" />
          </div>
          <Skeleton className="h-9 w-28" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <div className="space-y-1">
          <CardTitle className="text-sm flex items-center gap-2">
            <Puzzle className="h-4 w-4" /> Installed Plugins
          </CardTitle>
          <CardDescription>
            Enhance your application with various features and integrations
          </CardDescription>
        </div>
        {!readOnly && appPlugins.length > 0 && (
          <Button onClick={() => setInstallOpen(true)}>
            <Plus />
            Install Plugin
          </Button>
        )}
      </CardHeader>
      <CardContent>
        {appPlugins.length === 0 ? (
          <EmptyState
            title="No plugins installed"
            description="Browse available plugins to extend your application capabilities."
            icon={Puzzle}
            actionText="Browse Plugins"
            onAction={() => setInstallOpen(true)}
            actionIcon={ScanSearch}
          />
        ) : (
          <DataTable columns={columns} data={appPlugins} />
        )}
      </CardContent>

      <InstallPluginDialog
        appId={appId}
        projectId={projectId}
        open={installOpen}
        onOpenChange={setInstallOpen}
        installedPluginIds={appPlugins.map((p) => p.plugin_id)}
      />

      <EditAppPluginEnvDialog
        appId={appId}
        appPlugin={editEnvPlugin}
        open={!!editEnvPlugin}
        onOpenChange={(open) => !open && setEditEnvPlugin(null)}
      />

      <AlertDialog open={!!pluginToUninstall} onOpenChange={(open) => !open && setPluginToUninstall(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Uninstall Plugin?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the plugin and its configuration from your application.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => pluginToUninstall && uninstallMutation.mutate(pluginToUninstall)}
            >
              {uninstallMutation.isPending ? (
                <Loader2 className="animate-spin" />
              ) : (
                "Uninstall"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
