import { CheckSquare, FlaskConical, GalleryVerticalEnd, Goal } from "lucide-react"

import { PageHeader } from "@/components/layout/page-header"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import {
  Bug,
  FileText,
  ListTodo,
  User,
  Users
} from "lucide-react"
import { useEffect, useMemo, useState } from "react"

import { collaborationApi } from "@/api/collaboration"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { InputGroupAddon } from "@/components/ui/input-group"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useQuery } from "@tanstack/react-query"
import BacklogPage from "./backlog-page"
import DefectsPage from "./defects-page"
import RequirementsPage from "./requirements-page"
import SprintsPage from "./sprints-page"
import TasksPage from "./tasks-page"
import TestCasesPage from "./test-cases-page"

type Scope = "my-items" | "all-items"
type TaskViewMode = "list" | "kanban"
type SprintOption = { label: string; value: string }

const STORAGE_KEYS = {
  scope: "collab-scope",
  taskView: "collab-task-view",
  sprint: "collab-sprint",
  tab: "collab-tab",
} as const

function getStored<T extends string>(key: string, fallback: T): T {
  return (localStorage.getItem(key) as T) || fallback
}

const MY_ITEMS_TABS = ["tasks", "test-cases", "defects"] as const
const ALL_ITEMS_TABS = ["sprints", "tasks", "requirements", "backlog", "test-cases", "defects"] as const

