import { createPinia } from 'pinia'
import { createApp } from 'vue'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import App from './App.vue'
import './lib/validators'
import './style.css'

const routes: RouteRecordRaw[] = [
    {
        name: "home",
        path: "/",
        component: () => import('@/components/Home.vue'),
    },
    {
        name: "sign-in",
        path: "/sign-in", component: () => import('@/components/user/SignIn.vue'),
        beforeEnter: (_to: any, _from: any, next: any) => {
            if (localStorage.getItem('userID')) {
                next({ name: 'home' });
            } else {
                next();
            }
        }
    },
    {
        name: "sign-up",
        path: "/sign-up", component: () => import('@/components/user/SignUp.vue'),
    },
    {
        name: "admin",
        path: "/admin",
        component: () => import('@/components/Home.vue'),
        redirect: { name: 'adminOverview' },
        children: [
            { name: 'adminOverview', path: "overview", component: () => import('@/components/admin/Overview.vue') },
            { name: 'cluster', path: "cluster", component: () => import('@/components/cluster/ClusterManager.vue') },
            { name: "clusterPage", path: "cluster/:id", component: () => import('@/components/cluster/ClusterPage.vue') },
            { name: "node", path: "cluster/:id/node", component: () => import('@/components/cluster/node/NodeManager.vue') },
            { name: "nodePage", path: "cluster/:id/node/:nodeId", component: () => import('@/components/cluster/node/NodePage.vue') },
        ]
    },
    {
        name: "user",
        path: "/",
        component: () => import('@/components/Home.vue'),
        redirect: { name: 'userOverview' },
        children: [
            { name: 'userOverview', path: "overview", component: () => import('@/components/user/Overview.vue') },
            { name: 'env', path: "env", component: () => import('@/components/env/EnvManager.vue') },
            { name: "envPage", path: "env/:id", component: () => import('@/components/env/EnvPage.vue') },
            { name: 'app', path: "app", component: () => import('@/components/app/AppManager.vue') },
            { name: "appPage", path: "app/:id", component: () => import('@/components/app/AppPage.vue') },
            { name: 'member', path: "member", component: () => import('@/components/project/MemberManager.vue') },
        ]
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes
})


// router.beforeEach(async (to, _from, next) => {
//     if (to.path.startsWith('/console')) {
//         if (!localStorage.getItem('userID')) {
//             // const redirectPath = '/sign-in?redirectUrl=' + encodeURIComponent(to.fullPath);
//             // next(redirectPath);
//         } else {
//             next();
//         }
//     } else {
//         next();
//     }
// })

createApp(App).use(createPinia()).use(router).mount('#app')
