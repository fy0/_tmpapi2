export default {
  "home": {
    "mirrors": {
      "title": "访问镜像",
      "description": "选择就近线路接入同一个 API 服务",
      "currentSite": "主站",
      "default": "默认",
      "action": "登录后即可使用该线路创建和管理密钥"
    },
    "customLinks": {
      "title": "推荐链接",
      "description": "常用页面和外部资源"
    }
  },
  "nav": {
    "supportTickets": "工单/意见",
    "ticketManagement": "工单管理",
    "apiManagement": "API 管理",
    "services": "服务",
    "myInvoices": "我的发票",
    "invoiceManagement": "发票管理(测试中)"
  },
  "supportTickets": {
    "title": "工单/意见",
    "description": "提交问题、意见或服务请求并查看回复",
    "create": "新建工单",
    "view": "查看工单",
    "detail": "工单详情",
    "reply": "回复",
    "sendReply": "发送回复",
    "searchPlaceholder": "搜索工单...",
    "empty": "暂无工单",
    "emptyDescription": "提交问题或建议后会显示在这里。",
    "created": "工单已创建",
    "replied": "回复已发送",
    "failedToLoad": "加载工单失败",
    "failedToLoadDetail": "加载工单详情失败",
    "failedToCreate": "创建工单失败",
    "failedToReply": "发送回复失败",
    "filters": {
      "allStatus": "全部状态",
      "allCategories": "全部分类"
    },
    "columns": {
      "title": "标题",
      "category": "分类",
      "status": "状态",
      "priority": "优先级",
      "lastMessageAt": "最后消息"
    },
    "form": {
      "title": "标题",
      "category": "分类",
      "priority": "优先级",
      "status": "状态",
      "content": "内容"
    },
    "status": {
      "open": "待处理",
      "pending": "处理中",
      "resolved": "已解决",
      "closed": "已关闭"
    },
    "priority": {
      "low": "低",
      "normal": "普通",
      "high": "高"
    },
    "category": {
      "feedback": "提意见",
      "bug": "问题反馈",
      "billing": "计费/充值",
      "account": "账号",
      "other": "其他"
    },
    "role": {
      "user": "用户",
      "admin": "管理员"
    }
  },
  "invoice": {
    "title": "我的发票",
    "description": "管理可开票充值和发票文件",
    "management": "发票管理(测试中)",
    "managementDescription": "审核开票申请并上传已开票文件",
    "availableAmount": "可开票金额",
    "pendingAmount": "待开票金额",
    "issuedAmount": "已开票金额",
    "availableRecharges": "可开票充值",
    "createRequest": "开票申请",
    "invoiceTitle": "发票抬头",
    "taxNo": "税号",
    "note": "备注",
    "selectedCount": "已选充值",
    "selectedAmount": "已选金额",
    "submitRequest": "提交申请",
    "history": "开票记录",
    "id": "编号",
    "amount": "金额",
    "status": "状态",
    "createdAt": "申请时间",
    "usedAt": "充值时间",
    "code": "兑换码",
    "user": "用户",
    "recharges": "充值记录",
    "detail": "发票详情",
    "selectedRecharges": "已选充值记录",
    "fileName": "文件名",
    "fileSize": "文件大小",
    "uploadInvoice": "上传发票",
    "uploadConfirmTitle": "确认上传发票",
    "uploadConfirmMessage": "确定将文件「{file}」上传到发票 #{id} 吗？上传后该申请会标记为已开票。",
    "deleteFile": "删除文件",
    "deleteFileConfirmTitle": "确认删除发票文件",
    "deleteFileConfirmMessage": "确定删除发票 #{id} 的已上传文件吗？删除后该申请会回退为待开票状态。",
    "deleteFileSuccess": "发票文件已删除，状态已回退为待开票",
    "viewFile": "查看文件",
    "preview": "预览",
    "previewUnavailable": "该文件类型暂不支持预览，请下载后查看。",
    "exportPendingCsv": "导出待开票 CSV",
    "exportOlderThanSixHours": "只导出 6 小时以外",
    "exportSuccess": "待开票信息已导出",
    "settings": "发票设置",
    "minInvoiceAmount": "最低开票金额",
    "minInvoiceAmountHint": "低于该金额时用户不能提交开票申请，0 表示不限制。",
    "maxInvoiceAmount": "最高开票金额",
    "maxInvoiceAmountHint": "高于该金额时用户不能提交开票申请，0 表示不限制。",
    "invoiceAmountLimitHint": "低于最低金额或高于最高金额时用户不能提交开票申请，0 表示不限制。",
    "minInvoiceAmountNotice": "最低开票金额：{amount}。低于该金额无法提交开票申请。",
    "maxInvoiceAmountNotice": "最高开票金额：{amount}。高于该金额无法提交开票申请。",
    "belowMinInvoiceAmount": "当前已选金额低于最低开票金额 {amount}",
    "aboveMaxInvoiceAmount": "当前已选金额高于最高开票金额 {amount}",
    "saveSettings": "保存设置",
    "settingsSaved": "发票设置已保存",
    "download": "下载",
    "withdraw": "撤回",
    "withdrawConfirmTitle": "确认撤回开票申请",
    "withdrawConfirmMessage": "确定撤回发票 #{id} 的开票申请吗？撤回后所选充值会重新变为可开票。",
    "searchPlaceholder": "搜索用户、抬头或税号",
    "noRecharges": "暂无可开票充值",
    "noInvoices": "暂无开票记录",
    "createSuccess": "开票申请已提交",
    "withdrawSuccess": "开票申请已撤回",
    "uploadSuccess": "发票文件已上传",
    "statuses": {
      "pending": "待开票",
      "issued": "已开票",
      "rejected": "已拒绝",
      "withdrawn": "已撤回"
    },
    "errors": {
      "INVOICE_RECHARGE_UNAVAILABLE": "所选充值不可开票或已被使用",
      "INVOICE_FILE_UNAVAILABLE": "发票文件不可用",
      "INVOICE_FILE_TOO_LARGE": "发票文件过大",
      "INVOICE_AMOUNT_BELOW_MINIMUM": "所选金额低于最低开票金额 {minimum}",
      "INVOICE_AMOUNT_ABOVE_MAXIMUM": "所选金额高于最高开票金额 {maximum}",
      "INVOICE_WITHDRAW_UNAVAILABLE": "该开票申请当前不可撤回",
      "INVOICE_WITHDRAW_EXPIRED": "开票申请已超过 6 小时，无法撤回",
      "INVOICE_UPLOAD_UNAVAILABLE": "该开票申请当前不可上传发票",
      "INVOICE_MIN_AMOUNT_INVALID": "最低开票金额必须大于或等于 0",
      "INVOICE_MAX_AMOUNT_INVALID": "最高开票金额必须大于或等于 0",
      "INVOICE_AMOUNT_RANGE_INVALID": "最高开票金额必须大于或等于最低开票金额",
      "INVOICE_TITLE_REQUIRED": "请填写发票抬头",
      "INVOICE_RECHARGES_REQUIRED": "请选择至少一笔充值",
      "INVOICE_NOT_FOUND": "开票记录不存在"
    }
  },
  "keys": {
    "imageKey": "绘图用",
    "setImageKey": "设为绘图用",
    "imageKeyCurrent": "当前用于 {token} 的绘图密钥",
    "imageKeyIneligible": "仅活跃、未过期、绑定允许生图的 OpenAI 分组的密钥可设为绘图用",
    "imageKeySetSuccess": "已设为绘图用密钥",
    "failedToSetImageKey": "设置绘图用密钥失败",
    "ccsImport": {
      "title": "导入到 CCS",
      "description": "导入前请选择要写入 CCS 的访问镜像。",
      "clientSection": "客户端类型",
      "mirrorSection": "访问镜像",
      "currentSite": "主站",
      "defaultMirror": "默认",
      "selectMirror": "请选择一个访问镜像"
    }
  },
  "admin": {
    "groups": {
      "imagePricing": {
        "responsesImageGenerationRedirect": "Responses 图片分流目标",
        "responsesImageGenerationRedirectHint": "显式携带 image_generation 工具的 /responses 请求会转发到所选 OpenAI 图片分组；Codex 图片桥接也可借此启用。计价仍归属当前 API Key 分组。",
        "responsesImageGenerationRedirectGroupPlaceholder": "选择 OpenAI 图片分组",
        "noOpenAIImageGroupsAvailable": "暂无可用的 OpenAI 图片生成分组"
      }
    },
    "channels": {
      "form": {
        "codexImageGenerationBridgeHint": "开启后，OpenAI 分组的 Codex /responses 文本请求可能会被自动注入 image_generation 工具。可配合分组上的 Responses 图片分流目标使用。"
      }
    },
    "accounts": {
      "openai": {
        "codexCLIOnlyAllowClaudeCode": "额外放行 Claude Code 的 Codex 插件",
        "codexCLIOnlyAllowClaudeCodeDesc": "仅在上方开关开启时生效。额外放行通过 Claude Code 的 Codex 插件发起的请求（精确匹配 originator=Claude Code），不影响对其他非官方客户端的拦截。",
        "codexImageGenerationBridge": "Codex 图片生成桥接",
        "codexImageGenerationBridgeDesc": "账号级策略优先于渠道和全局配置。仅控制 Codex 走 /responses 文本端点时是否注入 image_generation 工具；可配合分组上的 Responses 图片分流目标使用，不影响独立图片生成接口。",
        "codexImageGenerationBridgeInherit": "跟随渠道",
        "codexImageGenerationBridgeInheritDesc": "不写入账号覆盖，继续使用渠道或全局策略。",
        "codexImageGenerationBridgeEnabled": "强制开启",
        "codexImageGenerationBridgeEnabledDesc": "允许 Codex /responses 请求获得图片工具注入。",
        "codexImageGenerationBridgeDisabled": "强制关闭",
        "codexImageGenerationBridgeDisabledDesc": "阻断 Codex /responses 的图片工具注入。",
        "codexImageGenerationBridgeBadgeInherit": "渠道策略",
        "codexImageGenerationBridgeBadgeEnabled": "账号开启",
        "codexImageGenerationBridgeBadgeDisabled": "账号关闭"
      }
    },
    "supportTickets": {
      "title": "工单管理",
      "description": "查看用户工单、回复并更新处理状态",
      "create": "新建用户工单",
      "edit": "更新工单",
      "view": "查看工单",
      "user": "用户",
      "userId": "用户 ID",
      "searchPlaceholder": "搜索标题、邮箱或用户名...",
      "empty": "暂无工单",
      "emptyDescription": "用户提交工单后会显示在这里。",
      "created": "工单已创建",
      "updated": "工单已更新",
      "failedToLoad": "加载工单失败",
      "failedToLoadDetail": "加载工单详情失败",
      "failedToCreate": "创建工单失败",
      "failedToUpdate": "更新工单失败"
    },
    "settings": {
      "userIdMaintenance": {
        "title": "用户 ID 维护",
        "description": "调整 users.id 自增序列，或在全表引用中迁移单个用户 ID。",
        "warning": "这是高风险维护操作。请先确认没有相关用户正在操作；修改用户 ID 会同步更新已知引用和缓存，但外部系统中保存的旧 ID 不会自动更新。",
        "maxUserId": "当前最大 ID",
        "nextUserId": "下一个自增 ID",
        "sequenceName": "序列名称",
        "setNextTitle": "设置当前自增值",
        "setNextHint": "只能设置为大于当前下一个 ID 且大于当前最大用户 ID 的值，不能回退。",
        "newNextUserId": "新的下一个用户 ID",
        "setNextValidation": "请输入一个大于当前下一个 ID 和当前最大用户 ID 的正整数。",
        "setNextButton": "更新自增值",
        "setNextSuccess": "用户 ID 自增值已更新",
        "changeTitle": "整体修改用户 ID",
        "changeHint": "会迁移 users 主键以及已知外键/非外键引用。不能修改当前登录管理员自己的 ID。",
        "oldUserId": "原用户 ID",
        "newUserId": "新用户 ID",
        "confirmation": "确认文本",
        "confirmationHint": "请输入 {text} 后才能执行。",
        "changeValidation": "请填写有效且不同的原/新用户 ID，并输入正确确认文本。",
        "changeConfirm": "确认要整体修改该用户 ID？此操作会更新多张表中的引用。",
        "changeButton": "修改用户 ID",
        "changeSuccess": "用户 ID 已修改",
        "submitting": "处理中..."
      },
      "site": {
        "supportTicketEntryVisibility": "工单/意见入口可见性",
        "supportTicketEntryVisibilityHint": "选择“全部用户可见”时，用户侧边栏会显示工单入口；选择“仅管理员可见”时，仅保留后台工单管理入口。",
        "supportTicketEntryVisibilityAll": "全部用户可见",
        "supportTicketEntryVisibilityAdmin": "仅管理员可见",
        "siteSubtitleHint": "显示在登录和注册页面，支持多行",
        "siteDescription": "站点简介",
        "siteDescriptionHint": "显示在首页副标题位置，支持多行；留空时首页使用站点副标题",
        "siteDescriptionPlaceholder": "尽量保证稳定的小站，不面向中国大陆地区提供服务",
        "apiKeyPageNotice": "API 密钥页公告",
        "apiKeyPageNoticeHint": "显示在 API Keys 页面顶部，支持多行 Markdown，留空则不显示",
        "apiKeyPageNoticePlaceholder": "例如：如果速度不理想，换用网络优化节点 [code-us.urpg.net](https://code-us.urpg.net)\n请不要公开分享自己的 API Key",
        "customEndpoints": {
          "remove": "删除",
          "moveUp": "上移",
          "moveDown": "下移"
        },
        "customHomeLinks": {
          "title": "首页自定义链接",
          "description": "添加展示在首页的自定义链接，支持外部 http(s) 地址或内部 / 开头路径",
          "itemLabel": "链接 #{n}",
          "linkTitle": "标题",
          "titlePlaceholder": "如：使用文档",
          "url": "URL",
          "urlPlaceholder": "https://docs.example.com 或 /dashboard",
          "descriptionLabel": "描述",
          "descriptionPlaceholder": "如：快速了解 API 使用方式",
          "openInNewWindow": "新窗口打开",
          "add": "添加链接",
          "remove": "删除",
          "moveUp": "上移",
          "moveDown": "下移"
        }
      },
      "customMenu": {
        "description": "添加自定义页面到侧边栏导航，可选择内嵌显示或外部新窗口打开。",
        "urlPlaceholder": "https://example.com/page?key={token}",
        "openMode": "打开方式",
        "openModeEmbedded": "内嵌页面",
        "openModeExternal": "外部页面",
        "openModeExternalConfirm": "外部页面 + 确认弹窗",
        "group": "菜单分组",
        "groupPlaceholder": "留空则显示在默认分组"
      }
    }
  },
  "customPage": {
    "externalConfirmTitle": "打开外部页面",
    "externalConfirmMessage": "即将打开外部页面“{label}”：{url}。是否继续？",
    "externalConfirmOpen": "继续打开",
    "externalOpenFailed": "外部页面链接无法打开，请检查菜单 URL 配置。",
    "imgKeyMissingTitle": "暂无可用绘图密钥",
    "imgKeyMissingDesc": "请先在 API 密钥页面创建或选择一个支持生图的 OpenAI 分组密钥。",
    "imgKeyErrorTitle": "绘图密钥加载失败",
    "imgKeyErrorDesc": "无法解析 {token}，请稍后重试。"
  }
} as const
