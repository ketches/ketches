import { formatDate } from "@/lib/utils"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Archive,
  ArrowLeft,
  ChevronRight,
  Clipboard,
  Copy,
  Download,
  File,
  FileArchive,
  FileClock,
  FileCode,
  FileImage,
  FileOutput,
  Folder,
  FolderOpen,
  FolderPlus,
  Home,
  LayoutGrid,
  List,
  Loader2,
  MoreHorizontal,
  MoreVertical,
  Pencil,
  RefreshCw,
  Save,
  Trash2,
  Upload,
  X
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { fileExplorerApi, type FileInfo } from "@/api/file-explorer"
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
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import type { AxiosError } from "axios"
import { Separator } from "../ui/separator"

// Copy text to clipboard with toast feedback
function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    toast.success("Path copied to clipboard")
  }).catch(() => {
    toast.error("Failed to copy path")
  })
}

interface FileExplorerPanelProps {
  appId: string
  instanceName: string
  containerName: string
}

const FILE_VIEW_MODE_KEY = "file_explorer_view_mode"

// Determine file icon based on extension
function getFileIcon(name: string, type: string) {
  if (type === "dir") return <Folder className="h-4 w-4 text-blue-400" />
  if (type === "link") return <File className="h-4 w-4 text-purple-400" />

  const ext = name.split(".").pop()?.toLowerCase() || ""
  const codeExts = ["js", "ts", "jsx", "tsx", "go", "py", "rs", "java", "c", "cpp", "h", "rb", "php", "sh", "bash", "zsh", "yaml", "yml", "toml", "json", "xml", "html", "css", "scss", "sql"]
  const imageExts = ["png", "jpg", "jpeg", "gif", "svg", "webp", "ico", "bmp"]
  const textExts = ["txt", "md", "log", "csv", "env", "conf", "cfg", "ini", "properties"]

  if (codeExts.includes(ext)) return <FileCode className="h-4 w-4 text-emerald-400" />
  if (imageExts.includes(ext)) return <FileImage className="h-4 w-4 text-orange-400" />
  if (textExts.includes(ext)) return <FileClock className="h-4 w-4 text-yellow-400" />
  return <File className="h-4 w-4 text-muted-foreground" />
}

