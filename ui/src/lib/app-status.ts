export type AppStatus =
    | "undeployed"
    | "starting"
    | "running"
    | "stopped"
    | "stopping"
    | "updating"
    | "abnormal"
    | "completed"
    | "debugging"
    | "unknown";

export const getAppStatusColor = (status: string): "gray" | "blue" | "green" | "sky" | "amber" | "red" | "yellow" | "orange" => {
    switch (status.toLowerCase()) {
        case "running":
        case "completed":
        case "succeeded":
            return "green";
        case "starting":
        case "updating":
        case "pending":
            return "blue";
        case "debugging":
            return "amber";
        case "stopping":
            return "orange";
        case "failed":
        case "abnormal":
        case "error":
            return "red";
        case "undeployed":
        case "stopped":
        case "unknown":
        default:
            return "gray";
    }
};
