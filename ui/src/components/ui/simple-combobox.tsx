import * as React from "react"

import { Combobox } from "@base-ui/react"
import {
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
  ComboboxValue,
} from "@/components/ui/combobox"
import { cn } from "@/lib/utils"

export type SimpleComboboxOption = {
  value: string
  label: string
  description?: string
  disabled?: boolean
}

interface SimpleComboboxProps {
  value?: string | null
  onValueChange?: (value: string | null) => void
  options: SimpleComboboxOption[]
  placeholder?: string
  disabled?: boolean
  className?: string
}

export function SimpleCombobox({
  value,
  onValueChange,
  options,
  placeholder = "Select...",
  disabled,
  className,
}: SimpleComboboxProps) {
  return (
    <Combobox.Root
      value={value ?? null}
      onValueChange={onValueChange}
      disabled={disabled}
    >
      <ComboboxTrigger
        className={cn(
          "border-input data-[placeholder]:text-muted-foreground bg-input/20 dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/30 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:aria-invalid:border-destructive/50 gap-1.5 rounded-md border px-2 py-1.5 text-xs/relaxed transition-colors focus-visible:ring-[2px] aria-invalid:ring-[2px] h-7 flex w-full items-center justify-between whitespace-nowrap outline-none disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
      >
        <ComboboxValue placeholder={placeholder} />
      </ComboboxTrigger>
      <ComboboxContent>
        <ComboboxList>
          {options.map((option) => (
            <ComboboxItem
              key={option.value}
              value={option.value}
              disabled={option.disabled}
            >
              {option.description ? (
                <div className="flex flex-col gap-0.5">
                  <span>{option.label}</span>
                  <span className="text-muted-foreground text-[10px] leading-relaxed">
                    {option.description}
                  </span>
                </div>
              ) : (
                option.label
              )}
            </ComboboxItem>
          ))}
        </ComboboxList>
        <ComboboxEmpty>No options found.</ComboboxEmpty>
      </ComboboxContent>
    </Combobox.Root>
  )
}
