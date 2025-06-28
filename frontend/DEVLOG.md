# Development Log

Initialize project.

```shell
yarn create vite vue-demo -- --template vue-ts && cd vue-demo
```

Add tailwindcss and vite plugin for tailwindcss.

```shell
yarn add tailwindcss @tailwindcss/vite
```

`src/style.css`:

> Replace all the content of `src/style.css` with the following:

```css
@import "tailwindcss";
```

`tsconfig.json`:
> Add the following to `tsconfig.json`:

```json
{
  // ...
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": [
        "./src/*"
      ]
    }
  }
  // ...
}
```

`tsconfig.app.json`:
> Add the following to `tsconfig.app.json`:

```json
{
  "compilerOptions": {
    // ...
    "baseUrl": ".",
    "paths": {
      "@/*": [
        "./src/*"
      ]
    }
    // ...
  }
}
```

Add node types for TypeScript.

```shell
yarn add -D @types/node
```

`vite.config.ts`:

> Add the following to `vite.config.ts`:

```ts
import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```

Add shadcn and initialize it.

```shell
yarn add shadcn-vue@latest
yarn shadcn-vue init
```

Design the layout and add the header.

```shell
yarn shadcn-vue add Login04
yarn shadcn-vue add Sidebar07
```
