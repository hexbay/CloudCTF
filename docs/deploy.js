#!/usr/bin/env node

/**
 * Ocean CTF 文档部署脚本
 * 
 * 使用方法:
 * 1. 设置环境变量: 
 *    export DEPLOY_SERVER=user@host
 *    export DEPLOY_PATH=/var/www/ctf-docs
 * 
 * 2. 运行部署:
 *    pnpm deploy
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// 配置
const config = {
  server: process.env.DEPLOY_SERVER || '',
  remotePath: process.env.DEPLOY_PATH || '/var/www/ctf-docs',
  distDir: path.join(__dirname, '.vitepress/dist'),
};

// 颜色输出
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  cyan: '\x1b[36m',
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function error(message) {
  log(`❌ ${message}`, 'red');
  process.exit(1);
}

function success(message) {
  log(`✅ ${message}`, 'green');
}

function info(message) {
  log(`ℹ️  ${message}`, 'cyan');
}

function warning(message) {
  log(`⚠️  ${message}`, 'yellow');
}

// 检查构建目录
function checkBuildDir() {
  if (!fs.existsSync(config.distDir)) {
    error('构建目录不存在，请先运行 pnpm docs:build');
  }
  success('构建目录检查通过');
}

// 部署到服务器
function deployToServer() {
  if (!config.server) {
    error('未设置 DEPLOY_SERVER 环境变量\n' +
          '请设置: export DEPLOY_SERVER=user@host');
  }

  info(`开始部署到: ${config.server}:${config.remotePath}`);

  try {
    // 使用 rsync 部署
    const rsyncCmd = `rsync -avz --delete ${config.distDir}/ ${config.server}:${config.remotePath}/`;
    
    info('执行命令: ' + rsyncCmd);
    execSync(rsyncCmd, { stdio: 'inherit' });
    
    success('部署完成!');
    info(`访问地址: http://ctf-docs.lostpeach.cn`);
  } catch (err) {
    error('部署失败: ' + err.message);
  }
}

// 本地预览
function localPreview() {
  info('启动本地预览...');
  try {
    execSync('pnpm docs:preview', { stdio: 'inherit' });
  } catch (err) {
    error('预览失败: ' + err.message);
  }
}

// 主函数
function main() {
  const args = process.argv.slice(2);
  const command = args[0] || 'deploy';

  log('\n🚀 Ocean CTF 文档部署工具\n', 'cyan');

  switch (command) {
    case 'deploy':
      checkBuildDir();
      deployToServer();
      break;
    
    case 'preview':
      checkBuildDir();
      localPreview();
      break;
    
    case 'help':
      log('使用方法:', 'yellow');
      log('  node deploy.js deploy   - 部署到服务器');
      log('  node deploy.js preview  - 本地预览');
      log('  node deploy.js help     - 显示帮助');
      log('\n环境变量:', 'yellow');
      log('  DEPLOY_SERVER - 服务器地址 (user@host)');
      log('  DEPLOY_PATH   - 远程路径 (默认: /var/www/ctf-docs)');
      break;
    
    default:
      warning(`未知命令: ${command}`);
      log('运行 node deploy.js help 查看帮助');
      break;
  }
}

main();
