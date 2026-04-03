# Gateway Exposure Guard And Image-Only Rollback Design

**Date:** 2026-04-03

**Goal:** Disable unsupported TCP/UDP public exposure in the current product while making deployment history and rollback work reliably for image changes only.

## Scope

This design covers:

- `internal/core/app_gateway.go`
- `internal/core/gateway_api.go`
- `internal/services/app.go`
- `internal/services/deployment_history.go`
- `internal/handlers/deployment_history.go`
- `internal/models/deployment_history.go`
- `ui/src/components/applications/gateway-editor.tsx`
- `ui/src/components/deployments/deployment-history-list.tsx`
- `ui/src/api/deployment-history.ts`

This design does not cover:

- Experimental Gateway API `TCPRoute` / `UDPRoute` support
- Rollback of non-image configuration such as replicas, resources, probes, or gateways
- Generic deployment revision management outside application image changes

## Current State

### Gateway Exposure

- The backend currently supports HTTPRoute creation for exposed HTTP/HTTPS gateways.
- `internal/core/app_gateway.go` still contains TODOs for `TCPRoute` / `UDPRoute`.
- The current environment-level Gateway in `internal/core/gateway_api.go` only creates HTTP and HTTPS listeners.
- The project depends on `sigs.k8s.io/gateway-api v1.5.1`, which includes `TCPRoute` and `UDPRoute` clients under `v1alpha2`, but these APIs are still part of the Gateway API Experimental Channel rather than the stable standard channel.

### Deployment History

- `ui/src/components/deployments/deployment-history-list.tsx` already renders a rollback UI.
- `internal/services/deployment_history.go` already has `RecordDeployment()` and `RollbackDeployment()`.
- The normal image update path in `internal/services/app.go:UpdateAppImage()` does not call `RecordDeployment()`.
- As a result, the deployment history list is usually empty unless rollback has already happened.

### Rollback Semantics

- `services.RollbackDeployment()` currently restores image, replicas, and resource settings from history.
- This conflicts with the approved requirement: rollback must only revert the image version.
- `executeRollbackAction()` in `internal/services/app.go` is still an unimplemented generic action path and cannot produce a meaningful rollback without a deployment history target.

## Constraints

- Do not implement `TCPRoute` / `UDPRoute` while they remain an Experimental Channel dependency for this product.
- Preserve current HTTP/HTTPS public exposure behavior.
- Rollback must only revert the deployed image.
- The UI must make this limit explicit so users do not assume full configuration rollback.
- Follow the existing repository rules:
  - No physical foreign keys
  - No `any` in frontend changes unless unavoidable
  - Keep backend comments in English

## Approach Options

### Option 1: Implement Experimental TCP/UDP Gateway API now

Pros:

- Resolves the backend TODOs directly
- Enables public exposure for all supported protocols

Cons:

- Requires experimental CRD/channel support in customer clusters
- Requires env Gateway TCP/UDP listeners in addition to route resources
- Expands validation, compatibility, and test surface significantly
- Conflicts with the approved direction to avoid experimental exposure features

### Option 2: Hide TCP/UDP public exposure in the UI only

Pros:

- Small frontend-only change
- Low implementation cost

Cons:

- Backend still accepts unsupported payloads if called directly
- Leaves server-side behavior inconsistent and review item only partially addressed

### Option 3: Recommended

Disable TCP/UDP public exposure in both UI and backend, and separately fix deployment history plus image-only rollback.

Pros:

- Matches the current product policy
- Prevents unsupported behavior from both UI and direct API access
- Fixes an actually broken user workflow: empty deployment history and misleading rollback
- Keeps the implementation bounded and stable

Cons:

- The Gateway API TODOs remain intentionally unresolved in implementation terms, but are neutralized by product behavior

## Architecture

The change is split into two vertical slices.

### Slice 1: Gateway Public Exposure Guard

- Frontend gateway editor enforces protocol-based capability:
  - For `http` and `https`, `Enable public access` works as it does today.
  - For non-HTTP protocols, the checkbox is forced to `false`, disabled, and accompanied by an info tooltip explaining that TCP/UDP public exposure is not available yet.
- Backend gateway create/update validation rejects `exposed=true` when protocol is not `http` or `https`.
- Existing HTTP/HTTPS sync logic remains unchanged.

This keeps the product honest: unsupported capabilities are not offered and cannot be forced through the API.

### Slice 2: Image History And Rollback

- Application image updates record a deployment history row whenever the effective image changes.
- Rollback uses a history target row and applies only the previous image from that row.
- After rollback succeeds, the system records a new deployment history row describing the rollback image change.
- UI wording is updated to explicitly state that rollback only restores the image version and does not revert other configuration.

This makes history and rollback behavior align with the approved semantics and with user expectations.

## Detailed Design

### 1. Gateway Editor UX

Modify `ui/src/components/applications/gateway-editor.tsx`.

Behavior:

- When protocol is `tcp` or `udp`:
  - `formData.exposed` is forced to `false`
  - the checkbox remains unchecked
  - the checkbox is disabled
  - an adjacent `Info` icon shows a hover tooltip such as:
    - `Public access is currently available only for HTTP/HTTPS gateways. TCP/UDP public exposure is not supported yet.`
- If the user switches from `http/https` to `tcp/udp`, the form resets:
  - `exposed = false`
  - gateway-public-access-only fields remain hidden or inactive as already applicable

