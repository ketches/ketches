import { authApi, type SignInRequest } from "@/api/auth";
import { platformUpdateApi } from "@/api/platform-update";
import { Button } from "@/components/ui/button";
import {
    Field,
    FieldContent,
    FieldError,
    FieldGroup,
    FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { clearManualLogoutMarker, getPostLoginTarget } from "@/lib/auth-redirect";
import { markSessionRefreshed } from "@/lib/auth-session";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { isAxiosError } from "axios";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import * as z from "zod";

const loginSchema = z.object({
    username: z.string().min(1, "Username is required"),
    password: z.string().min(1, "Password is required"),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export function LoginForm({
    className,
    ...props
}: React.ComponentProps<"form">) {
    const navigate = useNavigate();
    const location = useLocation();
    const setAuth = useAuthStore((state) => state.setAuth);
    const queryClient = useQueryClient();
    const [isLoading, setIsLoading] = useState(false);
    const [defaultPasswordNotice, setDefaultPasswordNotice] = useState<
        string | null
    >(null);

    const {
        register,
        handleSubmit,
        formState: { errors },
    } = useForm<LoginFormValues>({
        resolver: zodResolver(loginSchema),
    });

    useEffect(() => {
        if (!defaultPasswordNotice) {
            return;
        }
        toast.warning("Security Notice", {
            description: defaultPasswordNotice,
        });
    }, [defaultPasswordNotice]);

    const onSubmit = async (data: LoginFormValues) => {
        setIsLoading(true);
        setDefaultPasswordNotice(null);
        try {
            const payload: SignInRequest = data;
            const response = await authApi.signIn(payload);
            queryClient.clear();
            setAuth(response.user);
            markSessionRefreshed();
            setDefaultPasswordNotice(response.default_password_notice || null);
            if (response.user.role === "admin") {
                void platformUpdateApi.check({ mode: "auto" }).catch(() => undefined);
            }
            toast.success("Login successful", {
                description: `Welcome back, ${response.user.username}!`,
            });
            clearManualLogoutMarker();
            navigate(getPostLoginTarget(location.search), { replace: true });
        } catch (err: unknown) {
            const errMsg = isAxiosError<{ error?: string }>(err)
                ? err.response?.data?.error || "Invalid username or password"
                : "Invalid username or password";
            toast.error("Login Failed", {
                description: errMsg,
            });
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <form
            className={cn("flex flex-col gap-6", className)}
            onSubmit={handleSubmit(onSubmit)}
            {...props}
        >
            <FieldGroup>
                <div className="flex flex-col items-center gap-1 text-center">
                    <h1 className="text-2xl font-bold">
                        Login to your account
                    </h1>
                    <p className="text-muted-foreground text-sm text-balance">
                        Enter your username below to login to your account
                    </p>
                </div>
                <Field>
                    <FieldLabel htmlFor="username">Username *</FieldLabel>
                    <FieldContent>
                        <Input
                            id="username"
                            {...register("username")}
                        />
                    </FieldContent>
                    {errors.username && (
                        <FieldError>{errors.username.message}</FieldError>
                    )}
                </Field>
                <Field>
                    <div className="flex items-center">
                        <FieldLabel htmlFor="password">Password *</FieldLabel>
                        <a
                            href="#"
                            className="ml-auto text-xs underline-offset-4 hover:underline"
                            tabIndex={-1}
                        >
                            Forgot your password?
                        </a>
                    </div>
                    <FieldContent>
                        <Input
                            id="password"
                            type="password"
                            autoComplete="current-password"
                            {...register("password")}
                        />
                    </FieldContent>
                    {errors.password && (
                        <FieldError>{errors.password.message}</FieldError>
                    )}
                </Field>
                <Field>
                    <Button type="submit" disabled={isLoading}>
                        {isLoading ? "Logging in..." : "Login"}
                    </Button>
                </Field>
                {/* <FieldSeparator>Or continue with</FieldSeparator> */}
                <Field>
                    {/* <Button variant="outline" type="button" disabled={isLoading} className="gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" className="h-4 w-4">
              <path
                d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"
                fill="currentColor"
              />
            </svg>
            Login with GitHub
          </Button> */}
                    <div className="text-center text-xs">
                        Don&apos;t have an account?{" "}
                        <Link
                            to="/signup"
                            className="underline underline-offset-4 font-medium"
                        >
                            Sign up
                        </Link>
                    </div>
                </Field>
            </FieldGroup>
        </form>
    );
}
