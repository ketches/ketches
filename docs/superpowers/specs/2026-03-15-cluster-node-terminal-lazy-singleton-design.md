# Cluster Node Terminal Lazy Singleton Design

## Summary

This design changes the cluster node terminal flow from "create one temporary Pod per
terminal connection" to "lazily create and reuse one high-privilege terminal Pod per
node".

The first terminal connection to a node creates the helper Pod on demand. Later terminal
connections reuse the same Pod instead of creating a new one. The Pod is not deleted when
an exec session ends. Instead, a background cleanup job deletes idle or unhealthy node
terminal Pods after a bounded retention window.

## Goals

- Lazily create the node terminal Pod only when a user opens a node terminal.
- Reuse a single node terminal Pod per node instead of creating a new Pod per session.
- Keep the existing node terminal API route and terminal UI flow stable.
- Automatically clean up idle or unhealthy node terminal Pods.
- Keep the implementation focused on the node terminal path only.

## Non-Goals

- No changes to application container terminal behavior.
- No cluster-wide terminal pool or daemonized terminal system.
- No new database tables, migrations, or foreign keys.
- No redesign of the terminal panel UI.
- No new user-facing controls for manual node terminal cleanup in this iteration.

## User Experience

### First Terminal Open

When a user opens a node terminal for the first time on a given node:

- the existing terminal panel opens as it does today
- the first WebSocket-backed exec request triggers helper Pod creation
- once the Pod is running, the shell session starts

There is no separate pre-create API and no extra UI step before the terminal connects.

### Later Terminal Opens

When a user opens the same node terminal again before the idle timeout expires:

- the backend reuses the existing helper Pod
- the terminal connects faster because Pod creation is skipped

Multiple terminal sessions may still exist in the UI, but they all exec into the same
node terminal Pod for that node.

### Idle Cleanup

If the helper Pod is idle for 30 minutes, the backend cleanup job deletes it. The next
terminal open recreates it on demand.

## Backend Design

### Stable Pod Identity

The helper Pod name changes from a random value such as `node-terminal-<uuid>` to a stable
node-scoped name:

```txt
node-terminal-<sanitized-node-name>
```

Rules:

- one helper Pod per node per cluster
- continue using the `default` namespace in this iteration
- sanitize the node name so it is always a valid Kubernetes Pod name segment

The stable name is the key to reuse. A request no longer assumes that it must create a new
Pod.

### Pod Labels and Annotations

Each node terminal Pod carries explicit metadata so it can be discovered and cleaned up:

```txt
labels:
  ketches.io/node-terminal: "true"
  ketches.io/node-name: "<node name>"

annotations:
  ketches.io/last-active-at: "<RFC3339 timestamp>"
```

`last-active-at` is updated whenever a new terminal exec session is about to use the Pod.

### Get-or-Create Flow

`ExecClusterNodeTerminal` changes from "create, wait, exec, delete" to the following flow:

1. Compute the stable Pod name for the target node.
2. Read the Pod from Kubernetes.
3. If the Pod does not exist, create it.
4. If the Pod exists and is reusable, update `last-active-at` and continue.
5. If the Pod exists but is not reusable, delete it, recreate it, wait for `Running`, then
   continue.
6. Exec into the helper container with the existing `nsenter` command.

The API route and handler stay unchanged. Only the service-layer implementation changes.

### Reusable Pod Criteria

A node terminal Pod is reusable when all of the following are true:

- it exists with the expected stable name
- it has the expected node terminal labels
- it targets the requested node
- it is not in a terminal Pod phase such as `Succeeded` or `Failed`
- its container is not in a known broken waiting state such as image pull failure

If the Pod is `Pending`, the service may wait for it to become `Running` within the normal
startup timeout instead of recreating it immediately.

### Broken Pod Recreate Rules

The service deletes and recreates the helper Pod when any of the following is true:

- the Pod is in `Succeeded` or `Failed`
- the Pod was evicted
- the container is stuck in an unrecoverable wait state
- the Pod does not target the requested node
- the Pod metadata indicates it is not the expected node terminal helper
- the Pod does not become `Running` before the startup timeout expires

This keeps recovery deterministic and avoids indefinitely reusing damaged helper Pods.

### Pod Specification

The helper Pod keeps the current privilege model and shell strategy:

- `NodeName` pinned to the target node
- one helper container
- privileged security context
- `HostPID: true`
- `HostNetwork: true`
- long-running sleep command
- `nsenter` into PID 1 on exec

The initial image remains `alpine:latest` in this iteration so the change stays focused on
lifecycle behavior rather than image supply policy.

