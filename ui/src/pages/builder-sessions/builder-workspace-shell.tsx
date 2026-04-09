import { PageHeader } from "@/components/layout/page-header"

import { BuilderWorkspaceMainPanel } from "./components/builder-workspace-main-panel"
import { BuilderWorkspaceSidebar } from "./components/builder-workspace-sidebar"
import { useBuilderWorkspaceFiles } from "./hooks/use-builder-workspace-files"
import { useBuilderWorkspaceSession } from "./hooks/use-builder-workspace-session"

export function BuilderWorkspaceShell() {
  const workspace = useBuilderWorkspaceSession()
  const files = useBuilderWorkspaceFiles({
    projectId: workspace.projectId,
    sessionId: workspace.sessionId,
    hasFiles: workspace.hasFiles,
  })

  return (
    <div
      data-testid="builder-workspace-shell"
      className="flex h-full min-h-0 flex-1 flex-col overflow-hidden"
    >
      <PageHeader items={workspace.breadcrumbItems} />

      <div
        data-testid="builder-workspace-body"
        className="flex h-full min-h-0 flex-1 overflow-hidden bg-background"
      >
        <BuilderWorkspaceSidebar
          projectId={workspace.projectId}
          sessionId={workspace.sessionId}
          sessions={workspace.sessions}
          onNewConversation={workspace.handleOpenDraft}
          onSelectSession={(targetSessionId) => {
            workspace.navigate(`/builder-sessions/${targetSessionId}`)
          }}
          hasFiles={workspace.hasFiles}
          filesExpanded={files.filesExpanded}
          setFilesExpanded={files.setFilesExpanded}
          currentPath={files.currentPath}
          filesData={files.filesData}
          selectedFile={files.selectedFile}
          fileContent={files.fileContent}
          onSelectFile={files.handleSelectFile}
          onNavigateParent={files.handleNavigateParent}
          onDownloadFiles={files.handleDownloadFiles}
        />

        <section
          data-testid="builder-workspace-chat-column"
          className="flex h-full min-h-0 min-w-0 flex-1 overflow-hidden bg-muted/5"
        >
          <BuilderWorkspaceMainPanel
            sessionId={workspace.sessionId}
            selectedDetail={workspace.selectedDetail}
            latestRun={workspace.latestRun}
            visibleMessages={workspace.visibleMessages}
            activeStreamingRun={workspace.activeStreamingRun}
            streamingRunId={workspace.streamingRunId}
            streamingLog={workspace.streamingLog}
            queuedMessages={workspace.queuedMessages}
            shouldShowConversationLoader={workspace.shouldShowConversationLoader}
            shouldShowSessionError={workspace.shouldShowSessionError}
            isSessionLoading={workspace.isSessionLoading}
            isSubmitting={workspace.isSubmitting}
            messageInput={workspace.messageInput}
            composerRef={workspace.composerRef}
            handleComposerInputChange={workspace.handleComposerInputChange}
            handleSendMessage={workspace.handleSendMessage}
            draftModelKey={workspace.draftModelKey}
            draftModelOptions={workspace.draftModelOptions}
            setDraftModelKey={workspace.setDraftModelKey}
            modelSelectionHint={workspace.modelSelectionHint}
            composerStatusText={workspace.composerStatusText}
            draftError={workspace.draftError}
            sessionExports={workspace.sessionExports}
            promotingExportId={workspace.promotingExportId}
            setPromotingExportId={workspace.setPromotingExportId}
            promotionForm={workspace.promotionForm}
            setPromotionForm={workspace.setPromotionForm}
            buildPromotionForm={workspace.buildPromotionForm}
            setBuildPromotionForm={workspace.setBuildPromotionForm}
            deployBuildForm={workspace.deployBuildForm}
            setDeployBuildForm={workspace.setDeployBuildForm}
            exportPromotionPlan={workspace.exportPromotionPlan}
            createExportPending={workspace.createExportMutation.isPending}
            promoteExportPending={workspace.promoteExportMutation.isPending}
            promoteExportToBuildPending={workspace.promoteExportToBuildMutation.isPending}
            deployExportBuildPending={workspace.deployExportBuildMutation.isPending}
            promotedBuildData={workspace.promoteExportToBuildMutation.data}
            handleDownloadExport={workspace.handleDownloadExport}
            handlePromoteExport={workspace.handlePromoteExport}
            handlePromoteExportToBuild={workspace.handlePromoteExportToBuild}
            handleDeployExportBuild={workspace.handleDeployExportBuild}
            handleCreateExport={() => workspace.createExportMutation.mutate()}
            selectedPreview={workspace.selectedPreview}
            handleDownloadPreview={workspace.handleDownloadPreview}
            handleOpenPreview={workspace.handleOpenPreview}
            previewLaunch={workspace.previewLaunch}
          />
        </section>
      </div>
    </div>
  )
}
