import { FileExplorerContent } from "./file-explorer/components/file-explorer-content"
import { FileExplorerDialogs } from "./file-explorer/components/file-explorer-dialogs"
import { FileEditorView } from "./file-explorer/components/file-editor-view"
import { FileExplorerToolbar } from "./file-explorer/components/file-explorer-toolbar"
import { useFileExplorer } from "./file-explorer/hooks/use-file-explorer"
import { type FileExplorerPanelProps } from "./file-explorer/types"

export function FileExplorerPanel({ appId, instanceName, containerName }: FileExplorerPanelProps) {
  const explorer = useFileExplorer({ appId, instanceName, containerName })

  if (explorer.editingFile) {
    return (
      <FileEditorView
        editingFile={explorer.editingFile}
        isSaving={explorer.isSavingFile}
        onChangeContent={(content) => {
          explorer.setEditingFile((previous) => previous ? { ...previous, content } : previous)
        }}
        onClose={explorer.closeEditingFile}
        onForceClose={explorer.closeEditingFile}
        onSave={explorer.saveEditingFile}
      />
    )
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <FileExplorerToolbar
        currentPath={explorer.currentPath}
        pathSegments={explorer.pathSegments}
        isMultiSelect={explorer.isMultiSelect}
        isLoading={explorer.isLoading}
        viewMode={explorer.viewMode}
        onGoHome={explorer.handleGoHome}
        onNavigateUp={explorer.handleNavigateUp}
        onNavigateToRoot={() => explorer.navigateTo("/")}
        onNavigateToSegment={explorer.navigateToSegment}
        onCopyCurrentPath={explorer.copyCurrentPath}
        onBatchMove={explorer.handleBatchMove}
        onBatchCopy={explorer.handleBatchCopy}
        onCompressToContainer={explorer.handleCompressToContainer}
        onCompressAndDownload={explorer.handleCompressAndDownload}
        onBatchDelete={explorer.handleBatchDelete}
        onClearSelection={explorer.clearSelection}
        onRefresh={() => explorer.refetch()}
        onUpload={explorer.handleUpload}
        onOpenCreateFolderDialog={explorer.openCreateFolderDialog}
        onViewModeChange={explorer.setViewMode}
      />

      <input
        ref={explorer.fileInputRef}
        type="file"
        className="hidden"
        onChange={explorer.handleFileSelected}
      />

      <FileExplorerContent
        files={explorer.sortedFiles}
        isOpeningFile={explorer.isOpeningFile}
        isLoading={explorer.isLoading}
        errorMessage={explorer.errorMessage}
        hasError={explorer.hasError}
        viewMode={explorer.viewMode}
        selectedFile={explorer.selectedFile}
        selectedFiles={explorer.selectedFiles}
        onSelect={explorer.setSelectedFile}
        onToggleSelect={explorer.toggleFileSelection}
        onToggleSelectAll={explorer.toggleSelectAll}
        onOpen={explorer.handleOpen}
        onRename={explorer.openRenameDialog}
        onMove={explorer.openMoveDialog}
        onCopy={explorer.openCopyDialog}
        onDelete={explorer.handleDelete}
        onDownload={explorer.handleDownload}
        onCopyPath={explorer.handleCopyPath}
        onRetry={() => explorer.refetch()}
      />

      <div className="flex h-7 min-h-7 items-center justify-between border-t bg-muted/20 px-3 text-[10px] text-muted-foreground">
        <div className="flex items-center gap-3">
          <span>{explorer.sortedFiles.length} items</span>
          {explorer.isMultiSelect && <span>{explorer.selectedFiles.size} selected</span>}
          {!explorer.isMultiSelect && explorer.selectedFile && <span>Selected: {explorer.selectedFile.name}</span>}
        </div>
        <div className="flex items-center gap-2">
          <span className="font-mono">{explorer.currentPath}</span>
        </div>
      </div>

      <FileExplorerDialogs
        selectedFilesCount={explorer.selectedFiles.size}
        deleteTarget={explorer.deleteTarget}
        isCompressDialogOpen={explorer.isCompressDialogOpen}
        setIsCompressDialogOpen={explorer.setIsCompressDialogOpen}
        compressArchiveName={explorer.compressArchiveName}
        setCompressArchiveName={explorer.setCompressArchiveName}
        onCompressConfirm={explorer.handleCompressConfirm}
        compressPending={explorer.compressPending}
        isBatchMoveDialogOpen={explorer.isBatchMoveDialogOpen}
        setIsBatchMoveDialogOpen={explorer.setIsBatchMoveDialogOpen}
        batchMoveDestination={explorer.batchMoveDestination}
        setBatchMoveDestination={explorer.setBatchMoveDestination}
        onBatchMoveConfirm={explorer.handleBatchMoveConfirm}
        isBatchCopyDialogOpen={explorer.isBatchCopyDialogOpen}
        setIsBatchCopyDialogOpen={explorer.setIsBatchCopyDialogOpen}
        batchCopyDestination={explorer.batchCopyDestination}
        setBatchCopyDestination={explorer.setBatchCopyDestination}
        onBatchCopyConfirm={explorer.handleBatchCopyConfirm}
        isDeleteDialogOpen={explorer.isDeleteDialogOpen}
        setIsDeleteDialogOpen={explorer.setIsDeleteDialogOpen}
        onDeleteConfirm={explorer.handleDeleteConfirm}
        deletePending={explorer.deletePending}
        isBatchDeleteDialogOpen={explorer.isBatchDeleteDialogOpen}
        setIsBatchDeleteDialogOpen={explorer.setIsBatchDeleteDialogOpen}
        onBatchDeleteConfirm={explorer.handleBatchDeleteConfirm}
        isNewFolderDialogOpen={explorer.isNewFolderDialogOpen}
        setIsNewFolderDialogOpen={explorer.setIsNewFolderDialogOpen}
        newFolderName={explorer.newFolderName}
        setNewFolderName={explorer.setNewFolderName}
        onCreateFolder={explorer.handleCreateFolder}
        mkdirPending={explorer.mkdirPending}
        isRenameDialogOpen={explorer.isRenameDialogOpen}
        setIsRenameDialogOpen={explorer.setIsRenameDialogOpen}
        renameTarget={explorer.renameTarget}
        newName={explorer.newName}
        setNewName={explorer.setNewName}
        onRename={explorer.handleRename}
        movePending={explorer.movePending}
        isMoveDialogOpen={explorer.isMoveDialogOpen}
        setIsMoveDialogOpen={explorer.setIsMoveDialogOpen}
        moveTarget={explorer.moveTarget}
        moveDestination={explorer.moveDestination}
        setMoveDestination={explorer.setMoveDestination}
        onMove={explorer.handleMove}
        isCopyDialogOpen={explorer.isCopyDialogOpen}
        setIsCopyDialogOpen={explorer.setIsCopyDialogOpen}
        copyTarget={explorer.copyTarget}
        copyDestination={explorer.copyDestination}
        setCopyDestination={explorer.setCopyDestination}
        onCopy={explorer.handleCopy}
        copyPending={explorer.copyPending}
      />
    </div>
  )
}
