import DefaultTheme from "vitepress/theme";
import type { Theme } from "vitepress";
import { onMounted, watch, nextTick } from "vue";
import { useRoute } from "vitepress";
import { setupMermaidClient } from "./mermaid.client";

export default {
  extends: DefaultTheme,
  setup() {
    const route = useRoute();

    onMounted(() => {
      setupMermaidClient();
    });

    watch(
      () => route.path,
      () => {
        nextTick(() => {
          setupMermaidClient();
        });
      }
    );
  },
} satisfies Theme;
