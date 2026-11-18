# DocsHub 1.2.0 主题设置

DocsHub 提供 15 组、121 个设置，覆盖外观、导航、首页、文档列表、阅读、系列、讨论提示和页脚。进入后台「外观与主题」(`/admin/appearance/`)，调整后点击「保存更改」。预览不会保存设置；导出、导入和恢复默认值使用 NewPress 的统一设置流程。

默认值和校验范围以 [settings.schema.json](settings.schema.json) 为准。旧版配置缺少的新字段会补齐默认值，原有字段名称和类型保持兼容。配置由核心保存到数据目录下的 `theme-settings/docshub.json`；日常修改通过后台完成。

## 先理解设置的作用范围

- **菜单优先级**：在菜单管理中配置的 `primary`、`utility`、`footer` 菜单，分别替代主题默认主导航、更多资源和页脚入口。显式配置的空菜单保持为空。`show_series_navigation`、默认 RSS 和主题名称开关只影响默认菜单；`show_utility_navigation`、`show_footer_navigation` 可以隐藏整个对应区域。修改 `listing_title`、`series_title` 会同步默认主导航名称，自定义菜单的标签仍由菜单管理维护。
- **站点身份与页脚**：站点名、站点图标、语言、账号身份、版权、备案和自定义页脚内容归核心管理。`footer_description` 只在核心未设置自定义版权时显示；主题的页脚导航开关不隐藏版权、备案或核心页脚内容。核心隐藏署名设置优先于 `show_theme_credit`，RSS 还要求公开 API 开启。
- **外观优先级**：允许切换时，读者浏览器中保存的浅色/深色/跟随系统偏好优先于 `default_color_mode`。关闭 `allow_color_switch` 后统一使用默认外观并忽略旧偏好。预览页从当前预览配置开始，不读取或写入读者偏好；预览中的临时切换不影响公开页面。
- **首页数据数量**：`tutorials_count`、`latest_count` 各允许 1–8 篇，默认 4 篇，只截取核心已经提供的数据。需要显示 8 篇时，还要在站点阅读设置中提供至少 8 篇首页文章，并确保有足够公开内容。教程优先使用核心的精选集合（有数据时），否则使用首页文章；主题不会为了补足数量生成示例内容或额外查询。
- **入口复用**：快捷入口、首屏按钮和学习步骤复用「导航与学习入口」中的地址。`start_link` 对应主按钮/入门步骤，`advanced_link` 对应进阶，`practice_link` 对应实战，`community_link` 也用于阅读帮助。关闭教程推荐时，默认 `demo_link="/#tutorials"` 自动转向 `/explore/`；自定义链接由站长维护。更多链接需要文字和地址同时非空；公告文字为空时整条公告隐藏。
- **列表视图**：`list_default_view` 用于文档中心、作者页和归档页。启用 JavaScript 时，地址里的有效 `view=grid` 或 `view=list` 优先，非法值回退到默认设置。隐藏排序、视图或分类控件只隐藏入口；已有 URL 条件仍有效，已选关键词/分类仍可清除。默认排序由核心决定。
- **阅读与系列**：目录只收录正文存在的标题；`h2`、`h3`、`h4` 分别表示截至二、三、四级。关闭全部阅读侧栏内容时自动收为单栏。系列入口、上下篇和推荐依赖核心系列模块及真实系列关系，同系列推荐最多三篇。隐藏入口不会停用模块。
- **讨论与恢复**：主题只调整评论区标题、说明和输入提示；评论开关、登录要求、审核与验证由核心控制。404 文案和搜索可以配置；维护说明始终使用核心维护信息，维护页不显示搜索入口。

## 尺寸、空值与无脚本行为

`content_width` 为主内容区域上限（960–1800px，不含左侧导航），`sidebar_width` 为宽屏侧栏宽度（190–280px，只在 1180px 以上生效）。`reading_width` 为文章区域上限（960–1440px，包含右侧目录）；它仍受外层内容宽度约束。`reading_font_size` 范围 14–20px，手机正文自动减小 1px；`reading_line_height` 范围 1.5–2.4。小屏幕始终随窗口收缩。

