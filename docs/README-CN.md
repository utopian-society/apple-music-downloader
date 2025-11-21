[English](../README.md) / 简体中文


### ！！必须先安装[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)，并确认[MP4Box](https://gpac.io/downloads/gpac-nightly-builds/)已正确添加到环境变量

### 添加功能

1. 支持内嵌封面和LRC歌词（需要`media-user-token`，获取方式看最后的说明）
2. 支持获取逐词与未同步歌词
3. 支持下载歌手 `go run main.go https://music.apple.com/us/artist/taylor-swift/159260351` `--all-album` 自动选择歌手的所有专辑
4. 下载解密部分更换为Sendy McSenderson的代码，实现边下载边解密,解决大文件解密时内存不足
5. MV下载，需要安装[mp4decrypt](https://www.bento4.com/downloads/)
6. 支持交互式搜索 `go run main.go --search [song/album/artist] "搜索词"`
7. **批量下载支持** - 从文本文件批量下载多个专辑/播放列表 `go run main.go --batch urls.txt` 或多个文件 `go run main.go 1.txt 2.txt`
8. **分盘文件夹** - 多碟专辑自动按光盘分文件夹组织（可通过 config.yaml 中的 `separate-disc-folders` 配置）
9. **MV下载开关** - 通过配置文件 `download-music-video` 选项或 `--dl-mv` 命令行参数控制是否下载MV

### 特别感谢 `chocomint` 创建 `agent-arm64.js`
对于获取`aac-lc` `MV` `歌词` 必须填入有订阅的`media-user-token`

- `aac-he (audio-he)`
- `alac (audio-alac-stereo)`
- `ec3 (audio-atmos / audio-ec3)`
- `aac (audio-stereo)`
- `aac-lc (audio-stereo)`
- `aac-binaural (audio-stereo-binaural)`
- `aac-downmix (audio-stereo-downmix)`
- `MV`

**注意**: `aac-lc` 不适用于 HE-AAC（HE-AAC / HE-AACv2）流 — 它仅适用于普通的 AAC 变体，例如 binaural、downmix、stereo 等。

# Apple Music ALAC/杜比全景声下载器

原脚本由 Sorrow 编写。本人已修改，包含一些修复和改进。

## 使用方法
1. 确保解密程序 [wrapper](https://github.com/zhaarey/wrapper) 正在运行
2. 开始下载部分专辑：`go run main.go https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511`。
3. 开始下载单曲：`go run main.go --song https://music.apple.com/us/album/never-gonna-give-you-up-2022-remaster/1624945511?i=1624945512` 或 `go run main.go https://music.apple.com/us/song/you-move-me-2022-remaster/1624945520`。
4. 开始下载所选曲目：`go run main.go --select https://music.apple.com/us/album/whenever-you-need-somebody-2022-remaster/1624945511` 输入以空格分隔的数字。
5. 开始下载部分播放列表：`go run main.go https://music.apple.com/us/playlist/taylor-swift-essentials/pl.3950454ced8c45a3b0cc693c2a7db97b` 或 `go run main.go https://music.apple.com/us/playlist/hi-res-lossless-24-bit-192khz/pl.u-MDAWvpjt38370N`。
6. 对于杜比全景声 (Dolby Atmos)：`go run main.go --atmos https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`。
7. 对于 AAC (AAC)：`go run main.go --aac https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`。
8. 要查看音质：`go run main.go --debug https://music.apple.com/us/album/1989-taylors-version-deluxe/1713845538`。
9. **批量下载**：
   - 单个文件：`go run main.go --batch urls.txt`
   - 多个文件：`go run main.go 1.txt 2.txt 3.txt`
   - 或者：`go run main.go --batch file1.txt --batch file2.txt`
   
   （创建文本文件，每行一个 Apple Music URL。格式参考 `batch_example.txt`）。

[中文教程-详见方法三](https://telegra.ph/Apple-Music-Alac高解析度无损音乐下载教程-04-02-2)

## 下载歌词

1. 打开 [Apple Music](https://music.apple.com) 并登录
2. 打开开发者工具，点击“应用程序 -> 存储 -> Cookies -> https://music.apple.com”
3. 找到名为“media-user-token”的 Cookie 并复制其值
4. 将步骤 3 中获取的 Cookie 值粘贴到 config.yaml 文件中并保存
5. 正常启动脚本

## 配置选项

### 多批量文件下载

下载器支持一次处理多个批量文件：

- **单个文件**：`go run main.go --batch urls.txt`
- **多个文件（自动识别）**：`go run main.go 1.txt 2.txt 3.txt`
- **多个文件（显式指定）**：`go run main.go --batch file1.txt --batch file2.txt`

当您将 `.txt` 文件作为参数传递时，它们会自动被识别为批量文件。所有文件中的 URL 将被合并并按顺序处理。

**注意：** 所有下载都按顺序处理（一次一个），以确保稳定性和正确的文件处理。

### 多碟专辑组织

下载器现在支持将多碟专辑组织到单独的光盘文件夹中。这由 `config.yaml` 中的 `separate-disc-folders` 选项控制：

- **`separate-disc-folders: true`** - 为多碟专辑中的每张光盘创建"Disc 1"、"Disc 2"等子文件夹
- **`separate-disc-folders: false`**（默认）- 所有曲目直接保存在专辑文件夹中

**启用 `separate-disc-folders: true` 后的文件夹结构示例：**
```
歌手名/
└── 专辑名 [ALAC]/
    ├── cover.jpg
    ├── Disc 1/
    │   |
    │   ├── 01. 歌曲标题.m4a
    │   └── 02. 歌曲标题.m4a
    └── Disc 2/
        |
        ├── 01. 歌曲标题.m4a
        └── 02. 歌曲标题.m4a
```

**注意：** 单碟专辑不受此设置影响，将继续直接在专辑文件夹中保存曲目。

### MV下载控制

您现在可以使用配置文件或命令行参数来控制是否下载MV：

- **配置文件**：在 `config.yaml` 中设置 `download-music-video: true` 或 `download-music-video: false`
- **命令行**：运行程序时使用 `--dl-mv=true` 或 `--dl-mv=false` 参数

**示例：**
```bash
# 通过命令行禁用MV下载
go run main.go --dl-mv=false https://music.apple.com/us/album/example/123456

# 通过命令行启用MV下载（覆盖配置文件设置）
go run main.go --dl-mv=true https://music.apple.com/us/album/example/123456
```

当MV下载被禁用时：
- 专辑、播放列表和电台中的MV将被跳过
- 独立的MV URL将被跳过
- 将显示消息"Music video download is disabled, skipping"（MV下载已禁用，跳过）

**注意：** MV下载需要以下条件：
1. `download-music-video` 已启用（设置为 true）
2. config.yaml 中有有效的 `media-user-token`
3. 已安装 [mp4decrypt](https://www.bento4.com/downloads/) 并添加到 PATH 环境变量
