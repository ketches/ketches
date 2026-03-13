import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { Filter } from "lucide-react"

import { CollabPriorityOptions } from "@/api/collaboration"
import { projectsApi } from "@/api/projects"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { InputGroupAddon } from "@/components/ui/input-group"

interface StatusFilterProps {
  value: string
  onChange: (value: string) => void
  options: readonly { label: string; value: string }[]
}

export function StatusFilter({ value, onChange, options }: StatusFilterProps) {
  const allOptions = [{ label: "All Statuses", value: "" }, ...options]
  return (
    <Combobox value={value} onValueChange={(val) => onChange(val ?? "")} itemToStringLabel={(item) => allOptions.find(opt => opt.value === item)?.label || item}>
      <ComboboxInput placeholder="Status" className="w-36">
        <InputGroupAddon><Filter className="h-3.5 w-3.5" /></InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent>
        <ComboboxList>
          {allOptions.map((opt) => (
            <ComboboxItem key={opt.value} value={opt.value}>{opt.label}</ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}

interface PriorityFilterProps {
  value: string
  onChange: (value: string) => void
}

export function PriorityFilter({ value, onChange }: PriorityFilterProps) {
  const allOptions = [{ label: "All Priorities", value: "" }, ...CollabPriorityOptions]
  return (
    <Combobox value={value} onValueChange={(val) => onChange(val ?? "")} itemToStringLabel={(item) => allOptions.find(opt => opt.value === item)?.label || item}>
      <ComboboxInput placeholder="Priority" className="w-36">
        <InputGroupAddon><Filter className="h-3.5 w-3.5" /></InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent>
        <ComboboxList>
          {allOptions.map((opt) => (
            <ComboboxItem key={opt.value} value={opt.value}>{opt.label}</ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}

interface AssigneeFilterProps {
  projectId: string
  value: string
  onChange: (value: string) => void
}

export function AssigneeFilter({ projectId, value, onChange }: AssigneeFilterProps) {
  const { data } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => projectsApi.listMembers(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const memberOptions = useMemo(() => {
    const members = data?.items ?? []
    return [
      { label: "All Assignees", value: "" },
      ...members.map(m => ({ label: m.fullname || m.username, value: m.user_id })),
    ]
  }, [data?.items])
  
  return (
    <Combobox value={value} onValueChange={(val) => onChange(val ?? "")} itemToStringLabel={(item) => memberOptions.find(opt => opt.value === item)?.label || item}>
      <ComboboxInput placeholder="Assignee" className="w-36">
        <InputGroupAddon><Filter className="h-3.5 w-3.5" /></InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent>
        <ComboboxList>
          {memberOptions.map((opt) => (
            <ComboboxItem key={opt.value} value={opt.value}>{opt.label}</ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}