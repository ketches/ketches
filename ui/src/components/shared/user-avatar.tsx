import { AvatarFallback } from "@/components/ui/avatar"
import { getUserAvatarColorClass, getUserAvatarInitial } from "@/lib/user-avatar"
import { cn } from "@/lib/utils"

export function UserAvatarFallback({
  name,
  className,
  children,
  ...props
}: React.ComponentProps<typeof AvatarFallback> & {
  name?: string
}) {
  return (
    <AvatarFallback
      className={cn(getUserAvatarColorClass(name), className)}
      {...props}
    >
      {children ?? getUserAvatarInitial(name)}
    </AvatarFallback>
  )
}
