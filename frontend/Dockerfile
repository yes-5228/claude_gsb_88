# ---------- 构建阶段：编译前端静态资源 ----------
FROM node:22-alpine AS build
WORKDIR /app

# 默认使用国内镜像源，容器内安装依赖更稳定；可通过 --build-arg NPM_REGISTRY=... 覆盖
ARG NPM_REGISTRY=https://registry.npmmirror.com

COPY package.json package-lock.json* ./
RUN npm ci --registry=${NPM_REGISTRY} --no-audit --no-fund \
    || npm install --registry=${NPM_REGISTRY} --no-audit --no-fund

COPY . .
RUN npm run build

# ---------- 运行阶段：nginx 托管静态资源并反向代理 /api ----------
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
