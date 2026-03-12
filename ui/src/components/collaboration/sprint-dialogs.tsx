import { Button, buttonVariants } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { collaborationApi, SprintStatus, type Sprint, type CreateSprintRequest, type UpdateSprintRequest } from "@/api/collaboration"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Calendar } from "@/components/ui/calendar"
import { format } from "date-fns"
import { CalendarIcon } from "lucide-react"
import { cn } from "@/lib/utils"

interface CreateSprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onSuccess?: () => void
}

export function CreateSprintDialog({
  open,
  onOpenChange,
  projectId,
  onSuccess
}: CreateSprintDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState<CreateSprintRequest>({
    name: "",
    goal: "",
    status: SprintStatus.PLANNED,
    start_date: "",
    end_date: "",
  })
  const [startDate, setStartDate] = useState<Date | undefined>(undefined)
  const [endDate, setEndDate] = useState<Date | undefined>(undefined)

  useEffect(() => {
    if (open) {
      setFormData({
        name: "",
        goal: "",
        status: SprintStatus.PLANNED,
        start_date: "",
        end_date: "",
      })
      setStartDate(undefined)
      setEndDate(undefined)
    }
  }, [open])

  useEffect(() => {
    if (startDate) {
      setFormData(prev => ({ ...prev, start_date: startDate.toISOString() }))
    }
    if (endDate) {
      setFormData(prev => ({ ...prev, end_date: endDate.toISOString() }))
    }
  }, [startDate, endDate])

  const mutation = useMutation({
    mutationFn: (data: CreateSprintRequest) => {
      return collaborationApi.createSprint(projectId, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sprints", projectId] })
      toast.success("Sprint created")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to create sprint", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }



  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    mutation.mutate(formData)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Create Sprint</DialogTitle>
          <DialogDescription>
            Create a new sprint to plan your work.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="Sprint 1"
              required
            />
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="goal">Goal</Label>
            <Textarea
              id="goal"
              value={formData.goal || ""}
              onChange={(e) => setFormData({ ...formData, goal: e.target.value })}
              placeholder="Sprint goal..."
              className="min-h-24"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Status</Label>
              <Combobox
                value={formData.status}
                onValueChange={(val) => val && setFormData({ ...formData, status: val as SprintStatus })}
              >
                <ComboboxInput placeholder="Select status" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(SprintStatus).map((status) => (
                      <ComboboxItem key={status} value={status}>
                        {status.replace(/_/g, " ")}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 flex flex-col">
              <Label>Start Date</Label>
              <Popover>
                <PopoverTrigger
                  className={cn(
                    buttonVariants({ variant: "outline" }),
                    "w-full pl-3 text-left font-normal",
                    !startDate && "text-muted-foreground"
                  )}
                >
                  {startDate ? (
                    format(startDate, "PPP")
                  ) : (
                    <span>Pick a date</span>
                  )}
                  <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={startDate}
                    onSelect={setStartDate}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>

            <div className="space-y-2 flex flex-col">
              <Label>End Date</Label>
              <Popover>
                <PopoverTrigger
                  className={cn(
                    buttonVariants({ variant: "outline" }),
                    "w-full pl-3 text-left font-normal",
                    !endDate && "text-muted-foreground"
                  )}
                >
                  {endDate ? (
                    format(endDate, "PPP")
                  ) : (
                    <span>Pick a date</span>
                  )}
                  <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={endDate}
                    onSelect={setEndDate}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

interface EditSprintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  sprint: Sprint | null
  onSuccess?: () => void
}

export function EditSprintDialog({
  open,
  onOpenChange,
  projectId,
  sprint,
  onSuccess
}: EditSprintDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState<UpdateSprintRequest>({
    name: "",
    goal: "",
    status: SprintStatus.PLANNED,
    start_date: "",
    end_date: "",
  })
  const [startDate, setStartDate] = useState<Date | undefined>(undefined)
  const [endDate, setEndDate] = useState<Date | undefined>(undefined)

  useEffect(() => {
    if (sprint && open) {
      setFormData({
        name: sprint.name,
        goal: sprint.goal,
        status: sprint.status,
        start_date: sprint.start_date,
        end_date: sprint.end_date,
      })
      setStartDate(sprint.start_date ? new Date(sprint.start_date) : undefined)
      setEndDate(sprint.end_date ? new Date(sprint.end_date) : undefined)
    }
  }, [sprint, open])

  useEffect(() => {
    if (startDate) {
      setFormData(prev => ({ ...prev, start_date: startDate.toISOString() }))
    }
    if (endDate) {
      setFormData(prev => ({ ...prev, end_date: endDate.toISOString() }))
    }
  }, [startDate, endDate])

  const mutation = useMutation({
    mutationFn: (data: UpdateSprintRequest) => {
      if (!sprint) throw new Error("No sprint selected")
      return collaborationApi.updateSprint(projectId, sprint.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sprints", projectId] })
      toast.success("Sprint updated")
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: string } } }
      toast.error("Failed to update sprint", {
        description: err.response?.data?.error || "Unknown error occurred"
      })
    }

  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    mutation.mutate(formData)
  }

  if (!sprint) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Edit Sprint</DialogTitle>
          <DialogDescription>
            Update sprint details.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-name">Name</Label>
            <Input
              id="edit-name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="Sprint 1"
              required
            />
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="edit-goal">Goal</Label>
            <Textarea
              id="edit-goal"
              value={formData.goal || ""}
              onChange={(e) => setFormData({ ...formData, goal: e.target.value })}
              placeholder="Sprint goal..."
              className="min-h-24"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Status</Label>
              <Combobox
                value={formData.status}
                onValueChange={(val) => val && setFormData({ ...formData, status: val as SprintStatus })}
              >
                <ComboboxInput placeholder="Select status" />
                <ComboboxContent>
                  <ComboboxList>
                    {Object.values(SprintStatus).map((status) => (
                      <ComboboxItem key={status} value={status}>
                        {status.replace(/_/g, " ")}
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 flex flex-col">
              <Label>Start Date</Label>
              <Popover>
                <PopoverTrigger
                  className={cn(
                    buttonVariants({ variant: "outline" }),
                    "w-full pl-3 text-left font-normal",
                    !startDate && "text-muted-foreground"
                  )}
                >
                  {startDate ? (
                    format(startDate, "PPP")
                  ) : (
                    <span>Pick a date</span>
                  )}
                  <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={startDate}
                    onSelect={setStartDate}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>

            <div className="space-y-2 flex flex-col">
              <Label>End Date</Label>
              <Popover>
                <PopoverTrigger
                  className={cn(
                    buttonVariants({ variant: "outline" }),
                    "w-full pl-3 text-left font-normal",
                    !endDate && "text-muted-foreground"
                  )}
                >
                  {endDate ? (
                    format(endDate, "PPP")
                  ) : (
                    <span>Pick a date</span>
                  )}
                  <CalendarIcon className="ml-auto h-4 w-4 opacity-50" />
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={endDate}
                    onSelect={setEndDate}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? "Save Changes" : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
