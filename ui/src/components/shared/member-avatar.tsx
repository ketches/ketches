import { getUserAvatarColorClass, getUserAvatarInitial } from "@/lib/user-avatar"

export function MemberAvatar({ name }: { name?: string }) {
    const letter = getUserAvatarInitial(name)
    const colorClass = getUserAvatarColorClass(name)

    return (
        <span className={`inline-grid h-4 w-4 shrink-0 -translate-x-0.5 place-items-center rounded-full bg-muted text-[9px] leading-none font-medium ${colorClass}`}>
            {letter}
        </span>

    )
}
