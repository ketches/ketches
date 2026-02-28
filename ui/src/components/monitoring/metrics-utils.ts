/**
 * Returns a Tailwind text color class based on utilization percentage.
 * Blue: 0-70%, Orange: 70-85%, Red: 85%+
 */
export function getUtilizationColorClass(value: number): string {
  if (value >= 85) return "text-red-500"
  if (value >= 70) return "text-orange-500"
  return "text-blue-500"
}

/**
 * Returns a CSS color string for recharts stroke/fill based on utilization percentage.
 * Blue: 0-70%, Orange: 70-85%, Red: 85%+
 */
export function getUtilizationColor(value: number): string {
  if (value >= 85) return "hsl(0, 72%, 51%)"
  if (value >= 70) return "hsl(25, 95%, 53%)"
  return "hsl(217, 91%, 60%)"
}
