import { zodResolver } from "@hookform/resolvers/zod"
import { useQuery } from "@tanstack/react-query"
import { isAxiosError } from "axios"
import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { Link, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import * as z from "zod"

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
import { Send } from "lucide-react"
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "../ui/input-group"

const PROTECTED_FIELDS_TRANSITION_MS = 280

type SignupFormValues = {
  fullname: string
  username: string
  email: string
  password: string
  verificationCode: string
  confirmPassword: string
}

function createSignupSchema(emailVerificationRequired: boolean) {
  return z.object({
    fullname: z.string().min(1, "Full name is required"),
    username: z.string().min(3, "Username must be at least 3 characters"),
    email: z.string().email("Invalid email address"),
    password: z.string().refine(isStrongPassword, PASSWORD_POLICY_MESSAGE),
    verificationCode: z.string(),
    confirmPassword: z.string().min(1, "Please confirm your password"),
  }).superRefine((data, ctx) => {
    if (emailVerificationRequired && data.verificationCode.length !== 6) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Enter the 6-digit verification code",
        path: ["verificationCode"],
      })
    }
    if (data.password !== data.confirmPassword) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Passwords don't match",
        path: ["confirmPassword"],
      })
    }
  })
}

export function SignupForm({
  className,
  ...props
}: React.ComponentProps<"form">) {
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)
  const [isSendingCode, setIsSendingCode] = useState(false)
  const [hasSentVerificationCode, setHasSentVerificationCode] = useState(false)
  const [resendAfterSeconds, setResendAfterSeconds] = useState(0)

  const signUpConfigQuery = useQuery({
    queryKey: ["sign-up-config"],
    queryFn: authApi.getSignUpConfig,
  })
  const emailVerificationRequired = signUpConfigQuery.data?.email_verification_required ?? true
  const [shouldRenderProtectedFields, setShouldRenderProtectedFields] = useState(!emailVerificationRequired)
  const [isProtectedFieldsVisible, setIsProtectedFieldsVisible] = useState(!emailVerificationRequired)

  const {
    register,
    getValues,
    handleSubmit,
    formState: { errors },
  } = useForm<SignupFormValues>({
    resolver: zodResolver(createSignupSchema(emailVerificationRequired)),
    defaultValues: {
      verificationCode: "",
    },
  })
  const emailField = register("email")
  const shouldShowProtectedFields = !emailVerificationRequired || hasSentVerificationCode
  const isSubmitDisabled = isLoading || (emailVerificationRequired && !hasSentVerificationCode)

  useEffect(() => {
    if (resendAfterSeconds <= 0) {
      return
    }

    const timer = window.setTimeout(() => {
      setResendAfterSeconds((current) => Math.max(0, current - 1))
    }, 1000)

    return () => window.clearTimeout(timer)
  }, [resendAfterSeconds])

  useEffect(() => {
    if (!emailVerificationRequired) {
      setShouldRenderProtectedFields(true)
      setIsProtectedFieldsVisible(true)
      return
    }

    if (hasSentVerificationCode) {
      setShouldRenderProtectedFields(true)
      const frame = window.requestAnimationFrame(() => {
        setIsProtectedFieldsVisible(true)
      })
      return () => window.cancelAnimationFrame(frame)
    }

    setIsProtectedFieldsVisible(false)
    const timer = window.setTimeout(() => {
      setShouldRenderProtectedFields(false)
    }, PROTECTED_FIELDS_TRANSITION_MS)

    return () => window.clearTimeout(timer)
  }, [emailVerificationRequired, hasSentVerificationCode])

  const handleSendVerificationCode = async () => {
    const email = getValues("email")
    if (!email) {
      toast.error("Verification Failed", {
        description: "Enter your email address before requesting a verification code",
      })
      return
    }

    setIsSendingCode(true)
    try {
      const payload: SignUpVerificationCodeRequest = { email }
      const response = await authApi.sendSignUpVerificationCode(payload)
      setHasSentVerificationCode(true)
      setResendAfterSeconds(response.resend_after_seconds)
      toast.success("Verification code sent", {
        description: "Check your email inbox for the 6-digit code.",
      })
    } catch (err: unknown) {
      const errMsg = isAxiosError<{ error?: string }>(err)
        ? err.response?.data?.error || "Failed to send verification code"
        : "Failed to send verification code"
      toast.error("Verification Failed", {
        description: errMsg,
      })
    } finally {
      setIsSendingCode(false)
    }
  }

  const handleEmailChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    emailField.onChange(event)
    if (!hasSentVerificationCode) {
      return
    }

    setHasSentVerificationCode(false)
    setResendAfterSeconds(0)
  }

  const onSubmit = async (data: SignupFormValues) => {
    setIsLoading(true)
    try {
      const payload: SignUpRequest = {
        fullname: data.fullname,
        username: data.username,
        email: data.email,
        password: data.password,
        ...(emailVerificationRequired ? { verification_code: data.verificationCode ?? "" } : {}),
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
      toast.error("Registration Failed", {
        description: errMsg,
      })
    } finally {
      setIsLoading(false)
    }
  }

  const protectedFields = (
    <>
      {emailVerificationRequired && (
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
      )}
      <Field>
        <FieldLabel htmlFor="password">Password *</FieldLabel>
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
        <FieldLabel htmlFor="confirmPassword">Confirm Password *</FieldLabel>
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
    </>
  )

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
            {emailVerificationRequired
              ? shouldShowProtectedFields
                ? "Enter the verification code and finish setting your password."
                : "Enter your email address to receive a verification code."
              : "Complete the registration form."}
          </p>
        </div>
        <Field>
          <FieldLabel htmlFor="fullname">Full Name *</FieldLabel>
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
          <FieldLabel htmlFor="username">Username *</FieldLabel>
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
          <FieldLabel htmlFor="email">Email *</FieldLabel>
          <FieldContent className="flex gap-2">
            <InputGroup>
              <InputGroupInput
                id="email"
                type="email"
                placeholder="m@example.com"
                {...emailField}
                onChange={handleEmailChange}
              />
              <InputGroupAddon align="inline-end">
                {emailVerificationRequired && (
                  <InputGroupButton
                    type="button"
                    variant="secondary"
                    disabled={isSendingCode || resendAfterSeconds > 0}
                    onClick={handleSendVerificationCode}
                  >
                    <Send className="h-4 w-4 mr-1" />
                    {isSendingCode
                      ? "Sending..."
                      : resendAfterSeconds > 0
                        ? `Resend in ${resendAfterSeconds}s`
                        : "Send Code"}
                  </InputGroupButton>
                )}
              </InputGroupAddon>
            </InputGroup>
          </FieldContent>
          {emailVerificationRequired && (
            <FieldDescription>
              {shouldShowProtectedFields
                ? "Verification codes stay valid for 300 seconds."
                : "Send a verification code to continue registration."}
            </FieldDescription>
          )}
          {errors.email && (
            <FieldError>{errors.email.message}</FieldError>
          )}
        </Field>
        {!emailVerificationRequired && protectedFields}
        {emailVerificationRequired && shouldRenderProtectedFields && (
          <div
            aria-hidden={!isProtectedFieldsVisible}
            className={cn(
              "grid transition-[grid-template-rows,opacity] duration-300 ease-out motion-reduce:transition-none",
              isProtectedFieldsVisible ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
            )}
          >
            <div className="overflow-hidden">
              <FieldGroup
                className={cn(
                  "pt-1 transition-[transform,opacity] duration-300 ease-out motion-reduce:transition-none",
                  isProtectedFieldsVisible ? "translate-y-0 opacity-100" : "-translate-y-2 opacity-0 pointer-events-none",
                )}
              >
                {protectedFields}
              </FieldGroup>
            </div>
          </div>
        )}
        <Field>
          <Button type="submit" disabled={isSubmitDisabled}>
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
