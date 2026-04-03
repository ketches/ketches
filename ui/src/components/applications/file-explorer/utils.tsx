import { formatDate } from "@/lib/utils"
import { isAxiosError } from "axios"
import {
  File,
  FileClock,
  FileCode,
  FileImage,
  Folder,
} from "lucide-react"
import { toast } from "sonner"

export const FILE_VIEW_MODE_KEY = "file_explorer_view_mode"

export function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    toast.success("Path copied to clipboard")
  }).catch(() => {
    toast.error("Failed to copy path")
  })
}

export function getFileIcon(name: string, type: string, className: string = "h-4 w-4") {
  if (type === "dir") {
    return <Folder className={`${className} text-blue-400`} />
  }

  if (type === "link") {
    return <File className={`${className} text-purple-400`} />
  }

  const ext = name.split(".").pop()?.toLowerCase() || ""
  const codeExts = ["js", "ts", "jsx", "tsx", "go", "py", "rs", "java", "c", "cpp", "h", "rb", "php", "sh", "bash", "zsh", "yaml", "yml", "toml", "json", "xml", "html", "css", "scss", "sql"]
  const imageExts = ["png", "jpg", "jpeg", "gif", "svg", "webp", "ico", "bmp"]
  const textExts = ["txt", "md", "log", "csv", "env", "conf", "cfg", "ini", "properties"]

  if (codeExts.includes(ext)) {
    return <FileCode className={`${className} text-emerald-400`} />
  }

  if (imageExts.includes(ext)) {
    return <FileImage className={`${className} text-orange-400`} />
  }

  if (textExts.includes(ext)) {
    return <FileClock className={`${className} text-yellow-400`} />
  }

  return <File className={`${className} text-muted-foreground`} />
}

export function formatSize(bytes: number): string {
  if (bytes === 0) {
    return "0 B"
  }

  const units = ["B", "KB", "MB", "GB"]
  const index = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, index)).toFixed(index > 0 ? 1 : 0)} ${units[index]}`
}

export function formatTime(timestamp: number): string {
  if (timestamp === 0) {
    return "-"
  }

  return formatDate(timestamp * 1000)
}

export function isTextFile(name: string): boolean {
  const ext = name.split(".").pop()?.toLowerCase() || ""
  const textExts = [
    "txt", "md", "log", "csv", "env", "conf", "cfg", "ini", "properties",
    "js", "ts", "jsx", "tsx", "go", "py", "rs", "java", "c", "cpp", "h",
    "rb", "php", "sh", "bash", "zsh", "yaml", "yml", "toml", "json",
    "xml", "html", "css", "scss", "sql", "makefile", "dockerfile",
    "gitignore", "dockerignore", "editorconfig",
  ]
  const nameLower = name.toLowerCase()
  const noExtNames = ["makefile", "dockerfile", "jenkinsfile", "vagrantfile", "gemfile", "rakefile", "procfile", "license", "readme", "changelog"]
  return textExts.includes(ext) || noExtNames.includes(nameLower)
}

export function buildPath(segments: string[]): string {
  if (segments.length === 0) {
    return "/"
  }

  return `/${segments.join("/")}`
}

export function parsePath(path: string): string[] {
  return path.split("/").filter(Boolean)
}

export function getErrorMessage(error: unknown, fallback: string) {
  if (isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error || error.message || fallback
  }

  if (error instanceof Error) {
    return error.message
  }

  return fallback
}
