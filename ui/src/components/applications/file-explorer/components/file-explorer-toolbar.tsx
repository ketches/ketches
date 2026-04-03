import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import {
  Archive,
  ArrowLeft,
  ChevronRight,
  Clipboard,
  Copy,
  FileArchive,
  FileOutput,
  FolderPlus,
  Home,
  LayoutGrid,
  List,
  RefreshCw,
  Trash2,
  Upload,
  X,
} from "lucide-react"

import { type FileExplorerViewMode } from "../types"

interface FileExplorerToolbarProps {
  currentPath: string
  pathSegments: string[]
  isMultiSelect: boolean
  isLoading: boolean
  viewMode: FileExplorerViewMode
  onGoHome: () => void
  onNavigateUp: () => void
  onNavigateToRoot: () => void
  onNavigateToSegment: (index: number) => void
  onCopyCurrentPath: () => void
  onBatchMove: () => void
  onBatchCopy: () => void
  onCompressToContainer: () => void
  onCompressAndDownload: () => void
  onBatchDelete: () => void
  onClearSelection: () => void
  onRefresh: () => void
  onUpload: () => void
  onOpenCreateFolderDialog: () => void
  onViewModeChange: (mode: FileExplorerViewMode) => void
}

export function FileExplorerToolbar({
  currentPath,
  pathSegments,
  isMultiSelect,
  isLoading,
  viewMode,
  onGoHome,
  onNavigateUp,
  onNavigateToRoot,
  onNavigateToSegment,
  onCopyCurrentPath,
  onBatchMove,
  onBatchCopy,
  onCompressToContainer,
  onCompressAndDownload,
  onBatchDelete,
  onClearSelection,
  onRefresh,
  onUpload,
  onOpenCreateFolderDialog,
  onViewModeChange,
}: FileExplorerToolbarProps) {
  return (
    <div className="flex h-8 min-h-8 items-center justify-between border-b bg-muted/20 px-3">
      <div className="flex items-center gap-2 min-w-0 flex-1">
        <Tooltip>
          <TooltipTrigger
            delay={200}
            render={<Button variant="ghost" size="icon-sm" onClick={onGoHome} />}
          >
            <Home className="h-3.5 w-3.5" />
          </TooltipTrigger>
          <TooltipContent>Go to home directory</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            delay={200}
            render={(
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={onNavigateUp}
                disabled={currentPath === "/"}
              />
            )}
          >
            <ArrowLeft className="h-3.5 w-3.5" />
          </TooltipTrigger>
          <TooltipContent>Go up</TooltipContent>
        </Tooltip>

        <div className="flex items-center gap-0.5 text-xs min-w-0 overflow-x-auto no-scrollbar group/breadcrumb">
          <Tooltip>
            <TooltipTrigger>
              <button
                onClick={onNavigateToRoot}
                className={cn(
                  "px-1.5 py-0.5 rounded hover:bg-muted transition-colors font-mono shrink-0",
                  currentPath === "/" ? "text-foreground font-medium" : "text-muted-foreground"
                )}
              >
                /
              </button>
            </TooltipTrigger>
            <TooltipContent className="text-[10px]">
              Go to root directory
            </TooltipContent>
          </Tooltip>
          {pathSegments.map((segment, index) => (
            <div key={`${segment}-${index}`} className="contents">
              <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />
              <Tooltip>
                <TooltipTrigger>
                  <button
                    onClick={() => onNavigateToSegment(index)}
                    className={cn(
                      "px-1.5 py-0.5 rounded hover:bg-muted transition-colors font-mono truncate max-w-32 shrink-0",
                      index === pathSegments.length - 1 ? "text-foreground font-medium" : "text-muted-foreground"
                    )}
                  >
                    {segment}
                  </button>
                </TooltipTrigger>
                <TooltipContent className="text-[10px]">
                  {currentPath === "/" ? "/" : `/${pathSegments.slice(0, index + 1).join("/")}`}
                </TooltipContent>
              </Tooltip>
            </div>
          ))}
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={(
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="opacity-0 group-hover/breadcrumb:opacity-100 transition-opacity shrink-0"
                  onClick={onCopyCurrentPath}
                />
              )}
            >
              <Clipboard className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>Copy current path</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <div className="flex items-center gap-1 shrink-0">
        {isMultiSelect && (
          <>
            <Tooltip>
              <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onBatchMove} />}>
                <FileOutput className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Move selected</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onBatchCopy} />}>
                <Copy className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Copy selected</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onCompressToContainer} />}>
                <Archive className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Compress</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onCompressAndDownload} />}>
                <FileArchive className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Compress & Download</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={(
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                    onClick={onBatchDelete}
                  />
                )}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Delete selected</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onClearSelection} />}>
                <X className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Clear selection</TooltipContent>
            </Tooltip>
            <div className="h-4 w-px bg-border mx-0.5" />
          </>
        )}

        <Button variant="ghost" size="icon-sm" onClick={onRefresh} disabled={isLoading}>
          <RefreshCw className={cn("h-3.5 w-3.5", isLoading && "animate-spin")} />
        </Button>

        <Tooltip>
          <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onUpload} />}>
            <Upload className="h-3.5 w-3.5" />
          </TooltipTrigger>
          <TooltipContent>Upload file</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger delay={200} render={<Button variant="ghost" size="icon-sm" onClick={onOpenCreateFolderDialog} />}>
            <FolderPlus className="h-3.5 w-3.5" />
          </TooltipTrigger>
          <TooltipContent>New folder</TooltipContent>
        </Tooltip>

        <Separator orientation="vertical" className="mt-1 mb-1 mx-2" />

        <div className="flex items-center bg-muted rounded-md">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={(
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className={cn("rounded-r-none", viewMode === "list" && "bg-background shadow-sm")}
                  onClick={() => onViewModeChange("list")}
                />
              )}
            >
              <List className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>List view</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={(
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className={cn("rounded-l-none", viewMode === "grid" && "bg-background shadow-sm")}
                  onClick={() => onViewModeChange("grid")}
                />
              )}
            >
              <LayoutGrid className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>Grid view</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  )
}
