/** @vitest-environment jsdom */
import "@testing-library/jest-dom/vitest"
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const {
  mockNavigate,
  mockGetSignUpConfig,
  mockSendSignUpVerificationCode,
  mockGetValues,
} = vi.hoisted(() => ({
  mockNavigate: vi.fn(),
  mockGetSignUpConfig: vi.fn(),
  mockSendSignUpVerificationCode: vi.fn(),
  mockGetValues: vi.fn(),
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
    sendSignUpVerificationCode: mockSendSignUpVerificationCode,
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
    getValues: (name?: string) => mockGetValues(name),
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
  afterEach(() => {
    cleanup()
  })

  beforeEach(() => {
    mockGetSignUpConfig.mockReset()
    mockNavigate.mockReset()
    mockSendSignUpVerificationCode.mockReset()
    mockGetValues.mockReset()
  })

  it("hides email verification controls when verification is not required", () => {
    mockGetSignUpConfig.mockReturnValue({
      enabled: true,
      email_verification_required: false,
    })

    render(<SignupForm />)

    expect(screen.queryByRole("button", { name: /send code/i })).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/verification code/i)).not.toBeInTheDocument()
    expect(screen.getByLabelText(/^password/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^confirm password/i)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /create account/i })).toBeInTheDocument()
    expect(screen.getByText(/complete the registration form/i)).toBeInTheDocument()
  })

  it("shows verification and password fields only after the code is sent", async () => {
    mockGetSignUpConfig.mockReturnValue({
      enabled: true,
      email_verification_required: true,
    })
    mockGetValues.mockReturnValue("user@example.com")
    mockSendSignUpVerificationCode.mockResolvedValue({
      expires_in_seconds: 300,
      resend_after_seconds: 60,
    })

    render(<SignupForm />)

    expect(screen.getByRole("button", { name: /send code/i })).toBeInTheDocument()
    expect(screen.queryByLabelText(/verification code/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/^password/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/confirm password/i)).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /create account/i })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: /send code/i }))

    await waitFor(() => {
      expect(mockSendSignUpVerificationCode).toHaveBeenCalledWith({
        email: "user@example.com",
      })
    })

    expect(screen.getByLabelText(/verification code/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /create account/i })).toBeInTheDocument()
  })
})
