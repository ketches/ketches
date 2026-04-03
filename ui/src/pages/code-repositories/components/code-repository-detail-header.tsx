import { ChevronsUpDown, FolderGit2, Pencil, Play } from "lucide-react"

import type { CodeRepository } from "@/api/code-repositories"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

interface CodeRepositoryDetailHeaderProps {
  repo: CodeRepository
  safeRepos: CodeRepository[]
  isViewer: boolean
  hasBuildSettings: boolean
  onSelectRepo: (repoId: string) => void
  onEdit: () => void
  onBuild: () => void
}

export function CodeRepositoryDetailHeader({
  repo,
  safeRepos,
  isViewer,
  hasBuildSettings,
  onSelectRepo,
  onEdit,
  onBuild,
}: CodeRepositoryDetailHeaderProps) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div className="rounded-lg bg-primary/10 p-3 text-primary shrink-0">
            <FolderGit2 className="h-8 w-8" />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h1 className="truncate text-2xl font-bold tracking-tight">{repo.name}</h1>
              {safeRepos.length > 1 && (
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button variant="ghost" size="icon-sm">
                        <ChevronsUpDown />
                      </Button>
                    }
                  />
                  <DropdownMenuContent align="start" className="w-fit">
                    <DropdownMenuGroup>
                      {safeRepos.map((candidate) => (
                        <DropdownMenuItem key={candidate.id} onClick={() => onSelectRepo(candidate.id)}>
                          <FolderGit2 className="h-4 w-4" />
                          {candidate.name}
                        </DropdownMenuItem>
                      ))}
                    </DropdownMenuGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span className="font-mono">{repo.slug}</span>
              <span>•</span>
              {repo.description ? (
                <span className="truncate">{repo.description}</span>
              ) : (
                <span className="italic">No description</span>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {!isViewer && (
            <Button variant="outline" size="icon" onClick={onEdit}>
              <Pencil />
            </Button>
          )}
          {hasBuildSettings && !isViewer && (
            <Button onClick={onBuild}>
              <Play />
              Build
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