export function CollaborationsPage({ projectId: projectIdProp }: { projectId?: string } = {}) {
  const { activeProjectId, activeProjectName } = useProjectStore()
  const projectId = projectIdProp || activeProjectId
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")
  const currentUserId = useAuthStore((s) => s.user?.id) ?? ""

  const [scope, setScope] = useState<Scope>(() => getStored(STORAGE_KEYS.scope, "all-items"))
  const [taskViewMode, setTaskViewMode] = useState<TaskViewMode>(() => getStored(STORAGE_KEYS.taskView, "list"))
  const [selectedSprintId, setSelectedSprintId] = useState<string>(() => getStored(STORAGE_KEYS.sprint, ""))
  const [activeTab, setActiveTab] = useState<string>(() => {
    const stored = getStored(STORAGE_KEYS.tab, "")
    const validTabs = scope === "my-items" ? MY_ITEMS_TABS : ALL_ITEMS_TABS
    if (stored && (validTabs as readonly string[]).includes(stored)) return stored
    return validTabs[0]
  })

  // Persist preferences
  useEffect(() => { localStorage.setItem(STORAGE_KEYS.scope, scope) }, [scope])
  useEffect(() => { localStorage.setItem(STORAGE_KEYS.taskView, taskViewMode) }, [taskViewMode])
  useEffect(() => { localStorage.setItem(STORAGE_KEYS.sprint, selectedSprintId) }, [selectedSprintId])
  useEffect(() => { localStorage.setItem(STORAGE_KEYS.tab, activeTab) }, [activeTab])

  // Fetch sprints for filter
  const { data: sprintsData } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId ?? "", { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const sprintOptions = useMemo<SprintOption[]>(() => {
    const sprints = sprintsData?.items ?? []

    return [
      { label: "All Sprints", value: "" },
      ...sprints.map((s) => ({ label: s.name, value: s.id })),
    ]
  }, [sprintsData?.items])
  const getSprintOptionLabel = (item: SprintOption | string) => {
    if (typeof item !== "string") {
      return item.label
    }

    return sprintOptions.find((opt) => opt.value === item)?.label || item
  }

  // Auto-select first active sprint when sprints load and no valid selection exists
  useEffect(() => {
    const sprints = sprintsData?.items ?? []
    if (!sprints.length) return
    // Skip if a valid sprint is already selected
    if (selectedSprintId && sprints.some(s => s.id === selectedSprintId)) return

    const firstActive = sprints.find(s => s.status === "active")
    if (firstActive) {
      setSelectedSprintId(firstActive.id)
    }
  }, [sprintsData?.items, selectedSprintId])

  const assigneeId = scope === "my-items" ? currentUserId : undefined
  const handleScopeChange = (value: string) => {
    const nextScope = value as Scope
    const nextValidTabs = nextScope === "my-items" ? MY_ITEMS_TABS : ALL_ITEMS_TABS

    setScope(nextScope)
    setActiveTab((currentTab) => (
      (nextValidTabs as readonly string[]).includes(currentTab) ? currentTab : nextValidTabs[0]
    ))
  }

  const validTabs = scope === "my-items" ? MY_ITEMS_TABS : ALL_ITEMS_TABS

  const breadcrumbs: BreadcrumbItem[] = isAdmin ? [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    { label: activeProjectName || "Projects", icon: GalleryVerticalEnd, href: `/projects/${projectId}` },
  ] : []
  breadcrumbs.push({ label: "Collaborations", icon: CheckSquare })

  if (!projectId) {
    return (
      <div className="flex flex-col items-center justify-center h-full">
        <GalleryVerticalEnd className="h-12 w-12 text-muted-foreground mb-4" />
        <h2 className="text-lg font-semibold">Select a project to view collaborations</h2>
      </div>
    )
  }


  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />
      {!projectIdProp && (
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Collaborations</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage tasks, sprints, requirements and more with your team.
            </p>
          </div>
        </div>)}

      <div className="flex flex-col space-y-4">
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <div className="flex flex-wrap items-center gap-4 md:flex-nowrap">
            <Tabs value={scope} onValueChange={handleScopeChange}>
              <TabsList>
                <TabsTrigger value="my-items">
                  <User />
                  My Items
                </TabsTrigger>
                <TabsTrigger value="all-items">
                  <Users />
                  All Items
                </TabsTrigger>
              </TabsList>
            </Tabs>

            <div className="w-full sm:w-auto">
              <Combobox
                items={sprintOptions}
                value={selectedSprintId}
                onValueChange={(val) => setSelectedSprintId(typeof val === "string" ? val : val?.value ?? "")}
                itemToStringLabel={getSprintOptionLabel}
              >
                <ComboboxInput placeholder="Filter by sprint..." className="w-full sm:w-48 h-7" >
                  <InputGroupAddon>
                    <Goal />
                  </InputGroupAddon>
                </ComboboxInput>
                <ComboboxContent alignOffset={-24} className="w-auto sm:w-48">
                  <ComboboxList>
                    {(opt: SprintOption) => (
                      <ComboboxItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </ComboboxItem>
                    )}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>

            <TabsList className="ml-auto" >
              {(validTabs as readonly string[]).includes("sprints") && (
                <TabsTrigger value="sprints">
                  <Goal className="text-fuchsia-500" />
                  Sprints
                </TabsTrigger>
              )}
              <TabsTrigger value="tasks">
                <ListTodo className="text-green-500" />
                Tasks
              </TabsTrigger>
              {(validTabs as readonly string[]).includes("requirements") && (
                <TabsTrigger value="requirements">
                  <FileText className="text-blue-500" />
                  Requirements
                </TabsTrigger>
              )}
              {(validTabs as readonly string[]).includes("backlog") && (
                <TabsTrigger value="backlog">
                  <ListTodo className="text-yellow-500" />
                  Backlog
                </TabsTrigger>
              )}
              <TabsTrigger value="test-cases">
                <FlaskConical className="text-purple-500" />
                Test Cases
              </TabsTrigger>
              <TabsTrigger value="defects">
                <Bug className="text-red-500" />
                Defects
              </TabsTrigger>
            </TabsList>
          </div>

          {(validTabs as readonly string[]).includes("sprints") && (
            <TabsContent value="sprints" className="mt-2">
              <SprintsPage projectId={projectId} />
            </TabsContent>
          )}

          <TabsContent value="tasks" className="mt-2">
            <TasksPage
              projectId={projectId}
              viewMode={taskViewMode}
              onViewModeChange={setTaskViewMode}
              assigneeId={assigneeId}
              sprintId={selectedSprintId || undefined}
            />
          </TabsContent>

          {(validTabs as readonly string[]).includes("requirements") && (
            <TabsContent value="requirements" className="mt-2">
              <RequirementsPage
                projectId={projectId}
                assigneeId={assigneeId}
                sprintId={selectedSprintId || undefined}
              />
            </TabsContent>
          )}

          {(validTabs as readonly string[]).includes("backlog") && (
            <TabsContent value="backlog" className="mt-2">
              <BacklogPage projectId={projectId} />
            </TabsContent>
          )}

          <TabsContent value="test-cases" className="mt-2">
            <TestCasesPage
              projectId={projectId}
              sprintId={selectedSprintId || undefined}
            />
          </TabsContent>

          <TabsContent value="defects" className="mt-2">
            <DefectsPage
              projectId={projectId}
              assigneeId={assigneeId}
              sprintId={selectedSprintId || undefined}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

export default CollaborationsPage
