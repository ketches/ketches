import * as React from "react"
import { Popover as PopoverPrimitive } from "@base-ui/react/popover"

import { cn } from "@/lib/utils"

type HoverCardContextValue = {
  onTriggerEnter: () => void
  onTriggerLeave: () => void
  onContentEnter: () => void
  onContentLeave: () => void
}

const HoverCardContext = React.createContext<HoverCardContextValue | null>(null)

type HoverCardProps = Omit<PopoverPrimitive.Root.Props, "open" | "defaultOpen" | "onOpenChange"> & {
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
  openDelay?: number
  closeDelay?: number
}

function HoverCard({
  open,
  defaultOpen = false,
  onOpenChange,
  openDelay = 120,
  closeDelay = 120,
  ...props
}: HoverCardProps) {
  const [uncontrolledOpen, setUncontrolledOpen] = React.useState(defaultOpen)
  const openTimerRef = React.useRef<number | null>(null)
  const closeTimerRef = React.useRef<number | null>(null)

  const isControlled = open !== undefined
  const currentOpen = isControlled ? open : uncontrolledOpen

  const setOpen = React.useCallback(
    (nextOpen: boolean) => {
      if (!isControlled) {
        setUncontrolledOpen(nextOpen)
      }
      onOpenChange?.(nextOpen)
    },
    [isControlled, onOpenChange]
  )

  const clearOpenTimer = React.useCallback(() => {
    if (openTimerRef.current !== null) {
      window.clearTimeout(openTimerRef.current)
      openTimerRef.current = null
    }
  }, [])

  const clearCloseTimer = React.useCallback(() => {
    if (closeTimerRef.current !== null) {
      window.clearTimeout(closeTimerRef.current)
      closeTimerRef.current = null
    }
  }, [])

  React.useEffect(() => {
    return () => {
      clearOpenTimer()
      clearCloseTimer()
    }
  }, [clearCloseTimer, clearOpenTimer])

  const contextValue = React.useMemo<HoverCardContextValue>(
    () => ({
      onTriggerEnter: () => {
        clearCloseTimer()
        clearOpenTimer()
        openTimerRef.current = window.setTimeout(() => {
          setOpen(true)
        }, openDelay)
      },
      onTriggerLeave: () => {
        clearOpenTimer()
        clearCloseTimer()
        closeTimerRef.current = window.setTimeout(() => {
          setOpen(false)
        }, closeDelay)
      },
      onContentEnter: () => {
        clearCloseTimer()
      },
      onContentLeave: () => {
        clearCloseTimer()
        closeTimerRef.current = window.setTimeout(() => {
          setOpen(false)
        }, closeDelay)
      },
    }),
    [clearCloseTimer, clearOpenTimer, closeDelay, openDelay, setOpen]
  )

  return (
    <HoverCardContext.Provider value={contextValue}>
      <PopoverPrimitive.Root data-slot="hover-card" open={currentOpen} onOpenChange={setOpen} {...props} />
    </HoverCardContext.Provider>
  )
}

function HoverCardTrigger({
  onMouseEnter,
  onMouseLeave,
  onFocus,
  onBlur,
  ...props
}: PopoverPrimitive.Trigger.Props) {
  const context = React.useContext(HoverCardContext)

  return (
    <PopoverPrimitive.Trigger
      data-slot="hover-card-trigger"
      onMouseEnter={(event) => {
        context?.onTriggerEnter()
        onMouseEnter?.(event)
      }}
      onMouseLeave={(event) => {
        context?.onTriggerLeave()
        onMouseLeave?.(event)
      }}
      onFocus={(event) => {
        context?.onTriggerEnter()
        onFocus?.(event)
      }}
      onBlur={(event) => {
        context?.onTriggerLeave()
        onBlur?.(event)
      }}
      {...props}
    />
  )
}

function HoverCardContent({
  className,
  align = "center",
  alignOffset = 0,
  side = "bottom",
  sideOffset = 8,
  onMouseEnter,
  onMouseLeave,
  ...props
}: PopoverPrimitive.Popup.Props &
  Pick<
    PopoverPrimitive.Positioner.Props,
    "align" | "alignOffset" | "side" | "sideOffset"
  >) {
  const context = React.useContext(HoverCardContext)

  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Positioner
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
        className="isolate z-50"
      >
        <PopoverPrimitive.Popup
          data-slot="hover-card-content"
          onMouseEnter={(event) => {
            context?.onContentEnter()
            onMouseEnter?.(event)
          }}
          onMouseLeave={(event) => {
            context?.onContentLeave()
            onMouseLeave?.(event)
          }}
          className={cn(
            "bg-popover text-popover-foreground data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/10 rounded-lg p-3 text-xs shadow-md ring-1 duration-100 data-[side=inline-start]:slide-in-from-right-2 data-[side=inline-end]:slide-in-from-left-2 z-50 w-fit max-w-sm origin-(--transform-origin) outline-hidden",
            className
          )}
          {...props}
        />
      </PopoverPrimitive.Positioner>
    </PopoverPrimitive.Portal>
  )
}

export { HoverCard, HoverCardContent, HoverCardTrigger }
