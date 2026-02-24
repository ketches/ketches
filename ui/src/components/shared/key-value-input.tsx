import { Plus, Trash2 } from "lucide-react"
import * as React from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export interface KeyValuePair {
  key: string
  value: string
}

interface KeyValueInputProps {
  value: KeyValuePair[]
  onChange: (value: KeyValuePair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  className?: string
}

export function KeyValueInput({
  value,
  onChange,
  keyPlaceholder = "KEY",
  valuePlaceholder = "VALUE",
  className,
}: KeyValueInputProps) {
  const [newKey, setNewKey] = React.useState("")
  const [newValue, setNewValue] = React.useState("")

  const handleAdd = () => {
    if (!newKey.trim()) return

    onChange([...value, { key: newKey.trim(), value: newValue.trim() }])
    setNewKey("")
    setNewValue("")
  }

  const handleRemove = (index: number) => {
    onChange(value.filter((_, i) => i !== index))
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault()
      handleAdd()
    }
  }

  const canAdd = newKey.trim().length > 0

  return (
    <div className={className}>
      {value.length > 0 && (
        <div className="space-y-2 mb-3">
          {value.map((pair, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                value={pair.key}
                disabled
                className="flex-1 font-mono text-sm"
                placeholder={keyPlaceholder}
              />
              <Input
                value={pair.value}
                disabled
                className="flex-1 font-mono text-sm"
                placeholder={valuePlaceholder}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => handleRemove(index)}
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
              >
                <Trash2 />
                <span className="sr-only">Remove</span>
              </Button>
            </div>
          ))}
        </div>
      )}

      <div className="flex items-center gap-2">
        <Input
          placeholder={keyPlaceholder}
          value={newKey}
          onChange={(e) => setNewKey(e.target.value)}
          onKeyDown={handleKeyDown}
          className="flex-1 font-mono text-sm"
        />
        <Input
          placeholder={valuePlaceholder}
          value={newValue}
          onChange={(e) => setNewValue(e.target.value)}
          onKeyDown={handleKeyDown}
          className="flex-1 font-mono text-sm"
        />
        <Button
          type="button"
          variant="ghost"
          onClick={handleAdd}
          disabled={!canAdd}
          size="icon"
          className="gap-1.5"
        >
          <Plus />
        </Button>
      </div>
    </div>
  )
}
