import type {
  Requirement,
  Task,
  UpdateRequirementRequest,
  UpdateTaskRequest,
} from "@/api/collaboration"

export function buildTaskUpdateRequest(
  task: Task,
  overrides: Partial<UpdateTaskRequest> = {}
): UpdateTaskRequest {
  return {
    title: task.title,
    description: task.description,
    status: task.status,
    priority: task.priority,
    assignee_id: task.assignee_id,
    due_date: task.due_date,
    estimate_hours: task.estimate_hours,
    requirement_id: task.requirement_id,
    sprint_id: task.sprint_id,
    ...overrides,
  }
}

export function buildRequirementUpdateRequest(
  requirement: Requirement,
  overrides: Partial<UpdateRequirementRequest> = {}
): UpdateRequirementRequest {
  return {
    title: requirement.title,
    description: requirement.description,
    status: requirement.status,
    priority: requirement.priority,
    assignee_id: requirement.assignee_id,
    sprint_id: requirement.sprint_id,
    planning_status: requirement.planning_status,
    ...overrides,
  }
}
