import {
  Kanban,
  ListTodo,
  Bug,
  TestTube,
  Target,
  FileText
} from "lucide-react"
import { useState } from "react"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import SprintsPage from "./sprints-page"
import RequirementsPage from "./requirements-page"
import BacklogPage from "./backlog-page"
import TasksPage from "./tasks-page"
import TestCasesPage from "./test-cases-page"
import DefectsPage from "./defects-page"

interface CollaborationPageProps {
  projectId: string
}

export function CollaborationPage({ projectId }: CollaborationPageProps) {
  const [activeTab, setActiveTab] = useState("sprints")

  return (
    <div className="flex flex-col gap-4">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="sprints">
            <Kanban className="mr-2 h-4 w-4" />
            Sprints
          </TabsTrigger>
          <TabsTrigger value="requirements">
            <FileText className="mr-2 h-4 w-4" />
            Requirements
          </TabsTrigger>
          <TabsTrigger value="backlog">
            <ListTodo className="mr-2 h-4 w-4" />
            Backlog
          </TabsTrigger>
          <TabsTrigger value="tasks">
            <Target className="mr-2 h-4 w-4" />
            Tasks
          </TabsTrigger>
          <TabsTrigger value="test-cases">
            <TestTube className="mr-2 h-4 w-4" />
            Test Cases
          </TabsTrigger>
          <TabsTrigger value="defects">
            <Bug className="mr-2 h-4 w-4" />
            Defects
          </TabsTrigger>
        </TabsList>

        <TabsContent value="sprints" className="mt-4">
          <SprintsPage projectId={projectId} />
        </TabsContent>
        
        <TabsContent value="requirements" className="mt-4">
          <RequirementsPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="backlog" className="mt-4">
          <BacklogPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="tasks" className="mt-4">
          <TasksPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="test-cases" className="mt-4">
          <TestCasesPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="defects" className="mt-4">
          <DefectsPage projectId={projectId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
