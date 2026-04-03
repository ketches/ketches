import { type FileInfo } from "@/api/file-explorer"
import { Button } from "@/components/ui/button"
import { FolderOpen, Loader2 } from "lucide-react"

import { type FileExplorerItemActionHandlers, type FileExplorerViewMode } from "../types"
import { FileGridView } from "./file-grid-view"
import { FileListView } from "./file-list-view"

interface FileExplorerContentProps extends FileExplorerItemActionHandlers {
  files: FileInfo[]
  isOpeningFile: boolean
  isLoading: boolean
  errorMessage?: string
  hasError: boolean
  viewMode: FileExplorerViewMode
  selectedFile: FileInfo | null
  selectedFiles: Set<string>
  onSelect: (file: FileInfo) => void
  onToggleSelect: (fileName: string) => void
  onToggleSelectAll: () => void
  onOpen: (file: FileInfo) => void
  onRetry: () => void
}

export function FileExplorerContent({
  files,
  isOpeningFile,
  isLoading,
  errorMessage,
  hasError,
  viewMode,
  selectedFile,
  selectedFiles,
  onSelect,
  onToggleSelect,
  onToggleSelectAll,
  onOpen,
  onRename,
  onMove,
  onCopy,
  onDelete,
  onDownload,
  onCopyPath,
  onRetry,
}: FileExplorerContentProps) {
  return (
    <div className="relative min-h-0 flex-1 overflow-auto">
      {isOpeningFile && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/60 z-10">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Loading file...</span>
          </div>
        </div>
      )}
      {isLoading ? (
        <div className="flex items-center justify-center h-full">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : hasError ? (
        <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
          <FolderOpen className="h-8 w-8 opacity-30" />
          <p className="text-xs">Failed to load files</p>
          <p className="text-[10px]">{errorMessage}</p>
          <Button variant="outline" size="sm" onClick={onRetry} className="mt-2 h-7 text-xs">
            Retry
          </Button>
        </div>
      ) : files.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
          <FolderOpen className="h-8 w-8 opacity-30" />
          <p className="text-xs">Empty directory</p>
        </div>
      ) : viewMode === "list" ? (
        <FileListView
          files={files}
          selectedFile={selectedFile}
          selectedFiles={selectedFiles}
          onSelect={onSelect}
          onToggleSelect={onToggleSelect}
          onToggleSelectAll={onToggleSelectAll}
          onOpen={onOpen}
          onRename={onRename}
          onMove={onMove}
          onCopy={onCopy}
          onDelete={onDelete}
          onDownload={onDownload}
          onCopyPath={onCopyPath}
        />
      ) : (
        <FileGridView
          files={files}
          selectedFile={selectedFile}
          selectedFiles={selectedFiles}
          onSelect={onSelect}
          onToggleSelect={onToggleSelect}
          onOpen={onOpen}
          onRename={onRename}
          onMove={onMove}
          onCopy={onCopy}
          onDelete={onDelete}
          onDownload={onDownload}
          onCopyPath={onCopyPath}
        />
      )}
    </div>
  )
}
