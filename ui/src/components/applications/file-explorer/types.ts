import { type FileInfo } from "@/api/file-explorer"

export interface FileExplorerPanelProps {
  appId: string
  instanceName: string
  containerName: string
}

export interface EditingFileState {
  path: string
  content: string
  originalContent: string
}

export type FileExplorerViewMode = "list" | "grid"

export interface FileExplorerItemActionHandlers {
  onRename: (file: FileInfo) => void
  onMove: (file: FileInfo) => void
  onCopy: (file: FileInfo) => void
  onDelete: (file: FileInfo) => void
  onDownload: (file: FileInfo) => void
  onCopyPath: (file: FileInfo) => void
}
