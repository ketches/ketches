import Editor from "@monaco-editor/react"
import { useQuery } from "@tanstack/react-query"
import {
  Background,
  ControlButton,
  Controls,
  Handle,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  useReactFlow,
  type Edge,
  type Node
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import dagre from "dagre"
import {
  Box,
  Code,
  FileCog,
  FileKey,
  HardDrive,
  Layers,
  LayoutGrid,
  Loader2,
  Network,
  Route,
  Scale,
  Shapes
} from "lucide-react"
import * as React from "react"

import { appsApi } from "@/api/apps"
import { ColorBadge } from "@/components/shared/color-badge"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { getAppStatusColor } from "@/lib/app-status"

const nodeWidth = 180
const nodeHeight = 60

/** Top-bottom layout for main structure, with Service→Pod vertical connections. */
const LAYOUT_DIRECTION = "TB" as const
const RANK_SEP = 100
const NODE_SEP = 60

const HANDLE_IDS = { top: "top", bottom: "bottom", left: "left", right: "right" } as const

/** Determine if an edge should use vertical (top/bottom) or horizontal (left/right) handles. */
function isVerticalEdge(sourceId: string, targetId: string): boolean {
  // Workload → Pod: vertical
  if (sourceId.startsWith("workload-") && targetId.startsWith("pod-")) return true
  // Service → Pod: vertical
  if (sourceId.startsWith("svc-") && targetId.startsWith("pod-")) return true
  return false
}

/** Check if edge should be animated (Service → Pod). */
function isAnimatedEdge(sourceId: string, targetId: string): boolean {
  return sourceId.startsWith("svc-") && targetId.startsWith("pod-")
}

function getLayoutedElements(nodes: Node[], edges: Edge[], direction: string = LAYOUT_DIRECTION) {
  // Filter out Workload → Service edges (we'll replace them with Service → Pod)
  const filteredEdges = edges.filter((e) => {
    return !(e.source.startsWith("workload-") && e.target.startsWith("svc-"))
  })

  // Add Service → Pod edges (inferred from Workload → Pod + original Workload → Service)
  const workloadToPods = new Map<string, string[]>()
  const workloadToService = new Map<string, string>()

  edges.forEach((e) => {
    if (e.source.startsWith("workload-") && e.target.startsWith("pod-")) {
      if (!workloadToPods.has(e.source)) workloadToPods.set(e.source, [])
      workloadToPods.get(e.source)!.push(e.target)
    }
    if (e.source.startsWith("workload-") && e.target.startsWith("svc-")) {
      workloadToService.set(e.source, e.target)
    }
  })

  const inferredEdges: Edge[] = []
  workloadToPods.forEach((podIds, workloadId) => {
    const svcId = workloadToService.get(workloadId)
    if (svcId) {
      podIds.forEach((podId) => {
        inferredEdges.push({
          id: `${svcId}-${podId}`,
          source: svcId,
          target: podId,
        })
      })
    }
  })

  const allEdges = [...filteredEdges, ...inferredEdges]

  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: direction, ranksep: RANK_SEP, nodesep: NODE_SEP })

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight })
  })

  allEdges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  dagre.layout(dagreGraph)

  const sourceHandlesByNode = new Map<string, Set<string>>()
  const targetHandlesByNode = new Map<string, Set<string>>()

  const newEdges: Edge[] = allEdges.map((e, i) => {
    const isVertical = isVerticalEdge(e.source, e.target)
    const sourceHandle = isVertical ? HANDLE_IDS.bottom : HANDLE_IDS.right
    const targetHandle = isVertical ? HANDLE_IDS.top : HANDLE_IDS.left
    const animated = isAnimatedEdge(e.source, e.target)

    if (!sourceHandlesByNode.has(e.source)) sourceHandlesByNode.set(e.source, new Set())
    sourceHandlesByNode.get(e.source)!.add(sourceHandle)
    if (!targetHandlesByNode.has(e.target)) targetHandlesByNode.set(e.target, new Set())
    targetHandlesByNode.get(e.target)!.add(targetHandle)

    return {
      ...e,
      id: e.id ?? `e-${i}`,
      sourceHandle,
      targetHandle,
      animated,
    }
  })

  const newNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id)
    const sourceHandles = Array.from(sourceHandlesByNode.get(node.id) ?? [])
    const targetHandles = Array.from(targetHandlesByNode.get(node.id) ?? [])
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      },
      data: {
        ...node.data,
        sourceHandles,
        targetHandles,
      },
    }
  })

  return { nodes: newNodes, edges: newEdges }
}

