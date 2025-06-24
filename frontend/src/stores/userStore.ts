import { getUserInfo } from "@/api/user";
import type { userModel } from "@/types/user";
import { defineStore } from "pinia";

export const useUserStore = defineStore('userStore', {
    state: () => ({
        user: null as userModel | null,
    }),
    actions: {
        async initUser() {
            const userID = localStorage.getItem('userID');
            if (userID) {
                // Fetch user info from the server or cache
                const resp = await getUserInfo(userID);
                this.user = resp.data;
            } else {
                this.clearUser();
            }
        },
        getUser() {
            return this.user;
        },
        setUser(newUser: userModel) {
            this.user = newUser;
            localStorage.setItem('userID', newUser.userID);
        },
        clearUser() {
            this.user = null;
            localStorage.removeItem('userID');
        },
    },
})