可选说明留空后不生成空白段落。首屏第一行标题留空时显示站点名；教程区、社区标题和搜索提示留空时使用默认文字。带必填校验的标题、按钮或阅读帮助文字不能保存为空。颜色接受六位十六进制值，入口接受安全的站内路径或网址；范围、枚举和类型由核心校验。

禁用 JavaScript 时，站内链接、搜索表单、正文和后台配置的默认列表视图仍可使用。默认深色由服务端直接输出；跟随系统回退浅色。外观切换、动态目录、阅读进度/时长、复制按钮功能、相对日期和悬浮返回顶部需要 JavaScript。首页日期会保留完整日期。减少动态效果同时尊重主题开关和系统偏好。

## 三种配置示例

以下只列需要调整的字段，可在后台对应分组中设置；它们是配置对照，不是完整的导入文档。未列字段保留当前值，建议从默认配置开始。

**产品文档门户**：保留首页入口与分类检索，推荐内容更丰富，外观跟随读者设备。

```json
{
  "default_color_mode": "system",
  "allow_color_switch": true,
  "hero_eyebrow": "产品文档",
  "hero_title": "从这里开始",
  "hero_title_second": "找到每一步的答案",
  "tutorials_count": 6,
  "latest_count": 6,
  "card_cover": "all",
  "list_default_view": "grid",
  "show_category_filters": true,
  "toc_depth": "h3"
}
```

首页文章数据至少提供 6 篇；将快速开始、API 和常见问题地址替换为本站对应内容。

**精简知识库**：减少插画和首页模块，用紧凑列表呈现更新，保留顶部搜索。

```json
{
  "navigation_density": "compact",
  "show_hero_artwork": false,
  "show_hero_secondary": false,
  "show_quick_links": false,
  "show_tutorials": false,
  "show_community": false,
  "show_learning_path": false,
  "show_latest": true,
  "latest_count": 8,
  "list_default_view": "list",
  "card_cover": "none",
  "card_meta": "date",
  "show_card_excerpt": true,
  "reduce_motion": true
}
```

列表无封面时自动收回图片列。需要统一浅色展示时另设 `default_color_mode="light"`、`allow_color_switch=false`。

**课程阅读站**：从系列索引进入课程，使用更详细的目录与连续阅读工具。

```json
{
  "series_title": "课程目录",
  "start_link": "/series/",
  "hero_primary_label": "选择课程",
  "show_series_navigation": true,
  "show_learning_path": true,
  "show_learning_series_link": true,
  "toc_depth": "h4",
  "reading_font_size": 17,
  "reading_line_height": 2,
  "reading_width": 1160,
  "show_reading_series": true,
  "show_series_related": true,
  "show_chapter_excerpt": true,
  "show_chapter_meta": false
}
```

先在核心启用系列模块，创建公开系列并关联章节。需要把课程放到自定义主菜单时，在菜单管理中添加 `/series/` 入口。

## 分组字段索引

下表列出全部字段及默认值。`true` 表示开启，`false` 表示关闭，`空` 表示默认不填写。控件说明和校验范围可在后台查看。

### 品牌与外观

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `accent` | 强调色 | `#4c82ff` |
| `default_color_mode` | 默认外观 | `light` |
| `allow_color_switch` | 允许访客切换外观 | `true` |
| `content_width` | 内容区域最大宽度 | `1600` |
| `sidebar_width` | 桌面侧栏宽度 | `218` |
| `navigation_density` | 桌面导航密度 | `comfortable` |
| `reduce_motion` | 减少动态效果 | `false` |

### 导航与搜索

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `sticky_header` | 固定顶部工具栏 | `true` |
| `show_search` | 显示顶部搜索 | `true` |
| `search_placeholder` | 搜索框提示语 | `搜索文档、教程或关键词…` |
| `show_search_shortcut` | 显示搜索快捷键提示 | `true` |
| `show_help` | 显示桌面帮助菜单 | `true` |
| `show_language` | 显示桌面语言信息 | `true` |
| `show_account` | 显示账号入口 | `true` |
| `show_utility_navigation` | 显示更多资源导航 | `true` |
| `show_series_navigation` | 默认导航显示学习系列 | `true` |

