/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}'
  ],
  theme: {
    extend: {
      colors: {
        primary: '#3b82f6',
        'background-light': '#f8fafc',
        'sidebar-light': '#ffffff',
      },
      fontFamily: {
        display: ['Inter', 'Noto Sans SC', 'sans-serif'],
      },
      borderRadius: {
        DEFAULT: '12px',
      },
    },
  },
  plugins: [],
  corePlugins: {
    preflight: false,
  },
}
