import { useQuery } from "@tanstack/react-query"
import { ArrowDownToLine, Blocks, Loader2 } from "lucide-react"
import * as React from "react"

import { clustersApi, type Extension } from "@/api/clusters"
import { InstallExtensionDialog } from "@/components/cluster/install-extension-dialog"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { ColorBadge } from "../shared/color-badge"

interface BrowseExtensionsDialogProps {
  clusterId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  // Extension IDs already installed on this cluster (to filter them out)
  installedExtensionIds: string[]
}

export function BrowseExtensionsDialog({
  clusterId,
  open,
  onOpenChange,
  installedExtensionIds,
}: BrowseExtensionsDialogProps) {
  const [search, setSearch] = React.useState("")
  const [installTarget, setInstallTarget] =
    React.useState<Extension | null>(null)
  const [installOpen, setInstallOpen] = React.useState(false)

  // Fetch all extensions
  const { data: extension = [], isLoading } = useQuery({
    queryKey: ["extensions"],
    queryFn: () => clustersApi.listExtensions(),
    enabled: open,
  })

  const safeItems: Extension[] = Array.isArray(extension)
    ? extension
    : []

  // Filter: exclude already installed, apply search
  const filteredItems = safeItems.filter((item) => {
    const matchesSearch =
      (item.display_name || item.name)
        .toLowerCase()
        .includes(search.toLowerCase()) ||
      (item.description ?? "").toLowerCase().includes(search.toLowerCase())
    const notInstalled = !installedExtensionIds.includes(item.id)
    return matchesSearch && notInstalled
  })

  const handleInstallClick = (item: Extension) => {
    setInstallTarget(item)
    setInstallOpen(true)
  }

  const handleInstallDialogClose = (open: boolean) => {
    setInstallOpen(open)
    if (!open) {
      // Close browse dialog too after a successful install
      onOpenChange(false)
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-2xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>Install Extension</DialogTitle>
            <DialogDescription>
              Browse available extensions and install them to this cluster.
            </DialogDescription>
          </DialogHeader>

          <Input
            placeholder="Search extensions..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="shrink-0"
          />

          <div className="flex-1 overflow-y-auto min-h-60">
            {isLoading ? (
              <div className="flex items-center justify-center h-full py-12">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : filteredItems.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full py-12 text-center">
                <p className="text-sm font-medium text-muted-foreground">
                  {search
                    ? "No matching extensions found"
                    : "All available extensions are already installed"}
                </p>
              </div>
            ) : (
              <div className="grid gap-3">
                {filteredItems.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex items-start gap-3 min-w-0 flex-1">
                      <div className="p-1.5 bg-blue-500/10 rounded-md text-blue-600 shrink-0 mt-0.5">
                        <Blocks className="h-4 w-4" />
                      </div>
                      <div className="min-w-0 space-y-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <h4 className="font-medium leading-none">
                            {item.display_name || item.name}
                          </h4>
                          {item.builtin ? (
                            <ColorBadge color="purple" className="text-[10px]">
                              Built-in
                            </ColorBadge>
                          ) : (
                            <ColorBadge color="blue" className="text-[10px]">
                              Custom
                            </ColorBadge>
                          )}
                        </div>
                        {item.description && (
                          <p className="text-xs text-muted-foreground line-clamp-2">
                            {item.description}
                          </p>
                        )}
                        <p className="text-[10px] text-muted-foreground font-mono truncate">
                          {item.oci_url}
                        </p>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      className="ml-4 shrink-0"
                      onClick={() => handleInstallClick(item)}
                    >
                      <ArrowDownToLine className="h-3.5 w-3.5" />
                      Install
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Step 2: configure and confirm the install */}
      <InstallExtensionDialog
        open={installOpen}
        onOpenChange={handleInstallDialogClose}
        clusterId={clusterId}
        extension={installTarget}
      />
    </>
  )
}
