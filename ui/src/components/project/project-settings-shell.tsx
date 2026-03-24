import * as React from "react"

import { Button } from "@/components/ui/button"

export type ProjectSettingsSection = "general" | "ai-providers"

interface ProjectSettingsShellProps {
  activeSection: ProjectSettingsSection
  onSectionChange: (section: ProjectSettingsSection) => void
  children: React.ReactNode
}

const sections: Array<{ key: ProjectSettingsSection; label: string }> = [
  { key: "general", label: "General" },
  { key: "ai-providers", label: "AI Providers" },
]

export function ProjectSettingsShell({ activeSection, onSectionChange, children }: ProjectSettingsShellProps) {
  return (
    <div className="flex min-h-[420px] gap-6 overflow-hidden md:flex-row">
      <aside className="w-full shrink-0 border-b pb-4 md:w-56 md:border-b-0 md:border-r md:pb-0 md:pr-4">
        <div className="space-y-1">
          {sections.map((section) => (
            <Button
              key={section.key}
              type="button"
              variant={activeSection === section.key ? "secondary" : "ghost"}
              className="w-full justify-start"
              onClick={() => onSectionChange(section.key)}
            >
              {section.label}
            </Button>
          ))}
        </div>
      </aside>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}
