import * as React from "react"

import { Button } from "@/components/ui/button"

export type AccountSettingsSection = "profile" | "security" | "ai-providers"

interface AccountSettingsShellProps {
  activeSection: AccountSettingsSection
  onSectionChange: (section: AccountSettingsSection) => void
  children: React.ReactNode
}

const sections: Array<{ key: AccountSettingsSection; label: string }> = [
  { key: "profile", label: "Profile" },
  { key: "security", label: "Security" },
  { key: "ai-providers", label: "AI Providers" },
]

export function AccountSettingsShell({ activeSection, onSectionChange, children }: AccountSettingsShellProps) {
  return (
    <div className="flex min-h-105 gap-6 overflow-hidden md:flex-row">
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
