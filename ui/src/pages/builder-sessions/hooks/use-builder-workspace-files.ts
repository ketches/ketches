import { builderSessionsApi, type BuilderWorkspaceFile } from "@/api/builder-sessions"
import { useQuery } from "@tanstack/react-query"
import * as React from "react"
import { toast } from "sonner"

interface UseBuilderWorkspaceFilesOptions {
  projectId?: string
  sessionId?: string
  hasFiles: boolean
}

export function useBuilderWorkspaceFiles({
  projectId,
  sessionId,
  hasFiles,
}: UseBuilderWorkspaceFilesOptions) {
  const [filesExpanded, setFilesExpanded] = React.useState(false)
  const [currentPath, setCurrentPath] = React.useState("/")
  const [selectedFile, setSelectedFile] = React.useState<BuilderWorkspaceFile | null>(null)
  const [fileContent, setFileContent] = React.useState<string | null>(null)

  const { data: filesData } = useQuery({
    queryKey: ["builder-files", projectId, sessionId, currentPath],
    queryFn: () => builderSessionsApi.listFiles(projectId!, sessionId!, currentPath),
    enabled: !!projectId && !!sessionId && hasFiles,
  })

  React.useEffect(() => {
    setFilesExpanded(false)
    setCurrentPath("/")
    setSelectedFile(null)
    setFileContent(null)
  }, [sessionId])

  const handleSelectFile = React.useCallback(
    async (file: BuilderWorkspaceFile) => {
      if (!projectId || !sessionId) {
        return
      }

      if (file.type === "dir") {
        setSelectedFile(null)
        setFileContent(null)
        const nextPath = currentPath === "/" ? `/${file.name}/` : `${currentPath}${file.name}/`
        setCurrentPath(nextPath)
        return
      }

      setSelectedFile(file)
      setFileContent(null)

      try {
        const response = await builderSessionsApi.readFile(projectId, sessionId, `${currentPath}${file.name}`)
        setFileContent(response.content)
      } catch {
        setFileContent("Failed to load file preview.")
      }
    },
    [currentPath, projectId, sessionId]
  )

  const handleDownloadFiles = React.useCallback(async () => {
    if (!projectId || !sessionId) {
      return
    }

    try {
      await builderSessionsApi.downloadTarBlob(projectId, sessionId)
    } catch {
      toast.error("Failed to download files")
    }
  }, [projectId, sessionId])

  const handleNavigateParent = React.useCallback(() => {
    const parentSegments = currentPath.split("/").filter(Boolean)
    parentSegments.pop()
    const nextPath = parentSegments.length > 0 ? `/${parentSegments.join("/")}/` : "/"
    setCurrentPath(nextPath)
    setSelectedFile(null)
    setFileContent(null)
  }, [currentPath])

  return {
    filesExpanded,
    setFilesExpanded,
    currentPath,
    setCurrentPath,
    selectedFile,
    fileContent,
    filesData,
    handleSelectFile,
    handleDownloadFiles,
    handleNavigateParent,
  }
}
