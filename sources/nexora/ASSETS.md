# 素材生成与维护

本次通过 imagegen 技能的 API/CLI 模式调用用户指定的 `gpt-image-2.6-sunburst`，生成无文字建筑背景和山峦照片，再压缩为本地 WebP。调用目标为用户指定的兼容接口；没有保存凭据。浏览器不访问生图服务。

| 文件 | 用途 | 处理 |
| --- | --- | --- |
| `assets/images/headquarters.webp` | 第一张轮播 | 保留参考图原始摄影区域，替换文字覆盖区域，局部匹配色调 |
| `assets/images/headquarters-mobile.webp` | 较小尺寸备选 | 同图等比缩小 |
| `assets/images/about-architecture.webp` | 关于我们、第二张轮播、无封面新闻 | 原参考图建筑照片裁切 |
| `assets/images/innovation-mountains.webp` | 第三张轮播 | 生成照片降饱和度与对比度 |
| `assets/images/innovation-banner.webp` | 创新愿景横幅 | 原图顶部去字后，与生成的山峦延伸融合 |
| `assets/fonts/nexora-serif.woff2` | 中文展示标题 | Noto Serif SC 400 字重子集；按 OFL 重命名 |

首次调用的输入为参考图上方 1254×600 的首屏裁切。最终生成提示词如下。

## 建筑背景编辑提示词

```text
Use case: precise-object-edit
Asset type: photographic hero background for a corporate technology website.
Input image: edit target, a 1254 by 600 architectural photograph with website UI layered on it.
Primary request: Remove ALL overlaid website typography, logos, navigation, icons, button outlines, slide indicators and arrows. Reconstruct only the sky, trees and water behind those overlays. Return a clean full-bleed architectural photograph with no UI.
Invariants: Preserve the exact camera angle, crop, composition, position and size of the glass headquarters, the warm interior lights, every building edge, water reflections, trees, blue-hour sky, subdued dark navy left side and lighter slate-blue upper right. The right-hand two-story glass building occupies the right half, top corner at about x=940 y=120, ground line at y=465. Preserve original 2.09:1 aspect ratio. Do not zoom or move objects.
Constraints: Change only areas covered by white UI. No text, letters, numbers, logo, watermark, interface, controls or borders. Photorealism. Crisp detailed glass, subdued blue hour.
```

## 山峦生成提示词

```text
Use case: photorealistic-natural
Asset type: wide corporate technology website innovation banner.
Primary request: A panoramic blue-hour landscape of layered distant mountains in a restrained slate blue and cool silver palette, a thin line of pale mist in valleys and subtle snow veins on dark slopes. Realistic natural photograph, high detail, understated and tranquil.
Composition: ultra-wide panoramic framing, about 2.1:1, generous smooth desaturated blue-grey sky in the upper half for white web typography. Mountains form low successive layers across the bottom half, tallest rounded ridge on the left, smaller ridge at center, dark foreground ridge on right.
Lighting: soft light just before dawn, no orange or saturated colors. Match a blue glass corporate architecture photograph.
Constraints: no text, logos, UI, watermark, people, fantasy elements, neon, or artificial lights.
```

两次请求均使用 `size=auto`、`quality=high`。原始输出与本机合成脚本在 `output/imagegen/nexora/`，不属于安装包。图形与版式均由真实模板、CSS 和 SVG 实现，图片中没有页面文案或可点击控件。
