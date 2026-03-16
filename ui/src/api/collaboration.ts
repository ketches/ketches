import type { LucideIcon } from 'lucide-react'
import { AlertCircle, AlertOctagon, AlertTriangle, CheckCircle2, ChevronDown, ChevronUp, ChevronsUp, CircleAlert, CircleCheck, CircleDashed, CircleDot, CircleDotDashed, CircleSlash, Clock, ClockCheck, Cog, Equal, Info, XCircle } from 'lucide-react'
import client from './client'
import { type PaginationParams, type PaginationResponse } from './pagination'

// ── Shared Types ─────────────────────────────────────────────────────────────

export interface CollabOption {
  label: string
  value: string
  icon: LucideIcon
  color: string
}

// ── Constants & Enums ────────────────────────────────────────────────────────

export const SprintStatus = {
  PLANNED: 'planned',
  ACTIVE: 'active',
  CLOSED: 'closed',
} as const
export type SprintStatus = typeof SprintStatus[keyof typeof SprintStatus]
export const SprintStatusOptions: CollabOption[] = [
  { label: 'Planned', value: 'planned', icon: Clock, color: 'orange' },
  { label: 'Active', value: 'active', icon: ClockCheck, color: 'green' },
  { label: 'Closed', value: 'closed', icon: CircleCheck, color: 'gray' },
]

export const RequirementStatus = {
  TRIAGE: 'triage',
  CONFIRMED: 'confirmed',
  IN_PROGRESS: 'in_progress',
  DONE: 'done',
  CLOSED: 'closed',
} as const
export type RequirementStatus = typeof RequirementStatus[keyof typeof RequirementStatus]
export const RequirementStatusOptions: CollabOption[] = [
  { label: 'Triage', value: 'triage', icon: AlertTriangle, color: 'purple' },
  { label: 'Confirmed', value: 'confirmed', icon: CircleCheck, color: 'indigo' },
  { label: 'In Progress', value: 'in_progress', icon: CircleDotDashed, color: 'blue' },
  { label: 'Done', value: 'done', icon: CheckCircle2, color: 'green' },
  { label: 'Closed', value: 'closed', icon: CircleSlash, color: 'slate' },
]

export const PlanningStatus = {
  BACKLOG: 'backlog',
  PLANNED: 'planned',
  IN_SPRINT: 'in_sprint',
  DONE: 'done',
} as const
export type PlanningStatus = typeof PlanningStatus[keyof typeof PlanningStatus]
export const PlanningStatusOptions = [
  { label: 'Backlog', value: 'backlog' },
  { label: 'Planned', value: 'planned' },
  { label: 'In Sprint', value: 'in_sprint' },
  { label: 'Done', value: 'done' },
] as const

export const TaskStatus = {
  TODO: 'todo',
  IN_PROGRESS: 'in_progress',
  REVIEW: 'review',
  DONE: 'done',
  CANCELLED: 'cancelled',
} as const
export type TaskStatus = typeof TaskStatus[keyof typeof TaskStatus]
export const TaskStatusOptions: CollabOption[] = [
  { label: 'Todo', value: 'todo', icon: CircleDashed, color: "gray" },
  { label: 'In Progress', value: 'in_progress', icon: CircleDotDashed, color: "blue" },
  { label: 'Review', value: 'review', icon: CircleDot, color: "orange" },
  { label: 'Done', value: 'done', icon: CheckCircle2, color: "green" },
  { label: 'Cancelled', value: 'cancelled', icon: CircleSlash, color: "gray" },
]

export const TestRunStatus = {
  PASSED: 'passed',
  FAILED: 'failed',
  BLOCKED: 'blocked',
} as const
export type TestRunStatus = typeof TestRunStatus[keyof typeof TestRunStatus]
export const TestRunStatusOptions: CollabOption[] = [
  { label: 'Passed', value: 'passed', icon: CheckCircle2, color: "green" },
  { label: 'Failed', value: 'failed', icon: CircleSlash, color: "red" },
  { label: 'Blocked', value: 'blocked', icon: CircleDashed, color: "gray" },
]

