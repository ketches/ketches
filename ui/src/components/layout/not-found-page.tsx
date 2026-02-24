import { ArrowLeft, BadgeAlert, Home } from "lucide-react"
import { useNavigate } from "react-router-dom"

import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"

interface NotFoundPageProps {
  resourceType: string
  resourceName?: string
  backHref?: string
  backLabel?: string
}

export function NotFoundPage({
  resourceType,
  resourceName,
  backHref = "/",
  backLabel = "Back to Home",
}: NotFoundPageProps) {
  const navigate = useNavigate()

  return (
    <Empty className="border border-dashed">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <BadgeAlert />
        </EmptyMedia>
        <EmptyTitle>{resourceType} Not Found</EmptyTitle>
        <EmptyDescription>
          {resourceName ? (
            <>
              The {resourceType.toLowerCase()} <strong>"{resourceName}"</strong> does not exist or has been deleted.
            </>
          ) : (
            <>
              The requested {resourceType.toLowerCase()} does not exist or has been deleted.
            </>
          )}
        </EmptyDescription>
      </EmptyHeader>
      <EmptyContent className="flex-row justify-center gap-2">
        <Button
          onClick={() => navigate(backHref)}
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          {backLabel}
        </Button>
        <Button
          variant="outline"
          onClick={() => navigate("/")}
        >
          <Home className="h-4 w-4 mr-1" />
          Go to Dashboard
        </Button>
      </EmptyContent>
    </Empty>
  )
}

export default NotFoundPage
