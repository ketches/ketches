import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // `any` is acceptable in error callbacks (TanStack Query onError, axios errors, etc.)
      '@typescript-eslint/no-explicit-any': 'warn',

      // Allow unused vars when prefixed with `_` (intentionally ignored)
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          vars: 'all',
          varsIgnorePattern: '^_',
          args: 'after-used',
          argsIgnorePattern: '^_',
          caughtErrors: 'all',
          caughtErrorsIgnorePattern: '^_',
        },
      ],

      // Empty catch blocks are sometimes intentional; warn instead of blocking CI
      'no-empty': ['warn', { allowEmptyCatch: true }],

      // shadcn/ui and context files legitimately co-export variants/hooks with components
      'react-refresh/only-export-components': 'warn',

      // React Compiler memoization compatibility — informational, not a logic bug
      'react-hooks/incompatible-library': 'warn',

      // Calling setState in an effect is sometimes valid (e.g. syncing external prop)
      'react-hooks/set-state-in-effect': 'warn',

      // Dependency arrays are sometimes intentionally limited; warn to stay visible
      'react-hooks/exhaustive-deps': 'warn',

      // Prefer @ts-expect-error — keep as error, only one occurrence to fix
      '@typescript-eslint/ban-ts-comment': 'error',
    },
  },
])
