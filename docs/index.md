---
layout: home

hero:
  name: CloudCTF
  text: CTF 竞赛与漏洞训练平台
  tagline: 单二进制交付的企业级私有化部署方案
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/quick-start
    - theme: alt
      text: 在线预览
      link: http://cloud-ctf.lostpeach.cn/
    - theme: alt
      text: 在线文档
      link: http://ctf-docs.lostpeach.cn/
    - theme: alt
      text: 查看 GitHub
      link: https://github.com/hexbay/CloudCTF

features:
  - icon:
      src: /icons/trophy.svg
      alt: 多种赛制
    title: 多种赛制支持
    details: 支持个人赛、团队赛、理论赛等多种比赛模式，可同时举办多场比赛，满足不同场景需求

  - icon:
      src: /icons/chart.svg
      alt: 排行榜
    title: 实时排行榜
    details: 实时更新的赛事排行榜，支持多维度统计分析，首杀标记，让比赛更有竞争性

  - icon:
      src: /icons/code.svg
      alt: 练习平台
    title: CTF 练习平台
    details: 提供日常练习环境，支持多种题目类型（Web、Pwn、Reverse、Crypto、Misc），随时随地提升技能

  - icon:
      src: /icons/shield.svg
      alt: 漏洞复现
    title: 漏洞复现环境
    details: 集成 Vulhub 漏洞库，一键部署真实漏洞环境，支持 CVE/CNVD 漏洞复现，实战化培训

  - icon:
      src: /icons/docker.svg
      alt: 虚拟化
    title: 虚拟化支持
    details: 支持 Docker 运行时与模板化环境管理，兼容多种虚拟化资源，灵活承载动态靶机与漏洞环境

  - icon:
      src: /icons/target.svg
      alt: 动态靶场
    title: 动态靶场
    details: 自动化容器管理，动态 Flag 生成，每个用户独立环境，防止作弊，提升培训效果

---

## 为什么选择 CloudCTF？

CloudCTF 是专为企业和教育机构打造的 CTF 竞赛与安全培训平台，提供完整的私有化部署方案。

### 核心优势

- **🎯 功能完整**：覆盖比赛、练习、培训、考核全场景，一站式解决方案
- **📚 内置题库**：集成 Vulhub 漏洞库，提供数百个真实漏洞环境，开箱即用
- **📦 快速部署**：单二进制交付，前端内嵌，无需单独部署前端
- **💰 成本可控**：私有化部署，无需按用户付费，一次投入长期使用
- **🔧 灵活定制**：支持多种赛制和题型，满足不同培训需求
- **⚡ 性能强劲**：基于 Go 服务架构与灵活存储能力，轻松支持高并发竞赛场景
- **🛡️ 安全可靠**：容器隔离、权限控制、防作弊机制，保障比赛公平

### 快速开始

如果想先体验界面和基础流程，可以访问在线预览环境：

- [http://cloud-ctf.lostpeach.cn/](http://cloud-ctf.lostpeach.cn/)

```bash
# 1. 下载服务二进制
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64
chmod +x cloudctf-linux-amd64

# 2. 直接启动
./cloudctf-linux-amd64

# 3. 访问
# 用户端: http://localhost:8005/
# 管理端: http://localhost:8005/admin
```

详细部署步骤请查看 [快速开始](/guide/quick-start)，生产环境部署请查看 [生产部署](/guide/production-deploy)。

### 技术支持

如有问题，欢迎通过以下方式联系我们：

- 📧 Email: 5829662@qq.com
- 💬 GitHub Issues: [提交问题](https://github.com/hexbay/CloudCTF/issues)
- 📖 文档: [查看完整文档](/guide/introduction)
