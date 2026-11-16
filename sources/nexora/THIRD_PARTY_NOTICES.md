# Nexora 素材与许可

## Noto Serif SC / Nexora Serif

`assets/fonts/nexora-serif.woff2` 是 Noto Serif SC 的 400 字重字符子集，内部名称改为 Nexora Serif。由 Google Fonts 源码生成，保留 SIL Open Font License 1.1，完整许可见 `assets/fonts/OFL.txt`。

Copyright 2012 Google Inc. All Rights Reserved.

原始文件来源：

```text
https://github.com/google/fonts/tree/main/ofl/notoserifsc
NotoSerifSC[wght].ttf
```

字体遵循其 OFL 许可，不属于主题代码的 Apache-2.0 许可。不包含 Chunnel 的 B2 字体。

## 参考图与摄影素材

- 企业建筑和山峦构图来自用户于 2026-11-16 为本次实现提供的 NEXORA 参考图。
- `about-architecture.webp` 为参考图中不含文字的建筑照片裁切。
- `headquarters.webp` 为原图主体与指定模型生成的无文字区域合成；手机版资源从相同构图缩放得到。
- `innovation-mountains.webp` 为用户指定服务与模型生成的山峦图；`innovation-banner.webp` 结合参考图开头区域与生成图延伸。
- 这些图片只作为本次企业官网主题的默认演示素材，不将输入图或第三方服务输出重新声明为 Apache-2.0 摄影作品。发布者可以在主题设置中替换为企业自有图片。

模型及提示词记录见 `ASSETS.md`。凭据未保存到主题、构建脚本、安装包或浏览器代码。

## 代码与图形

主题代码遵循 Apache-2.0。N 形标志和线性图标由本次实现以 SVG 路径重建。`nexora-verification.source.js` 由 NewPress 核心验证展示适配器改编，保留其 Apache-2.0 来源；使用 Nexora 自有命名、浅色配置和错误提示，不依赖 Chunnel 主题或其全局对象。
