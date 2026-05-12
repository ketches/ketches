import { CollabPriorityOptions, DefectSeverityOptions, type CollabOption } from "@/api/collaboration"
import { projectsApi } from "@/api/projects"
import { MemberAvatar } from "@/components/shared/member-avatar"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { InputGroupAddon } from "@/components/ui/input-group"
import { useQuery } from "@tanstack/react-query"
import { User } from "lucide-react"
import { useMemo } from "react"

const colorMap: Record<string, string> = {
  gray: "text-gray-500",
  slate: "text-slate-500",
  blue: "text-blue-500",
  green: "text-green-500",
  red: "text-red-500",
  orange: "text-orange-500",
  amber: "text-amber-500",
  purple: "text-purple-500",
  indigo: "text-indigo-500",
  yellow: "text-yellow-600",
}

interface InlineComboboxEditorProps {
  value: string
  options: CollabOption[]
  onValueChange: (newValue: string) => void
  disabled?: boolean
}

function InlineComboboxEditor({ value, options, onValueChange, disabled }: InlineComboboxEditorProps) {
  const current = options.find((o) => o.value === value)
  const CurrentIcon = current?.icon
  const currentColor = colorMap[current?.color ?? ""] ?? "text-muted-foreground"

  return (
    <Combobox
      value={value}
      onValueChange={(val: string | null) => {
        if (val && val !== value) onValueChange(val)
      }}
      itemToStringLabel={(v) => options.find((o) => o.value === v)?.label ?? v ?? ""}
      disabled={disabled}
    >
      <ComboboxInput className="w-36">
        <InputGroupAddon>
          {CurrentIcon && <CurrentIcon className={`size-3.5 ${currentColor}`} />}
        </InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent alignOffset={-24} className="w-36">
        <ComboboxList>
          {options.map((opt) => {
            const Icon = opt.icon
            const color = colorMap[opt.color] ?? "text-muted-foreground"
            return (
              <ComboboxItem key={opt.value} value={opt.value}>
                <Icon className={`size-3.5 ${color}`} />
                {opt.label}
              </ComboboxItem>
            )
          })}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}

interface InlineStatusEditorProps {
  currentStatus: string
  statusOptions: CollabOption[]
  onStatusChange: (newStatus: string) => void
  disabled?: boolean
}

export function InlineStatusEditor({ currentStatus, statusOptions, onStatusChange, disabled }: InlineStatusEditorProps) {
  return (
    <InlineComboboxEditor
      value={currentStatus}
      options={statusOptions}
      onValueChange={onStatusChange}
      disabled={disabled}
    />
  )
}

interface InlinePriorityEditorProps {
  currentPriority: string
  onPriorityChange: (newPriority: string) => void
  disabled?: boolean
}

export function InlinePriorityEditor({ currentPriority, onPriorityChange, disabled }: InlinePriorityEditorProps) {
  return (
    <InlineComboboxEditor
      value={currentPriority}
      options={CollabPriorityOptions}
      onValueChange={onPriorityChange}
      disabled={disabled}
    />
  )
}

interface InlineSeverityEditorProps {
  currentSeverity: string
  onSeverityChange: (newSeverity: string) => void
  disabled?: boolean
}

export function InlineSeverityEditor({ currentSeverity, onSeverityChange, disabled }: InlineSeverityEditorProps) {
  return (
    <InlineComboboxEditor
      value={currentSeverity}
      options={DefectSeverityOptions}
      onValueChange={onSeverityChange}
      disabled={disabled}
    />
  )
}

interface InlineAssigneeEditorProps {
  projectId: string
  currentAssigneeId?: string
  onAssigneeChange: (assigneeId: string) => void
  disabled?: boolean
}

export function InlineAssigneeEditor({ projectId, currentAssigneeId, onAssigneeChange, disabled }: InlineAssigneeEditorProps) {
  const { data } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => projectsApi.listMembers(projectId, { page: 1, page_size: 100 }),
    enabled: !!projectId,
  })

  const members = data?.items ?? []

  const options = useMemo(() => [
    // { label: "Unassigned", value: "" },
    ...members.map(m => ({ label: m.fullname || m.username, value: m.user_id })),
  ], [members])

  const current = options.find(o => o.value === (currentAssigneeId ?? ""))

  return (
    <Combobox
      value={currentAssigneeId ?? ""}
      onValueChange={(val: string | null) => {
        const next = val ?? ""
        if (next !== (currentAssigneeId ?? "")) onAssigneeChange(next)
      }}
      itemToStringLabel={(v) => options.find(o => o.value === v)?.label ?? v ?? ""}
      disabled={disabled}
    >
      <ComboboxInput className="w-36">
        <InputGroupAddon>
          {current && current.value ? (
            <MemberAvatar name={current.label} />
          ) : (
            <User className="size-3.5 text-muted-foreground" />
          )}
        </InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent alignOffset={-24} className="w-36">
        <ComboboxList>
          {options.map((opt) => (
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

export { MemberAvatar }
