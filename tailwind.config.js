/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/templates/**/*.tmpl", "./static/sites/**/*.html"],
  theme: {
    extend: {},
  },
  plugins: [require("@tailwindcss/typography")],
};
