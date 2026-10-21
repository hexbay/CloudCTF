# Ocean CTF 文档构建脚本 (PowerShell)

param(
    [Parameter(Position=0)]
    [string]$Command = "help",
    
    [Parameter()]
    [string]$Server = ""
)

function Show-Help {
    Write-Host "Ocean CTF 文档构建工具" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "可用命令:" -ForegroundColor Yellow
    Write-Host "  .\build.ps1 install    - 安装依赖"
    Write-Host "  .\build.ps1 dev        - 启动开发服务器"
    Write-Host "  .\build.ps1 build      - 构建生产版本"
    Write-Host "  .\build.ps1 deploy     - 部署到服务器"
    Write-Host "  .\build.ps1 clean      - 清理构建文件"
    Write-Host ""
}

function Install-Dependencies {
    Write-Host "安装依赖..." -ForegroundColor Green
    pnpm install
}

function Start-Dev {
    Write-Host "启动开发服务器..." -ForegroundColor Green
    pnpm docs:dev
}

function Build-Docs {
    Write-Host "构建生产版本..." -ForegroundColor Green
    pnpm docs:build
}

function Deploy-Docs {
    if (-not $Server) {
        Write-Host "错误: 请指定服务器地址" -ForegroundColor Red
        Write-Host "用法: .\build.ps1 deploy -Server user@host" -ForegroundColor Yellow
        exit 1
    }
    
    Write-Host "开始部署..." -ForegroundColor Green
    Build-Docs
    
    Write-Host "复制文件到服务器..." -ForegroundColor Green
    scp -r .vitepress/dist/* ${Server}:/var/www/ctf-docs/
    
    Write-Host "部署完成!" -ForegroundColor Green
}

function Clean-Build {
    Write-Host "清理构建文件..." -ForegroundColor Green
    
    if (Test-Path ".vitepress/dist") {
        Remove-Item -Recurse -Force ".vitepress/dist"
    }
    
    if (Test-Path ".vitepress/cache") {
        Remove-Item -Recurse -Force ".vitepress/cache"
    }
    
    Write-Host "清理完成!" -ForegroundColor Green
}

# 执行命令
switch ($Command.ToLower()) {
    "install" { Install-Dependencies }
    "dev" { Start-Dev }
    "build" { Build-Docs }
    "deploy" { Deploy-Docs }
    "clean" { Clean-Build }
    "help" { Show-Help }
    default {
        Write-Host "未知命令: $Command" -ForegroundColor Red
        Show-Help
        exit 1
    }
}
