import * as React from "react"
import { Bot, Settings } from "lucide-react"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "@/components/ui/sidebar"

export type ProjectSettingsSection = "general" | "ai-providers"

interface ProjectSettingsShellProps {
  activeSection: ProjectSettingsSection
  onSectionChange: (section: ProjectSettingsSection) => void
  children: React.ReactNode
}

const sections: Array<{ key: ProjectSettingsSection; label: string; icon: React.ElementType }> = [
  { key: "general", label: "General", icon: Settings },
  { key: "ai-providers", label: "AI Providers", icon: Bot },
]

export function ProjectSettingsShell({ activeSection, onSectionChange, children }: ProjectSettingsShellProps) {
  return (
    <SidebarProvider
      className="flex min-h-[480px] w-full overflow-hidden"
      style={{ "--sidebar-width": "200px" } as React.CSSProperties}
    >
      <Sidebar collapsible="none" className="border-r">
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {sections.map((section) => (
                  <SidebarMenuItem key={section.key}>
                    <SidebarMenuButton
                      isActive={activeSection === section.key}
                      onClick={() => onSectionChange(section.key)}
                    >
                      <section.icon />
                      <span>{section.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
      </Sidebar>
      <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
        {children}
      </div>
    </SidebarProvider>
  )
}
