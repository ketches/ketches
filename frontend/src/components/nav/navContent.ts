import { Grid2X2, Package, Telescope, UsersRound } from "lucide-vue-next";

export const userNavContents = [
    {
        title: "总览",
        icon: Telescope,
        route: { name: "user-overview" },
        isActive: true,
    },
    {
        title: "环境",
        icon: Grid2X2,
        route: { name: "env" },
    },
    {
        title: "应用",
        icon: Package,
        route: { name: "app" },
    },
    {
        title: "成员",
        icon: UsersRound,
        route: { name: "member" },
    },
];