### Exec Semantics

Terminal sessions still create a fresh `exec` stream for each WebSocket connection. Reusing
the Pod does not imply sharing one shell process. It only avoids repeated Pod creation.

The `defer delete pod` behavior is removed. Closing a terminal session only ends the exec
stream. Pod lifecycle is now controlled by the cleanup job and by health-based recreation.

### Concurrency Handling

Two requests may try to lazily create the same helper Pod at the same time. The service must
handle this as a normal race:

- try to create the Pod
- if creation fails because the Pod already exists, read the Pod again
- continue with the Pod if it is reusable or becomes reusable after waiting

Create conflict is not treated as a fatal error.

### Idle Cleanup Job

The backend runs a lightweight periodic cleanup job after startup:

- interval: every 10 minutes
- scope: enabled clusters only
- target: Pods with `ketches.io/node-terminal=true`

Deletion rules:

- delete if `last-active-at` is older than 30 minutes
- delete if the Pod is in a terminal phase
- delete if the Pod is clearly unhealthy or not ready for too long

The first implementation can live in the service layer and be started during backend
initialization. A dedicated controller is unnecessary for this scope.

### Suggested Service Helpers

To keep `cluster.go` readable and testable, the node terminal path should be split into
small helpers such as:

- `buildNodeTerminalPodName(nodeName string) string`
- `buildNodeTerminalPod(nodeName, podName string) *corev1.Pod`
- `isReusableNodeTerminalPod(pod *corev1.Pod, nodeName string) bool`
- `shouldRecreateNodeTerminalPod(pod *corev1.Pod, now time.Time) bool`
- `touchNodeTerminalPodActivity(...) error`
- `cleanupIdleNodeTerminalPods(...) error`

These helpers give the implementation clear decision points that can be unit tested without
forcing every case through a full exec path.

## Frontend Design

### Terminal Panel Behavior

The frontend remains intentionally close to current behavior:

- clicking node `Terminal` opens the existing bottom terminal panel
- the first terminal session connects through the existing WebSocket path
- no extra API call is added for Pod creation or Pod status polling

This means lazy creation happens naturally on the first real terminal use.

### Multi-Session UI

The current terminal panel supports multiple sessions. That UI stays unchanged in this
iteration.

For node terminals, multiple sessions now map to:

- one helper Pod per node
- one exec stream per terminal tab

This keeps the user interaction stable while still meeting the singleton Pod goal.

### No Pre-Creation Behavior

The frontend does not create node terminal resources in advance during page load, panel mount,
or node list rendering. The only creation trigger remains the first node terminal exec
connection.

## Error Handling

### Terminal Startup Failures

If helper Pod creation or startup fails:

- the WebSocket session returns the error to the terminal as it does today
- no partially healthy Pod is kept if it is known to be unrecoverable
- the next terminal attempt can retry from a clean state

### Exec Failures

If `exec` fails but the helper Pod is otherwise healthy, the service does not immediately
delete the Pod. A transient exec transport failure should not cause Pod churn.

If `exec` fails because the Pod is no longer usable, the Pod should be deleted so the next
terminal open recreates it cleanly.

## Testing

### Backend

Add service-layer coverage for:

- stable helper Pod naming
- lazy creation when the Pod does not exist
- reuse when a healthy helper Pod already exists
- delete-and-recreate when the Pod is unhealthy or mismatched
- concurrent create conflict fallback
- idle cleanup of expired helper Pods
- preservation of application container terminal behavior

Where direct client-go mocking is too heavy, isolate pure helper logic into focused
testable functions.

### Frontend

Add or update terminal panel coverage to confirm:

- opening a node terminal still uses the existing WebSocket exec path
- node terminal flow does not issue any separate pre-create request
- terminal reconnection behavior remains compatible with the reused helper Pod model

### Manual Verification

Validate the implemented behavior with a live cluster:

1. Open a node terminal for a node with no helper Pod and confirm the helper Pod is created.
2. Close the terminal and reopen it within 30 minutes, then confirm the same helper Pod is
   reused.
3. Let the helper Pod sit idle past 30 minutes and confirm the cleanup job deletes it.
4. Open the terminal again and confirm the helper Pod is recreated on demand.

## Risks

- A long-lived privileged helper Pod increases security exposure if cleanup fails, so the
  cleanup job must be reliable.
- `alpine:latest` plus `nsenter` assumes the image contents remain compatible with the
  terminal strategy.
- Helper Pod reuse introduces more lifecycle branches than the current fire-and-forget flow,
  so test coverage must focus on recreate and cleanup decisions.
