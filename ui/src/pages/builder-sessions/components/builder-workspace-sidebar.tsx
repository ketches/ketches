import type { BuilderSession, BuilderWorkspaceFile } from "@/api/builder-sessions"
import { Button } from "@/components/ui/button"
import { BuilderSessionHistoryRail } from "@/pages/builder-sessions/builder-session-history-rail"
import { ChevronLeft, ChevronRight, FileCode2, FileText, Folder, Loader2 } from "lucide-react"

interface BuilderWorkspaceSidebarProps {
  projectId?: string
  sessionId?: string
  sessions: BuilderSession[]
  onNewConversation: () => void
  onSelectSession: (sessionId: string) => void
  hasFiles: boolean
  filesExpanded: boolean
  setFilesExpanded: React.Dispatch<React.SetStateAction<boolean>>
  currentPath: string
  filesData?: { files: BuilderWorkspaceFile[] }
  selectedFile: BuilderWorkspaceFile | null
  fileContent: string | null
  onSelectFile: (file: BuilderWorkspaceFile) => Promise<void>
  onNavigateParent: () => void
  onDownloadFiles: () => Promise<void>
}

export function BuilderWorkspaceSidebar({
  projectId,
  sessionId,
  sessions,
  onNewConversation,
  onSelectSession,
  hasFiles,
  filesExpanded,
  setFilesExpanded,
  currentPath,
  filesData,
  selectedFile,
  fileContent,
  onSelectFile,
  onNavigateParent,
  onDownloadFiles,
}: BuilderWorkspaceSidebarProps) {
  return (
    <>
      <BuilderSessionHistoryRail
        sessions={sessions}
        selectedSessionId={sessionId}
        onNewConversation={onNewConversation}
        onSelectSession={(targetSessionId) => {
          if (!projectId) {
            return
          }
          onSelectSession(targetSessionId)
        }}
      />

      {hasFiles ? (
        <aside
          data-testid="builder-files-rail"
          className={`flex min-h-0 shrink-0 bg-muted/10 ${filesExpanded ? "w-80" : "w-14"}`}
        >
          <div className="flex min-h-0 w-full flex-col">
            <div className={`p-2 ${filesExpanded ? "flex items-center justify-between" : "flex justify-center"}`}>
              <Button
                data-testid="builder-files-rail-toggle"
                variant="ghost"
                size="icon"
                onClick={() => setFilesExpanded((current) => !current)}
              >
                {filesExpanded ? <ChevronRight className="h-4 w-4" /> : <FileCode2 className="h-4 w-4" />}
                <span className="sr-only">Toggle files rail</span>
              </Button>
              {filesExpanded ? (
                <Button variant="ghost" size="sm" onClick={() => void onDownloadFiles()}>
                  Download
                </Button>
              ) : null}
            </div>

            {filesExpanded ? (
              <div className="flex min-h-0 flex-1 flex-col">
                <div data-testid="builder-files-list" className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
                  {currentPath !== "/" ? (
                    <button
                      type="button"
                      className="mb-1 flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                      onClick={onNavigateParent}
                    >
                      <ChevronLeft className="h-4 w-4" />
                      <span>Parent directory</span>
                    </button>
                  ) : null}

                  {filesData?.files.map((file) => (
                    <button
                      key={`${currentPath}${file.name}`}
                      type="button"
                      className="flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
                      onClick={() => {
                        void onSelectFile(file)
                      }}
                    >
                      {file.type === "dir" ? (
                        <Folder className="h-4 w-4 shrink-0" />
                      ) : (
                        <FileText className="h-4 w-4 shrink-0" />
                      )}
                      <span className="truncate">{file.name}</span>
                    </button>
                  ))}
                </div>

                {selectedFile ? (
                  <div className="min-h-0 px-3 py-3">
                    <div className="mb-2 flex items-center gap-2">
                      <FileText className="h-4 w-4 shrink-0" />
                      <span className="truncate text-sm font-medium">{selectedFile.name}</span>
                    </div>
                    {fileContent === null ? (
                      <div className="flex items-center text-sm text-muted-foreground">
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Loading preview…
                      </div>
                    ) : (
                      <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-all text-xs text-muted-foreground">
                        {fileContent}
                      </pre>
                    )}
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
        </aside>
      ) : null}
    </>
  )
}
