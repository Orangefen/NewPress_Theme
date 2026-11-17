# DocsHub 素材记录

六张装饰 WebP 来自用户在本次主题开发任务中提供的 DocsHub 参考图（1254 × 1254），依照用户要求裁切与去除文字，用于保持参考图的材质和构图。没有访问外部图片库，也没有把整页截图作为页面界面。

| 文件 | 处理 |
| --- | --- |
| `glass-ribbon.webp` | 裁切参考图首屏背景，移除标题、按钮及顶栏文字，保留玻璃丝带 |
| `tutorial-workspace.webp` | 第一张教程封面 |
| `tutorial-components.webp` | 第二张教程封面 |
| `tutorial-code.webp` | 第三张教程封面 |
| `tutorial-performance.webp` | 第四张教程封面 |
| `community-mountains.webp` | 社区卡片背景，移除原有文字与按钮 |
| `preview.webp` | 已实现主题在隔离 NewPress 实例中的实际页面截图 |

素材提取脚本位于仓库 `scripts/prepare_docshub_assets.py`，可使用原始参考 PNG 重现，开发时需要 Pillow、NumPy 与 OpenCV。运行安装包不需要这些工具。原始参考图不随安装包分发。

图标与品牌标记为本主题手写内联 SVG；字体为 Noto Sans SC 可变字体的本地 WOFF2 子集，采用 SIL Open Font License 1.1，许可证位于 `assets/fonts/OFL.txt`；包含 GB2312 字符、ASCII 和常用标点，其他文字回退至系统字体。字体源来自 `notofonts/noto-cjk` 官方仓库的 `Sans/Variable/TTF/Subset/NotoSansSC-VF.ttf`。参考图片的权利不因代码采用 Apache-2.0 而发生变化，来源按用户提供素材记录。
