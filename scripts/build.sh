#!/bin/bash

set -e

# 颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building Rx-ui...${NC}"

# 进入项目根目录
cd "$(dirname "$0")/.."

# 1. 构建前端
echo -e "${GREEN}Step 1: Building frontend...${NC}"
cd web
npm install
npm run build
cd ..

# 2. 复制前端资源到 internal/web
echo -e "${GREEN}Step 2: Copying frontend assets...${NC}"
rm -rf internal/web/dist
cp -r web/dist internal/web/

# 3. 构建后端
echo -e "${GREEN}Step 3: Building backend...${NC}"
go build -o rx-ui main.go

echo -e "${GREEN}Build complete!${NC}"
echo "Binary: ./rx-ui"