export const DefectSeverity = {
  CRITICAL: 'critical',
  HIGH: 'high',
  MEDIUM: 'medium',
  LOW: 'low',
} as const
export type DefectSeverity = typeof DefectSeverity[keyof typeof DefectSeverity]
export const DefectSeverityOptions: CollabOption[] = [
  { label: 'Critical', value: 'critical', icon: AlertOctagon, color: 'red' },
  { label: 'High', value: 'high', icon: AlertTriangle, color: 'orange' },
  { label: 'Medium', value: 'medium', icon: AlertCircle, color: 'yellow' },
  { label: 'Low', value: 'low', icon: Info, color: 'green' },
]

export const DefectStatus = {
  NEW: 'new',
  PROCESSING: 'processing',
  PENDING_VERIFY: 'pending_verify',
  CLOSED: 'closed',
  REJECTED: 'rejected',
} as const
export type DefectStatus = typeof DefectStatus[keyof typeof DefectStatus]
export const DefectStatusOptions: CollabOption[] = [
  { label: 'New', value: 'new', icon: CircleAlert, color: 'red' },
  { label: 'Processing', value: 'processing', icon: Cog, color: 'orange' },
  { label: 'Pending Verify', value: 'pending_verify', icon: Clock, color: 'amber' },
  { label: 'Closed', value: 'closed', icon: CheckCircle2, color: 'green' },
  { label: 'Rejected', value: 'rejected', icon: XCircle, color: 'slate' },
]

export const CollabPriority = {
  P0: 'p0',
  P1: 'p1',
  P2: 'p2',
  P3: 'p3',
} as const
export type CollabPriority = typeof CollabPriority[keyof typeof CollabPriority]
export const CollabPriorityOptions: CollabOption[] = [
  { label: 'P0', value: 'p0', icon: ChevronsUp, color: 'red' },
  { label: 'P1', value: 'p1', icon: ChevronUp, color: 'orange' },
  { label: 'P2', value: 'p2', icon: Equal, color: 'blue' },
  { label: 'P3', value: 'p3', icon: ChevronDown, color: 'green' },
]

// ── Interfaces ───────────────────────────────────────────────────────────────

export interface CollabFilterParams extends PaginationParams {
  status?: string
  assignee_id?: string
  sprint_id?: string
  priority?: string
}

