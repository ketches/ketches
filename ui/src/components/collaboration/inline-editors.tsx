import { CollabPriorityOptions } from "@/api/collaboration"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn } from "@/lib/utils"
import { Check } from "lucide-react"
import { useState } from "react"
import { PriorityBadge, StatusBadge } from "./collab-badges"

interface InlineStatusEditorProps {
  currentStatus: string
  statusOptions: readonly { label: string; value: string }[]
  onStatusChange: (newStatus: string) => void
}

export function InlineStatusEditor({
  currentStatus,
  statusOptions,
  onStatusChange,
}: InlineStatusEditorProps) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<button className="cursor-pointer rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" />}>
        <StatusBadge status={currentStatus} />
      </PopoverTrigger>
      <PopoverContent className="w-48 p-1">
        <div className="space-y-1">
          {statusOptions.map((option) => (
            <button
              key={option.value}
              className={cn(
                "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground outline-none",
                currentStatus === option.value && "bg-accent/50"
              )}
              onClick={() => {
                if (currentStatus !== option.value) {
                  onStatusChange(option.value)
                }
                setOpen(false)
              }}
            >
              <span className="flex-1 text-left">{option.label}</span>
              {currentStatus === option.value && <Check className="h-4 w-4" />}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}

interface InlinePriorityEditorProps {
  currentPriority: string
  onPriorityChange: (newPriority: string) => void
}

export function InlinePriorityEditor({
  currentPriority,
  onPriorityChange,
}: InlinePriorityEditorProps) {
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<button className="cursor-pointer rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" />}>
        <PriorityBadge priority={currentPriority} />
      </PopoverTrigger>
      <PopoverContent className="w-48 p-1">
        <div className="space-y-1">
          {CollabPriorityOptions.map((option) => (
            <button
              key={option.value}
              className={cn(
                "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground outline-none",
                currentPriority === option.value && "bg-accent/50"
              )}
              onClick={() => {
                if (currentPriority !== option.value) {
                  onPriorityChange(option.value)
                }
                setOpen(false)
              }}
            >
              <span className="flex-1 text-left">{option.label}</span>
              {currentPriority === option.value && <Check className="h-4 w-4" />}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
