import { useQuery } from "@tanstack/react-query"
import { GitBranch, Loader2, Tag } from "lucide-react"

import { codeRepositoriesApi } from "@/api/code-repositories"
import { Combobox } from "@base-ui/react"
import {
  ComboboxContent,
  ComboboxGroup,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
  ComboboxTrigger,
  ComboboxValue,
} from "@/components/ui/combobox"
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
    <Combobox.Root value={value} onValueChange={onValueChange} disabled={disabled || isLoading}>
      <ComboboxTrigger
        className={cn(
          "border-input data-[placeholder]:text-muted-foreground bg-input/20 dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/30 gap-1.5 rounded-md border px-2 py-1.5 text-xs/relaxed transition-colors focus-visible:ring-[2px] h-7 flex w-full items-center justify-between whitespace-nowrap outline-none disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
      >
        <ComboboxValue placeholder={placeholder}>
          {isLoading ? (
            <div className="flex items-center gap-2">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>Loading...</span>
            </div>
          ) : (
            value
          )}
        </ComboboxValue>
      </ComboboxTrigger>
      <ComboboxContent>
        <ComboboxList>
          {value && !hasValueInRefs && (
            <ComboboxGroup>
              <ComboboxLabel>Current</ComboboxLabel>
              <ComboboxItem value={value}>{value}</ComboboxItem>
            </ComboboxGroup>
          )}
          {branches.length > 0 && (
            <ComboboxGroup>
              <ComboboxLabel className="flex items-center gap-2">
                <GitBranch className="h-3 w-3" />
                Branches
              </ComboboxLabel>
              {branches.map((ref) => (
                <ComboboxItem key={ref.name} value={ref.name}>
                  {ref.name}
                </ComboboxItem>
              ))}
            </ComboboxGroup>
          )}
          {tags.length > 0 && (
            <ComboboxGroup>
              <ComboboxLabel className="flex items-center gap-2">
                <Tag className="h-3 w-3" />
                Tags
              </ComboboxLabel>
              {tags.map((ref) => (
                <ComboboxItem key={ref.name} value={ref.name}>
                  {ref.name}
                </ComboboxItem>
              ))}
            </ComboboxGroup>
          )}
          {refs.length === 0 && !isLoading && (
            <div className="p-2 text-xs text-muted-foreground text-center">
              No branches or tags found
            </div>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox.Root>
  )
}
