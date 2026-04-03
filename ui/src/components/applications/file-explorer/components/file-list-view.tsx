import { type FileInfo } from "@/api/file-explorer"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { cn } from "@/lib/utils"
import { Download, MoreVertical } from "lucide-react"

import { type FileExplorerItemActionHandlers } from "../types"
import { formatSize, formatTime, getFileIcon } from "../utils"
import { FileContextMenu } from "./file-context-menu"

interface FileListViewProps extends FileExplorerItemActionHandlers {
  files: FileInfo[]
  selectedFile: FileInfo | null
  selectedFiles: Set<string>
  onSelect: (file: FileInfo) => void
  onToggleSelect: (fileName: string) => void
  onToggleSelectAll: () => void
  onOpen: (file: FileInfo) => void
}

export function FileListView({
  files,
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
}: FileListViewProps) {
  const allFilesSelected = files.length > 0 && selectedFiles.size === files.length
  const someFilesSelected = selectedFiles.size > 0 && selectedFiles.size < files.length

  return (
    <div className="w-full">
      <div className="grid grid-cols-[24px_1fr_70px_100px_50px_56px] gap-2 px-3 py-1.5 text-[10px] font-medium text-muted-foreground uppercase tracking-wider border-b bg-muted/10">
        <div className="flex items-center justify-center">
          <Checkbox
            checked={allFilesSelected}
            data-state={someFilesSelected ? "indeterminate" : undefined}
            onCheckedChange={onToggleSelectAll}
            className="h-3.5 w-3.5"
          />
        </div>
        <span>Name</span>
        <span className="text-right">Size</span>
        <span>Modified</span>
        <span>Perm</span>
        <span className="text-right">Actions</span>
      </div>
      {files.map((file) => (
        <div
          key={file.name}
          className={cn(
            "grid grid-cols-[24px_1fr_70px_100px_50px_56px] gap-2 px-3 py-1.5 text-xs items-center cursor-pointer transition-colors hover:bg-muted/50",
            selectedFile?.name === file.name && !selectedFiles.size && "bg-primary/10",
            selectedFiles.has(file.name) && "bg-primary/10"
          )}
          onClick={() => onSelect(file)}
          onDoubleClick={() => onOpen(file)}
        >
          <div className="flex items-center justify-center" onClick={(event) => event.stopPropagation()}>
            <Checkbox
              checked={selectedFiles.has(file.name)}
              onCheckedChange={() => onToggleSelect(file.name)}
              className="h-3.5 w-3.5"
            />
          </div>
          <div className="flex items-center gap-2 min-w-0">
            {getFileIcon(file.name, file.type)}
            <span className="truncate font-mono">{file.name}</span>
          </div>
          <span className="text-right text-muted-foreground font-mono">
            {file.type === "dir" ? "-" : formatSize(file.size)}
          </span>
          <span className="text-muted-foreground truncate">{formatTime(file.modTime)}</span>
          <span className="text-muted-foreground font-mono">{file.permissions}</span>
          <div className="flex items-center justify-end gap-0.5">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(event) => {
                event.stopPropagation()
                onDownload(file)
              }}
              title={`Download${file.type === "dir" ? " (.tar)" : ""}`}
            >
              <Download />
            </Button>
            <FileContextMenu
              file={file}
              onRename={onRename}
              onMove={onMove}
              onCopy={onCopy}
              onDelete={onDelete}
              onDownload={onDownload}
              onCopyPath={onCopyPath}
            >
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={(event) => event.stopPropagation()}
              >
                <MoreVertical />
              </Button>
            </FileContextMenu>
          </div>
        </div>
      ))}
    </div>
  )
}
