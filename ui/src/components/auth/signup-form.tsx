import { useQuery } from "@tanstack/react-query"
import { zodResolver } from "@hookform/resolvers/zod"
import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { Link, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import * as z from "zod"
import { isAxiosError } from "axios"

import { authApi, type SignUpRequest, type SignUpVerificationCodeRequest } from "@/api/auth"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { PASSWORD_POLICY_MESSAGE, isStrongPassword } from "@/lib/password-policy"
import { cn } from "@/lib/utils"

const signupSchema = z.object({
  fullname: z.string().min(1, "Full name is required"),
  username: z.string().min(3, "Username must be at least 3 characters"),
  email: z.string().email("Invalid email address"),
  password: z.string().refine(isStrongPassword, PASSWORD_POLICY_MESSAGE),
  verificationCode: z.string().length(6, "Enter the 6-digit verification code"),
  confirmPassword: z.string().min(1, "Please confirm your password"),
}).refine((data) => data.password === data.confirmPassword, {
  message: "Passwords don't match",
  path: ["confirmPassword"],
})

type SignupFormValues = z.infer<typeof signupSchema>

export function SignupForm({
  className,
  ...props
}: React.ComponentProps<"form">) {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [resendAfterSeconds, setResendAfterSeconds] = useState(0)

  const signUpConfigQuery = useQuery({
    queryKey: ["sign-up-config"],
    queryFn: authApi.getSignUpConfig,
  })

  const {
    register,
    getValues,
    handleSubmit,
    formState: { errors },
  } = useForm<SignupFormValues>({
    resolver: zodResolver(signupSchema),
  })

  useEffect(() => {
    if (resendAfterSeconds <= 0) {
      return
    }

    const timer = window.setTimeout(() => {
      setResendAfterSeconds((current) => Math.max(0, current - 1))
    }, 1000)

    return () => window.clearTimeout(timer)
  }, [resendAfterSeconds])

  const handleSendVerificationCode = async () => {
    const email = getValues("email")
    if (!email) {
      setError("Enter your email address before requesting a verification code")
      return
    }

    setIsSendingCode(true)
    setError(null)
    try {
      const payload: SignUpVerificationCodeRequest = { email }
      const response = await authApi.sendSignUpVerificationCode(payload)
      setResendAfterSeconds(response.resend_after_seconds)
      toast.success("Verification code sent", {
        description: "Check your email inbox for the 6-digit code.",
      })
    } catch (err: unknown) {
      const errMsg = isAxiosError<{ error?: string }>(err)
        ? err.response?.data?.error || "Failed to send verification code"
        : "Failed to send verification code"
      setError(errMsg)
      toast.error("Verification Failed", {
        description: errMsg,
      })
    } finally {
      setIsSendingCode(false)
    }
  }

  const onSubmit = async (data: SignupFormValues) => {
    setIsLoading(true)
    setError(null)
    try {
      const payload: SignUpRequest = {
        fullname: data.fullname,
        username: data.username,
        email: data.email,
        password: data.password,
        verification_code: data.verificationCode,
      }
      await authApi.signUp(payload)
      toast.success("Account created", {
        description: "You can now sign in with your credentials.",
      })
      navigate("/login")
    } catch (err: unknown) {
      const errMsg = isAxiosError<{ error?: string }>(err)
        ? err.response?.data?.error || "Failed to create account"
        : "Failed to create account"
      setError(errMsg)
      toast.error("Registration Failed", {
        description: errMsg,
      })
    } finally {
      setIsLoading(false)
    }
  }

  if (signUpConfigQuery.isLoading) {
    return (
      <div className={cn("text-center text-sm text-muted-foreground", className)}>
        Loading registration settings...
      </div>
    )
  }

  if (signUpConfigQuery.data && !signUpConfigQuery.data.enabled) {
    return (
      <div className={cn("space-y-3 text-center", className)}>
        <h1 className="text-2xl font-bold">Public registration is disabled</h1>
        <p className="text-sm text-muted-foreground">
          Contact an administrator if you need an account.
        </p>
        <div className="text-xs">
          Already have an account?{" "}
          <Link to="/login" className="underline underline-offset-4 font-medium">
            Sign in
          </Link>
        </div>
      </div>
    )
  }

  return (
    <form
      className={cn("flex flex-col gap-6", className)}
      onSubmit={handleSubmit(onSubmit)}
      {...props}
    >
      <FieldGroup>
        <div className="flex flex-col items-center gap-1 text-center">
          <h1 className="text-2xl font-bold">Create your account</h1>
          <p className="text-muted-foreground text-sm text-balance">
            Verify your email, then complete the registration form.
          </p>
        </div>
        {error && (
          <div className="text-sm font-medium text-destructive text-center">
            {error}
          </div>
        )}
        <Field>
          <FieldLabel htmlFor="fullname">Full Name</FieldLabel>
          <FieldContent>
            <Input
              id="fullname"
              placeholder="John Doe"
              {...register("fullname")}
            />
          </FieldContent>
          {errors.fullname && (
            <FieldError>{errors.fullname.message}</FieldError>
          )}
        </Field>
        <Field>
          <FieldLabel htmlFor="username">Username</FieldLabel>
          <FieldContent>
            <Input
              id="username"
              placeholder="username"
              {...register("username")}
            />
          </FieldContent>
          {errors.username && (
            <FieldError>{errors.username.message}</FieldError>
          )}
        </Field>
        <Field>
          <FieldLabel htmlFor="email">Email</FieldLabel>
          <FieldContent className="flex gap-2">
            <Input
              id="email"
              type="email"
              placeholder="m@example.com"
              {...register("email")}
            />
            <Button
              type="button"
              variant="outline"
              disabled={isSendingCode || resendAfterSeconds > 0}
              onClick={handleSendVerificationCode}
            >
              {isSendingCode ? "Sending..." : resendAfterSeconds > 0 ? `Resend in ${resendAfterSeconds}s` : "Send Code"}
            </Button>
          </FieldContent>
          <FieldDescription>
            Verification codes stay valid for 300 seconds.
          </FieldDescription>
          {errors.email && (
            <FieldError>{errors.email.message}</FieldError>
          )}
        </Field>
        <Field>
          <FieldLabel htmlFor="verificationCode">Verification Code</FieldLabel>
          <FieldContent>
            <Input
              id="verificationCode"
              inputMode="numeric"
              maxLength={6}
              placeholder="123456"
              {...register("verificationCode")}
            />
          </FieldContent>
          {errors.verificationCode && (
            <FieldError>{errors.verificationCode.message}</FieldError>
          )}
        </Field>
        <Field>
          <FieldLabel htmlFor="password">Password</FieldLabel>
          <FieldContent>
            <Input
              id="password"
              type="password"
              autoComplete="new-password"
              {...register("password")}
            />
          </FieldContent>
          <FieldDescription>{PASSWORD_POLICY_MESSAGE}</FieldDescription>
          {errors.password && (
            <FieldError>{errors.password.message}</FieldError>
          )}
        </Field>
        <Field>
          <FieldLabel htmlFor="confirmPassword">Confirm Password</FieldLabel>
          <FieldContent>
            <Input
              id="confirmPassword"
              type="password"
              {...register("confirmPassword")}
            />
          </FieldContent>
          {errors.confirmPassword && (
            <FieldError>{errors.confirmPassword.message}</FieldError>
          )}
        </Field>
        <Field>
          <Button type="submit" disabled={isLoading}>
            {isLoading ? "Creating Account..." : "Create Account"}
          </Button>
        </Field>
        <Field>
          <div className="text-center text-xs">
            Already have an account?{" "}
            <Link to="/login" className="underline underline-offset-4 font-medium">
              Sign in
            </Link>
          </div>
        </Field>
      </FieldGroup>
    </form>
  )
}
