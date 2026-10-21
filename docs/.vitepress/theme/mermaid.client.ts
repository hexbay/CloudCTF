import mermaid from "mermaid";

async function renderMermaid() {
  if (typeof window === "undefined") return;

  // 检测当前主题（暗色或亮色）
  const isDark = document.documentElement.classList.contains("dark");
  const currentTheme = isDark ? "dark" : "default";

  // 每次都重新初始化以应用当前主题
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: "loose",
    theme: currentTheme,
  });

  console.log(`[Mermaid] Using theme: ${currentTheme}`);

  // 查找所有 mermaid 代码块容器和已渲染的图表
  const codeContainers = document.querySelectorAll<HTMLElement>(
    "div[class*='language-mermaid']"
  );
  const renderedContainers = document.querySelectorAll<HTMLElement>(
    ".vp-mermaid[data-mermaid-source]"
  );

  console.log(
    `[Mermaid] Found ${codeContainers.length} code blocks, ${renderedContainers.length} rendered diagrams`
  );

  // 重新渲染已存在的图表（主题切换时）
  for (const container of renderedContainers) {
    const source = container.getAttribute("data-mermaid-source");
    if (!source) continue;

    try {
      const id = `mermaid-${Math.random().toString(36).slice(2, 11)}`;
      const { svg } = await mermaid.render(id, source);
      container.innerHTML = svg;
      console.log(`[Mermaid] Re-rendered: ${id}`);
    } catch (error) {
      console.error("[Mermaid] Re-render error:", error);
    }
  }

  // 渲染新的代码块
  for (const container of codeContainers) {
    const codeElement = container.querySelector("code");
    if (!codeElement) continue;

    const source = codeElement.textContent ?? "";
    if (!source.trim()) continue;

    try {
      const id = `mermaid-${Math.random().toString(36).slice(2, 11)}`;
      const { svg } = await mermaid.render(id, source);

      // 创建新容器并替换
      const wrapper = document.createElement("div");
      wrapper.className = "vp-mermaid";
      wrapper.innerHTML = svg;
      wrapper.setAttribute("data-mermaid-source", source);

      container.replaceWith(wrapper);
      console.log(`[Mermaid] Rendered: ${id}`);
    } catch (error) {
      console.error("[Mermaid] Render error:", error);
      container.setAttribute("data-mermaid-error", "true");
    }
  }
}

export async function setupMermaidClient() {
  if (typeof window === "undefined") return;

  console.log("[Mermaid] Setup client called");

  // 延迟一点确保 DOM 完全加载
  await new Promise((resolve) => setTimeout(resolve, 100));
  await renderMermaid();
}
