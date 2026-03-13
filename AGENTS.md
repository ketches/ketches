# Ketches Project Agents

This document defines the specialized agents involved in the development and maintenance of the Ketches platform.

## System Architect Agent

- **Role**: Oversees the overall technical design and integration between the backend (ketches) and frontend (ketches-ui).
- **Responsibilities**:
  - Ensure adherence to the technical design specified in `docs/TECHNICAL_DESIGN.md`.
  - Maintain architectural consistency across the project.
  - Review cross-module integrations.

## Product & Requirements Agent

- **Role**: Maintains the product vision and requirement alignment.
- **Responsibilities**:
  - Keep the PRD (`docs/PRD.md`) and technical design (`docs/TECHNICAL_DESIGN.md`) up to date.
  - Verify that implemented features match the prioritized functional modules (P0, P1, P2).

## Backend Engineer Agent

- **Focus**: Go, Gin, API Handlers, and Services.
- **Responsibilities**:
  - Implement RESTful APIs following the designs in `docs/TECHNICAL_DESIGN.md`.
  - Maintain business logic in the `internal/services` layer.
  - Ensure robust error handling and request validation.

## Kubernetes Specialist Agent

- **Focus**: `client-go`, K8s Resource Management.
- **Responsibilities**:
  - Manage Kubernetes resource construction (Deployments, StatefulSets, Services, etc.) in the `internal/core` layer.
  - Implement cluster client management and resource CRUD operations.
  - Handle real-time event processing and status calculation.

## DB & ORM Specialist Agent

- **Focus**: GORM, PostgreSQL/MySQL/SQLite, Database Entities.
- **Responsibilities**:
  - Define and maintain database entities in `internal/db/entities`.
  - Manage database migrations and ORM query encapsulations.
  - Optimize database interactions and ensure data integrity.

### Foreign Key Policy (MANDATORY)

- **Never create physical foreign keys** in database DDL/migrations.
- Keep `gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}` enabled.
- **Do not use GORM association tags** in entity structs (e.g. `foreignKey`, `references`, `constraint`).
- Entity structs must keep only scalar columns/IDs. Relationship data must be loaded via explicit query models (DTOs) + `JOIN`/manual queries.
- For legacy databases, use DDL scripts to drop existing foreign keys and normalize nullable ID columns (empty string -> `NULL`).

## Security & Auth Agent

- **Focus**: JWT, RBAC, Middlewares.
- **Responsibilities**:
  - Implement secure JWT-based authentication.
  - Maintain permission-checking middlewares (AdminOnly, ProjectMember, etc.).
  - Ensure sensitive data (like KubeConfigs and passwords) is handled securely.

## Frontend Engineer Agent

- **Focus**: React, TypeScript, Component Architecture.
- **Responsibilities**:
  - Implement application pages and routing logic in `ui/src/pages` and `ui/src/routes`.
  - Maintain reusable business components in `ui/src/components/ui`.
  - Ensure type safety across the entire frontend application.
  - Use `Combobox` component for all dropdowns and selection inputs, adhering to the design system, never use `Select`.
  - Always use `min-h-?` for height constraints, never use `"min-h-[?px]` for that purpose.
  - Always use `space-y-?` for vertical spacing, never use `gap-y-?` for that purpose.

## UI/UX Design Agent

- **Focus**: shadcn/ui, Tailwind CSS, Visual Consistency.
- **Responsibilities**:
  - Maintain the design system using shadcn/ui and Tailwind CSS.
  - Ensure a consistent and user-friendly interface across all modules.
  - Optimize responsive layouts and accessibility.

## State & Data Agent

- **Focus**: Zustand, TanStack Query, Axios.
- **Responsibilities**:
  - Manage global application state using Zustand.
  - Implement efficient server-state management with TanStack Query.
  - Maintain the API client and data fetching logic in `ui/src/api`.

## DevOps UI Specialist Agent

- **Focus**: xterm.js, WebSockets, Real-time Visuals.
- **Responsibilities**:
  - Implement and optimize real-time features like terminal emulation and log viewing.
  - Manage WebSocket connections for live application updates.
  - Ensure high performance when handling large volumes of log data.

## Coding rules

### Backend

- Always use English code comments;
- Use `any` rather than `interface{}` for generic types;
- Never add physical foreign keys or GORM association tags in entities; use query-model + `JOIN` patterns instead;

### Frontend

- Always use English code comments;
- Always use shadcn components;
- Do not write css directly, use tailwind;
- The src/components/ui directory contains standard components. Please do not modify any content within it;
- Avoid using the `any` type;
- Prioritize the use of `Field`, `FieldLabel`, and `FieldContent` in forms.
- Combobox itmes better use const values. For example: `[{ label: "Option 1", value: "option1" },{ label: "Option 2", value: "option2" }]`.
- Combobox should use `itemToStringLabel` to specify the label field, never use `value` for that purpose. For example: `itemToStringLabel={(v) => FILE_MODE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}`

### Docs

- Compliant with markdown lint:
  - Keep table pipe space to the right and left for style "compact"
  - Code blocks should have a language specified, if not, use `txt` as default;
