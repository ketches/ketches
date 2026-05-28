import { readFileSync } from "node:fs"
import { describe, expect, it } from "vitest"

const detailPages = [
  "applications/application-detail-page.tsx",
  "clusters/cluster-detail-page.tsx",
  "clusters/cluster-node-detail-page.tsx",
  "code-repositories/code-repository-detail-page.tsx",
  "environments/environment-detail-page.tsx",
  "projects/project-detail-page.tsx",
] as const

describe("detail page scroll areas", () => {
  it.each(detailPages)("%s uses the shared themed detail scroll area", (pagePath) => {
    const source = readFileSync(new URL(pagePath, import.meta.url), "utf8")

    expect(source).toContain("@/components/layout/detail-page-scroll-area")
    expect(source).toContain("<DetailPageScrollArea")
  })

  it("keeps the themed scrollbar overlaid outside the detail content flow", () => {
    const detailScrollAreaSource = readFileSync(
      "src/components/layout/detail-page-scroll-area.tsx",
      "utf8"
    )
    const scrollAreaSource = readFileSync(
      "src/components/ui/scroll-area.tsx",
      "utf8"
    )

    expect(detailScrollAreaSource).toContain('"flex flex-col gap-6 px-px pb-4"')
    expect(detailScrollAreaSource).not.toContain("pr-3")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:absolute")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:right-0")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:translate-x-4")
    expect(detailScrollAreaSource).not.toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:translate-x-full")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:absolute")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:bottom-0")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-orientation=horizontal]]:translate-y-full")
    expect(scrollAreaSource).toContain("data-vertical:h-full")
    expect(scrollAreaSource).not.toContain("data-vertical:absolute")
    expect(scrollAreaSource).not.toContain("data-horizontal:absolute")
  })

  it("auto-hides the detail scrollbar while idle without changing the shared ScrollArea", () => {
    const detailScrollAreaSource = readFileSync(
      "src/components/layout/detail-page-scroll-area.tsx",
      "utf8"
    )
    const scrollAreaSource = readFileSync(
      "src/components/ui/scroll-area.tsx",
      "utf8"
    )

    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar]]:opacity-0")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar]]:transition-opacity")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar]:hover]:opacity-100")
    expect(detailScrollAreaSource).toContain("[&_[data-slot=scroll-area-scrollbar][data-scrolling]]:opacity-100")
    expect(detailScrollAreaSource).not.toContain("[data-hovering]")
    expect(scrollAreaSource).not.toContain("opacity-0")
    expect(scrollAreaSource).not.toContain("data-hovering")
    expect(scrollAreaSource).not.toContain("data-scrolling")
  })
})