This is a product-level guard, not just a visual hint.

### 2. Backend Gateway Validation

Modify request validation in `internal/handlers/app_gateway.go`.

Behavior:

- For protocol `http` and `https`, preserve current `exposed` validation flow.
- For protocol `tcp` and `udp`:
  - reject requests where `exposed == true`
  - return a clear validation error such as:
    - `public access is currently supported only for HTTP/HTTPS gateways`

This prevents API clients or stale frontends from bypassing the UI restriction.

No Gateway API resource changes are required for this slice because TCP/UDP public exposure is explicitly disabled.

### 3. Deployment History Recording Rules

Modify `internal/services/app.go:UpdateAppImage()` and reuse `internal/services/deployment_history.go`.

Behavior:

- Before mutating the app record, capture the previous app snapshot.
- After saving and successfully applying the new image, compare `before.ContainerImage` and `after.ContainerImage`.
- If the image changed, write a deployment history row with:
  - `image_before = previous image`
  - `image_after = new image`
  - current replicas/resource fields copied as context only
  - `deploy_type = "manual"`
  - `reason = "Image updated from application settings"` or equivalent
- If the image did not change, do not create a duplicate history row.

This is intentionally image-driven history, not generic config history.

### 4. Image-Only Rollback Semantics

Modify `internal/services/deployment_history.go`.

Behavior:

- `RollbackDeployment(appID, historyID)` loads the chosen history row and the current app context.
- Only set:
  - `appCtx.App.ContainerImage = history.ImageBefore`
- Do not modify:
  - replicas
  - CPU/memory requests or limits
  - any other app configuration
- Save the app and apply it to Kubernetes.
- Record a new history row representing the rollback event:
  - `deploy_type = "rollback"`
  - `reason = "Rollback image to previous deployment"`
  - `image_before = current image before rollback`
  - `image_after = restored previous image`

The history row structure remains unchanged for now to avoid a schema migration. Non-image fields in the row remain contextual snapshots, but rollback logic must ignore them.

### 5. Generic Rollback Action

Modify `internal/services/app.go:executeRollbackAction()`.

Behavior:

- Do not leave the action as a misleading stub.
- The generic action route has no history target, so it cannot safely determine which image to restore.

Recommended behavior:

- Return an explicit application error such as:
  - `rollback requires a deployment history target`

This makes the generic action honest and directs callers to the existing history-based rollback endpoint instead of pretending rollback is implemented.

If later the product wants a “rollback to latest previous image” quick action, that should be a separate design with explicit selection rules.

### 6. Frontend Rollback Wording

Modify `ui/src/components/deployments/deployment-history-list.tsx`.

Behavior:

- Update card description from generic rollback wording to image-specific wording.
- Update confirmation dialog copy to make the limitation explicit.

Target wording:

- Description:
  - `Track image deployment changes and roll back to a previous image version when needed.`
- Confirmation:
  - `This rollback only restores the application image version. Other configuration changes are not reverted.`

The preview section in the dialog should prioritize the image being restored and can keep secondary contextual fields if useful, but it must not imply those values will be applied.

## Data Flow

### Image Update

1. User updates app image in the UI.
2. `UpdateAppImage()` loads app context and stores the pre-change snapshot.
3. Service saves app image changes.
4. Service applies the updated workload to Kubernetes.
5. If apply succeeds and image changed, service records deployment history.
6. UI deployment history list can now fetch and display the new row.

### Rollback

1. User opens deployment history and chooses a row.
2. Frontend calls `POST /v1/apps/:appID/deployment-history/rollback`.
3. Backend loads the selected history row and current app context.
4. Backend updates only the app image to `history.image_before`.
5. Backend applies the workload to Kubernetes.
6. Backend records a new rollback history row.
7. Frontend refreshes app detail and deployment history.

## Error Handling

### Gateway Guard

- Frontend tooltip explains the disabled state before submission.
- Backend validation returns a clear 400 error if an unsupported payload is submitted directly.

### Deployment History

- Do not record history when `ApplyApp()` fails.
- If `ApplyApp()` succeeds but history recording fails, return the error to the caller and log it clearly. The app state has changed, so the error message should reflect that history persistence failed after apply.
- Rollback should fail fast if:
  - the history row does not exist
  - the app does not exist
  - applying the image to Kubernetes fails

## Testing Strategy

### Backend

- Add service tests for `UpdateAppImage()`:
  - image changed -> history row created
  - image unchanged -> no new history row
- Add service tests for `RollbackDeployment()`:
  - only image changes
  - replicas/resources remain unchanged
  - rollback creates a new history row
- Add handler/validation tests for gateway create/update:
  - `tcp/udp + exposed=true` -> 400
  - `http/https + exposed=true` remains valid
- Add a test for `executeRollbackAction()` returning the explicit unsupported error.

### Frontend

- Add/update gateway editor tests:
  - protocol switch to `tcp/udp` disables and unchecks `Enable public access`
  - info tooltip text is present
- Add/update deployment history dialog tests:
  - image-only rollback wording is rendered

## Success Criteria

- TCP/UDP gateways can no longer be configured as publicly exposed from the UI or API.
- HTTP/HTTPS gateway exposure behavior remains unchanged.
- Updating an application image creates deployment history entries.
- Rollback restores only the image version.
- Deployment history UI clearly communicates that rollback does not revert other configuration.
