import type { BuilderMessage, BuilderModelOption, BuilderRun, BuilderSessionDetail } from "@/api/builder-sessions"
import { EmptyState } from "@/components/shared/empty-state"
import { BuilderComposer } from "@/pages/builder-sessions/builder-composer"
import { BuilderPreviewFrame } from "@/pages/builder-sessions/builder-preview-frame"
import { BuilderPreviewPanel } from "@/pages/builder-sessions/builder-preview-panel"
import { Bot, Brain, Loader2, User } from "lucide-react"
import * as React from "react"

import { BuilderWorkspaceExportPanel } from "./builder-workspace-export-panel"

interface BuilderQueuedMessageItem {
  id: string
  content: string
  createdAt: string
  optimistic?: boolean
}

interface BuilderWorkspaceMainPanelProps {
  sessionId?: string
  selectedDetail?: BuilderSessionDetail
  latestRun?: BuilderRun
  visibleMessages: BuilderMessage[]
  activeStreamingRun: BuilderRun | null
  streamingRunId: string | null
  streamingLog: string
  queuedMessages: BuilderQueuedMessageItem[]
  shouldShowConversationLoader: boolean
  shouldShowSessionError: boolean
  isSessionLoading: boolean
  isSubmitting: boolean
  messageInput: string
  composerRef: React.RefObject<HTMLTextAreaElement | null>
  handleComposerInputChange: (value: string) => void
  handleSendMessage: () => void
  draftModelKey: string | null
  draftModelOptions: BuilderModelOption[]
  setDraftModelKey: (value: string | null) => void
  modelSelectionHint: string
  composerStatusText?: string
  draftError: string
  sessionExports: import("@/api/builder-sessions").BuilderExport[]
  promotingExportId: string | null
  setPromotingExportId: React.Dispatch<React.SetStateAction<string | null>>
  promotionForm: import("@/api/builder-sessions").BuilderExportPromotionRequest
  setPromotionForm: React.Dispatch<React.SetStateAction<import("@/api/builder-sessions").BuilderExportPromotionRequest>>
  buildPromotionForm: import("@/api/builder-sessions").BuilderExportInitialBuildPromotionRequest
  setBuildPromotionForm: React.Dispatch<React.SetStateAction<import("@/api/builder-sessions").BuilderExportInitialBuildPromotionRequest>>
  deployBuildForm: import("@/api/builder-sessions").BuilderExportDeployBuildRequest
  setDeployBuildForm: React.Dispatch<React.SetStateAction<import("@/api/builder-sessions").BuilderExportDeployBuildRequest>>
  exportPromotionPlan?: import("@/api/builder-sessions").BuilderExportPromotionPlan
  createExportPending: boolean
  promoteExportPending: boolean
  promoteExportToBuildPending: boolean
  deployExportBuildPending: boolean
  promotedBuildData?: { build: { id: string } }
  handleDownloadExport: (exportItem: import("@/api/builder-sessions").BuilderExport) => Promise<void>
  handlePromoteExport: (exportId: string) => void
  handlePromoteExportToBuild: (exportId: string) => void
  handleDeployExportBuild: () => void
  handleCreateExport: () => void
  selectedPreview?: import("@/api/builder-sessions").BuilderPreviewSummary
  handleDownloadPreview: () => void
  handleOpenPreview: () => void
  previewLaunch?: import("@/api/builder-sessions").BuilderPreviewLaunch | null
}

function ConversationMessage({ message }: { message: BuilderMessage }) {
  if (message.role === "system") {
    return <div className="text-sm text-muted-foreground">{message.content}</div>
  }

  const isUserMessage = message.role === "user"

  return (
    <div className={`flex gap-2.5 ${isUserMessage ? "flex-row-reverse" : "flex-row"}`}>
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
        {isUserMessage ? <User className="h-3.5 w-3.5" /> : <Bot className="h-3.5 w-3.5" />}
      </div>
      <div
        className={`max-w-3xl rounded-[1.25rem] px-4 py-2.5 text-sm ${isUserMessage ? "bg-primary text-primary-foreground shadow-sm" : "bg-muted/55 text-foreground"}`}
      >
        {message.content}
      </div>
    </div>
  )
}

