import { UserRoundCog, UserRoundPen, UserRoundSearch } from "lucide-vue-next";

export const projectRoleRefs =
{
    'owner': {
        label: '所有者',
        icon: UserRoundCog,
        style: 'text-green-500'
    },
    'developer': {
        label: '开发者',
        icon: UserRoundPen,
        style: 'text-blue-500'
    },
    'viewer': {
        label: '观察者',
        icon: UserRoundSearch,
        style: 'text-gray-500'
    }
}
