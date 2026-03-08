import { useQuery } from "@tanstack/react-query"
import {
  Background,
  Controls,
  Handle,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Edge,
  type Node
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import dagre from "dagre"
import {
  Box,
  CheckCircle2,
  ExternalLink,
  FolderGit2,
  Hammer,
  Orbit,
  XCircle
} from "lucide-react"
import * as React from "react"
import { Link } from "react-router-dom"

import { codeRepositoriesApi } from "@/api/code-repositories"
import { ColorBadge } from "@/components/shared/color-badge"
import { useTheme } from "@/components/theme-provider/theme-provider"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

const nodeWidth = 180
const nodeHeight = 80

/** Left-right layout so nodes align with right/left connection points and lines are not twisted. */
const LAYOUT_DIRECTION = "LR" as const

const HANDLE_IDS = { top: "top", bottom: "bottom", left: "left", right: "right" } as const

function getLayoutedElements(nodes: Node[], edges: Edge[], direction: string = LAYOUT_DIRECTION) {
  const dagreGraph = new dagre.graphlib.Graph()
  dagreGraph.setDefaultEdgeLabel(() => ({}))
  dagreGraph.setGraph({ rankdir: direction })

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight })
  })

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  dagre.layout(dagreGraph)

  const sourceHandlesByNode = new Map<string, Set<string>>()
  const targetHandlesByNode = new Map<string, Set<string>>()
  const newEdges: Edge[] = edges.map((e, i) => {
    const sourceHandle = HANDLE_IDS.right
    const targetHandle = HANDLE_IDS.left
    if (!sourceHandlesByNode.has(e.source)) sourceHandlesByNode.set(e.source, new Set())
    sourceHandlesByNode.get(e.source)!.add(sourceHandle)
    if (!targetHandlesByNode.has(e.target)) targetHandlesByNode.set(e.target, new Set())
    targetHandlesByNode.get(e.target)!.add(targetHandle)
    return {
      ...e,
      id: e.id ?? `e-${i}`,
      sourceHandle,
      targetHandle,
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

const HANDLE_POSITIONS: Record<string, Position> = {
  top: Position.Top,
  bottom: Position.Bottom,
  left: Position.Left,
  right: Position.Right,
}

const getStatusBadgeColor = (status: string) => {
  if (status === "succeeded" || status === "Running") return "green"
  if (status === "failed" || status === "Failed") return "red"
  if (status === "Pending") return "yellow"
  return "gray"
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
  const appId = data.id?.startsWith("app-") ? data.id.slice(4) : null
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
      {appId && (
        <Link
          to={`/applications/${appId}`}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          className="absolute top-1 right-1 flex h-6 w-6 z-10 items-center justify-center rounded text-muted-foreground hover:text-foreground hover:bg-muted"
        >
          <ExternalLink className="h-3 w-3" />
        </Link>
      )}
      <CardContent className={appId ? "p-0 pr-8" : "p-0"}>
        <div className="flex items-center gap-2">
          <div className="p-2 bg-primary/10 rounded-md text-primary shrink-0">
            <Icon className="h-4 w-4" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <p className="text-xs font-medium text-muted-foreground tracking-wider">{data.type}</p>
              {data.status && (
                <ColorBadge color={getStatusBadgeColor(data.status)} className="text-[10px] px-1.5 py-0 shrink-0 gap-0.5">
                  {(data.status === "succeeded" || data.status === "Running") ? <CheckCircle2 className="h-2.5 w-2.5" /> : <XCircle className="h-2.5 w-2.5" />}
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

const getTypeIcon = (type: string) => {
  switch (type) {
    case "CodeRepository": return FolderGit2
    case "BuildConfig": return Hammer
    case "Environment": return Orbit
    case "Application": return Box
    default: return Box
  }
}

export function RepoTopologyView({ repoId }: { repoId: string }) {
  const { theme } = useTheme()
  const [nodes, , onNodesChange] = useNodesState<Node>([])
  const [edges, , onEdgesChange] = useEdgesState<Edge>([])
  const colorMode = theme === "system" ? "system" : theme

  const { data, isLoading } = useQuery({
    queryKey: ["repo-topology", repoId],
    queryFn: () => codeRepositoriesApi.getTopology(repoId),
    enabled: !!repoId,
  })

  // Derive layout from API data so cached data shows immediately when tab is switched back
  const layouted = React.useMemo(() => {
    if (!data) return null
    const initialNodes: Node[] = (data.nodes || []).map((n) => ({
      id: n.id,
      type: "custom",
      data: {
        label: n.name,
        name: n.name,
        type: n.type,
        status: n.status,
        icon: getTypeIcon(n.type),
        id: n.id,
      },
      position: { x: 0, y: 0 },
    }))
    const initialEdges: Edge[] = (data.edges || []).map((e, i) => ({
      id: `e-${i}`,
      source: e.source,
      target: e.target,
      animated: e.source.startsWith("bc") || e.source.startsWith("env"),
    }))
    return getLayoutedElements(initialNodes, initialEdges)
  }, [data])

  const displayNodes: Node[] = layouted?.nodes ?? nodes
  const displayEdges: Edge[] = layouted?.edges ?? edges

  if (isLoading) {
    return (
      <div className="h-100 w-full flex items-center justify-center border rounded-lg bg-muted/5">
        <Skeleton className="h-full w-full" />
      </div>
    )
  }

  return (
    <div className="w-full h-150 overflow-hidden border rounded-lg bg-muted/5">
      <ReactFlow
        nodes={displayNodes}
        edges={displayEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ maxZoom: 1 }}
        maxZoom={1}
        colorMode={colorMode}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  )
}
