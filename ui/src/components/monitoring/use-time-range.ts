import { useState } from "react"

export type TimeRange = "5m" | "15m" | "30m" | "1h" | "4h"

interface TimeRangeConfig {
  rangeSeconds: number
  step: string
  label: string
}

const TIME_RANGE_CONFIG: Record<TimeRange, TimeRangeConfig> = {
  "5m":  { rangeSeconds: 300,   step: "15",  label: "5 min" },
  "15m": { rangeSeconds: 900,   step: "30",  label: "15 min" },
  "30m": { rangeSeconds: 1800,  step: "60",  label: "30 min" },
  "1h":  { rangeSeconds: 3600,  step: "60",  label: "1 hour" },
  "4h":  { rangeSeconds: 14400, step: "120", label: "4 hours" },
}

export const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: "5m", label: "5m" },
  { value: "15m", label: "15m" },
  { value: "30m", label: "30m" },
  { value: "1h", label: "1h" },
  { value: "4h", label: "4h" },
]

export function useTimeRange(defaultRange: TimeRange = "1h") {
  const [timeRange, setTimeRange] = useState<TimeRange>(defaultRange)
  const config = TIME_RANGE_CONFIG[timeRange]
  return {
    timeRange,
    setTimeRange,
    rangeSeconds: config.rangeSeconds,
    step: config.step,
    label: config.label,
  }
}
