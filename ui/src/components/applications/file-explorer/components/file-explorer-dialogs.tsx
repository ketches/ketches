import { type FileInfo } from "@/api/file-explorer"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Loader2 } from "lucide-react"

interface FileExplorerDialogsProps {
  selectedFilesCount: number
  deleteTarget: FileInfo | null
  isCompressDialogOpen: boolean
  setIsCompressDialogOpen: (open: boolean) => void
  compressArchiveName: string
  setCompressArchiveName: (value: string) => void
  onCompressConfirm: () => void
  compressPending: boolean
  isBatchMoveDialogOpen: boolean
  setIsBatchMoveDialogOpen: (open: boolean) => void
  batchMoveDestination: string
  setBatchMoveDestination: (value: string) => void
  onBatchMoveConfirm: () => void
  isBatchCopyDialogOpen: boolean
  setIsBatchCopyDialogOpen: (open: boolean) => void
  batchCopyDestination: string
  setBatchCopyDestination: (value: string) => void
  onBatchCopyConfirm: () => void
  isDeleteDialogOpen: boolean
  setIsDeleteDialogOpen: (open: boolean) => void
  onDeleteConfirm: () => void
  deletePending: boolean
  isBatchDeleteDialogOpen: boolean
  setIsBatchDeleteDialogOpen: (open: boolean) => void
  onBatchDeleteConfirm: () => void
  isNewFolderDialogOpen: boolean
  setIsNewFolderDialogOpen: (open: boolean) => void
  newFolderName: string
  setNewFolderName: (value: string) => void
  onCreateFolder: () => void
  mkdirPending: boolean
  isRenameDialogOpen: boolean
  setIsRenameDialogOpen: (open: boolean) => void
  renameTarget: FileInfo | null
  newName: string
  setNewName: (value: string) => void
  onRename: () => void
  movePending: boolean
  isMoveDialogOpen: boolean
  setIsMoveDialogOpen: (open: boolean) => void
  moveTarget: FileInfo | null
  moveDestination: string
  setMoveDestination: (value: string) => void
  onMove: () => void
  isCopyDialogOpen: boolean
  setIsCopyDialogOpen: (open: boolean) => void
  copyTarget: FileInfo | null
  copyDestination: string
  setCopyDestination: (value: string) => void
  onCopy: () => void
  copyPending: boolean
}

export function FileExplorerDialogs({
  selectedFilesCount,
  deleteTarget,
  isCompressDialogOpen,
  setIsCompressDialogOpen,
  compressArchiveName,
  setCompressArchiveName,
  onCompressConfirm,
  compressPending,
  isBatchMoveDialogOpen,
  setIsBatchMoveDialogOpen,
  batchMoveDestination,
  setBatchMoveDestination,
  onBatchMoveConfirm,
  isBatchCopyDialogOpen,
  setIsBatchCopyDialogOpen,
  batchCopyDestination,
  setBatchCopyDestination,
  onBatchCopyConfirm,
  isDeleteDialogOpen,
  setIsDeleteDialogOpen,
  onDeleteConfirm,
  deletePending,
  isBatchDeleteDialogOpen,
  setIsBatchDeleteDialogOpen,
  onBatchDeleteConfirm,
  isNewFolderDialogOpen,
  setIsNewFolderDialogOpen,
  newFolderName,
  setNewFolderName,
  onCreateFolder,
  mkdirPending,
  isRenameDialogOpen,
  setIsRenameDialogOpen,
  renameTarget,
  newName,
  setNewName,
  onRename,
  movePending,
  isMoveDialogOpen,
  setIsMoveDialogOpen,
  moveTarget,
  moveDestination,
  setMoveDestination,
  onMove,
  isCopyDialogOpen,
  setIsCopyDialogOpen,
  copyTarget,
  copyDestination,
  setCopyDestination,
  onCopy,
  copyPending,
}: FileExplorerDialogsProps) {
  return (
    <>
      <Dialog open={isCompressDialogOpen} onOpenChange={setIsCompressDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Compress Files</DialogTitle>
            <DialogDescription>Compress {selectedFilesCount} selected file(s) into an archive in the current directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Archive name (e.g., archive.tar.gz)"
              value={compressArchiveName}
              onChange={(event) => setCompressArchiveName(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onCompressConfirm()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsCompressDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onCompressConfirm} disabled={!compressArchiveName.trim() || compressPending}>
                {compressPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Compress
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isBatchMoveDialogOpen} onOpenChange={setIsBatchMoveDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Move Files</DialogTitle>
            <DialogDescription>Move {selectedFilesCount} selected file(s) to a new directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Destination path (e.g., /tmp)"
              value={batchMoveDestination}
              onChange={(event) => setBatchMoveDestination(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onBatchMoveConfirm()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsBatchMoveDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onBatchMoveConfirm} disabled={!batchMoveDestination.trim()}>
                Move
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isBatchCopyDialogOpen} onOpenChange={setIsBatchCopyDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Copy Files</DialogTitle>
            <DialogDescription>Copy {selectedFilesCount} selected file(s) to a new directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Destination path (e.g., /tmp)"
              value={batchCopyDestination}
              onChange={(event) => setBatchCopyDestination(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onBatchCopyConfirm()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsBatchCopyDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onBatchCopyConfirm} disabled={!batchCopyDestination.trim()}>
                Copy
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deleteTarget?.type === "dir" ? "Folder" : "File"}?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deleteTarget?.name}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={onDeleteConfirm}
              variant="destructive"
            >
              {deletePending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={isBatchDeleteDialogOpen} onOpenChange={setIsBatchDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {selectedFilesCount} File(s)?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedFilesCount} selected file(s)? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={onBatchDeleteConfirm}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={isNewFolderDialogOpen} onOpenChange={setIsNewFolderDialogOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>New Folder</DialogTitle>
            <DialogDescription>Create a new folder in the current directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Folder name"
              value={newFolderName}
              onChange={(event) => setNewFolderName(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onCreateFolder()}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsNewFolderDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onCreateFolder} disabled={!newFolderName.trim() || mkdirPending}>
                {mkdirPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Create
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isRenameDialogOpen} onOpenChange={setIsRenameDialogOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Rename</DialogTitle>
            <DialogDescription>Enter a new name for "{renameTarget?.name}".</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="New name"
              value={newName}
              onChange={(event) => setNewName(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onRename()}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsRenameDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onRename} disabled={!newName.trim() || movePending}>
                {movePending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Rename
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isMoveDialogOpen} onOpenChange={setIsMoveDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Move</DialogTitle>
            <DialogDescription>Move "{moveTarget?.name}" to a new location.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Destination path (e.g., /tmp/myfile)"
              value={moveDestination}
              onChange={(event) => setMoveDestination(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onMove()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsMoveDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onMove} disabled={!moveDestination.trim() || movePending}>
                {movePending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Move
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isCopyDialogOpen} onOpenChange={setIsCopyDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Copy</DialogTitle>
            <DialogDescription>Copy "{copyTarget?.name}" to a new location.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Destination path (e.g., /tmp/myfile.copy)"
              value={copyDestination}
              onChange={(event) => setCopyDestination(event.target.value)}
              onKeyDown={(event) => event.key === "Enter" && onCopy()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsCopyDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={onCopy} disabled={!copyDestination.trim() || copyPending}>
                {copyPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Copy
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
