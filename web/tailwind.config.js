/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#000000',
        canvas: '#000000',
        surface: '#050505',
        'surface-elevated': '#0a0a0a',
        'surface-highlight': '#141414',
        border: 'rgba(255, 255, 255, 0.22)',
        'border-strong': 'rgba(255, 255, 255, 0.48)',
        text: '#ffffff',
        muted: '#a3a3a3',
        'text-dim': '#737373',
        primary: {
          DEFAULT: '#ffffff',
          hover: '#e5e5e5',
          light: '#ffffff',
        },
        cyan: {
          DEFAULT: '#d4d4d4',
          hover: '#ffffff',
        },
        healthy: '#22c55e',
        warning: '#f59e0b',
        critical: '#ef4444',
        offline: '#737373',
      },
      fontFamily: {
        sans: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
        mono: ['JetBrains Mono', 'Fira Code', 'ui-monospace', 'monospace'],
      },
      boxShadow: {
        'glass': 'none',
        'glow-primary': '0 0 0 1px rgba(255, 255, 255, 0.45)',
        'glow-healthy': '0 0 0 1px rgba(34, 197, 94, 0.45)',
        'glow-warning': '0 0 0 1px rgba(245, 158, 11, 0.45)',
        'glow-critical': '0 0 0 1px rgba(239, 68, 68, 0.45)',
        'glow-cyan': '0 0 0 1px rgba(255, 255, 255, 0.35)',
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'mesh-glow': 'linear-gradient(to right, rgba(255, 255, 255, 0.03) 1px, transparent 1px), linear-gradient(to bottom, rgba(255, 255, 255, 0.03) 1px, transparent 1px)',
      },
      keyframes: {
        pulseGlow: {
          '0%, 100%': { opacity: 1, transform: 'scale(1)' },
          '50%': { opacity: 0.6, transform: 'scale(1.15)' },
        },
      },
      animation: {
        'pulse-glow': 'pulseGlow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
    },
  },
  plugins: [],
}