function StreamingConversationMessage({
  run,
  streamingLog,
}: {
  run: BuilderRun
  streamingLog: string
}) {
  const statusLabel = run.status === "queued" ? "Builder queued the next run" : "Builder is working"
  const hasLog = streamingLog.trim().length > 0

  return (
    <div className="flex gap-2.5">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <Bot className="h-3.5 w-3.5" />
      </div>
      <div
        data-testid="builder-streaming-message"
        className="max-w-3xl rounded-[1.25rem] bg-muted/55 px-4 py-3 text-sm text-foreground"
      >
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>{statusLabel}</span>
        </div>
        <div className="mt-2">
          {hasLog ? (
            <pre className="whitespace-pre-wrap break-words text-sm text-foreground">{streamingLog}</pre>
          ) : (
            <span className="inline-flex items-center gap-1 text-sm text-muted-foreground">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current" />
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:120ms]" />
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-current [animation-delay:240ms]" />
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

export function BuilderWorkspaceMainPanel({
  sessionId,
  selectedDetail,
  latestRun,
  visibleMessages,
  activeStreamingRun,
  streamingRunId,
  streamingLog,
  queuedMessages,
  shouldShowConversationLoader,
  shouldShowSessionError,
  isSessionLoading,
  isSubmitting,
  messageInput,
  composerRef,
  handleComposerInputChange,
  handleSendMessage,
  draftModelKey,
  draftModelOptions,
  setDraftModelKey,
  modelSelectionHint,
  composerStatusText,
  draftError,
  sessionExports,
  promotingExportId,
  setPromotingExportId,
  promotionForm,
  setPromotionForm,
  buildPromotionForm,
  setBuildPromotionForm,
  deployBuildForm,
  setDeployBuildForm,
  exportPromotionPlan,
  createExportPending,
  promoteExportPending,
  promoteExportToBuildPending,
  deployExportBuildPending,
  promotedBuildData,
  handleDownloadExport,
  handlePromoteExport,
  handlePromoteExportToBuild,
  handleDeployExportBuild,
  handleCreateExport,
  selectedPreview,
  handleDownloadPreview,
  handleOpenPreview,
  previewLaunch,
}: BuilderWorkspaceMainPanelProps) {
  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      <div className="flex min-h-0 flex-1 flex-col">
        <div className={`min-h-0 flex-1 overflow-y-auto px-5 py-5 ${sessionId ? "" : "flex items-center justify-center"}`}>
          {shouldShowConversationLoader ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Opening Builder workspace…
            </div>
          ) : shouldShowSessionError ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              Failed to load this Builder session.
            </div>
          ) : sessionId && isSessionLoading && !selectedDetail ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Loading conversation…
            </div>
          ) : sessionId ? (
            <div className="mx-auto flex w-full max-w-4xl flex-col gap-4">
              {latestRun?.executor_policy_key || latestRun?.execution_image_ref ? (
                <div className="rounded-2xl border bg-background px-4 py-3 text-sm text-muted-foreground shadow-sm">
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                    {latestRun.planned_project_kind ? <span>Project kind: {latestRun.planned_project_kind}</span> : null}
                    {latestRun.phase ? <span>Phase: {latestRun.phase}</span> : null}
                    {latestRun.executor_policy_key ? <span>Executor: {latestRun.executor_policy_key}</span> : null}
                    {latestRun.execution_image_ref ? <span>Image: {latestRun.execution_image_ref}</span> : null}
                    {latestRun.error_class ? <span>Error class: {latestRun.error_class}</span> : null}
                  </div>
                </div>
              ) : null}

              <BuilderWorkspaceExportPanel
                sessionExports={sessionExports}
                promotingExportId={promotingExportId}
                setPromotingExportId={setPromotingExportId}
                promotionForm={promotionForm}
                setPromotionForm={setPromotionForm}
                buildPromotionForm={buildPromotionForm}
                setBuildPromotionForm={setBuildPromotionForm}
                deployBuildForm={deployBuildForm}
                setDeployBuildForm={setDeployBuildForm}
                exportPromotionPlan={exportPromotionPlan}
                createExportPending={createExportPending}
                promoteExportPending={promoteExportPending}
                promoteExportToBuildPending={promoteExportToBuildPending}
                deployExportBuildPending={deployExportBuildPending}
                promotedBuildData={promotedBuildData}
                onCreateExport={handleCreateExport}
                onDownloadExport={handleDownloadExport}
                onPromoteExport={handlePromoteExport}
                onPromoteExportToBuild={handlePromoteExportToBuild}
                onDeployExportBuild={handleDeployExportBuild}
              />

              {selectedPreview ? (
                <BuilderPreviewPanel
                  preview={selectedPreview}
                  onDownload={handleDownloadPreview}
                  onOpenPreview={handleOpenPreview}
                />
              ) : null}
              {previewLaunch?.frame_url ? <BuilderPreviewFrame frameUrl={previewLaunch.frame_url} /> : null}
              {visibleMessages.length === 0 && !activeStreamingRun ? (
                <div className="flex min-h-36 flex-col items-center justify-center rounded-[1.75rem] border bg-background px-7 py-8 text-center text-sm text-muted-foreground shadow-sm">
                  <Bot className="mb-2.5 h-5 w-5" />
                  No messages yet. Send the next instruction to keep building.
                </div>
              ) : (
                <>
                  {visibleMessages.map((message) => <ConversationMessage key={message.id || `${message.role}-${message.created_at}`} message={message} />)}
                  {activeStreamingRun ? (
                    <StreamingConversationMessage
                      run={activeStreamingRun}
                      streamingLog={streamingRunId === activeStreamingRun.id ? streamingLog : ""}
                    />
                  ) : null}
                </>
              )}
            </div>
          ) : (
            <div className="mx-auto flex w-full max-w-3xl flex-col gap-4">
              <EmptyState
                title="New conversation"
                description="Describe what you want to build. The draft stays unsaved until your first send."
                icon={Brain}
              />

              <div className="mx-auto w-full max-w-3xl">
                <BuilderComposer
                  centered
                  composerRef={composerRef}
                  value={messageInput}
                  onValueChange={handleComposerInputChange}
                  onSubmit={handleSendMessage}
                  isSubmitting={isSubmitting}
                  modelValue={draftModelKey}
                  modelOptions={draftModelOptions}
                  onModelValueChange={setDraftModelKey}
                  modelSelectionHint={modelSelectionHint}
                  statusText={composerStatusText}
                  statusError={draftError || undefined}
                />
              </div>
            </div>
          )}
        </div>

        {sessionId ? (
          <div className="shrink-0 space-y-3">
            {queuedMessages.length > 0 ? (
              <div
                data-testid="builder-queued-messages"
                className="mx-4 shrink-0 rounded-2xl border border-amber-200 bg-amber-50/80 px-4 py-3 text-sm shadow-sm"
              >
                <div className="mb-2 font-medium text-amber-950">Queued next</div>
                <div className="space-y-2">
                  {queuedMessages.map((message) => (
                    <div
                      key={message.id}
                      className="rounded-xl border border-amber-200/80 bg-background/80 px-3 py-2 text-foreground"
                    >
                      {message.content}
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
            <BuilderComposer
              composerRef={composerRef}
              value={messageInput}
              onValueChange={handleComposerInputChange}
              onSubmit={handleSendMessage}
              isSubmitting={isSubmitting}
              modelValue={draftModelKey}
              modelOptions={draftModelOptions}
              onModelValueChange={setDraftModelKey}
              modelSelectionHint={modelSelectionHint}
              statusText={undefined}
              statusError={undefined}
            />
          </div>
        ) : null}
      </div>
    </div>
  )
}
