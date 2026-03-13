import { CollabPriorityOptions } from "@/api/collaboration"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Check } from "lucide-react"
import { useState } from "react"
import { PriorityBadge, StatusBadge, type EntityType } from "./collab-badges"

interface InlineStatusEditorProps {
  currentStatus: string
  entityType: EntityType
  statusOptions: readonly { label: string; value: string }[]
  onStatusChange: (newStatus: string) => void
}

// Inline status editor that renders a StatusBadge as a clickable trigger
// and shows a popover with available status options
export function InlineStatusEditor({ currentStatus, entityType, statusOptions, onStatusChange }: InlineStatusEditorProps) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<button type="button" className="cursor-pointer" />}>
        <StatusBadge status={currentStatus} entityType={entityType} />
      </PopoverTrigger>
      <PopoverContent className="w-48 p-1" align="start">
        {statusOptions.map((opt) => (
          <button
            key={opt.value}
            type="button"
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent cursor-pointer"
            onClick={() => {
              if (opt.value !== currentStatus) {
                onStatusChange(opt.value)
              }
              setOpen(false)
            }}
          >
            <Check className={`h-3.5 w-3.5 ${opt.value === currentStatus ? "opacity-100" : "opacity-0"}`} />
            {opt.label}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  )
}

interface InlinePriorityEditorProps {
  currentPriority: string
  onPriorityChange: (newPriority: string) => void
}

// Inline priority editor that renders a PriorityBadge as a clickable trigger
// and shows a popover with available priority options
export function InlinePriorityEditor({ currentPriority, onPriorityChange }: InlinePriorityEditorProps) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<button type="button" className="cursor-pointer" />}>
        <PriorityBadge priority={currentPriority} />
      </PopoverTrigger>
      <PopoverContent className="w-48 p-1" align="start">
        {CollabPriorityOptions.map((opt) => (
          <button
            key={opt.value}
            type="button"
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent cursor-pointer"
            onClick={() => {
              if (opt.value !== currentPriority) {
                onPriorityChange(opt.value)
              }
              setOpen(false)
            }}
          >
            <Check className={`h-3.5 w-3.5 ${opt.value === currentPriority ? "opacity-100" : "opacity-0"}`} />
            {opt.label}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  )
}
