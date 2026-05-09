import { FileExplorerContent } from "./file-explorer/components/file-explorer-content"
import { FileExplorerDialogs } from "./file-explorer/components/file-explorer-dialogs"
import { FileEditorView } from "./file-explorer/components/file-editor-view"
import { FileExplorerToolbar } from "./file-explorer/components/file-explorer-toolbar"
import { useFileExplorer } from "./file-explorer/hooks/use-file-explorer"
import { type FileExplorerPanelProps } from "./file-explorer/types"

export function FileExplorerPanel({ appId, instanceName, containerName }: FileExplorerPanelProps) {
  const {
    fileInputRef,
    currentPath,
    pathSegments,
    viewMode,
    setViewMode,
    selectedFile,
    setSelectedFile,
    selectedFiles,
    isMultiSelect,
    editingFile,
    setEditingFile,
    closeEditingFile,
    saveEditingFile,
    isSavingFile,
    isOpeningFile,
    sortedFiles,
    isLoading,
    hasError,
    errorMessage,
    refetch,
    toggleFileSelection,
    toggleSelectAll,
    clearSelection,
    navigateTo,
    navigateToSegment,
    handleGoHome,
    handleNavigateUp,
    handleOpen,
    handleCompressToContainer,
    handleCompressConfirm,
    handleCompressAndDownload,
    handleBatchMove,
    handleBatchMoveConfirm,
    handleBatchDelete,
    handleBatchDeleteConfirm,
    handleBatchCopy,
    handleBatchCopyConfirm,
    handleCreateFolder,
    handleRename,
    handleMove,
    handleCopy,
    handleDelete,
    handleDeleteConfirm,
    handleDownload,
    handleCopyPath,
    handleUpload,
    handleFileSelected,
    openCreateFolderDialog,
    openRenameDialog,
    openMoveDialog,
    openCopyDialog,
    copyCurrentPath,
    isNewFolderDialogOpen,
    setIsNewFolderDialogOpen,
    newFolderName,
    setNewFolderName,
    isRenameDialogOpen,
    setIsRenameDialogOpen,
    renameTarget,
    newName,
    setNewName,
    isMoveDialogOpen,
    setIsMoveDialogOpen,
    moveTarget,
    moveDestination,
    setMoveDestination,
    isCopyDialogOpen,
    setIsCopyDialogOpen,
    copyTarget,
    copyDestination,
    setCopyDestination,
    isCompressDialogOpen,
    setIsCompressDialogOpen,
    compressArchiveName,
    setCompressArchiveName,
    isBatchMoveDialogOpen,
    setIsBatchMoveDialogOpen,
    batchMoveDestination,
    setBatchMoveDestination,
    isBatchCopyDialogOpen,
    setIsBatchCopyDialogOpen,
    batchCopyDestination,
    setBatchCopyDestination,
    isDeleteDialogOpen,
    setIsDeleteDialogOpen,
    deleteTarget,
    isBatchDeleteDialogOpen,
    setIsBatchDeleteDialogOpen,
    mkdirPending,
    movePending,
    copyPending,
    deletePending,
    compressPending,
  } = useFileExplorer({ appId, instanceName, containerName })

  if (editingFile) {
    return (
      <FileEditorView
        editingFile={editingFile}
        isSaving={isSavingFile}
        onChangeContent={(content) => {
          setEditingFile((previous) => previous ? { ...previous, content } : previous)
        }}
        onClose={closeEditingFile}
        onForceClose={closeEditingFile}
        onSave={saveEditingFile}
      />
    )
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <FileExplorerToolbar
        currentPath={currentPath}
        pathSegments={pathSegments}
        isMultiSelect={isMultiSelect}
        isLoading={isLoading}
        viewMode={viewMode}
        onGoHome={handleGoHome}
        onNavigateUp={handleNavigateUp}
        onNavigateToRoot={() => navigateTo("/")}
        onNavigateToSegment={navigateToSegment}
        onCopyCurrentPath={copyCurrentPath}
        onBatchMove={handleBatchMove}
        onBatchCopy={handleBatchCopy}
        onCompressToContainer={handleCompressToContainer}
        onCompressAndDownload={handleCompressAndDownload}
        onBatchDelete={handleBatchDelete}
        onClearSelection={clearSelection}
        onRefresh={() => refetch()}
        onUpload={handleUpload}
        onOpenCreateFolderDialog={openCreateFolderDialog}
        onViewModeChange={setViewMode}
      />

      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileSelected}
      />

      <FileExplorerContent
        files={sortedFiles}
        isOpeningFile={isOpeningFile}
        isLoading={isLoading}
        errorMessage={errorMessage}
        hasError={hasError}
        viewMode={viewMode}
        selectedFile={selectedFile}
        selectedFiles={selectedFiles}
        onSelect={setSelectedFile}
        onToggleSelect={toggleFileSelection}
        onToggleSelectAll={toggleSelectAll}
        onOpen={handleOpen}
        onRename={openRenameDialog}
        onMove={openMoveDialog}
        onCopy={openCopyDialog}
        onDelete={handleDelete}
        onDownload={handleDownload}
        onCopyPath={handleCopyPath}
        onRetry={() => refetch()}
      />

      <div className="flex h-7 min-h-7 items-center justify-between border-t bg-muted/20 px-3 text-[10px] text-muted-foreground">
        <div className="flex items-center gap-3">
          <span>{sortedFiles.length} items</span>
          {isMultiSelect && <span>{selectedFiles.size} selected</span>}
          {!isMultiSelect && selectedFile && <span>Selected: {selectedFile.name}</span>}
        </div>
        <div className="flex items-center gap-2">
          <span className="font-mono">{currentPath}</span>
        </div>
      </div>

      <FileExplorerDialogs
        selectedFilesCount={selectedFiles.size}
        deleteTarget={deleteTarget}
        isCompressDialogOpen={isCompressDialogOpen}
        setIsCompressDialogOpen={setIsCompressDialogOpen}
        compressArchiveName={compressArchiveName}
        setCompressArchiveName={setCompressArchiveName}
        onCompressConfirm={handleCompressConfirm}
        compressPending={compressPending}
        isBatchMoveDialogOpen={isBatchMoveDialogOpen}
        setIsBatchMoveDialogOpen={setIsBatchMoveDialogOpen}
        batchMoveDestination={batchMoveDestination}
        setBatchMoveDestination={setBatchMoveDestination}
        onBatchMoveConfirm={handleBatchMoveConfirm}
        isBatchCopyDialogOpen={isBatchCopyDialogOpen}
        setIsBatchCopyDialogOpen={setIsBatchCopyDialogOpen}
        batchCopyDestination={batchCopyDestination}
        setBatchCopyDestination={setBatchCopyDestination}
        onBatchCopyConfirm={handleBatchCopyConfirm}
        isDeleteDialogOpen={isDeleteDialogOpen}
        setIsDeleteDialogOpen={setIsDeleteDialogOpen}
        onDeleteConfirm={handleDeleteConfirm}
        deletePending={deletePending}
        isBatchDeleteDialogOpen={isBatchDeleteDialogOpen}
        setIsBatchDeleteDialogOpen={setIsBatchDeleteDialogOpen}
        onBatchDeleteConfirm={handleBatchDeleteConfirm}
        isNewFolderDialogOpen={isNewFolderDialogOpen}
        setIsNewFolderDialogOpen={setIsNewFolderDialogOpen}
        newFolderName={newFolderName}
        setNewFolderName={setNewFolderName}
        onCreateFolder={handleCreateFolder}
        mkdirPending={mkdirPending}
        isRenameDialogOpen={isRenameDialogOpen}
        setIsRenameDialogOpen={setIsRenameDialogOpen}
        renameTarget={renameTarget}
        newName={newName}
        setNewName={setNewName}
        onRename={handleRename}
        movePending={movePending}
        isMoveDialogOpen={isMoveDialogOpen}
        setIsMoveDialogOpen={setIsMoveDialogOpen}
        moveTarget={moveTarget}
        moveDestination={moveDestination}
        setMoveDestination={setMoveDestination}
        onMove={handleMove}
        isCopyDialogOpen={isCopyDialogOpen}
        setIsCopyDialogOpen={setIsCopyDialogOpen}
        copyTarget={copyTarget}
        copyDestination={copyDestination}
        setCopyDestination={setCopyDestination}
        onCopy={handleCopy}
        copyPending={copyPending}
      />
    </div>
  )
}
