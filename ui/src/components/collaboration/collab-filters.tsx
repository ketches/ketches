import { useQuery } from "@tanstack/react-query"
import { Filter, User } from "lucide-react"
import { useMemo } from "react"

import { CollabPriorityOptions } from "@/api/collaboration"
import { projectsApi } from "@/api/projects"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { InputGroupAddon } from "@/components/ui/input-group"
import { MemberAvatar } from "./inline-editors"

interface StatusFilterProps {
  value: string
  onChange: (value: string) => void
  options: { label: string; value: string; icon: React.ComponentType<{ className?: string }>; color: string }[]
}

export function StatusFilter({ value, onChange, options }: StatusFilterProps) {
  const allOptions = [{ label: "All Statuses", value: "", icon: Filter, color: "gray" }, ...options]
  return (
    <Combobox value={value} onValueChange={(val) => onChange(val ?? "")} itemToStringLabel={(item) => allOptions.find(opt => opt.value === item)?.label || item}>
      <ComboboxInput placeholder="Status" className="w-36">
        <InputGroupAddon><Filter className="h-3.5 w-3.5" /></InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent alignOffset={-24} className="w-36">
        <ComboboxList>
          {allOptions.map((opt) => (
            <ComboboxItem key={opt.value} value={opt.value}>
              <opt.icon className={`h-3.5 w-3.5 text-${opt.color}-500`} />
              {opt.label}
            </ComboboxItem>
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
  const allOptions = [{ label: "All Priorities", value: "", icon: Filter, color: "gray" }, ...CollabPriorityOptions]
  return (
    <Combobox value={value} onValueChange={(val) => onChange(val ?? "")} itemToStringLabel={(item) => allOptions.find(opt => opt.value === item)?.label || item}>
      <ComboboxInput placeholder="Priority" className="w-36">
        <InputGroupAddon><Filter className="h-3.5 w-3.5" /></InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent alignOffset={-24} className="w-36">
        <ComboboxList>
          {allOptions.map((opt) => (
            <ComboboxItem key={opt.value} value={opt.value}>
              <opt.icon className={`h-3.5 w-3.5 text-${opt.color}-500`} />
              {opt.label}
            </ComboboxItem>
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
      <ComboboxContent alignOffset={-24} className="w-36">
        <ComboboxList>
          {memberOptions.map((opt) => (
            <ComboboxItem key={opt.value} value={opt.value}>
              {opt.value ? <MemberAvatar name={opt.label} /> : <User className="size-3.5 text-muted-foreground" />}
              {opt.label}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}