export interface Sprint {
  id: string
  project_id: string
  name: string
  goal?: string
  status: SprintStatus
  start_date: string
  end_date: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface Requirement {
  id: string
  project_id: string
  sprint_id?: string
  title: string
  description?: string
  status: RequirementStatus
  priority: CollabPriority
  assignee_id?: string
  planning_status: PlanningStatus
  backlog_rank: number
  parent_requirement_id?: string
  depth: number
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface Task {
  id: string
  project_id: string
  sprint_id?: string
  requirement_id?: string
  title: string
  description?: string
  status: TaskStatus
  priority: CollabPriority
  assignee_id?: string
  due_date?: string
  estimate_hours?: number
  parent_task_id?: string
  depth: number
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface TestCase {
  id: string
  project_id: string
  sprint_id?: string
  requirement_id?: string
  task_id?: string
  title: string
  precondition?: string
  steps: string
  expected_result: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface TestRun {
  id: string
  project_id: string
  test_case_id: string
  status: TestRunStatus
  executed_by: string
  executed_at: string
  comment?: string
  created_at: string
  updated_at: string
}

export interface Defect {
  id: string
  project_id: string
  sprint_id?: string
  requirement_id?: string
  task_id?: string
  test_case_id?: string
  test_run_id?: string
  title: string
  description: string
  severity: DefectSeverity
  status: DefectStatus
  assignee_id?: string
  reproduction_steps?: string
  fix_note?: string
  runtime_context_json?: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

// ── Request Interfaces ───────────────────────────────────────────────────────

export interface CreateSprintRequest {
  name: string
  goal?: string
  status: SprintStatus
  start_date: string
  end_date: string
}

export type UpdateSprintRequest = CreateSprintRequest

export interface CreateRequirementRequest {
  title: string
  description?: string
  status: RequirementStatus
  priority: CollabPriority
  assignee_id?: string
  sprint_id?: string
  parent_requirement_id?: string
}

export interface UpdateRequirementRequest {
  title: string
  description?: string
  status: RequirementStatus
  priority: CollabPriority
  assignee_id?: string
  sprint_id?: string
  planning_status?: PlanningStatus
}

export interface CreateTaskRequest {
  title: string
  description?: string
  status: TaskStatus
  priority: CollabPriority
  assignee_id?: string
  due_date?: string
  estimate_hours?: number
  requirement_id?: string
  sprint_id?: string
  parent_task_id?: string
}

export type UpdateTaskRequest = Omit<CreateTaskRequest, 'parent_task_id'>

export interface CreateTestCaseRequest {
  title: string
  sprint_id?: string
  precondition?: string
  steps: string
  expected_result: string
  requirement_id?: string
  task_id?: string
}

export type UpdateTestCaseRequest = CreateTestCaseRequest

export interface CreateTestRunRequest {
  status: TestRunStatus
  comment?: string
}

export interface CreateDefectRequest {
  title: string
  description: string
  severity: DefectSeverity
  status: DefectStatus
  assignee_id?: string
  sprint_id?: string
  requirement_id?: string
  task_id?: string
  test_case_id?: string
  test_run_id?: string
  reproduction_steps?: string
}

export interface UpdateDefectRequest extends CreateDefectRequest {
  fix_note?: string
  runtime_context_json?: string
}

export interface BacklogReorderItem {
  requirement_id: string
  rank: number
}

// ── API Client ──────────────────────────────────────────────────────────────

export const collaborationApi = {
  // ── Sprints ──
  listSprints: async (projectId: string, params?: CollabFilterParams) => {
    return client.get(`/v1/projects/${projectId}/sprints`, { params }) as Promise<{ items: Sprint[], pagination: PaginationResponse }>
  },
  getSprint: async (projectId: string, sprintId: string) => {
    return client.get(`/v1/projects/${projectId}/sprints/${sprintId}`) as Promise<Sprint>
  },
  createSprint: async (projectId: string, data: CreateSprintRequest) => {
    return client.post(`/v1/projects/${projectId}/sprints`, data) as Promise<Sprint>
  },
  updateSprint: async (projectId: string, sprintId: string, data: UpdateSprintRequest) => {
    return client.put(`/v1/projects/${projectId}/sprints/${sprintId}`, data) as Promise<Sprint>
  },
  deleteSprint: async (projectId: string, sprintId: string) => {
    return client.delete(`/v1/projects/${projectId}/sprints/${sprintId}`)
  },

  // ── Requirements ──
  listRequirements: async (projectId: string, params?: CollabFilterParams) => {
    return client.get(`/v1/projects/${projectId}/requirements`, { params }) as Promise<{ items: Requirement[], pagination: PaginationResponse }>
  },
  getRequirement: async (projectId: string, requirementId: string) => {
    return client.get(`/v1/projects/${projectId}/requirements/${requirementId}`) as Promise<Requirement>
  },
  createRequirement: async (projectId: string, data: CreateRequirementRequest) => {
    return client.post(`/v1/projects/${projectId}/requirements`, data) as Promise<Requirement>
  },
  updateRequirement: async (projectId: string, requirementId: string, data: UpdateRequirementRequest) => {
    return client.put(`/v1/projects/${projectId}/requirements/${requirementId}`, data) as Promise<Requirement>
  },
  deleteRequirement: async (projectId: string, requirementId: string) => {
    return client.delete(`/v1/projects/${projectId}/requirements/${requirementId}`)
  },
  createRequirementChild: async (projectId: string, requirementId: string, data: CreateRequirementRequest) => {
    return client.post(`/v1/projects/${projectId}/requirements/${requirementId}/children`, data) as Promise<Requirement>
  },
  transitionRequirement: async (projectId: string, requirementId: string, status: RequirementStatus) => {
    return client.post(`/v1/projects/${projectId}/requirements/${requirementId}/transition`, { status })
  },

  // ── Backlog ──
  listBacklog: async (projectId: string, params?: CollabFilterParams) => {
    // Backlog typically returns a list of requirements without pagination in some systems, 
    // but the backend handler suggests it uses ListRequirements internally or similar.
    // Based on `handlers.ListBacklog` it likely returns list of requirements.
    // Checking `internal/routes/v1.go`: `projectsRead.GET("/:projectID/backlog", handlers.ListBacklog)`
    // It returns requirements.
    return client.get(`/v1/projects/${projectId}/backlog`, { params }) as Promise<{ items: Requirement[], pagination: PaginationResponse }>
  },
  reorderBacklog: async (projectId: string, items: BacklogReorderItem[]) => {
    return client.post(`/v1/projects/${projectId}/backlog/reorder`, { items })
  },
  planToSprint: async (projectId: string, requirementIds: string[], sprintId: string) => {
    return client.post(`/v1/projects/${projectId}/backlog/plan-to-sprint`, { requirement_ids: requirementIds, sprint_id: sprintId })
  },
  returnToBacklog: async (projectId: string, requirementIds: string[]) => {
    return client.post(`/v1/projects/${projectId}/backlog/return`, { requirement_ids: requirementIds })
  },

  // ── Tasks ──
  listTasks: async (projectId: string, params?: CollabFilterParams) => {
    return client.get(`/v1/projects/${projectId}/tasks`, { params }) as Promise<{ items: Task[], pagination: PaginationResponse }>
  },
  getTask: async (projectId: string, taskId: string) => {
    return client.get(`/v1/projects/${projectId}/tasks/${taskId}`) as Promise<Task>
  },
  createTask: async (projectId: string, data: CreateTaskRequest) => {
    return client.post(`/v1/projects/${projectId}/tasks`, data) as Promise<Task>
  },
  updateTask: async (projectId: string, taskId: string, data: UpdateTaskRequest) => {
    return client.put(`/v1/projects/${projectId}/tasks/${taskId}`, data) as Promise<Task>
  },
  deleteTask: async (projectId: string, taskId: string) => {
    return client.delete(`/v1/projects/${projectId}/tasks/${taskId}`)
  },
  createTaskChild: async (projectId: string, taskId: string, data: CreateTaskRequest) => {
    return client.post(`/v1/projects/${projectId}/tasks/${taskId}/children`, data) as Promise<Task>
  },
  transitionTask: async (projectId: string, taskId: string, status: TaskStatus) => {
    return client.post(`/v1/projects/${projectId}/tasks/${taskId}/transition`, { status })
  },

  // ── Test Cases ──
  listTestCases: async (projectId: string, params?: CollabFilterParams) => {
    return client.get(`/v1/projects/${projectId}/test-cases`, { params }) as Promise<{ items: TestCase[], pagination: PaginationResponse }>
  },
  getTestCase: async (projectId: string, testCaseId: string) => {
    return client.get(`/v1/projects/${projectId}/test-cases/${testCaseId}`) as Promise<TestCase>
  },
  createTestCase: async (projectId: string, data: CreateTestCaseRequest) => {
    return client.post(`/v1/projects/${projectId}/test-cases`, data) as Promise<TestCase>
  },
  updateTestCase: async (projectId: string, testCaseId: string, data: UpdateTestCaseRequest) => {
    return client.put(`/v1/projects/${projectId}/test-cases/${testCaseId}`, data) as Promise<TestCase>
  },
  deleteTestCase: async (projectId: string, testCaseId: string) => {
    return client.delete(`/v1/projects/${projectId}/test-cases/${testCaseId}`)
  },
  createTestRun: async (projectId: string, testCaseId: string, data: CreateTestRunRequest) => {
    return client.post(`/v1/projects/${projectId}/test-cases/${testCaseId}/runs`, data) as Promise<TestRun>
  },

  // ── Defects ──
  listDefects: async (projectId: string, params?: CollabFilterParams) => {
    return client.get(`/v1/projects/${projectId}/defects`, { params }) as Promise<{ items: Defect[], pagination: PaginationResponse }>
  },
  getDefect: async (projectId: string, defectId: string) => {
    return client.get(`/v1/projects/${projectId}/defects/${defectId}`) as Promise<Defect>
  },
  createDefect: async (projectId: string, data: CreateDefectRequest) => {
    return client.post(`/v1/projects/${projectId}/defects`, data) as Promise<Defect>
  },
  updateDefect: async (projectId: string, defectId: string, data: UpdateDefectRequest) => {
    return client.put(`/v1/projects/${projectId}/defects/${defectId}`, data) as Promise<Defect>
  },
  deleteDefect: async (projectId: string, defectId: string) => {
    return client.delete(`/v1/projects/${projectId}/defects/${defectId}`)
  },
  transitionDefect: async (projectId: string, defectId: string, status: DefectStatus) => {
    return client.post(`/v1/projects/${projectId}/defects/${defectId}/transition`, { status })
  },
}
