const AVATAR_COLOR_CLASSES = [
    "bg-red-100 text-red-700",
    "bg-orange-100 text-orange-700",
    "bg-amber-100 text-amber-700",
    "bg-green-100 text-green-700",
    "bg-teal-100 text-teal-700",
    "bg-sky-100 text-sky-700",
    "bg-blue-100 text-blue-700",
    "bg-indigo-100 text-indigo-700",
    "bg-pink-100 text-pink-700",
] as const

export function MemberAvatar({ name }: { name?: string }) {
    const letter = (name ?? "?")[0].toUpperCase()
    const colorClass =
        AVATAR_COLOR_CLASSES[letter.charCodeAt(0) % AVATAR_COLOR_CLASSES.length]

    return (
        <span className={`inline-grid h-4 w-4 shrink-0 -translate-x-0.5 place-items-center rounded-full bg-muted text-[9px] leading-none font-medium ${colorClass}`}>
            {letter}
        </span>

    )
}