// Format file size
function formatSize(bytes: number): string {
  if (bytes === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

// Format timestamp to human-readable
function formatTime(timestamp: number): string {
  if (timestamp === 0) return "-"
  return formatDate(timestamp * 1000)
}

// Check if a file is likely text/editable
function isTextFile(name: string): boolean {
  const ext = name.split(".").pop()?.toLowerCase() || ""
  const textExts = [
    "txt", "md", "log", "csv", "env", "conf", "cfg", "ini", "properties",
    "js", "ts", "jsx", "tsx", "go", "py", "rs", "java", "c", "cpp", "h",
    "rb", "php", "sh", "bash", "zsh", "yaml", "yml", "toml", "json",
    "xml", "html", "css", "scss", "sql", "makefile", "dockerfile",
    "gitignore", "dockerignore", "editorconfig",
  ]
  // Also consider files without extension (like Makefile, Dockerfile)
  const nameLower = name.toLowerCase()
  const noExtNames = ["makefile", "dockerfile", "jenkinsfile", "vagrantfile", "gemfile", "rakefile", "procfile", "license", "readme", "changelog"]
  return textExts.includes(ext) || noExtNames.includes(nameLower)
}

// Build path from segments
function buildPath(segments: string[]): string {
  if (segments.length === 0) return "/"
  return "/" + segments.join("/")
}

// Parse path into segments
function parsePath(path: string): string[] {
  return path.split("/").filter(Boolean)
}

export function FileExplorerPanel({ appId, instanceName, containerName }: FileExplorerPanelProps) {
  const queryClient = useQueryClient()
  const [currentPath, setCurrentPath] = React.useState("/")
  const [viewMode, setViewMode] = React.useState<"list" | "grid">(() => {
    const saved = localStorage.getItem(FILE_VIEW_MODE_KEY)
    return (saved === "list" || saved === "grid") ? saved : "list"
  })
  const [selectedFile, setSelectedFile] = React.useState<FileInfo | null>(null)
  const [selectedFiles, setSelectedFiles] = React.useState<Set<string>>(new Set())
  const [editingFile, setEditingFile] = React.useState<{ path: string; content: string; originalContent: string } | null>(null)
  const [isNewFolderDialogOpen, setIsNewFolderDialogOpen] = React.useState(false)
  const [newFolderName, setNewFolderName] = React.useState("")
  const [isRenameDialogOpen, setIsRenameDialogOpen] = React.useState(false)
  const [renameTarget, setRenameTarget] = React.useState<FileInfo | null>(null)
  const [newName, setNewName] = React.useState("")
  const [isMoveDialogOpen, setIsMoveDialogOpen] = React.useState(false)
  const [moveTarget, setMoveTarget] = React.useState<FileInfo | null>(null)
  const [moveDestination, setMoveDestination] = React.useState("")
  const [isCopyDialogOpen, setIsCopyDialogOpen] = React.useState(false)
  const [copyTarget, setCopyTarget] = React.useState<FileInfo | null>(null)
  const [copyDestination, setCopyDestination] = React.useState("")
  const [isCompressDialogOpen, setIsCompressDialogOpen] = React.useState(false)
  const [compressArchiveName, setCompressArchiveName] = React.useState("")
  const [isBatchMoveDialogOpen, setIsBatchMoveDialogOpen] = React.useState(false)
  const [batchMoveDestination, setBatchMoveDestination] = React.useState("")
  const [isBatchCopyDialogOpen, setIsBatchCopyDialogOpen] = React.useState(false)
  const [batchCopyDestination, setBatchCopyDestination] = React.useState("")
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = React.useState(false)
  const [deleteTarget, setDeleteTarget] = React.useState<FileInfo | null>(null)
  const [isBatchDeleteDialogOpen, setIsBatchDeleteDialogOpen] = React.useState(false)
  const [isOpeningFile, setIsOpeningFile] = React.useState(false)
  const fileInputRef = React.useRef<HTMLInputElement>(null)

  React.useEffect(() => {
    localStorage.setItem(FILE_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const filesQueryKey = ["files", appId, instanceName, containerName, currentPath]

  // Fetch home directory on mount
  const { data: homeData } = useQuery({
    queryKey: ["home-dir", appId, instanceName, containerName],
    queryFn: () => fileExplorerApi.getHomeDir(appId, instanceName, containerName),
    retry: 1,
    staleTime: Infinity,
  })

  const { data: filesData, isLoading, error, refetch } = useQuery({
    queryKey: filesQueryKey,
    queryFn: () => fileExplorerApi.listFiles(appId, instanceName, containerName, currentPath),
    retry: 1,
  })

  const pathSegments = parsePath(currentPath)

  // Sort files: directories first, then by name
  const sortedFiles = React.useMemo(() => {
    if (!filesData?.files) return []
    return [...filesData.files].sort((a, b) => {
      if (a.type === "dir" && b.type !== "dir") return -1
      if (a.type !== "dir" && b.type === "dir") return 1
      return a.name.localeCompare(b.name)
    })
  }, [filesData?.files])

  const isMultiSelect = selectedFiles.size > 0

  // Toggle selection of a file
  const toggleFileSelection = React.useCallback((fileName: string) => {
    setSelectedFiles((prev) => {
      const next = new Set(prev)
      if (next.has(fileName)) {
        next.delete(fileName)
      } else {
        next.add(fileName)
      }
      return next
    })
  }, [])

  // Toggle select all
  const toggleSelectAll = React.useCallback(() => {
    if (selectedFiles.size === sortedFiles.length) {
      setSelectedFiles(new Set())
    } else {
      setSelectedFiles(new Set(sortedFiles.map((f) => f.name)))
    }
  }, [selectedFiles.size, sortedFiles])

  // Clear multi-select
  const clearSelection = React.useCallback(() => {
    setSelectedFiles(new Set())
  }, [])

  // Navigate to directory
  const navigateTo = React.useCallback((path: string) => {
    setCurrentPath(path)
    setSelectedFile(null)
    setSelectedFiles(new Set())
  }, [])

  // Navigate to a path segment
  const navigateToSegment = React.useCallback((index: number) => {
    const newPath = buildPath(pathSegments.slice(0, index + 1))
    navigateTo(newPath)
  }, [pathSegments, navigateTo])

  // Open file editor
  const openFileEditor = React.useCallback((file: FileInfo) => {
    const filePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`
    setIsOpeningFile(true)
    fileExplorerApi.readFile(appId, instanceName, containerName, filePath)
      .then((result) => {
        setEditingFile({
          path: filePath,
          content: result.content,
          originalContent: result.content,
        })
      })
      .catch((err) => {
        toast.error("Failed to read file", {
          description: err.response?.data?.error || err.message,
        })
      })
      .finally(() => {
        setIsOpeningFile(false)
      })
  }, [appId, instanceName, containerName, currentPath])

  // Handle file/folder double click
  const handleOpen = React.useCallback((file: FileInfo) => {
    if (file.type === "dir") {
      const newPath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`
      navigateTo(newPath)
    } else if (isTextFile(file.name)) {
      openFileEditor(file)
    }
  }, [currentPath, navigateTo, openFileEditor])

  // Mutations
  const writeFileMutation = useMutation({
    mutationFn: ({ path, content }: { path: string; content: string }) =>
      fileExplorerApi.writeFile(appId, instanceName, containerName, path, content),
    onSuccess: () => {
      toast.success("File saved successfully")
      if (editingFile) {
        setEditingFile({ ...editingFile, originalContent: editingFile.content })
      }
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to save file", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  const mkdirMutation = useMutation({
    mutationFn: (path: string) =>
      fileExplorerApi.mkdir(appId, instanceName, containerName, path),
    onSuccess: () => {
      toast.success("Directory created")
      setIsNewFolderDialogOpen(false)
      setNewFolderName("")
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to create directory", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (path: string) =>
      fileExplorerApi.deleteFile(appId, instanceName, containerName, path),
    onSuccess: () => {
      toast.success("Deleted successfully")
      setSelectedFile(null)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  const moveMutation = useMutation({
    mutationFn: ({ source, destination }: { source: string; destination: string }) =>
      fileExplorerApi.moveFile(appId, instanceName, containerName, source, destination),
    onSuccess: () => {
      toast.success("Moved/renamed successfully")
      setIsRenameDialogOpen(false)
      setIsMoveDialogOpen(false)
      setRenameTarget(null)
      setMoveTarget(null)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to move/rename", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  const copyMutation = useMutation({
    mutationFn: ({ source, destination }: { source: string; destination: string }) =>
      fileExplorerApi.copyFile(appId, instanceName, containerName, source, destination),
    onSuccess: () => {
      toast.success("Copied successfully")
      setIsCopyDialogOpen(false)
      setCopyTarget(null)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to copy", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) =>
      fileExplorerApi.uploadFile(appId, instanceName, containerName, currentPath, file),
    onSuccess: () => {
      toast.success("File uploaded successfully")
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to upload file", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  const compressMutation = useMutation({
    mutationFn: ({ fileNames, destPath }: { fileNames: string[]; destPath: string }) =>
      fileExplorerApi.compressFiles(appId, instanceName, containerName, currentPath, fileNames, destPath),
    onSuccess: () => {
      toast.success("Files compressed successfully")
      setIsCompressDialogOpen(false)
      setSelectedFiles(new Set())
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to compress files", {
        description: err.response?.data?.error || err.message,
      })
    },
  })

  // Handlers
  const handleGoHome = React.useCallback(() => {
    const home = homeData?.path || "/"
    navigateTo(home)
  }, [homeData, navigateTo])

  const handleCompressToContainer = () => {
    if (selectedFiles.size === 0) return
    const defaultName = `archive-${Date.now()}.tar.gz`
    setCompressArchiveName(defaultName)
    setIsCompressDialogOpen(true)
  }

  const handleCompressConfirm = () => {
    if (!compressArchiveName.trim()) return
    const destPath = currentPath === "/" ? `/${compressArchiveName}` : `${currentPath}/${compressArchiveName}`
    compressMutation.mutate({ fileNames: Array.from(selectedFiles), destPath })
  }

  const handleCompressAndDownload = () => {
    if (selectedFiles.size === 0) return
    const archiveName = `archive-${Date.now()}.tar.gz`
    const fileNames = Array.from(selectedFiles)
    toast.promise(
      fileExplorerApi.compressAndDownload(appId, instanceName, containerName, currentPath, fileNames, archiveName),
      {
        loading: `Compressing ${fileNames.length} file(s)...`,
        success: `Downloaded ${archiveName}`,
        error: (err) => `Failed: ${err.message}`,
      },
    )
  }

  const handleBatchMove = () => {
    if (selectedFiles.size === 0) return
    setBatchMoveDestination(currentPath === "/" ? "/" : currentPath)
    setIsBatchMoveDialogOpen(true)
  }

  const handleBatchMoveConfirm = async () => {
    if (!batchMoveDestination.trim() || selectedFiles.size === 0) return
    const fileNames = Array.from(selectedFiles)
    let successCount = 0
    let failCount = 0

    for (const fileName of fileNames) {
      const source = currentPath === "/" ? `/${fileName}` : `${currentPath}/${fileName}`
      const destination = batchMoveDestination === "/" ? `/${fileName}` : `${batchMoveDestination}/${fileName}`
      try {
        await fileExplorerApi.moveFile(appId, instanceName, containerName, source, destination)
        successCount++
      } catch {
        failCount++
      }
    }

    if (successCount > 0) {
      toast.success(`Moved ${successCount} file(s) successfully`)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    }
    if (failCount > 0) {
      toast.error(`Failed to move ${failCount} file(s)`)
    }

    setIsBatchMoveDialogOpen(false)
    setSelectedFiles(new Set())
  }

  const handleBatchDelete = () => {
    if (selectedFiles.size === 0) return
    setIsBatchDeleteDialogOpen(true)
  }

  const handleBatchDeleteConfirm = async () => {
    const fileNames = Array.from(selectedFiles)
    let successCount = 0
    let failCount = 0

    await Promise.all(
      fileNames.map(async (fileName) => {
        const filePath = currentPath === "/" ? `/${fileName}` : `${currentPath}/${fileName}`
        try {
          await fileExplorerApi.deleteFile(appId, instanceName, containerName, filePath)
          successCount++
        } catch {
          failCount++
        }
      })
    )

    if (successCount > 0) {
      toast.success(`Deleted ${successCount} file(s) successfully`)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    }
    if (failCount > 0) {
      toast.error(`Failed to delete ${failCount} file(s)`)
    }
    setSelectedFiles(new Set())
    setIsBatchDeleteDialogOpen(false)
  }

  const handleBatchCopy = () => {
    if (selectedFiles.size === 0) return
    setBatchCopyDestination(currentPath === "/" ? "/" : currentPath)
    setIsBatchCopyDialogOpen(true)
  }

  const handleBatchCopyConfirm = async () => {
    if (!batchCopyDestination.trim() || selectedFiles.size === 0) return
    const fileNames = Array.from(selectedFiles)
    let successCount = 0
    let failCount = 0

    for (const fileName of fileNames) {
      const source = currentPath === "/" ? `/${fileName}` : `${currentPath}/${fileName}`
      const destination = batchCopyDestination === "/" ? `/${fileName}` : `${batchCopyDestination}/${fileName}`
      try {
        await fileExplorerApi.copyFile(appId, instanceName, containerName, source, destination)
        successCount++
      } catch {
        failCount++
      }
    }

    if (successCount > 0) {
      toast.success(`Copied ${successCount} file(s) successfully`)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    }
    if (failCount > 0) {
      toast.error(`Failed to copy ${failCount} file(s)`)
    }

    setIsBatchCopyDialogOpen(false)
    setSelectedFiles(new Set())
  }

  const handleCreateFolder = () => {
    if (!newFolderName.trim()) return
    const path = currentPath === "/" ? `/${newFolderName}` : `${currentPath}/${newFolderName}`
    mkdirMutation.mutate(path)
  }

  const handleRename = () => {
    if (!renameTarget || !newName.trim()) return
    const source = currentPath === "/" ? `/${renameTarget.name}` : `${currentPath}/${renameTarget.name}`
    const destination = currentPath === "/" ? `/${newName}` : `${currentPath}/${newName}`
    moveMutation.mutate({ source, destination })
  }

  const handleMove = () => {
    if (!moveTarget || !moveDestination.trim()) return
    const source = currentPath === "/" ? `/${moveTarget.name}` : `${currentPath}/${moveTarget.name}`
    moveMutation.mutate({ source, destination: moveDestination })
  }

  const handleCopy = () => {
    if (!copyTarget || !copyDestination.trim()) return
    const source = currentPath === "/" ? `/${copyTarget.name}` : `${currentPath}/${copyTarget.name}`
    copyMutation.mutate({ source, destination: copyDestination })
  }

  const handleDelete = (file: FileInfo) => {
    setDeleteTarget(file)
    setIsDeleteDialogOpen(true)
  }

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return
    const filePath = currentPath === "/" ? `/${deleteTarget.name}` : `${currentPath}/${deleteTarget.name}`
    deleteMutation.mutate(filePath)
    setIsDeleteDialogOpen(false)
    setDeleteTarget(null)
  }

  const handleDownload = (file: FileInfo) => {
    const filePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`
    if (file.type === "dir") {
      toast.promise(
        fileExplorerApi.downloadDir(appId, instanceName, containerName, filePath),
        {
          loading: `Downloading ${file.name}.tar...`,
          success: `Downloaded ${file.name}.tar`,
          error: (err) => `Failed to download: ${err.message}`,
        }
      )
    } else {
      toast.promise(
        fileExplorerApi.downloadFile(appId, instanceName, containerName, filePath),
        {
          loading: `Downloading ${file.name}...`,
          success: `Downloaded ${file.name}`,
          error: (err) => `Failed to download: ${err.message}`,
        }
      )
    }
  }

  const handleCopyPath = (file: FileInfo) => {
    const filePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`
    copyToClipboard(filePath)
  }

  const handleUpload = () => {
    fileInputRef.current?.click()
  }

  const handleFileSelected = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      uploadMutation.mutate(file)
    }
    // Reset the input
    if (fileInputRef.current) {
      fileInputRef.current.value = ""
    }
  }

  const openRenameDialog = (file: FileInfo) => {
    setRenameTarget(file)
    setNewName(file.name)
    setIsRenameDialogOpen(true)
  }

  const openMoveDialog = (file: FileInfo) => {
    setMoveTarget(file)
    const filePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`
    setMoveDestination(filePath)
    setIsMoveDialogOpen(true)
  }

  const openCopyDialog = (file: FileInfo) => {
    setCopyTarget(file)
    const filePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`
    setCopyDestination(filePath + ".copy")
    setIsCopyDialogOpen(true)
  }

  // File editor view
  if (editingFile) {
    return (
      <FileEditorView
        editingFile={editingFile}
        setEditingFile={setEditingFile}
        writeFileMutation={writeFileMutation}
      />
    )
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      {/* Toolbar */}
      <div className="flex h-8 min-h-8 items-center justify-between border-b bg-muted/20 px-3">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          {/* Navigation buttons */}
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={<Button variant="ghost" size="icon-sm" onClick={handleGoHome} />}
            >
              <Home className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>Go to home directory</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => {
                    const parent = pathSegments.slice(0, -1)
                    navigateTo(buildPath(parent))
                  }}
                  disabled={currentPath === "/"}
                />
              }
            >
              <ArrowLeft className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>Go up</TooltipContent>
          </Tooltip>

          {/* Breadcrumb path */}
          <div className="flex items-center gap-0.5 text-xs min-w-0 overflow-x-auto no-scrollbar group/breadcrumb">
            <Tooltip>
              <TooltipTrigger>
                <button
                  onClick={() => navigateTo("/")}
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
              <React.Fragment key={index}>
                <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />
                <Tooltip>
                  <TooltipTrigger>
                    <button
                      onClick={() => navigateToSegment(index)}
                      className={cn(
                        "px-1.5 py-0.5 rounded hover:bg-muted transition-colors font-mono truncate max-w-32 shrink-0",
                        index === pathSegments.length - 1 ? "text-foreground font-medium" : "text-muted-foreground"
                      )}
                    >
                      {segment}
                    </button>
                  </TooltipTrigger>
                  <TooltipContent className="text-[10px]">
                    {buildPath(pathSegments.slice(0, index + 1))}
                  </TooltipContent>
                </Tooltip>
              </React.Fragment>
            ))}
            {/* Copy current path button - visible on breadcrumb hover */}
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="opacity-0 group-hover/breadcrumb:opacity-100 transition-opacity shrink-0"
                    onClick={() => copyToClipboard(currentPath)}
                  />
                }
              >
                <Clipboard className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Copy current path</TooltipContent>
            </Tooltip>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          {/* Multi-select actions */}
          {isMultiSelect && (
            <>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<Button variant="ghost" size="icon-sm" onClick={handleBatchMove} />}
                >
                  <FileOutput className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Move selected</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<Button variant="ghost" size="icon-sm" onClick={handleBatchCopy} />}
                >
                  <Copy className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Copy selected</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<Button variant="ghost" size="icon-sm" onClick={handleCompressToContainer} />}
                >
                  <Archive className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Compress</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<Button variant="ghost" size="icon-sm" onClick={handleCompressAndDownload} />}
                >
                  <FileArchive className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Compress & Download</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={handleBatchDelete}
                    />
                  }
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Delete selected</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={<Button variant="ghost" size="icon-sm" onClick={clearSelection} />}
                >
                  <X className="h-3.5 w-3.5" />
                </TooltipTrigger>
                <TooltipContent>Clear selection</TooltipContent>
              </Tooltip>
              <div className="h-4 w-px bg-border mx-0.5" />
            </>
          )}

          <Button variant="ghost" size="icon-sm" onClick={() => refetch()} disabled={isLoading}>
            <RefreshCw className={cn("h-3.5 w-3.5", isLoading && "animate-spin")} />
          </Button>

          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={<Button variant="ghost" size="icon-sm" onClick={handleUpload} />}
            >
              <Upload className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>Upload file</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => {
                    setNewFolderName("")
                    setIsNewFolderDialogOpen(true)
                  }}
                />
              }
            >
              <FolderPlus className="h-3.5 w-3.5" />
            </TooltipTrigger>
            <TooltipContent>New folder</TooltipContent>
          </Tooltip>

          <Separator orientation="vertical" className="mt-1 mb-1 mx-2" />

          <div className="flex items-center bg-muted rounded-md">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className={cn("rounded-r-none", viewMode === "list" && "bg-background shadow-sm")}
                    onClick={() => setViewMode("list")}
                  />
                }
              >
                <List className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>List view</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className={cn("rounded-l-none", viewMode === "grid" && "bg-background shadow-sm")}
                    onClick={() => setViewMode("grid")}
                  />
                }
              >
                <LayoutGrid className="h-3.5 w-3.5" />
              </TooltipTrigger>
              <TooltipContent>Grid view</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </div>


      {/* Hidden file input for upload */}
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        onChange={handleFileSelected}
      />

      {/* File list content */}
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
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
            <FolderOpen className="h-8 w-8 opacity-30" />
            <p className="text-xs">Failed to load files</p>
            <p className="text-[10px]">{(error as any)?.response?.data?.error || (error as any)?.message}</p>
            <Button variant="outline" size="sm" onClick={() => refetch()} className="mt-2 h-7 text-xs">
              Retry
            </Button>
          </div>
        ) : sortedFiles.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
            <FolderOpen className="h-8 w-8 opacity-30" />
            <p className="text-xs">Empty directory</p>
          </div>
        ) : viewMode === "list" ? (
          <ListFileView
            files={sortedFiles}
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
          />
        ) : (
          <GridFileView
            files={sortedFiles}
            selectedFile={selectedFile}
            selectedFiles={selectedFiles}
            onSelect={setSelectedFile}
            onToggleSelect={toggleFileSelection}
            onOpen={handleOpen}
            onRename={openRenameDialog}
            onMove={openMoveDialog}
            onCopy={openCopyDialog}
            onDelete={handleDelete}
            onDownload={handleDownload}
            onCopyPath={handleCopyPath}
          />
        )}
      </div>

      {/* Compress Dialog */}
      <Dialog open={isCompressDialogOpen} onOpenChange={setIsCompressDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Compress Files</DialogTitle>
            <DialogDescription>Compress {selectedFiles.size} selected file(s) into an archive in the current directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Archive name (e.g., archive.tar.gz)"
              value={compressArchiveName}
              onChange={(e) => setCompressArchiveName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCompressConfirm()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsCompressDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleCompressConfirm} disabled={!compressArchiveName.trim() || compressMutation.isPending}>
                {compressMutation.isPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Compress
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Batch Move Dialog */}
      <Dialog open={isBatchMoveDialogOpen} onOpenChange={setIsBatchMoveDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Move Files</DialogTitle>
            <DialogDescription>Move {selectedFiles.size} selected file(s) to a new directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Destination path (e.g., /tmp)"
              value={batchMoveDestination}
              onChange={(e) => setBatchMoveDestination(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleBatchMoveConfirm()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsBatchMoveDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleBatchMoveConfirm} disabled={!batchMoveDestination.trim()}>
                Move
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Batch Copy Dialog */}
      <Dialog open={isBatchCopyDialogOpen} onOpenChange={setIsBatchCopyDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Copy Files</DialogTitle>
            <DialogDescription>Copy {selectedFiles.size} selected file(s) to a new directory.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-2">
            <Input
              placeholder="Destination path (e.g., /tmp)"
              value={batchCopyDestination}
              onChange={(e) => setBatchCopyDestination(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleBatchCopyConfirm()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsBatchCopyDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleBatchCopyConfirm} disabled={!batchCopyDestination.trim()}>
                Copy
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
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
              onClick={handleDeleteConfirm}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Batch Delete Confirmation Dialog */}
      <AlertDialog open={isBatchDeleteDialogOpen} onOpenChange={setIsBatchDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {selectedFiles.size} File(s)?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedFiles.size} selected file(s)? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleBatchDeleteConfirm}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Status bar */}
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

      {/* New Folder Dialog */}
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
              onChange={(e) => setNewFolderName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCreateFolder()}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsNewFolderDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleCreateFolder} disabled={!newFolderName.trim() || mkdirMutation.isPending}>
                {mkdirMutation.isPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Create
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Rename Dialog */}
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
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleRename()}
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsRenameDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleRename} disabled={!newName.trim() || moveMutation.isPending}>
                {moveMutation.isPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Rename
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Move Dialog */}
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
              onChange={(e) => setMoveDestination(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleMove()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsMoveDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleMove} disabled={!moveDestination.trim() || moveMutation.isPending}>
                {moveMutation.isPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Move
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Copy Dialog */}
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
              onChange={(e) => setCopyDestination(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCopy()}
              autoFocus
              className="font-mono text-xs"
            />
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setIsCopyDialogOpen(false)}>Cancel</Button>
              <Button size="sm" onClick={handleCopy} disabled={!copyDestination.trim() || copyMutation.isPending}>
                {copyMutation.isPending && <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />}
                Copy
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div >
  )
}