### 站点公告

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `announcement_text` | 公告内容 | 空 |
| `announcement_link` | 公告详情链接 | 空 |
| `announcement_link_label` | 公告链接文字 | `查看详情` |

### 首页首屏

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `hero_eyebrow` | 首屏眉题 | 空 |
| `hero_title` | 首屏标题 · 第一行 | `开发文档` |
| `hero_title_second` | 首屏标题 · 第二行 | `与使用教程` |
| `hero_description` | 首屏介绍 · 第一行 | `查阅安装步骤、功能说明和接口用法，` |
| `hero_description_second` | 首屏介绍 · 第二行 | `也可以按章节阅读教程和示例。` |
| `show_hero_artwork` | 显示玻璃丝带插画 | `true` |
| `hero_primary_label` | 首屏主按钮文字 | `开始阅读` |
| `show_hero_secondary` | 显示首屏次按钮 | `true` |
| `hero_secondary_label` | 首屏次按钮文字 | `浏览教程` |

### 首页模块

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `show_quick_links` | 显示快捷入口 | `true` |
| `show_tutorials` | 显示教程推荐 | `true` |
| `show_latest` | 显示最新文档 | `true` |
| `show_community` | 显示社区卡片 | `true` |
| `show_learning_path` | 显示学习路径 | `true` |

### 首页文档内容

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `tutorials_title` | 教程区标题 | `教程推荐` |
| `tutorials_description` | 教程区说明 | 空 |
| `tutorials_count` | 推荐教程数量 | `4` |
| `tutorials_more_label` | 教程区更多链接文字 | `查看全部` |
| `tutorials_more_link` | 教程区更多链接 | `/explore/` |
| `latest_title` | 最新文档标题 | `最新文档` |
| `latest_description` | 最新文档说明 | 空 |
| `latest_count` | 最新文档数量 | `4` |
| `latest_more_label` | 最新文档更多链接文字 | `查看全部` |
| `latest_more_link` | 最新文档更多链接 | `/explore/?sort=published` |
| `show_latest_excerpt` | 显示最新文档摘要 | `true` |
| `show_latest_date` | 显示最新文档日期 | `true` |
| `relative_dates` | 最新文档使用相对时间 | `true` |

### 社区与学习路径

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `community_title` | 社区卡片标题 | `社区交流` |
| `community_description` | 社区卡片说明 | `提交问题，交流使用经验。` |
| `community_button_label` | 社区按钮文字 | `前往社区` |
| `show_community_artwork` | 显示社区山景插画 | `true` |
| `learning_path_title` | 学习路径标题 | `学习路径` |
| `show_learning_series_link` | 显示“全部系列”入口 | `true` |
| `learning_start_title` | 学习步骤 1 · 标题 | `快速入门` |
| `learning_start_description` | 学习步骤 1 · 说明 | `了解基本概念与环境搭建` |
| `learning_advanced_title` | 学习步骤 2 · 标题 | `进阶提升` |
| `learning_advanced_description` | 学习步骤 2 · 说明 | `了解功能用法与配置方法` |
| `learning_practice_title` | 学习步骤 3 · 标题 | `实战应用` |
| `learning_practice_description` | 学习步骤 3 · 说明 | `参考项目示例与操作步骤` |

### 导航与学习入口

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `start_link` | 快速开始链接 | `/explore/?q=入门` |
| `advanced_link` | 进阶教程链接 | `/explore/?q=进阶` |
| `api_link` | API 参考链接 | `/explore/?q=API` |
| `practice_link` | 最佳实践链接 | `/explore/?q=实战` |
| `tools_link` | 开发工具链接 | `/explore/?q=工具` |
| `faq_link` | 常见问题链接 | `/explore/?q=常见问题` |
| `changelog_link` | 更新日志链接 | `/explore/?q=更新` |
| `community_link` | 社区入口链接 | `/account/` |
| `projects_link` | 开源项目链接 | `/explore/?q=开源` |
| `about_link` | 关于我们链接 | `/explore/?q=关于` |
| `demo_link` | 演示教程链接 | `/#tutorials` |

