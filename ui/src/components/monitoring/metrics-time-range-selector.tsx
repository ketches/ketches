import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { type TimeRange, TIME_RANGES } from "./use-time-range"

interface MetricsTimeRangeSelectorProps {
  value: TimeRange
  onChange: (range: TimeRange) => void
}

export function MetricsTimeRangeSelector({ value, onChange }: MetricsTimeRangeSelectorProps) {
  return (
    <Tabs value={value} onValueChange={(v) => onChange(v as TimeRange)} className="h-7">
      <TabsList>
        {TIME_RANGES.map((range) => (
          <TabsTrigger key={range.value} value={range.value}>
            {range.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}
