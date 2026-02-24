import { useQuery } from "@tanstack/react-query"
import { GitBranch, Loader2, Tag } from "lucide-react"

import { codeRepositoriesApi } from "@/api/code-repositories"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

interface GitRefSelectProps {
  repoId: string
  value: string
  onValueChange: (value: string | null) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

export function GitRefSelect({
  repoId,
  value,
  onValueChange,
  placeholder = "Select branch or tag",
  className,
  disabled,
}: GitRefSelectProps) {
  const { data, isLoading } = useQuery({
    queryKey: ["code-repository-refs", repoId],
    queryFn: () => codeRepositoriesApi.listRefs(repoId),
    enabled: !!repoId,
  })

  const refs = data?.refs ?? []
  const branches = refs.filter((r) => r.type === "branch")
  const tags = refs.filter((r) => r.type === "tag")

  const hasValueInRefs = refs.some((r) => r.name === value)

  return (
    <Select value={value} onValueChange={onValueChange} disabled={disabled || isLoading}>
      <SelectTrigger className={cn("w-full", className)}>
        <SelectValue placeholder={placeholder}>
          {isLoading ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>Loading...</span>
            </div>
          ) : (
            value
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {value && !hasValueInRefs && (
          <SelectGroup>
            <SelectLabel>Current</SelectLabel>
            <SelectItem value={value}>{value}</SelectItem>
          </SelectGroup>
        )}
        {branches.length > 0 && (
          <SelectGroup>
            <SelectLabel className="flex items-center gap-2">
              <GitBranch className="h-3 w-3" />
              Branches
            </SelectLabel>
            {branches.map((ref) => (
              <SelectItem key={ref.name} value={ref.name}>
                {ref.name}
              </SelectItem>
            ))}
          </SelectGroup>
        )}
        {tags.length > 0 && (
          <SelectGroup>
            <SelectLabel className="flex items-center gap-2">
              <Tag className="h-3 w-3" />
              Tags
            </SelectLabel>
            {tags.map((ref) => (
              <SelectItem key={ref.name} value={ref.name}>
                {ref.name}
              </SelectItem>
            ))}
          </SelectGroup>
        )}
        {refs.length === 0 && !isLoading && (
          <div className="p-2 text-xs text-muted-foreground text-center">
            No branches or tags found
          </div>
        )}
      </SelectContent>
    </Select>
  )
}
