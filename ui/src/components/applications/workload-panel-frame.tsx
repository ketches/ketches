import * as React from "react"

import { cn } from "@/lib/utils"

interface WorkloadPanelFrameProps {
    toolbar: React.ReactNode
    status: React.ReactNode
    children: React.ReactNode
    className?: string
}

export function WorkloadPanelFrame({ toolbar, status, children, className }: WorkloadPanelFrameProps) {
    return (
        <div className={cn("flex h-full min-h-0 flex-col", className)}>
            <div className="flex h-8 min-h-8 items-center justify-between border-b bg-muted/20 px-3">
                {toolbar}
            </div>
            <div className="min-h-0 flex-1 overflow-hidden">
                {children}
            </div>
            <div className="flex h-7 min-h-7 items-center justify-between border-t bg-muted/20 px-3 text-[10px] text-muted-foreground">
                {status}
            </div>
        </div>
    )
}
