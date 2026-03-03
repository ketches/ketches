import { useState } from "react"

export type TimeRange = "5m" | "15m" | "30m" | "1h" | "4h" | "8h" | "12h" | "1d"

interface TimeRangeConfig {
  rangeSeconds: number
  step: string
  label: string
}

const TIME_RANGE_CONFIG: Record<TimeRange, TimeRangeConfig> = {
  "5m": { rangeSeconds: 300, step: "15", label: "5 min" },
  "15m": { rangeSeconds: 900, step: "30", label: "15 min" },
  "30m": { rangeSeconds: 1800, step: "60", label: "30 min" },
  "1h": { rangeSeconds: 3600, step: "60", label: "1 hour" },
  "4h": { rangeSeconds: 14400, step: "120", label: "4 hours" },
  "8h": { rangeSeconds: 28800, step: "120", label: "8 hours" },
  "12h": { rangeSeconds: 43200, step: "300", label: "12 hours" },
  "1d": { rangeSeconds: 86400, step: "300", label: "1 day" },
}

export const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: "5m", label: "5m" },
  { value: "15m", label: "15m" },
  { value: "30m", label: "30m" },
  { value: "1h", label: "1h" },
  { value: "4h", label: "4h" },
  { value: "8h", label: "8h" },
  { value: "12h", label: "12h" },
  { value: "1d", label: "1d" },
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
