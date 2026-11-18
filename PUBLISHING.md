# 主题目录发布

使用已加入主题市场支持的 NewPress 核心。GitHub 主仓库为 `Orangefen/NewPress_Theme`，Gitee 镜像为 `chunyangnet_0/NewPress_Theme`，统一使用 `main` 分支。签名私钥必须位于仓库之外；目录仅接受公网 GitHub raw / Gitee raw 地址，两个镜像必须同步相同的目录、ZIP 与封面。

当前目录 `catalog.themes-v1.json` 使用 schema 2、序号 4，包含 Nexora 1.0.1、Nexora 1.1.0、DocsHub 1.0.0 和 DocsHub 1.2.0，有效期至 2027-11-19T00:00:00Z。完整发行元数据保存在 `releases/catalog.themes-v1.sequence-4.metadata.json`，序号 1、2 和 3 的元数据继续保留。后续发布需要更高的序号，保留既有 ZIP 字节与历史发行时间。

发布前核对主题根目录的 `LICENSE`、`THIRD_PARTY_NOTICES.md`、`ASSETS.md` 和随字体携带的 `assets/fonts/OFL.txt`，打包后再次确认这些文件实际进入 ZIP。代码保持 Apache-2.0，第三方字体继续遵守其独立许可。项目配图为 AI 生成素材，不另列来源署名；更新图片时同步更新预览封面和素材清单。

补许可文件同样会改变 ZIP 的摘要和长度，应作为新版本打包并递增目录序号、重新签名；不能原地修补已有签名包。历史版本的许可归属和已获得的授权不因新增说明而改变。

1. 修改 `sources/<slug>/theme.json` 的版本。封面字段示例：`"cover": "assets/images/preview.webp"`。仅支持 PNG、JPEG、WebP，最大 2 MiB、4096×4096，最多 1200 万像素；建议 1600×1000 或 960×600。
2. 构建、测试、打包。先运行 `npm run build`、`npm run check`、`npm test`，再对改动主题运行 `python scripts/check_core.py --core ../newpress --theme <slug>`。Nexora 1.0.1、Nexora 1.1.0、DocsHub 1.0.0 和 DocsHub 1.2.0 安装包已经存在；新版本使用 `python scripts/build_extension.py --kind theme --source sources/<slug> --output packages/themes/<slug>-<新版本>.zip`，不要覆盖已有文件。
3. 在核心仓库运行发布工具（以下 `<...>` 需替换为实际值；先将源码、包和发行说明路径转换为绝对路径）：

```text
go run ./cmd/pluginmarket -keygen -key <仓库之外的签名私钥路径>
go run ./cmd/pluginmarket -schema 2 -themes-only -catalog-name catalog.themes-v1.json -key <签名私钥路径> -out <新的发行输出目录> -sequence <递增序号> -expires <未来的RFC3339到期时间> -themes <逗号分隔的全部主题ZIP绝对路径> -data-compatible -metadata <本次完整发行元数据JSON绝对路径>
```

本仓库已建立独立签名身份，正常发布使用原私钥，**不要重新运行 `-keygen`**；该命令仅供新建独立分发源时参考。`-data-compatible` 表示确认新旧版本均能使用现有设置。工具验证完整主题包，导出封面，计算所有 SHA-256 与字节数，再签署目录；不会自动上传。`-themes-only` 不包含内置插件。从最近一次完整元数据复制新文件，原样保留已有 `theme/<slug>@<version>` 条目，再添加新发行版的说明、通道、实际发行时间和兼容声明。使用 `-metadata` 提供完整元数据，`-themes` 用逗号分隔全部需要保留的主题 ZIP；元数据中不能包含没有对应 ZIP 的条目。

4. 将输出的 `catalog.themes-v1.json`、`catalog.themes-v1.readable.json`、`packages/`、`previews/` 合并到主题仓库并发布到 GitHub/Gitee。保留旧包，不能同版本改包或同序号改目录。更新目录有效期也需要递增序号并重新签名。
5. 已内置官方主题源的配套核心无需额外配置。其他支持独立主题源的部署可直接采用 [部署变量](deployment.env.example)，重启核心，在「主题市场」刷新目录并试装。自定义公钥与目录地址必须成对配置；不要填写私钥。

主题目录使用独立的最高序号记录，不与插件目录互相覆盖。配套核心默认读取本仓库；插件源配置不再影响主题源。需要继续使用旧的统一目录时，显式把该目录和对应公钥填写为主题源。没有有效目录时仍可管理本地已安装主题。

## 同步两个远程

```text
git push -u origin main
git push gitee main
```

推送同一提交，不要为两个平台分别重新签名。推送后分别匿名下载目录、封面和 ZIP，并用 NewPress 市场客户端验证签名及安装；`git push` 成功只代表仓库接收提交，不代表 raw 下载已可用。

`signing-public-key.txt` 是公钥副本，仅用于核对。核心内置公钥或显式部署配置才是信任根。私钥仅在本机仓库外保管并备份，不进入 Git、发行包或 CI；更换签名身份需同时通过可信渠道更新核心或部署配置。

独立仓库可使用 GitHub/Gitee 静态 raw 文件托管，不需要部署额外市场服务。当前协议不支持任意下载域名或直接 GitHub Releases 下载地址；需要更换分发方式时应先扩展核心校验协议。
