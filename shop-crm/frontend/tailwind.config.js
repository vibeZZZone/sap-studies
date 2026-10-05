/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Brand colours from the prototype: red for the till, green for stock and
        // confirmations.
        brand: {
          red: '#d32f2f',
          redDark: '#a92222',
          redLight: '#fdecea',
          green: '#2e7d32',
          greenDark: '#1b5e20',
          greenLight: '#e8f5e9',
        },
      },
      fontFamily: {
        receipt: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
}