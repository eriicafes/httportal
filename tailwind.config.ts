import { Config } from "tailwindcss";
import defaultTheme from "tailwindcss/defaultTheme";

export default {
  content: ["views/**/*.{html,ts}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Nunito Sans"', ...defaultTheme.fontFamily.sans],
      },
    },
  },
  plugins: [],
} satisfies Config;
