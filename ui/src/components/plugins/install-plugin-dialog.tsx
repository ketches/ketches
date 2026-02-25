import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ArrowDownToLine, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { pluginsApi } from "@/api/plugins"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"

interface InstallPluginDialogProps {
  appId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  installedPluginIds: string[]
}

export function InstallPluginDialog({
  appId,
  open,
  onOpenChange,
  installedPluginIds,
}: InstallPluginDialogProps) {
  const [search, setSearch] = React.useState("")
  const queryClient = useQueryClient()

  const { data: plugins = [], isLoading } = useQuery({
    queryKey: ["plugins-simple"],
    queryFn: () => pluginsApi.listPluginsSimple(),
  })

  const installMutation = useMutation({
    mutationFn: ({ pluginId, envVars }: { pluginId: string; envVars?: { key: string, value: string }[] }) =>
      pluginsApi.installPlugin(appId, pluginId, envVars),
    onSuccess: () => {
      toast.success("Plugin installed successfully")
      queryClient.invalidateQueries({ queryKey: ["app-plugins", appId] })
      onOpenChange(false)
    },
    onError: (error: any) => {
      toast.error("Failed to install plugin", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const filteredPlugins = plugins.filter((plugin) => {
    const matchesSearch =
      plugin.name.toLowerCase().includes(search.toLowerCase()) ||
      plugin.description.toLowerCase().includes(search.toLowerCase())
    const isNotInstalled = !installedPluginIds.includes(plugin.id)
    return matchesSearch && isNotInstalled
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Install Plugin</DialogTitle>
          <DialogDescription>
            Browse and install plugins to enhance your application.
          </DialogDescription>
        </DialogHeader>

        <Input
          placeholder="Search plugins..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />

        <div className="flex-1 overflow-y-auto min-h-75">
          {isLoading ? (
            <div className="flex items-center justify-center h-full">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : filteredPlugins.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-center p-4">
              <p className="text-sm font-medium text-muted-foreground">
                {search ? "No matching plugins found" : "All available plugins are installed"}
              </p>
            </div>
          ) : (
            <div className="grid gap-4">
              {filteredPlugins.map((plugin) => (
                <div
                  key={plugin.id}
                  className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <h4 className="font-medium leading-none">{plugin.name}</h4>
                      <Badge variant={plugin.plugin_type === "init" ? "default" : "secondary"}>
                        {plugin.plugin_type}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {plugin.description}
                    </p>
                  </div>
                  <Button
                    onClick={() => installMutation.mutate({ pluginId: plugin.id, envVars: plugin.env_vars })}
                    disabled={installMutation.isPending}
                  >
                    {installMutation.isPending ? (
                      <Loader2 className="animate-spin" />
                    ) : (
                      <>
                        <ArrowDownToLine />
                        Install
                      </>
                    )}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
