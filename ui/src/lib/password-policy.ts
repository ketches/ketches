export const PASSWORD_POLICY_MESSAGE = "Use at least 8 characters with uppercase, lowercase, number, and special character"

export function isStrongPassword(password: string): boolean {
  if (password.length < 8) {
    return false
  }

  const hasUpper = /[A-Z]/.test(password)
  const hasLower = /[a-z]/.test(password)
  const hasDigit = /[0-9]/.test(password)
  const hasSpecial = /[^A-Za-z0-9]/.test(password)

  return hasUpper && hasLower && hasDigit && hasSpecial
}
