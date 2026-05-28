import { isAxiosError } from "axios"
import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const formatDate = (dateString: string | number | Date) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return "-"
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

export const toTitleCase = (str: string) => {
  return str.replace(/\w\S*/g, (txt) => {
    return txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase()
  })
}

export function capitalizeDisplayMessage(message: string): string {
  return message.replace(/^(\s*)([a-z])/, (_, leading: string, firstLetter: string) => {
    return `${leading}${firstLetter.toUpperCase()}`
  })
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (isAxiosError<{ error?: string }>(error)) {
    return capitalizeDisplayMessage(error.response?.data?.error || error.message || fallback)
  }

  if (error instanceof Error) {
    return capitalizeDisplayMessage(error.message)
  }

  return capitalizeDisplayMessage(fallback)
}
