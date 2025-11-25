/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            colors: {
                background: "#0f172a", // slate-950
                surface: "#1e293b",    // slate-800
                primary: "#5560FF",    // Custom Blue/Purple
                secondary: "#10b981",  // Emerald-500
                accent: "#06b6d4",     // Cyan-500
                border: "#334155",     // slate-700
            },
            fontFamily: {
                sans: ['Inter', 'sans-serif'],
            },
        },
    },
    plugins: [],
}
