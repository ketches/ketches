/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockNavigate,
  mockGetSignUpConfig,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockGetSignUpConfig: vi.fn(),
}))

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockGetSignUpConfig(),
    isLoading: false,
  }),
}))

vi.mock("@/api/auth", () => ({
  authApi: {
    getSignUpConfig: () => mockGetSignUpConfig(),
    sendSignUpVerificationCode: vi.fn(),
    signUp: vi.fn(),
  },
}))

vi.mock("@hookform/resolvers/zod", () => ({
  zodResolver: () => undefined,
}))

vi.mock("react-hook-form", () => ({
  useForm: () => ({
    register: (name: string) => ({
      name,
      onChange: () => undefined,
      ref: () => undefined,
    }),
    getValues: () => "",
    handleSubmit: (submit: () => Promise<void>) => async (event?: Event) => {
      event?.preventDefault?.()
      await submit()
    },
    formState: {
      errors: {},
    },
  }),
}))

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => <a href={to}>{children}</a>,
  useNavigate: () => mockNavigate,
}))

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

import { SignupForm } from "./signup-form"

describe("SignupForm", () => {
  beforeEach(() => {
    mockGetSignUpConfig.mockReset()
    mockNavigate.mockReset()
  })

  it("hides email verification controls when verification is not required", () => {
    mockGetSignUpConfig.mockReturnValue({
      enabled: true,
      email_verification_required: false,
    })

    render(<SignupForm />)

    expect(screen.queryByRole("button", { name: /send code/i })).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/verification code/i)).not.toBeInTheDocument()
    expect(screen.getByText(/complete the registration form/i)).toBeInTheDocument()
  })
})
