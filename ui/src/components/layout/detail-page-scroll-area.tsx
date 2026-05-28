import * as React from "react"

import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"

interface DetailPageScrollAreaProps extends React.ComponentProps<typeof ScrollArea> {
  contentClassName?: string
}

const detailScrollAreaClassName = [
  "min-h-0 flex-1",
  "[&_[data-slot=scroll-area-scrollbar]]:opacity-0",
  "[&_[data-slot=scroll-area-scrollbar]]:transition-opacity",
  "[&_[data-slot=scroll-area-scrollbar]]:duration-150",
  "[&_[data-slot=scroll-area-scrollbar]:hover]:opacity-100",
  "[&_[data-slot=scroll-area-scrollbar][data-scrolling]]:opacity-100",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:absolute",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:inset-x-0",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:bottom-0",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:translate-y-full",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:absolute",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:inset-y-0",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:right-0",
  "[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:translate-x-4",
].join(" ")

export function DetailPageScrollArea({
  children,
  className,
  contentClassName,
  ...props
}: DetailPageScrollAreaProps) {
  return (
    <ScrollArea
      data-detail-page-scroll-area="true"
      className={cn(
        detailScrollAreaClassName,
        className
      )}
      {...props}
    >
      <div
        data-slot="detail-page-scroll-content"
        className={cn("flex flex-col gap-6 px-px pb-4", contentClassName)}
      >
        {children}
      </div>
    </ScrollArea>
  )
}
