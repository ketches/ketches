import { describe, expect, it } from "vitest"

import { CollabPriority, RequirementStatus, TaskStatus, type Requirement, type Task } from "@/api/collaboration"

import {
  buildRequirementUpdateRequest,
  buildTaskUpdateRequest,
} from "./collaboration-update-payloads"

describe("collaboration update payload helpers", () => {
  it("builds a full task update payload when only assignee changes", () => {
    const task: Task = {
      id: "task-1",
      project_id: "project-1",
      sprint_id: "sprint-1",
      requirement_id: "requirement-1",
      title: "Implement auth",
      description: "Task description",
      status: TaskStatus.IN_PROGRESS,
      priority: CollabPriority.P1,
      assignee_id: "user-1",
      due_date: "2026-03-20",
      estimate_hours: 8,
      parent_task_id: "",
      depth: 0,
      created_by: "user-1",
      updated_by: "user-1",
      created_at: "2026-03-01T00:00:00Z",
      updated_at: "2026-03-01T00:00:00Z",
    }

    expect(buildTaskUpdateRequest(task, { assignee_id: "user-2" })).toEqual({
      title: "Implement auth",
      description: "Task description",
      status: TaskStatus.IN_PROGRESS,
      priority: CollabPriority.P1,
      assignee_id: "user-2",
      due_date: "2026-03-20",
      estimate_hours: 8,
      requirement_id: "requirement-1",
      sprint_id: "sprint-1",
    })
  })

  it("builds a full requirement update payload when only priority changes", () => {
    const requirement: Requirement = {
      id: "requirement-1",
      project_id: "project-1",
      sprint_id: "sprint-1",
      title: "Track deployment status",
      description: "Requirement description",
      status: RequirementStatus.CONFIRMED,
      priority: CollabPriority.P2,
      assignee_id: "user-1",
      planning_status: "planned",
      backlog_rank: 10,
      parent_requirement_id: "",
      depth: 0,
      created_by: "user-1",
      updated_by: "user-1",
      created_at: "2026-03-01T00:00:00Z",
      updated_at: "2026-03-01T00:00:00Z",
    }

    expect(buildRequirementUpdateRequest(requirement, { priority: CollabPriority.P0 })).toEqual({
      title: "Track deployment status",
      description: "Requirement description",
      status: RequirementStatus.CONFIRMED,
      priority: CollabPriority.P0,
      assignee_id: "user-1",
      sprint_id: "sprint-1",
      planning_status: "planned",
    })
  })
})