type TopologyViewContextValue = {
  appId: string
  openYamlDialog: (nodeId: string) => void
  isViewer: boolean
}


const TopologyViewContext = React.createContext<TopologyViewContextValue | null>(null)

const HANDLE_POSITIONS: Record<string, Position> = {
  top: Position.Top,
  bottom: Position.Bottom,
  left: Position.Left,
  right: Position.Right,
}

type CustomNodeData = {
  icon?: React.ComponentType<{ className?: string }>
  id?: string
  name?: string
  type?: string
  status?: string
  sourceHandles?: string[]
  targetHandles?: string[]
}

const CustomNode = ({ data }: { data: CustomNodeData }) => {
  const Icon = data.icon || Box
  const ctx = React.useContext(TopologyViewContext)
  const canViewYaml = ctx?.appId && data.id && !String(data.id).startsWith("app-") && !ctx?.isViewer
  const targetHandles = data.targetHandles ?? []
  const sourceHandles = data.sourceHandles ?? []

  return (
    <Card className="w-fit h-fit shadow-sm border-muted-foreground/20 overflow-visible relative p-2">
      {targetHandles.map((hid) => (
        <Handle
          key={hid}
          type="target"
          id={hid}
          position={HANDLE_POSITIONS[hid] ?? Position.Top}
          className="w-2 h-2 bg-primary!"
        />
      ))}
      {canViewYaml && (
        <Button
          variant="ghost"
          size="icon-xs"
          className="absolute top-1 right-1 h-6 w-6 z-10 text-muted-foreground hover:text-foreground"
          title="View resource YAML"
          onClick={(e) => {
            e.stopPropagation()
            ctx?.openYamlDialog(data.id!)
          }}
        >
          <Code className="h-3 w-3" />
        </Button>
      )}
      <CardContent className={canViewYaml ? "p-0 pr-8" : "p-0"}>
        <div className="flex items-center gap-2">
          <div className="p-2 bg-primary/10 rounded-md text-primary shrink-0">
            <Icon className="h-4 w-4" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <p className="text-xs font-medium text-muted-foreground tracking-wider">{data.type}</p>
              {data.status && (
                <ColorBadge color={getAppStatusColor(data.status)} className="text-[10px] px-1.5 py-0 shrink-0">
                  {data.status}
                </ColorBadge>
              )}
            </div>
            <p className="text-xs font-medium truncate" title={data.name}>{data.name}</p>
          </div>
        </div>
      </CardContent>
      {sourceHandles.map((hid) => (
        <Handle
          key={hid}
          type="source"
          id={hid}
          position={HANDLE_POSITIONS[hid] ?? Position.Bottom}
          className="w-2 h-2 bg-primary!"
        />
      ))}
    </Card>
  )
}

const nodeTypes = {
  custom: CustomNode,
}

function FlowControls({
  onResetLayout,
  registerFitView,
}: {
  onResetLayout: () => void
  registerFitView: (fn: (() => void) | undefined) => void
}) {
  const { fitView } = useReactFlow()
  React.useEffect(() => {
    registerFitView(() => fitView())
    return () => registerFitView(undefined)
  }, [fitView, registerFitView])

  return (
    <Controls>
      <ControlButton onClick={onResetLayout} title="Reset layout">
        <LayoutGrid className="h-4 w-4" />
      </ControlButton>
    </Controls>
  )
}

const getTypeIcon = (type: string) => {
  switch (type) {
    case "Application": return Box
    case "Deployment": return Layers
    case "StatefulSet": return Layers
    case "Pod": return Shapes
    case "Service": return Network
    case "HTTPRoute": return Route
    case "ConfigMap": return FileCog
    case "Secret": return FileKey
    case "PVC": return HardDrive
    case "PV": return HardDrive
    case "HPA": return Scale
    default: return Box
  }
}

