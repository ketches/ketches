import { type FileInfo } from "@/api/file-explorer"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { cn } from "@/lib/utils"
import { Clipboard, Download, Folder, MoreHorizontal } from "lucide-react"

import { type FileExplorerItemActionHandlers } from "../types"
import { formatSize, getFileIcon } from "../utils"
import { FileContextMenu } from "./file-context-menu"

interface FileGridViewProps extends FileExplorerItemActionHandlers {
  files: FileInfo[]
  selectedFile: FileInfo | null
  selectedFiles: Set<string>
  onSelect: (file: FileInfo) => void
  onToggleSelect: (fileName: string) => void
  onOpen: (file: FileInfo) => void
}

export function FileGridView({
  files,
  selectedFile,
  selectedFiles,
  onSelect,
  onToggleSelect,
  onOpen,
  onRename,
  onMove,
  onCopy,
  onDelete,
  onDownload,
  onCopyPath,
}: FileGridViewProps) {
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(100px,1fr))] gap-1 p-2">
      {files.map((file) => (
        <div
          key={file.name}
          className={cn(
            "flex flex-col items-center gap-1 p-2 rounded-md cursor-pointer transition-colors hover:bg-muted/50 group relative",
            selectedFile?.name === file.name && !selectedFiles.size && "bg-primary/10",
            selectedFiles.has(file.name) && "bg-primary/10"
          )}
          onClick={() => onSelect(file)}
          onDoubleClick={() => onOpen(file)}
        >
          <div
            className="absolute top-1 left-1 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={(event) => event.stopPropagation()}
          >
            <Checkbox
              checked={selectedFiles.has(file.name)}
              onCheckedChange={() => onToggleSelect(file.name)}
              className="h-3.5 w-3.5"
            />
          </div>
          <div className="absolute top-1 right-1 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button
              variant="ghost"
              size="icon-xs"
              className="h-5 w-5"
              onClick={(event) => {
                event.stopPropagation()
                onCopyPath(file)
              }}
              title="Copy path"
            >
              <Clipboard className="h-2.5 w-2.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              className="h-5 w-5"
              onClick={(event) => {
                event.stopPropagation()
                onDownload(file)
              }}
              title="Download"
            >
              <Download className="h-2.5 w-2.5" />
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
                size="icon-xs"
                className="h-5 w-5"
                onClick={(event) => event.stopPropagation()}
              >
                <MoreHorizontal className="h-3 w-3" />
              </Button>
            </FileContextMenu>
          </div>
          <div className="p-2">
            {file.type === "dir" ? (
              <Folder className="h-8 w-8 text-blue-400" />
            ) : (
              getFileIcon(file.name, file.type, "h-8 w-8")
            )}
          </div>
          <span className="text-[10px] font-mono text-center truncate w-full" title={file.name}>
            {file.name}
          </span>
          <span className="text-[9px] text-muted-foreground">
            {file.type === "dir" ? "Folder" : formatSize(file.size)}
          </span>
        </div>
      ))}
    </div>
  )
}
