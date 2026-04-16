import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { Avatar } from "@/components/ui/avatar"
import { MemberAvatar } from "./member-avatar"
import { UserAvatarFallback } from "./user-avatar"

describe("user avatar helpers", () => {
  it("uses the same color mapping for compact and regular user avatar fallbacks", () => {
    render(
      <div>
        <div data-testid="member-avatar">
          <MemberAvatar name="Alice" />
        </div>
        <Avatar>
          <UserAvatarFallback name="Alice" data-testid="user-avatar-fallback" />
        </Avatar>
      </div>
    )

    const memberAvatar = screen.getByTestId("member-avatar").firstElementChild as HTMLElement | null
    const userAvatarFallback = screen.getByTestId("user-avatar-fallback")

    expect(memberAvatar).not.toBeNull()
    expect(memberAvatar!.className).toContain("bg-amber-100")
    expect(memberAvatar!.className).toContain("text-amber-700")
    expect(userAvatarFallback.className).toContain("bg-amber-100")
    expect(userAvatarFallback.className).toContain("text-amber-700")
  })

  it("falls back to a question mark when no user name is provided", () => {
    render(
      <Avatar>
        <UserAvatarFallback data-testid="empty-user-avatar" />
      </Avatar>
    )

    expect(screen.getByTestId("empty-user-avatar").textContent).toBe("?")
  })
})
