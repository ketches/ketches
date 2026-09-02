import { fileExplorerApi, type FileInfo } from "@/api/file-explorer"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isCancel, type AxiosError } from "axios"
import * as React from "react"
import { toast } from "sonner"

import { type EditingFileState, type FileExplorerPanelProps, type FileExplorerViewMode } from "../types"
import { FILE_VIEW_MODE_KEY, buildPath, copyToClipboard, getErrorMessage, isTextFile, parsePath } from "../utils"

function buildFilePath(currentPath: string, fileName: string) {
  return currentPath === "/" ? `/${fileName}` : `${currentPath}/${fileName}`
}

export function useFileExplorer({
  appId,
  instanceName,
  containerName,
}: FileExplorerPanelProps) {
  const queryClient = useQueryClient()
  const [currentPath, setCurrentPath] = React.useState("/")
  const [viewMode, setViewMode] = React.useState<FileExplorerViewMode>(() => {
    const saved = localStorage.getItem(FILE_VIEW_MODE_KEY)
    return saved === "list" || saved === "grid" ? saved : "list"
  })
  const [selectedFile, setSelectedFile] = React.useState<FileInfo | null>(null)
  const [selectedFiles, setSelectedFiles] = React.useState<Set<string>>(new Set())
  const [editingFile, setEditingFile] = React.useState<EditingFileState | null>(null)
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
  const openFileRequestRef = React.useRef<AbortController | null>(null)

  React.useEffect(() => () => {
    openFileRequestRef.current?.abort()
  }, [appId, containerName, instanceName])

  React.useEffect(() => {
    localStorage.setItem(FILE_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const filesQueryKey = React.useMemo(
    () => ["files", appId, instanceName, containerName, currentPath],
    [appId, containerName, currentPath, instanceName]
  )

  const { data: homeData } = useQuery({
    queryKey: ["home-dir", appId, instanceName, containerName],
    queryFn: ({ signal }) => fileExplorerApi.getHomeDir(appId, instanceName, containerName, signal),
    retry: 1,
    staleTime: Infinity,
  })

  const { data: filesData, isLoading, error, refetch } = useQuery({
    queryKey: filesQueryKey,
    queryFn: ({ signal }) => fileExplorerApi.listFiles(appId, instanceName, containerName, currentPath, signal),
    retry: 1,
  })

  const pathSegments = parsePath(currentPath)
  const errorMessage = getErrorMessage(error, "Failed to load files")

  const sortedFiles = React.useMemo(() => {
    if (!filesData?.files) {
      return []
    }

    return [...filesData.files].sort((a, b) => {
      if (a.type === "dir" && b.type !== "dir") {
        return -1
      }
      if (a.type !== "dir" && b.type === "dir") {
        return 1
      }
      return a.name.localeCompare(b.name)
    })
  }, [filesData])

  const isMultiSelect = selectedFiles.size > 0

  const toggleFileSelection = React.useCallback((fileName: string) => {
    setSelectedFiles((previous) => {
      const next = new Set(previous)
      if (next.has(fileName)) {
        next.delete(fileName)
      } else {
        next.add(fileName)
      }
      return next
    })
  }, [])

  const toggleSelectAll = React.useCallback(() => {
    if (selectedFiles.size === sortedFiles.length) {
      setSelectedFiles(new Set())
      return
    }

    setSelectedFiles(new Set(sortedFiles.map((file) => file.name)))
  }, [selectedFiles.size, sortedFiles])

  const clearSelection = React.useCallback(() => {
    setSelectedFiles(new Set())
  }, [])

  const navigateTo = React.useCallback((path: string) => {
    openFileRequestRef.current?.abort()
    setCurrentPath(path)
    setSelectedFile(null)
    setSelectedFiles(new Set())
  }, [])

  const navigateToSegment = React.useCallback((index: number) => {
    navigateTo(buildPath(pathSegments.slice(0, index + 1)))
  }, [navigateTo, pathSegments])

  const openFileEditor = React.useCallback((file: FileInfo) => {
    const filePath = buildFilePath(currentPath, file.name)
    openFileRequestRef.current?.abort()
    const controller = new AbortController()
    openFileRequestRef.current = controller
    setIsOpeningFile(true)
    fileExplorerApi.readFile(appId, instanceName, containerName, filePath, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return
        setEditingFile({
          path: filePath,
          content: result.content,
          originalContent: result.content,
        })
      })
      .catch((error: unknown) => {
        if (isCancel(error) || controller.signal.aborted) return
        toast.error("Failed to read file", {
          description: getErrorMessage(error, "Failed to read file"),
        })
      })
      .finally(() => {
        if (openFileRequestRef.current !== controller) return
        openFileRequestRef.current = null
        setIsOpeningFile(false)
      })
  }, [appId, instanceName, containerName, currentPath])

  const handleOpen = React.useCallback((file: FileInfo) => {
    if (file.type === "dir") {
      navigateTo(buildFilePath(currentPath, file.name))
      return
    }

    if (isTextFile(file.name)) {
      openFileEditor(file)
    }
  }, [currentPath, navigateTo, openFileEditor])

  const writeFileMutation = useMutation({
    mutationFn: ({ path, content }: { path: string; content: string }) =>
      fileExplorerApi.writeFile(appId, instanceName, containerName, path, content),
    onSuccess: () => {
      toast.success("File saved successfully")
      setEditingFile((previous) => previous ? { ...previous, originalContent: previous.content } : previous)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to save file", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const mkdirMutation = useMutation({
    mutationFn: (path: string) => fileExplorerApi.mkdir(appId, instanceName, containerName, path),
    onSuccess: () => {
      toast.success("Directory created")
      setIsNewFolderDialogOpen(false)
      setNewFolderName("")
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to create directory", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (path: string) => fileExplorerApi.deleteFile(appId, instanceName, containerName, path),
    onSuccess: () => {
      toast.success("Deleted successfully")
      setSelectedFile(null)
      queryClient.invalidateQueries({ queryKey: filesQueryKey })
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete", {
        description: error.response?.data?.error || error.message,
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
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to move/rename", {
        description: error.response?.data?.error || error.message,
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
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to copy", {
        description: error.response?.data?.error || error.message,
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
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to upload file", {
        description: error.response?.data?.error || error.message,
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
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to compress files", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const handleGoHome = React.useCallback(() => {
    navigateTo(homeData?.path || "/")
  }, [homeData?.path, navigateTo])

  const handleNavigateUp = React.useCallback(() => {
    navigateTo(buildPath(pathSegments.slice(0, -1)))
  }, [navigateTo, pathSegments])

  const handleCompressToContainer = React.useCallback(() => {
    if (selectedFiles.size === 0) {
      return
    }

    setCompressArchiveName(`archive-${Date.now()}.tar.gz`)
    setIsCompressDialogOpen(true)
  }, [selectedFiles.size])

  const handleCompressConfirm = React.useCallback(() => {
    if (!compressArchiveName.trim()) {
      return
    }

    const destPath = currentPath === "/" ? `/${compressArchiveName}` : `${currentPath}/${compressArchiveName}`
    compressMutation.mutate({ fileNames: Array.from(selectedFiles), destPath })
  }, [compressArchiveName, compressMutation, currentPath, selectedFiles])

  const handleCompressAndDownload = React.useCallback(() => {
    if (selectedFiles.size === 0) {
      return
    }

    const archiveName = `archive-${Date.now()}.tar.gz`
    const fileNames = Array.from(selectedFiles)
    toast.promise(
      fileExplorerApi.compressAndDownload(appId, instanceName, containerName, currentPath, fileNames, archiveName),
      {
        loading: `Compressing ${fileNames.length} file(s)...`,
        success: `Downloaded ${archiveName}`,
        error: (error) => `Failed: ${getErrorMessage(error, "Compress & download failed")}`,
      }
    )
  }, [appId, containerName, currentPath, instanceName, selectedFiles])

  const handleBatchMove = React.useCallback(() => {
    if (selectedFiles.size === 0) {
      return
    }

    setBatchMoveDestination(currentPath === "/" ? "/" : currentPath)
    setIsBatchMoveDialogOpen(true)
  }, [currentPath, selectedFiles.size])

  const handleBatchMoveConfirm = React.useCallback(async () => {
    if (!batchMoveDestination.trim() || selectedFiles.size === 0) {
      return
    }

    const fileNames = Array.from(selectedFiles)
    let successCount = 0
    let failCount = 0

    for (const fileName of fileNames) {
      const source = buildFilePath(currentPath, fileName)
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
  }, [appId, batchMoveDestination, containerName, currentPath, filesQueryKey, instanceName, queryClient, selectedFiles])

  const handleBatchDelete = React.useCallback(() => {
    if (selectedFiles.size === 0) {
      return
    }

    setIsBatchDeleteDialogOpen(true)
  }, [selectedFiles.size])

  const handleBatchDeleteConfirm = React.useCallback(async () => {
    const fileNames = Array.from(selectedFiles)
    let successCount = 0
    let failCount = 0

    await Promise.all(
      fileNames.map(async (fileName) => {
        const filePath = buildFilePath(currentPath, fileName)
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
  }, [appId, containerName, currentPath, filesQueryKey, instanceName, queryClient, selectedFiles])

  const handleBatchCopy = React.useCallback(() => {
    if (selectedFiles.size === 0) {
      return
    }

    setBatchCopyDestination(currentPath === "/" ? "/" : currentPath)
    setIsBatchCopyDialogOpen(true)
  }, [currentPath, selectedFiles.size])

  const handleBatchCopyConfirm = React.useCallback(async () => {
    if (!batchCopyDestination.trim() || selectedFiles.size === 0) {
      return
    }

    const fileNames = Array.from(selectedFiles)
    let successCount = 0
    let failCount = 0

    for (const fileName of fileNames) {
      const source = buildFilePath(currentPath, fileName)
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
  }, [appId, batchCopyDestination, containerName, currentPath, filesQueryKey, instanceName, queryClient, selectedFiles])

  const handleCreateFolder = React.useCallback(() => {
    if (!newFolderName.trim()) {
      return
    }

    mkdirMutation.mutate(buildFilePath(currentPath, newFolderName))
  }, [currentPath, mkdirMutation, newFolderName])

  const handleRename = React.useCallback(() => {
    if (!renameTarget || !newName.trim()) {
      return
    }

    moveMutation.mutate({
      source: buildFilePath(currentPath, renameTarget.name),
      destination: buildFilePath(currentPath, newName),
    })
  }, [currentPath, moveMutation, newName, renameTarget])

  const handleMove = React.useCallback(() => {
    if (!moveTarget || !moveDestination.trim()) {
      return
    }

    moveMutation.mutate({
      source: buildFilePath(currentPath, moveTarget.name),
      destination: moveDestination,
    })
  }, [currentPath, moveDestination, moveMutation, moveTarget])

  const handleCopy = React.useCallback(() => {
    if (!copyTarget || !copyDestination.trim()) {
      return
    }

    copyMutation.mutate({
      source: buildFilePath(currentPath, copyTarget.name),
      destination: copyDestination,
    })
  }, [copyDestination, copyMutation, copyTarget, currentPath])

  const handleDelete = React.useCallback((file: FileInfo) => {
    setDeleteTarget(file)
    setIsDeleteDialogOpen(true)
  }, [])

  const handleDeleteConfirm = React.useCallback(() => {
    if (!deleteTarget) {
      return
    }

    deleteMutation.mutate(buildFilePath(currentPath, deleteTarget.name))
    setIsDeleteDialogOpen(false)
    setDeleteTarget(null)
  }, [currentPath, deleteMutation, deleteTarget])

  const handleDownload = React.useCallback((file: FileInfo) => {
    const filePath = buildFilePath(currentPath, file.name)
    if (file.type === "dir") {
      toast.promise(
        fileExplorerApi.downloadDir(appId, instanceName, containerName, filePath),
        {
          loading: `Downloading ${file.name}.tar...`,
          success: `Downloaded ${file.name}.tar`,
          error: (error) => `Failed to download: ${getErrorMessage(error, "Download failed")}`,
        }
      )
      return
    }

    toast.promise(
      fileExplorerApi.downloadFile(appId, instanceName, containerName, filePath),
      {
        loading: `Downloading ${file.name}...`,
        success: `Downloaded ${file.name}`,
        error: (error) => `Failed to download: ${getErrorMessage(error, "Download failed")}`,
      }
    )
  }, [appId, containerName, currentPath, instanceName])

  const handleCopyPath = React.useCallback((file: FileInfo) => {
    copyToClipboard(buildFilePath(currentPath, file.name))
  }, [currentPath])

  const handleUpload = React.useCallback(() => {
    fileInputRef.current?.click()
  }, [])

  const handleFileSelected = React.useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file) {
      uploadMutation.mutate(file)
    }

    if (fileInputRef.current) {
      fileInputRef.current.value = ""
    }
  }, [uploadMutation])

  const openCreateFolderDialog = React.useCallback(() => {
    setNewFolderName("")
    setIsNewFolderDialogOpen(true)
  }, [])

  const openRenameDialog = React.useCallback((file: FileInfo) => {
    setRenameTarget(file)
    setNewName(file.name)
    setIsRenameDialogOpen(true)
  }, [])

  const openMoveDialog = React.useCallback((file: FileInfo) => {
    setMoveTarget(file)
    setMoveDestination(buildFilePath(currentPath, file.name))
    setIsMoveDialogOpen(true)
  }, [currentPath])

  const openCopyDialog = React.useCallback((file: FileInfo) => {
    setCopyTarget(file)
    setCopyDestination(`${buildFilePath(currentPath, file.name)}.copy`)
    setIsCopyDialogOpen(true)
  }, [currentPath])

  const closeEditingFile = React.useCallback(() => {
    setEditingFile(null)
  }, [])

  const saveEditingFile = React.useCallback(() => {
    if (!editingFile) {
      return
    }

    writeFileMutation.mutate({
      path: editingFile.path,
      content: editingFile.content,
    })
  }, [editingFile, writeFileMutation])

  return {
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
    isSavingFile: writeFileMutation.isPending,
    isOpeningFile,
    sortedFiles,
    isLoading,
    hasError: Boolean(error),
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
    copyCurrentPath: () => copyToClipboard(currentPath),
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
    mkdirPending: mkdirMutation.isPending,
    movePending: moveMutation.isPending,
    copyPending: copyMutation.isPending,
    deletePending: deleteMutation.isPending,
    compressPending: compressMutation.isPending,
  }
}
