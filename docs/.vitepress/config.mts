import { defineConfig } from "vitepress";

export default defineConfig({
  title: "CloudCTF",
  description: "CTF 竞赛平台 - 私有化部署文档",
  lang: "zh-CN",
  base: "/CloudCTF/",

  head: [["link", { rel: "icon", href: "/logo.svg", type: "image/svg+xml" }]],

  themeConfig: {
    logo: {
      light: "/logo.svg",
      dark: "/logo-light.svg",
    },

    nav: [
      { text: "指南", link: "/guide/introduction" },
      { text: "API", link: "/api/overview" },
      { text: "常见问题", link: "/guide/faq" },
    ],

    sidebar: {
      "/guide/": [
        {
          text: "开始使用",
          items: [
            { text: "项目介绍", link: "/guide/introduction" },
            { text: "功能特性", link: "/guide/features" },
            { text: "快速开始", link: "/guide/quick-start" },
            { text: "生产部署", link: "/guide/production-deploy" },
            { text: "Docker 部署", link: "/deployment/docker" },
          ],
        },

        {
          text: "用户指南",
          items: [
            { text: "管理员指南", link: "/guide/admin-guide" },
            { text: "参赛者指南", link: "/guide/user-guide" },
          ],
        },
        {
          text: "开发",
          items: [{ text: "技术架构", link: "/guide/architecture" }],
        },
        {
          text: "其他",
          items: [
            { text: "常见问题", link: "/guide/faq" },
            { text: "更新日志", link: "/guide/changelog" },
          ],
        },
      ],

      "/api/": [
        {
          text: "API 文档",
          items: [
            { text: "API 概览", link: "/api/overview" },
            { text: "认证授权", link: "/api/authentication" },
            { text: "用户 API", link: "/api/users" },
            { text: "题目 API", link: "/api/challenges" },
            { text: "比赛 API", link: "/api/competitions" },
            { text: "MCP 接入", link: "/api/mcp" },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: "github", link: "https://github.com/hexbay/CloudCTF" },
    ],

    footer: {
      message: "Released under the MIT License.",
      copyright: "Copyright © 2026-present CloudCTF contributors",
    },

    search: {
      provider: "local",
    },

    editLink: {
      pattern: "https://github.com/hexbay/CloudCTF/edit/main/docs/:path",
      text: "在 GitHub 上编辑此页",
    },

    lastUpdated: {
      text: "最后更新于",
      formatOptions: {
        dateStyle: "short",
        timeStyle: "medium",
      },
    },
  },
});
