import { collaborationApi, SprintStatus, SprintStatusOptions, type CreateSprintRequest, type Sprint, type UpdateSprintRequest } from "@/api/collaboration"
import { Button, buttonVariants } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Combobox, ComboboxContent, ComboboxInput, ComboboxItem, ComboboxList } from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"

import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { format, isValid, parse } from "date-fns"
import { CalendarIcon } from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

const SPRINT_DATE_FORMAT = "yyyy-MM-dd"

const EMPTY_SPRINT_FORM: Omit<CreateSprintRequest, "start_date" | "end_date"> = {
  name: "",
  goal: "",
  status: SprintStatus.PLANNED,
}

function formatSprintDate(date: Date) {
  return format(date, SPRINT_DATE_FORMAT)
}

function parseSprintDate(value?: string) {
  if (!value) {
    return undefined
  }

  const parsed = parse(value.slice(0, 10), SPRINT_DATE_FORMAT, new Date())
  return isValid(parsed) ? parsed : undefined
}

function createSprintRequest(
  formData: Omit<CreateSprintRequest, "start_date" | "end_date">,
  startDate?: Date,
  endDate?: Date,
) {
  if (!startDate || !endDate) {
    return null
  }

  return {
    ...formData,
    start_date: formatSprintDate(startDate),
    end_date: formatSprintDate(endDate),
  }
}

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
  const [formData, setFormData] = useState(EMPTY_SPRINT_FORM)
  const [startDate, setStartDate] = useState<Date | undefined>(undefined)
  const [endDate, setEndDate] = useState<Date | undefined>(undefined)

  const resetForm = () => {
    setFormData(EMPTY_SPRINT_FORM)
    setStartDate(undefined)
    setEndDate(undefined)
  }

  const handleOpenStateChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      resetForm()
    }
    onOpenChange(nextOpen)
  }

  const mutation = useMutation({
    mutationFn: (data: CreateSprintRequest) => {
      return collaborationApi.createSprint(projectId, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sprints", projectId] })
      toast.success("Sprint created")
      resetForm()
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

    const request = createSprintRequest(formData, startDate, endDate)
    if (!request) {
      toast.error("Start date and end date are required")
      return
    }

    mutation.mutate(request)
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenStateChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Create Sprint</SheetTitle>
          <SheetDescription>
            Create a new sprint to plan your work.
          </SheetDescription>
        </SheetHeader>
        <div className="grid flex-1 auto-rows-min gap-4 px-4">
          <div className="grid grid-cols-3 gap-4">
            <Field className="col-span-2">
              <FieldLabel>Name</FieldLabel>
              <FieldContent>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Sprint 1"
                  required
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Status</FieldLabel>
              <FieldContent>
                <Combobox
                  value={formData.status}
                  onValueChange={(val) => val && setFormData({ ...formData, status: val as SprintStatus })}
                  itemToStringLabel={(item) => SprintStatusOptions.find(opt => opt.value === item)?.label || item}
                >
                  <ComboboxInput placeholder="Select status" />
                  <ComboboxContent>
                    <ComboboxList>
                      {SprintStatusOptions.map((opt) => (
                        <ComboboxItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel>Goal</FieldLabel>
            <FieldContent>
              <Textarea
                id="goal"
                value={formData.goal || ""}
                onChange={(e) => setFormData({ ...formData, goal: e.target.value })}
                placeholder="Sprint goal..."
                className="min-h-24"
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Start Date</FieldLabel>
              <FieldContent>
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
                      autoFocus
                    />
                  </PopoverContent>
                </Popover>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>End Date</FieldLabel>
              <FieldContent>
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
                  <PopoverContent className="w-full p-0" align="start">
                    <Calendar
                      mode="single"
                      selected={endDate}
                      onSelect={setEndDate}
                      autoFocus
                    />
                  </PopoverContent>
                </Popover>
              </FieldContent>
            </Field>
          </div>
        </div>

        <SheetFooter>
          <div className="flex w-full items-center justify-end space-x-2">
            <Button type="button" variant="outline" onClick={() => handleOpenStateChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending || !formData.name} onClick={handleSubmit}>
              {mutation.isPending ? "Creating..." : "Create"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
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
  const [formData, setFormData] = useState<Omit<UpdateSprintRequest, "start_date" | "end_date">>(() => ({
    name: sprint?.name ?? "",
    goal: sprint?.goal ?? "",
    status: sprint?.status ?? SprintStatus.PLANNED,
  }))
  const [startDate, setStartDate] = useState<Date | undefined>(() => parseSprintDate(sprint?.start_date))
  const [endDate, setEndDate] = useState<Date | undefined>(() => parseSprintDate(sprint?.end_date))

  const handleOpenStateChange = (nextOpen: boolean) => {
    onOpenChange(nextOpen)
  }

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

    const request = createSprintRequest(formData, startDate, endDate)
    if (!request) {
      toast.error("Start date and end date are required")
      return
    }

    mutation.mutate(request)
  }

  if (!sprint) return null

  return (
    <Sheet open={open} onOpenChange={handleOpenStateChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Edit Sprint</SheetTitle>
          <SheetDescription>
            Update sprint details.
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit}>
          <div className="grid flex-1 auto-rows-min gap-4 px-4">
            <div className="grid grid-cols-3 gap-4">
              <Field className="col-span-2">
                <FieldLabel>Name</FieldLabel>
                <FieldContent>
                  <Input
                    id="edit-name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Sprint 1"
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel>Status</FieldLabel>
                <FieldContent>
                  <Combobox
                    value={formData.status}
                    onValueChange={(val) => val && setFormData({ ...formData, status: val as SprintStatus })}
                    itemToStringLabel={(item) => SprintStatusOptions.find(opt => opt.value === item)?.label || item}
                  >
                    <ComboboxInput placeholder="Select status" />
                    <ComboboxContent>
                      <ComboboxList>
                        {SprintStatusOptions.map((opt) => (
                          <ComboboxItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </FieldContent>
              </Field>
            </div>

            <Field>
              <FieldLabel>Goal</FieldLabel>
              <FieldContent>
                <Textarea
                  id="edit-goal"
                  value={formData.goal || ""}
                  onChange={(e) => setFormData({ ...formData, goal: e.target.value })}
                  placeholder="Sprint goal..."
                  className="min-h-24"
                />
              </FieldContent>
            </Field>

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Start Date</FieldLabel>
                <FieldContent>
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
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel>End Date</FieldLabel>
                <FieldContent>
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
                </FieldContent>
              </Field>
            </div>
          </div>
          <SheetFooter>
            <div className="flex w-full items-center justify-end space-x-2">
              <Button type="button" variant="outline" onClick={() => handleOpenStateChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Save Changes" : "Save"}
              </Button>
            </div>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
