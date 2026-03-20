import type { Config } from 'tailwindcss'

export default {
  content: ['./src/**/*.{ts,tsx}', './index.html'],
  corePlugins: {
    preflight: false,
  },
} satisfies Config
