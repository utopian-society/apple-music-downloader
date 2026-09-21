# 🌐 Cloud Server Proxy Setup

> **Fixes:** `Failed to get song response.` · `503 Service Unavailable` · `failed to get lyrics`

[English](#english) | [简体中文](#简体中文)

---

<a name="english"></a>

## English

Cloud hosting providers (like **DigitalOcean**, **AWS**, **Hetzner**, **Vultr**, etc.) often have their IP ranges **blocked by Apple Music's API** (`amp-api.music.apple.com`). This causes the following errors on the server even though everything works fine locally:

```
Failed to get song response.
Failed to rip song: 503 Service Unavailable
failed to get lyrics
```

To fix this, you can route all Apple API HTTP requests through a local proxy. This **does not touch your server's main IP address or break SSH in any way**.

---

### 1. Config Option

In your `config.yaml`, set the `proxy` field:

```yaml
# SOCKS5 or HTTP proxy URL.
# Leave empty "" for direct connection (local PC / no proxy needed).
# For cloud servers blocked by Apple, use one of the options below.
proxy: "socks5://127.0.0.1:1080"
```

---

### 2. Option A — Cloudflare WARP (Recommended)

Cloudflare WARP in **proxy mode** runs a local SOCKS5 proxy on port `1080`. It does **not** replace your server's system IP or route all traffic — only the Go process's HTTP calls go through it.

#### Step 1: Install Cloudflare WARP

```bash
# Add Cloudflare GPG key and repository
curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | sudo gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflare-client.list

# Update and install
sudo apt update && sudo apt install cloudflare-warp -y
```

#### Step 2: Configure WARP in Proxy Mode

```bash
# Register a new WARP account
warp-cli registration new

# CRITICAL: Set to proxy mode (prevents SSH disconnects / IP changes)
warp-cli mode proxy

# Set the local SOCKS5 proxy port
warp-cli proxy port 1080

# Connect
warp-cli connect
```

#### Step 3: Enable Auto-Start on Boot

```bash
sudo systemctl enable warp-svc
```

#### Step 4: Update config.yaml

```yaml
proxy: "socks5://127.0.0.1:1080"
```

#### Verify It Works

```bash
# Your server's real IP (unchanged)
curl https://api.ipify.org

# Traffic through WARP (Cloudflare proxy IP)
curl --socks5 127.0.0.1:1080 https://www.cloudflare.com/cdn-cgi/trace
```

> If the second command shows `warp=on`, the proxy is working correctly.

---

### 3. Option B — SSH Reverse Tunnel (Zero Cost, No Install)

If you don't want to install anything on the server, you can forward your **local machine's internet connection** to the server via an SSH tunnel.

#### On your LOCAL machine (Windows / macOS / Linux):

```bash
ssh -R 1080 -N user@YOUR_SERVER_IP
```

Keep this terminal open while downloading. This opens a SOCKS5 proxy on `127.0.0.1:1080` **on the server**, tunnelled through your local internet connection.

#### On the server, set config.yaml:

```yaml
proxy: "socks5://127.0.0.1:1080"
```

> ✅ SSH itself is **never** affected. This tunnel only makes port 1080 available locally on the server.

---

### 4. Local PC (Windows / macOS) — No Proxy Needed

If you're running the tool on your own computer, Apple's API is usually accessible directly. Leave the proxy setting empty:

```yaml
proxy: ""
```

---

### 5. Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| `503 Service Unavailable` | Server IP blocked by Apple | Set proxy in `config.yaml` |
| `failed to get lyrics` | `amp-api.music.apple.com` blocked | Set proxy in `config.yaml` |
| `proxy config error: ...` | Invalid proxy URL format | Check URL format: `socks5://host:port` |
| ALAC/Atmos works but lyrics/AAC fail | Only API calls blocked, not CDN | Normal — proxy fixes API calls |
| SSH disconnected after WARP install | WARP set to VPN mode instead of proxy mode | Run `warp-cli mode proxy` again |

---

<a name="简体中文"></a>

## 简体中文

云服务器（如 **DigitalOcean**、**AWS**、**Hetzner**、**Vultr** 等）的 IP 段经常被 Apple Music API（`amp-api.music.apple.com`）屏蔽，导致在服务器上运行时出现以下错误（而本地电脑完全正常）：

```
Failed to get song response.
Failed to rip song: 503 Service Unavailable
failed to get lyrics
```

解决方法是将所有 Apple API 请求通过本地代理转发。**这不会改变服务器主 IP 地址，也不会影响 SSH 连接。**

---

### 1. 配置选项

在 `config.yaml` 中设置 `proxy` 字段：

```yaml
# SOCKS5 或 HTTP 代理地址。
# 本地电脑无需代理，留空即可。
# 云服务器被 Apple 屏蔽时，使用以下任一选项。
proxy: "socks5://127.0.0.1:1080"
```

---

### 2. 方案 A — Cloudflare WARP（推荐）

Cloudflare WARP 的**代理模式**会在本地启动一个 SOCKS5 代理（端口 `1080`）。它**不会**替换服务器系统 IP 或路由全局流量 —— 只有 Go 程序的 HTTP 请求会通过代理。

#### 第一步：安装 Cloudflare WARP

```bash
# 添加 Cloudflare GPG 密钥和软件源
curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | sudo gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/cloudflare-client.list

# 更新并安装
sudo apt update && sudo apt install cloudflare-warp -y
```

#### 第二步：配置 WARP 代理模式

```bash
# 注册新的 WARP 账户
warp-cli registration new

# 关键：设置为代理模式（防止 SSH 断连 / IP 被替换）
warp-cli mode proxy

# 设置本地 SOCKS5 代理端口
warp-cli proxy port 1080

# 连接
warp-cli connect
```

#### 第三步：设置开机自启

```bash
sudo systemctl enable warp-svc
```

#### 第四步：修改 config.yaml

```yaml
proxy: "socks5://127.0.0.1:1080"
```

#### 验证是否正常工作

```bash
# 服务器真实 IP（保持不变）
curl https://api.ipify.org

# 通过 WARP 的流量（Cloudflare 代理 IP）
curl --socks5 127.0.0.1:1080 https://www.cloudflare.com/cdn-cgi/trace
```

> 若第二条命令输出包含 `warp=on`，代理配置成功。

---

### 3. 方案 B — SSH 反向隧道（零成本，无需安装）

如果不想在服务器上安装任何软件，可以通过 SSH 将本地电脑的网络连接转发到服务器。

#### 在你的**本地电脑**上执行（Windows / macOS / Linux）：

```bash
ssh -R 1080 -N user@YOUR_SERVER_IP
```

下载期间保持此终端开启。这会在**服务器的** `127.0.0.1:1080` 上开放一个 SOCKS5 代理，通过本地网络进行中转。

#### 在服务器的 config.yaml 中设置：

```yaml
proxy: "socks5://127.0.0.1:1080"
```

> ✅ SSH 连接**不受任何影响**。此隧道仅在服务器本地开放 1080 端口。

---

### 4. 本地电脑（Windows / macOS）— 无需代理

如果在自己的电脑上运行，通常可以直接访问 Apple 接口，无需代理，留空即可：

```yaml
proxy: ""
```

---

### 5. 故障排查

| 错误 | 原因 | 解决方法 |
|------|------|----------|
| `503 Service Unavailable` | 服务器 IP 被 Apple 屏蔽 | 在 `config.yaml` 中配置代理 |
| `failed to get lyrics` | `amp-api.music.apple.com` 被屏蔽 | 在 `config.yaml` 中配置代理 |
| `proxy config error: ...` | 代理 URL 格式错误 | 检查格式：`socks5://host:port` |
| ALAC/Atmos 正常但歌词/AAC 失败 | 仅 API 接口被屏蔽，CDN 正常 | 正常现象，配置代理后修复 |
| 安装 WARP 后 SSH 断连 | WARP 设置为 VPN 模式而非代理模式 | 重新运行 `warp-cli mode proxy` |
