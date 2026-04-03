import { type FileInfo } from "@/api/file-explorer"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Clipboard, Copy, Download, FileOutput, Pencil, Trash2 } from "lucide-react"
import * as React from "react"

import { type FileExplorerItemActionHandlers } from "../types"

interface FileContextMenuProps extends FileExplorerItemActionHandlers {
  file: FileInfo
  children: React.ReactElement
}

export function FileContextMenu({
  file,
  children,
  onRename,
  onMove,
  onCopy,
  onDelete,
  onDownload,
  onCopyPath,
}: FileContextMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={children} />
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuItem onClick={() => onCopyPath(file)}>
          <Clipboard className="h-3.5 w-3.5 mr-2" />
          Copy Path
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => onRename(file)}>
          <Pencil className="h-3.5 w-3.5 mr-2" />
          Rename
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onMove(file)}>
          <FileOutput className="h-3.5 w-3.5 mr-2" />
          Move
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onCopy(file)}>
          <Copy className="h-3.5 w-3.5 mr-2" />
          Copy
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onDownload(file)}>
          <Download className="h-3.5 w-3.5 mr-2" />
          Download{file.type === "dir" ? " (.tar)" : ""}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => onDelete(file)} className="text-destructive hover:text-destructive hover:bg-destructive/10">
          <Trash2 className="h-3.5 w-3.5 mr-2" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
