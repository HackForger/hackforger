# 本机内网 ingress(`*.inside.h2os.cloud`)

本机所有 `*.inside.h2os.cloud` 的 HTTPS 反向代理由**宿主机原生 Caddy**(经 launchd 常驻)提供,
通过 **AliDNS DNS-01** 自动签发 `*.inside.h2os.cloud` 泛域名证书。仅经 Tailscale 内网访问。

> ⚠️ **权威部署是下面的「原生 Caddy」**。本目录里的 `Dockerfile` / `Caddyfile` 是**早期 Docker 版,已废弃**
> (只代理 hackforger 一个服务、且没编 `layer4` 插件),保留仅作历史参考,见文末「Legacy」。

---

## 一、现行部署:原生 Caddy(launchd 常驻)

所有运行态文件都在宿主机 `~/.config/caddy/`(**不在本仓库**,因为它服务多个项目、且含密钥):

| 位置 | 作用 |
|---|---|
| launchd `com.h2os.caddy` | 常驻守护(`RunAtLoad` + `KeepAlive`);plist:`~/Library/LaunchAgents/com.h2os.caddy.plist` |
| `~/.config/caddy/run.sh` | 入口:`source ~/.config/caddy/env` 注入密钥后 `exec caddy run` |
| `~/.config/caddy/env` | `ALICLOUD_ACCESS_KEY_ID` / `ALICLOUD_ACCESS_KEY_SECRET`(权限 600,**不入库**) |
| `~/.config/caddy/Caddyfile` | 路由 + TLS 配置(含 `layer4`) |
| `~/.config/caddy/caddy.log` | 运行日志(stdout/stderr) |
| `/usr/local/bin/caddy` | 二进制,**v2.11.2,带 `layer4` + `alidns` 插件** |

**当前代理的服务:**

| 入口 | 上游 | 备注 |
|---|---|---|
| `hackforger.inside.h2os.cloud` | `localhost:3000` | HackForger / Gitea |
| `saneledger.inside.h2os.cloud` | `localhost:54722` | 前置 oauth2-proxy(`127.0.0.1:4180`,`forward_auth`) |
| `zchat.inside.h2os.cloud`(SNI) | `127.0.0.1:6667` | **layer4** 在 `:6697` 按 SNI 透传 TLS |
| 其它 `*.inside.h2os.cloud` | — | 兜底 `404 Not Found` |

### 启动链路

```
launchd(com.h2os.caddy) → ~/.config/caddy/run.sh → source env(ALICLOUD_*) → caddy run --config ~/.config/caddy/Caddyfile
```

### 常用运维

```bash
# 看日志(证书签发 / 反代错误)
tail -f ~/.config/caddy/caddy.log

# 改了 ~/.config/caddy/Caddyfile 后生效:
# 注意 Caddyfile 里设了 `admin off`,所以不能用 `caddy reload`,要重启服务:
launchctl kickstart -k gui/$(id -u)/com.h2os.caddy

# 停 / 起
launchctl bootout  gui/$(id -u)/com.h2os.caddy        # 停
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.h2os.caddy.plist   # 起

# 验证(经 Tailscale,且对应上游需在跑)
curl -I https://hackforger.inside.h2os.cloud
```

### 重建 caddy 二进制(需带 layer4 + alidns)

`/usr/local/bin/caddy` 必须同时带 `layer4`(zchat 透传)和 `alidns`(泛域名证书)两个插件:

```bash
xcaddy build \
  --with github.com/mholt/caddy-l4 \
  --with github.com/caddy-dns/alidns
sudo install -m 0755 ./caddy /usr/local/bin/caddy
launchctl kickstart -k gui/$(id -u)/com.h2os.caddy
```

> 国内构建设 `GOPROXY=https://goproxy.cn,direct` 即可拉动 Go module。

### 新增一个内网服务

编辑 `~/.config/caddy/Caddyfile`,仿照 `@hackforger` 块加一段,然后 `launchctl kickstart -k …` 重启:

```caddyfile
@foo host foo.inside.h2os.cloud
handle @foo {
    reverse_proxy localhost:PORT
}
```

---

## 二、Legacy:Docker 版(已废弃,勿用于现行部署)

本目录的 `Dockerfile` / `Caddyfile` 是更早的 Docker 方案,**约 2026-04 起已被上面的原生 Caddy 取代**,
容器 `inside-caddy` 早已停用。差异:

- 只代理 `hackforger` 一个服务,**缺 `layer4`**(没有 zchat / saneledger)。
- 密钥变量名是 `ALIDNS_ACCESS_KEY_*`(原生版是 `ALICLOUD_ACCESS_KEY_*`)。
- 上游用 `host.docker.internal:3000`(原生版直接 `localhost:3000`)。

保留它们仅供历史参考。如确需在容器里跑(例如换机临时验证),镜像构建为:

```bash
cd deploy/caddy && docker build -t caddy-alidns:latest .   # Dockerfile 已内置 GOPROXY
```

证书数据曾存于 Docker 具名卷 `caddy_data`;**与原生 Caddy 的 `~/.config/caddy` 存储相互独立**。