### 文档列表与卡片

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `listing_title` | 文档中心标题 | `文档与教程` |
| `listing_description` | 文档中心介绍 | `按分类浏览文档，或搜索你需要的操作步骤与示例。` |
| `list_default_view` | 默认展示方式 | `grid` |
| `show_category_filters` | 显示分类筛选 | `true` |
| `show_sort_control` | 显示排序控件 | `true` |
| `show_view_switch` | 显示视图切换 | `true` |
| `card_cover` | 文档卡片封面 | `all` |
| `show_card_category` | 显示卡片分类 | `true` |
| `show_card_excerpt` | 显示卡片摘要 | `true` |
| `card_meta` | 卡片底部信息 | `full` |

### 文章阅读

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `show_toc` | 显示文章目录 | `true` |
| `toc_depth` | 目录层级 | `h3` |
| `show_reading_progress` | 显示阅读进度 | `true` |
| `show_reading_time` | 显示预计阅读时间 | `true` |
| `show_code_copy` | 显示代码复制按钮 | `true` |
| `show_copy_link` | 显示复制文章链接 | `true` |
| `show_article_author` | 显示文章作者 | `true` |
| `show_article_date` | 显示最后更新日期 | `true` |
| `show_article_excerpt` | 显示文章摘要 | `true` |
| `show_article_tags` | 显示文章标签 | `true` |
| `show_reading_help` | 显示阅读帮助入口 | `true` |
| `reading_help_label` | 阅读帮助文案 | `遇到问题？到社区聊聊` |
| `show_reading_back_to_top` | 显示阅读侧栏返回顶部 | `true` |
| `reading_font_size` | 正文字号 | `15` |
| `reading_line_height` | 正文行高 | `1.95` |
| `reading_width` | 阅读区域最大宽度 | `1240` |

### 学习系列

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `series_title` | 系列索引标题 | `学习系列` |
| `series_description` | 系列默认介绍 | `按主题整理的文档，可按章节顺序阅读。` |
| `show_series_search` | 显示系列搜索 | `true` |
| `show_chapter_excerpt` | 显示章节摘要 | `true` |
| `show_chapter_meta` | 显示章节分类与日期 | `true` |
| `show_reading_series` | 显示系列上下篇与阅读位置 | `true` |
| `show_series_related` | 显示同系列推荐文档 | `false` |

### 讨论区

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `comments_title` | 讨论区标题 | `交流与讨论` |
| `comments_intro` | 讨论区说明 | 空 |
| `comments_placeholder` | 评论输入提示 | 空 |

### 错误与恢复

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `not_found_title` | 未找到页面的标题 | `文档不存在` |
| `not_found_description` | 未找到页面的说明 | `这篇文档可能已被移动。搜索关键词，或从文档中心继续浏览。` |
| `show_error_search` | 404 页面显示搜索 | `true` |

### 页脚与辅助入口

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `footer_description` | 页脚站点说明 | `文档与教程` |
| `show_footer_navigation` | 显示页脚导航 | `true` |
| `show_footer_rss` | 默认页脚显示 RSS 订阅 | `true` |
| `show_theme_credit` | 默认页脚显示主题名称 | `true` |
| `show_back_to_top` | 显示悬浮返回顶部 | `true` |

### 首页快捷入口

| 字段 | 后台名称 | 默认值 |
|---|---|---|
| `quick_start_title` | 快捷入口 1 · 标题 | `快速上手` |
| `quick_start_description` | 快捷入口 1 · 说明 | `环境准备与基础操作` |
| `quick_advanced_title` | 快捷入口 2 · 标题 | `进阶教程` |
| `quick_advanced_description` | 快捷入口 2 · 说明 | `功能用法与实现示例` |
| `quick_api_title` | 快捷入口 3 · 标题 | `API 参考` |
| `quick_api_description` | 快捷入口 3 · 说明 | `接口参数与调用示例` |
| `quick_tools_title` | 快捷入口 4 · 标题 | `开发工具` |
| `quick_tools_description` | 快捷入口 4 · 说明 | `工具安装与配置说明` |

配置值以实际保存结果为准。修改后可分别打开首页、文档中心、文章和系列页检查；同时检查手机宽度、深色模式以及关闭模块后的布局。
