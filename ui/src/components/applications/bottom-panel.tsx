import { useQuery } from "@tanstack/react-query"
import {
  ChevronDown,
  ChevronUp,
  FileText,
  FolderOpen,
  Layers2,
  Maximize2,
  Minimize2,
  Terminal,
  X,
  Zap,
} from "lucide-react"
import * as React from "react"

import { appsApi } from "@/api/apps"
import { clustersApi, type K8sNode } from "@/api/clusters"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useSidebar } from "@/components/ui/sidebar"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useBottomPanel } from "@/contexts/bottom-panel-context"
import { cn } from "@/lib/utils"

import { FileExplorerPanel } from "./file-explorer-panel"
import { LogPanel } from "./log-panel"
import { TerminalPanel } from "./terminal-panel"

export function BottomPanel() {
  const {
    isOpen,
    isMinimized,
    panelState,
    closePanel,
    toggleMinimize,
    switchType,
    switchContainer,
    switchInstance,
  } = useBottomPanel()

  const { state: sidebarState, isMobile } = useSidebar()
  const [isMaximized, setIsMaximized] = React.useState(false)

  const { data: instances = [] } = useQuery({
    queryKey: ["app-instances", panelState?.appId],
    queryFn: () => appsApi.listInstances(panelState!.appId),
    enabled: !!panelState?.appId && panelState.targetType !== "node",
  })

  const { data: nodes = [] } = useQuery<K8sNode[]>({
    queryKey: ["cluster-nodes", panelState?.appId],
    queryFn: () => clustersApi.listNodes(panelState!.appId),
    enabled: !!panelState?.appId && panelState.targetType === "node",
  })

  const isNode = panelState?.targetType === "node"

  React.useEffect(() => {
    if (isNode && panelState?.type !== "terminal") {
      switchType("terminal")
    }
  }, [isNode, panelState?.type, switchType])

  const handleToggleMaximize = React.useCallback(() => {
    setIsMaximized(prev => !prev)
  }, [])

  const handleInstanceChange = React.useCallback((value: string | null) => {
    if (!value || !panelState) return

    if (panelState.targetType === "node") {
      switchInstance(value, ["shell"])
      return
    }

    const safeInstances = Array.isArray(instances) ? instances : []
    const instance = safeInstances.find(i => i.instanceName === value)
    if (instance) {
      switchInstance(value, instance.containers || [panelState.containerName], instance.initContainers)
    }
  }, [panelState, instances, switchInstance])

  if (!isOpen || !panelState) {
    return null
  }

  const safeInstances = Array.isArray(instances) ? instances : []
  const safeNodes = Array.isArray(nodes) ? nodes : []

  const items = isNode
    ? safeNodes.map(n => ({ name: n.metadata.name }))
    : safeInstances.map(i => ({ name: i.instanceName }))

  const panelHeight = isMinimized ? "h-12" : isMaximized ? "h-[100vh]" : "h-[45vh]"
  const sidebarWidth = isMobile ? "0rem" : sidebarState === "collapsed" ? "3rem" : "16rem"

  const hasMultipleContainers = !isNode && (panelState.containers.length > 1 || (panelState.initContainers?.length || 0) > 0)

  return (
    <div
      className={cn(
        "fixed bottom-0 right-0 z-50 bg-background border-t shadow-2xl transition-all duration-300 ease-in-out",
        panelHeight
      )}
      style={{
        left: sidebarWidth,
      }}
    >
      <div className="flex items-center justify-between h-12 px-4 border-b bg-muted/30">
        <div className="flex items-center gap-3">
          <Tabs value={panelState.type} onValueChange={(v) => switchType(v as any)} className="w-auto">
            <TabsList className="h-8">
              {!isNode && (
                <TabsTrigger value="logs" className="text-xs px-3">
                  <FileText className="h-3.5 w-3.5 mr-1.5" />
                  Logs
                </TabsTrigger>
              )}
              <TabsTrigger value="terminal" className="text-xs px-3">
                <Terminal className="h-3.5 w-3.5 mr-1.5" />
                Terminal
              </TabsTrigger>
              {!isNode && (
                <TabsTrigger value="files" className="text-xs px-3">
                  <FolderOpen className="h-3.5 w-3.5 mr-1.5" />
                  Files
                </TabsTrigger>
              )}
            </TabsList>
          </Tabs>

          <div className="h-4 w-px bg-border" />

          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-medium text-foreground">{panelState.appName}</span>
            <span>/</span>

            {items.length > 1 ? (
              <Select
                value={panelState.instanceName}
                onValueChange={handleInstanceChange}
              >
                <SelectTrigger className="h-7 w-auto min-w-40 text-xs font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {items.map((item) => (
                    <SelectItem
                      key={item.name}
                      value={item.name}
                      className="text-xs font-mono"
                    >
                      {item.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <span className="font-mono">{panelState.instanceName}</span>
            )}

            {hasMultipleContainers && (
              <>
                <span>/</span>
                <Select
                  value={panelState.containerName}
                  onValueChange={(value) => value && switchContainer(value)}
                >
                  <SelectTrigger className="h-7 w-auto min-w-30 text-xs font-mono">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {panelState.initContainers && panelState.initContainers.length > 0 && (
                      <SelectGroup>
                        <SelectLabel className="flex items-center gap-1.5 py-1">
                          <Zap className="h-3 w-3" />
                          Init Containers
                        </SelectLabel>
                        {panelState.initContainers.map((container) => (
                          <SelectItem key={container} value={container} className="text-xs pl-6">
                            {container}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    )}
                    {panelState.initContainers && panelState.initContainers.length > 0 && panelState.containers.length > 0 && (
                      <SelectSeparator />
                    )}
                    <SelectGroup>
                      <SelectLabel className="flex items-center gap-1.5 py-1">
                        <Layers2 className="h-3 w-3" />
                        Containers
                      </SelectLabel>
                      {panelState.containers.map((container) => (
                        <SelectItem key={container} value={container} className="text-xs pl-6">
                          {container}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </>
            )}
          </div>
        </div>

        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={toggleMinimize}
            title={isMinimized ? "Expand" : "Minimize"}
          >
            {isMinimized ? (
              <ChevronUp />
            ) : (
              <ChevronDown />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={handleToggleMaximize}
            title={isMaximized ? "Restore" : "Maximize"}
            disabled={isMinimized}
          >
            {isMaximized ? (
              <Minimize2 />
            ) : (
              <Maximize2 />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={closePanel}
            title="Close"
          >
            <X />
          </Button>
        </div>
      </div>

      {!isMinimized && (
        <div className="h-[calc(100%-3rem)] overflow-hidden">
          <Tabs value={panelState.type} className="w-full h-full">
            <TabsContent value="logs" className="h-full mt-0">
              <LogPanel
                key={`${panelState.appId}-${panelState.instanceName}-${panelState.containerName}`}
                appId={panelState.appId}
                instanceName={panelState.instanceName}
                containerName={panelState.containerName}
              />
            </TabsContent>
            <TabsContent value="terminal" className="h-full mt-0">
              <TerminalPanel
                key={`${panelState.appId}-${panelState.instanceName}-${panelState.containerName}`}
                appId={panelState.appId}
                instanceName={panelState.instanceName}
                containerName={panelState.containerName}
                targetType={panelState.targetType}
              />
            </TabsContent>
            <TabsContent value="files" className="h-full mt-0">
              <FileExplorerPanel
                key={`${panelState.appId}-${panelState.instanceName}-${panelState.containerName}`}
                appId={panelState.appId}
                instanceName={panelState.instanceName}
                containerName={panelState.containerName}
              />
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  )
}
