import { Button } from "@/components/ui/button"
import { type TimeRange, TIME_RANGES } from "./use-time-range"

interface MetricsTimeRangeSelectorProps {
  value: TimeRange
  onChange: (range: TimeRange) => void
}

export function MetricsTimeRangeSelector({ value, onChange }: MetricsTimeRangeSelectorProps) {
  return (
    <div className="flex items-center gap-1">
      {TIME_RANGES.map((range) => (
        <Button
          key={range.value}
          variant={value === range.value ? "default" : "outline"}
          size="sm"
          className="h-7 px-2.5 text-xs"
          onClick={() => onChange(range.value)}
        >
          {range.label}
        </Button>
      ))}
    </div>
  )
}