// Extracted file editor view component with proper Ctrl+S support
function FileEditorView({
  editingFile,
  setEditingFile,
  writeFileMutation,
}: {
  editingFile: { path: string; content: string; originalContent: string }
  setEditingFile: (file: { path: string; content: string; originalContent: string } | null) => void
  writeFileMutation: ReturnType<typeof useMutation<unknown, any, { path: string; content: string }>>
}) {
  const hasChanges = editingFile.content !== editingFile.originalContent
  const textareaRef = React.useRef<HTMLTextAreaElement>(null)
  const [isFileLoading] = React.useState(false)
  const [isUnsavedDialogOpen, setIsUnsavedDialogOpen] = React.useState(false)

  // Handle Ctrl+S / Cmd+S keyboard shortcut
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault()
        if (hasChanges && !writeFileMutation.isPending) {
          writeFileMutation.mutate({ path: editingFile.path, content: editingFile.content })
        }
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [hasChanges, writeFileMutation, editingFile.path, editingFile.content])

  // Confirm before closing with unsaved changes
  const handleClose = React.useCallback(() => {
    if (hasChanges) {
      setIsUnsavedDialogOpen(true)
    } else {
      setEditingFile(null)
    }
  }, [hasChanges, setEditingFile])

  const handleDiscardChanges = React.useCallback(() => {
    setIsUnsavedDialogOpen(false)
    setEditingFile(null)
  }, [setEditingFile])

  return (
    <div className="flex flex-col h-full">
      {/* Editor toolbar */}
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
            disabled={!hasChanges || writeFileMutation.isPending}
            onClick={() => writeFileMutation.mutate({ path: editingFile.path, content: editingFile.content })}
          >
            {writeFileMutation.isPending ? (
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

      {/* Editor content */}
      <div className="flex-1 overflow-hidden relative">
        {isFileLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-background/80 z-10">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        )}
        <Textarea
          ref={textareaRef}
          value={editingFile.content}
          onChange={(e) => setEditingFile({ ...editingFile, content: e.target.value })}
          className="h-full w-full resize-none rounded-none border-0 font-mono text-xs leading-relaxed focus-visible:ring-0 focus-visible:ring-offset-0 p-3"
          spellCheck={false}
          autoFocus
        />
      </div>

      {/* Editor status bar */}
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

      {/* Unsaved Changes Confirmation Dialog */}
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
              onClick={handleDiscardChanges}
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

// Context menu for file operations
function FileContextMenu({
  file,
  children,
  onRename,
  onMove,
  onCopy,
  onDelete,
  onDownload,
  onCopyPath,
}: {
  file: FileInfo
  children: React.ReactElement
  onRename: (file: FileInfo) => void
  onMove: (file: FileInfo) => void
  onCopy: (file: FileInfo) => void
  onDelete: (file: FileInfo) => void
  onDownload: (file: FileInfo) => void
  onCopyPath: (file: FileInfo) => void
}) {
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
        <DropdownMenuItem onClick={() => onDelete(file)} className="text-destructive focus:text-destructive">
          <Trash2 className="h-3.5 w-3.5 mr-2" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

// Shared props type for file view components
interface FileViewProps {
  files: FileInfo[]
  selectedFile: FileInfo | null
  selectedFiles: Set<string>
  onSelect: (file: FileInfo) => void
  onToggleSelect: (fileName: string) => void
  onOpen: (file: FileInfo) => void
  onRename: (file: FileInfo) => void
  onMove: (file: FileInfo) => void
  onCopy: (file: FileInfo) => void
  onDelete: (file: FileInfo) => void
  onDownload: (file: FileInfo) => void
  onCopyPath: (file: FileInfo) => void
}

interface ListFileViewProps extends FileViewProps {
  onToggleSelectAll: () => void
}

// List view component
function ListFileView({
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
}: ListFileViewProps) {
  return (
    <div className="w-full">
      {/* Header */}
      <div className="grid grid-cols-[24px_1fr_70px_100px_50px_56px] gap-2 px-3 py-1.5 text-[10px] font-medium text-muted-foreground uppercase tracking-wider border-b bg-muted/10">
        <div className="flex items-center justify-center">
          <Checkbox
            checked={files.length > 0 && selectedFiles.size === files.length}
            indeterminate={selectedFiles.size > 0 && selectedFiles.size < files.length}
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
      {/* Rows */}
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
          <div className="flex items-center justify-center" onClick={(e) => e.stopPropagation()}>
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
              onClick={(e) => { e.stopPropagation(); onDownload(file) }}
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
                onClick={(e) => e.stopPropagation()}
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

// Grid view component
function GridFileView({
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
}: FileViewProps) {
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
          {/* Checkbox */}
          <div
            className="absolute top-1 left-1 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={(e) => e.stopPropagation()}
          >
            <Checkbox
              checked={selectedFiles.has(file.name)}
              onCheckedChange={() => onToggleSelect(file.name)}
              className="h-3.5 w-3.5"
            />
          </div>
          {/* Action buttons */}
          <div className="absolute top-1 right-1 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button
              variant="ghost"
              size="icon-xs"
              className="h-5 w-5"
              onClick={(e) => { e.stopPropagation(); onCopyPath(file) }}
              title="Copy path"
            >
              <Clipboard className="h-2.5 w-2.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              className="h-5 w-5"
              onClick={(e) => { e.stopPropagation(); onDownload(file) }}
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
                onClick={(e) => e.stopPropagation()}
              >
                <MoreHorizontal className="h-3 w-3" />
              </Button>
            </FileContextMenu>
          </div>
          <div className="p-2">
            {file.type === "dir" ? (
              <Folder className="h-8 w-8 text-blue-400" />
            ) : (
              React.cloneElement(getFileIcon(file.name, file.type) as React.ReactElement<{ className?: string }>, {
                className: "h-8 w-8",
              })
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
