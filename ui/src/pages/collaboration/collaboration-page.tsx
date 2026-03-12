import {
  Kanban,
  ListTodo,
  Bug,
  TestTube,
  Target,
  FileText,
  LayoutList,
  LayoutDashboard,
  Users,
  User,
} from "lucide-react"
import { useCallback, useEffect, useMemo, useState } from "react"

import { collaborationApi } from "@/api/collaboration"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { useAuthStore } from "@/stores/auth"
import { useQuery } from "@tanstack/react-query"
import SprintsPage from "./sprints-page"
import RequirementsPage from "./requirements-page"
import BacklogPage from "./backlog-page"
import TasksPage from "./tasks-page"
import TestCasesPage from "./test-cases-page"
import DefectsPage from "./defects-page"

interface CollaborationPageProps {
  projectId: string
}

type Scope = "my-items" | "all-items"
type TaskViewMode = "list" | "kanban"

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

export function CollaborationPage({ projectId }: CollaborationPageProps) {
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

  // When scope changes, validate active tab
  useEffect(() => {
    const validTabs = scope === "my-items" ? MY_ITEMS_TABS : ALL_ITEMS_TABS
    if (!(validTabs as readonly string[]).includes(activeTab)) {
      setActiveTab(validTabs[0])
    }
  }, [scope, activeTab])

  // Fetch sprints for filter
  const { data: sprintsData } = useQuery({
    queryKey: ["sprints", projectId, "all"],
    queryFn: () => collaborationApi.listSprints(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const sprints = sprintsData?.items ?? []
  const sprintOptions = useMemo(() => [
    { label: "All Sprints", value: "" },
    ...sprints.map((s) => ({ label: s.name, value: s.id })),
  ], [sprints])

  const assigneeId = scope === "my-items" ? currentUserId : undefined

  const handleScopeChange = useCallback((values: string[]) => {
    if (values.length > 0) setScope(values[0] as Scope)
  }, [])

  const handleTaskViewChange = useCallback((values: string[]) => {
    if (values.length > 0) setTaskViewMode(values[0] as TaskViewMode)
  }, [])

  const validTabs = scope === "my-items" ? MY_ITEMS_TABS : ALL_ITEMS_TABS

  return (
    <div className="flex flex-col space-y-4">
      {/* Top controls bar */}
      <div className="flex flex-wrap items-center gap-4">
        <ToggleGroup value={[scope]} onValueChange={handleScopeChange}>
          <ToggleGroupItem value="my-items">
            <User className="mr-1 h-3.5 w-3.5" />
            My Items
          </ToggleGroupItem>
          <ToggleGroupItem value="all-items">
            <Users className="mr-1 h-3.5 w-3.5" />
            All Items
          </ToggleGroupItem>
        </ToggleGroup>

        <Combobox
          value={selectedSprintId}
          onValueChange={(val) => setSelectedSprintId(val ?? "")}
        >
          <ComboboxInput placeholder="Filter by sprint..." itemToStringLabel={(item) => item.label} className="w-48" />
          <ComboboxContent>
            <ComboboxList>
              {sprintOptions.map((opt) => (
                <ComboboxItem key={opt.value} value={opt.value} label={opt.label}>
                  {opt.label}
                </ComboboxItem>
              ))}
            </ComboboxList>
          </ComboboxContent>
        </Combobox>

        {activeTab === "tasks" && (
          <ToggleGroup value={[taskViewMode]} onValueChange={handleTaskViewChange} className="ml-auto">
            <ToggleGroupItem value="list">
              <LayoutList className="mr-1 h-3.5 w-3.5" />
              List
            </ToggleGroupItem>
            <ToggleGroupItem value="kanban">
              <LayoutDashboard className="mr-1 h-3.5 w-3.5" />
              Kanban
            </ToggleGroupItem>
          </ToggleGroup>
        )}
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          {(validTabs as readonly string[]).includes("sprints") && (
            <TabsTrigger value="sprints">
              <Kanban className="mr-2 h-4 w-4" />
              Sprints
            </TabsTrigger>
          )}
          <TabsTrigger value="tasks">
            <Target className="mr-2 h-4 w-4" />
            Tasks
          </TabsTrigger>
          {(validTabs as readonly string[]).includes("requirements") && (
            <TabsTrigger value="requirements">
              <FileText className="mr-2 h-4 w-4" />
              Requirements
            </TabsTrigger>
          )}
          {(validTabs as readonly string[]).includes("backlog") && (
            <TabsTrigger value="backlog">
              <ListTodo className="mr-2 h-4 w-4" />
              Backlog
            </TabsTrigger>
          )}
          <TabsTrigger value="test-cases">
            <TestTube className="mr-2 h-4 w-4" />
            Test Cases
          </TabsTrigger>
          <TabsTrigger value="defects">
            <Bug className="mr-2 h-4 w-4" />
            Defects
          </TabsTrigger>
        </TabsList>

        {(validTabs as readonly string[]).includes("sprints") && (
          <TabsContent value="sprints" className="mt-4">
            <SprintsPage projectId={projectId} />
          </TabsContent>
        )}

        <TabsContent value="tasks" className="mt-4">
          <TasksPage
            projectId={projectId}
            viewMode={taskViewMode}
            assigneeId={assigneeId}
            sprintId={selectedSprintId || undefined}
          />
        </TabsContent>

        {(validTabs as readonly string[]).includes("requirements") && (
          <TabsContent value="requirements" className="mt-4">
            <RequirementsPage
              projectId={projectId}
              assigneeId={assigneeId}
              sprintId={selectedSprintId || undefined}
            />
          </TabsContent>
        )}

        {(validTabs as readonly string[]).includes("backlog") && (
          <TabsContent value="backlog" className="mt-4">
            <BacklogPage projectId={projectId} />
          </TabsContent>
        )}

        <TabsContent value="test-cases" className="mt-4">
          <TestCasesPage
            projectId={projectId}
            sprintId={selectedSprintId || undefined}
          />
        </TabsContent>

        <TabsContent value="defects" className="mt-4">
          <DefectsPage
            projectId={projectId}
            assigneeId={assigneeId}
            sprintId={selectedSprintId || undefined}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
