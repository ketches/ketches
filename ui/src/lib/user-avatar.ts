const USER_AVATAR_COLOR_CLASSES = [
  "bg-red-500/10 text-red-600",
  "bg-orange-500/10 text-orange-600",
  "bg-amber-500/10 text-amber-600",
  "bg-green-500/10 text-green-600",
  "bg-teal-500/10 text-teal-600",
  "bg-sky-500/10 text-sky-600",
  "bg-blue-500/10 text-blue-600",
  "bg-indigo-500/10 text-indigo-600",
  "bg-pink-500/10 text-pink-600",
] as const

export function getUserAvatarInitial(name?: string): string {
  const trimmed = name?.trim()
  return (trimmed?.[0] ?? "?").toUpperCase()
}

export function getUserAvatarColorClass(name?: string): string {
  const initial = getUserAvatarInitial(name)
  const codePoint = initial.codePointAt(0) ?? 0

  return USER_AVATAR_COLOR_CLASSES[codePoint % USER_AVATAR_COLOR_CLASSES.length]
}
