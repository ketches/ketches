import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { ArrowLeft, Clipboard, FileCode, Loader2, Save, X } from "lucide-react"
import * as React from "react"

import { type EditingFileState } from "../types"
import { copyToClipboard, formatSize } from "../utils"

interface FileEditorViewProps {
  editingFile: EditingFileState
  isSaving: boolean
  onChangeContent: (content: string) => void
  onClose: () => void
  onForceClose: () => void
  onSave: () => void
}

export function FileEditorView({
  editingFile,
  isSaving,
  onChangeContent,
  onClose,
  onForceClose,
  onSave,
}: FileEditorViewProps) {
  const hasChanges = editingFile.content !== editingFile.originalContent
  const [isUnsavedDialogOpen, setIsUnsavedDialogOpen] = React.useState(false)

  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key === "s") {
        event.preventDefault()
        if (hasChanges && !isSaving) {
          onSave()
        }
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [hasChanges, isSaving, onSave])

  const handleClose = React.useCallback(() => {
    if (hasChanges) {
      setIsUnsavedDialogOpen(true)
      return
    }

    onClose()
  }, [hasChanges, onClose])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/20">
        <div className="flex items-center gap-2 min-w-0">
          <Button variant="ghost" size="icon-xs" onClick={handleClose}>
            <ArrowLeft className="h-3.5 w-3.5" />
          </Button>
          <FileCode className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
          <Tooltip>
            <TooltipTrigger>
              <span className="text-xs font-mono truncate cursor-default">{editingFile.path}</span>
            </TooltipTrigger>
            <TooltipContent className="text-[10px] font-mono">{editingFile.path}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger>
              <Button
                variant="ghost"
                size="icon-xs"
                className="h-5 w-5 shrink-0"
                onClick={() => copyToClipboard(editingFile.path)}
              >
                <Clipboard className="h-3 w-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent className="text-[10px]">Copy file path</TooltipContent>
          </Tooltip>
          {hasChanges && (
            <span className="text-[10px] bg-yellow-500/20 text-yellow-500 px-1.5 py-0.5 rounded shrink-0">Modified</span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="default"
            size="sm"
            className="h-7 text-xs"
            disabled={!hasChanges || isSaving}
            onClick={onSave}
          >
            {isSaving ? (
              <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />
            ) : (
              <Save className="h-3 w-3 mr-1.5" />
            )}
            Save
          </Button>
          <Button variant="ghost" size="icon-xs" onClick={handleClose}>
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-hidden relative">
        <Textarea
          value={editingFile.content}
          onChange={(event) => onChangeContent(event.target.value)}
          className="h-full w-full resize-none rounded-none border-0 font-mono text-xs leading-relaxed focus-visible:ring-0 focus-visible:ring-offset-0 p-3"
          spellCheck={false}
          autoFocus
        />
      </div>

      <div className="flex items-center justify-between px-3 py-1 border-t bg-muted/20 text-[10px] text-muted-foreground">
        <div className="flex items-center gap-3">
          <span>Lines: {editingFile.content.split("\n").length}</span>
          <span>Size: {formatSize(new Blob([editingFile.content]).size)}</span>
        </div>
        <div className="flex items-center gap-2">
          <kbd className="px-1 py-0.5 bg-muted rounded text-[9px]">{navigator.platform.includes("Mac") ? "Cmd" : "Ctrl"}+S</kbd>
          <span>to save</span>
        </div>
      </div>

      <AlertDialog open={isUnsavedDialogOpen} onOpenChange={setIsUnsavedDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
            <AlertDialogDescription>
              You have unsaved changes. Are you sure you want to close without saving?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setIsUnsavedDialogOpen(false)
                onForceClose()
              }}
              variant="destructive"
            >
              Discard Changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
