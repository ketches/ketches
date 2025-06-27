import { toast } from "vue-sonner";

export function copyToClipboard(text: string) {
    if (text) {
        navigator.clipboard.writeText(text);
        toast.success("复制成功", {
            position: "top-center",
            duration: 1500,
        });
    } else {
        toast.warning("没有内容可以复制", {
            position: "top-center",
        });
    }
}