export function TopologyView({ appId, isViewer }: { appId: string; isViewer?: boolean }) {
  const { theme } = useTheme()
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])
  const [yamlDialogNodeId, setYamlDialogNodeId] = React.useState<string | null>(null)
  const fitViewRef = React.useRef<(() => void) | undefined>(undefined)
  const colorMode = theme === "system" ? "system" : theme

  const registerFitView = React.useCallback((fn: (() => void) | undefined) => {
    fitViewRef.current = fn
  }, [])

  const onResetLayout = React.useCallback(() => {
    const minimalNodes = nodes.map((n) => ({ ...n, position: { x: 0, y: 0 } }))
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(minimalNodes, edges)
    setNodes(layoutedNodes)
    setEdges(layoutedEdges)
    setTimeout(() => fitViewRef.current?.(), 50)
  }, [nodes, edges, setNodes, setEdges])

  const ctxValue = React.useMemo<TopologyViewContextValue>(
    () => ({
      appId,
      openYamlDialog: (nodeId) => setYamlDialogNodeId(nodeId),
      isViewer: isViewer ?? false,
    }),
    [appId, isViewer]
  )


  const { data, isLoading } = useQuery({
    queryKey: ["app-topology", appId],
    queryFn: () => appsApi.getTopology(appId),
    enabled: !!appId,
  })

  // Sync API data to React Flow state (needed when remounting with cached data, e.g. after tab switch)
  React.useEffect(() => {
    if (!data) return
    const initialNodes: Node[] = data.nodes.map((n) => ({
      id: n.id,
      type: "custom",
      data: {
        label: n.name,
        name: n.name,
        type: n.type,
        status: n.status,
        icon: getTypeIcon(n.type),
        id: n.id,
        appId,
      },
      position: { x: 0, y: 0 },
    }))
    const initialEdges: Edge[] = data.edges.map((e, i) => ({
      id: `e-${i}`,
      source: e.source,
      target: e.target,
    }))
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      initialNodes,
      initialEdges
    )
    setNodes(layoutedNodes)
    setEdges(layoutedEdges)
  }, [data, appId, setNodes, setEdges])

  if (isLoading) {
    return (
      <div className="h-100 w-full flex items-center justify-center border rounded-lg bg-muted/5">
        <Skeleton className="h-full w-full" />
      </div>
    )
  }

  return (
    <TopologyViewContext.Provider value={ctxValue}>
      <div className="w-full h-100 overflow-hidden border rounded-lg bg-muted/5">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ maxZoom: 1 }}
          maxZoom={1}
          colorMode={colorMode}
        >
          <Background />
          <FlowControls onResetLayout={onResetLayout} registerFitView={registerFitView} />
        </ReactFlow>
      </div>

      <ResourceYamlDialog
        appId={appId}
        nodeId={yamlDialogNodeId}
        open={yamlDialogNodeId !== null}
        onOpenChange={(open) => !open && setYamlDialogNodeId(null)}
      />
    </TopologyViewContext.Provider>
  )
}

function ResourceYamlDialog({
  appId,
  nodeId,
  open,
  onOpenChange,
}: {
  appId: string
  nodeId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["app-topology-resource-yaml", appId, nodeId],
    queryFn: () => appsApi.getTopologyResourceYaml(appId, nodeId!),
    enabled: open && !!appId && !!nodeId,
  })

  const { theme } = useTheme()

  // Resolved theme for Monaco
  const [monacoTheme, setMonacoTheme] = React.useState<"vs" | "vs-dark">("vs")
  React.useEffect(() => {
    const resolve = () => {
      if (theme === "dark") return "vs-dark" as const
      if (theme === "light") return "vs" as const
      return document.documentElement.classList.contains("dark")
        ? ("vs-dark" as const)
        : ("vs" as const)
    }
    setMonacoTheme(resolve())
    if (theme !== "system") return
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => setMonacoTheme(resolve())
    media.addEventListener("change", handler)
    return () => media.removeEventListener("change", handler)
  }, [theme])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex flex-col h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] overflow-hidden sm:h-[90vh] sm:max-h-[90vh] sm:max-w-[90vw]">
        <DialogHeader>
          <DialogTitle>Resource YAML</DialogTitle>
        </DialogHeader>
        <div className="flex-1 min-h-0 overflow-hidden rounded-md border">
          {isLoading ? (
            <div className="flex items-center justify-center h-full gap-2 text-muted-foreground bg-muted/30">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading...
            </div>
          ) : error ? (
            <div className="flex items-center justify-center h-full p-4">
              <p className="text-destructive">
                {(error as Error)?.message || "Failed to load resource YAML"}
              </p>
            </div>
          ) : data?.yaml != null ? (
            <Editor
              height="100%"
              language="yaml"
              theme={monacoTheme}
              value={data.yaml}
              options={{
                readOnly: true,
                fontSize: 12,
                lineNumbers: 'on',
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                wordWrap: 'on',
                automaticLayout: true,
                padding: { top: 16, bottom: 16 },
              }}
              loading={
                <div className="flex items-center justify-center h-full bg-zinc-950 text-zinc-500">
                  <Loader2 className="h-6 w-6 animate-spin mr-2" />
                  Loading editor...
                </div>
              }
            />
          ) : (
            <div className="flex items-center justify-center h-full text-muted-foreground bg-muted/30">
              No content available
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
