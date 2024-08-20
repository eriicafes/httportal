import { Alpine } from "alpinejs";
import htmx from "htmx.org";

declare global {
  interface Window {
    Alpine: Alpine;
    htmx: typeof htmx;
  }
}
