.PHONY: build deploy

# 腾讯云服务器配置
TX_SERVER ?= tx
TX_PATH ?= /var/www/ctf-docs

# 构建文档
build:
	@echo "正在构建文档..."
	pnpm docs:build
	@echo "构建完成!"

# 部署到腾讯云服务器
deploy: build
	@echo "正在同步到腾讯云服务器..."
	@echo "服务器: $(TX_SERVER)"
	@echo "路径: $(TX_PATH)"
	scp -r .vitepress/dist/* $(TX_SERVER):$(TX_PATH)/
	@echo "部署完成!"
	@echo "访问地址: http://ctf-docs.lostpeach.cn"
