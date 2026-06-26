# LEGACY — 早期 Docker 版 inside-caddy 的镜像,已被宿主机原生 launchd Caddy 取代。
# 注意:此镜像未编 layer4 插件,功能弱于现行原生部署。详见本目录 README.md。
FROM caddy:2-builder AS builder

# 国内拉取 Go module 走镜像,避免 xcaddy build 卡在 proxy.golang.org / GitHub
ENV GOPROXY=https://goproxy.cn,direct

RUN xcaddy build \
    --with github.com/caddy-dns/alidns

FROM caddy:2-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
