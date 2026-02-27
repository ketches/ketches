import { Upload } from "lucide-react"
import { useRef, useState } from "react"
import { toast } from "sonner"

import { usersApi } from "@/api/users"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const FILE_TYPES = [
  { value: "json", label: "JSON", description: "Array of user objects" },
  { value: "csv", label: "CSV", description: "Comma-separated values" },
  { value: "excel", label: "Excel", description: "Microsoft Excel (.xlsx)" },
] as const

interface ImportUsersDialogProps {
  onSuccess?: () => void
}

export function ImportUsersDialog({ onSuccess }: ImportUsersDialogProps) {
  const [open, setOpen] = useState(false)
  const [isPending, setIsPending] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [fileType, setFileType] = useState<"json" | "csv" | "excel">("json")
  const [error, setError] = useState<string>("")
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setSelectedFile(file)
      setError("")

      // Auto-detect file type from extension
      const ext = file.name.split(".").pop()?.toLowerCase()
      if (ext === "csv") {
        setFileType("csv")
      } else if (ext === "json") {
        setFileType("json")
      } else if (ext === "xlsx" || ext === "xls") {
        setFileType("excel")
      }
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!selectedFile) {
      setError("Please select a file to import")
      return
    }

    setIsPending(true)
    try {
      const result = await usersApi.importUsers(selectedFile, fileType)

      let message = `Successfully imported ${result.succeeded} user(s)`
      if (result.failed > 0) {
        message += `. ${result.failed} failed.`
        if (result.errors.length > 0) {
          message += ` First error: ${result.errors[0].message}`
        }
      }
      toast.success(message)

      setOpen(false)
      setSelectedFile(null)
      setError("")
      onSuccess?.()
    } catch (err: any) {
      setError(err.response?.data?.error || "Failed to import users")
    } finally {
      setIsPending(false)
    }
  }

  const handleClose = () => {
    setOpen(false)
    setSelectedFile(null)
    setError("")
    if (fileInputRef.current) {
      fileInputRef.current.value = ""
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>
        <Button variant="outline">
          <Upload />
          Import Users
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Import Users</DialogTitle>
            <DialogDescription>
              Import users from a JSON, CSV, or Excel file. Each user will have a default project created.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>File Format *</FieldLabel>
              <FieldContent>
                <Select
                  value={fileType}
                  onValueChange={(value) => value && setFileType(value as "json" | "csv" | "excel")}
                >
                  <SelectTrigger id="fileType" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {FILE_TYPES.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        <div className="flex flex-col">
                          <span>{type.label}</span>
                          <span className="text-xs text-muted-foreground">
                            {type.description}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel>File *</FieldLabel>
              <FieldContent>
                <Input id="file" type="file" accept=".json,.csv,.xlsx,.xls" onChange={handleFileChange} />
              </FieldContent>
              {selectedFile && (
                <p className="text-sm text-muted-foreground mt-1">
                  Selected: {selectedFile.name} ({(selectedFile.size / 1024).toFixed(1)} KB)
                </p>
              )}
            </Field>

            {error && <FieldError>{error}</FieldError>}

            <div className="rounded-md bg-muted p-4">
              <p className="text-sm font-medium mb-2">File format examples:</p>
              {fileType === "json" ? (
                <pre className="text-xs text-muted-foreground">
                  {`[
  {
    "username": "john",
    "email": "john@example.com",
    "password": "password123",
    "fullname": "John Doe",
    "role": "user"
  }
]`}
                </pre>
              ) : fileType === "csv" ? (
                <pre className="text-xs text-muted-foreground">
                  {`username,email,password,fullname,role
john,john@example.com,password123,John Doe,user
jane,jane@example.com,password456,Jane Doe,admin`}
                </pre>
              ) : (
                <pre className="text-xs text-muted-foreground">
                  {`username | email | password | fullname | role
john    | john@example.com | password123 | John Doe | user
jane    | jane@example.com | password456 | Jane Doe | admin`}
                </pre>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !selectedFile}>
              {isPending ? "Importing..." : "Import Users"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
