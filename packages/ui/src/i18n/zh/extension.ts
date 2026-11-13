import type { extension as english } from "../en/extension.ts";

export const extension: typeof english = {
  "extension.link.title": "关联到 Ravenpass",
  "extension.link.description":
    "在此电脑上的 Ravenpass 中，打开“设置”→“自动填充”，选择“关联扩展”，然后复制密钥。",
  "extension.link.key": "连接密钥",
  "extension.link.paste": "粘贴密钥",
  "extension.link.action": "关联",
  "extension.link.busy": "正在关联…",
  "extension.link.note": "该密钥仅可使用一次，5 分钟后过期。",
  "extension.link.error.not-key":
    "这不是有效的 Ravenpass 密钥。请从 Ravenpass 重新复制。",
  "extension.link.error.expired":
    "此密钥已过期。请在 Ravenpass 中生成新密钥并粘贴到此处。",
  "extension.link.error.not-open":
    "此电脑上的 Ravenpass 尚未打开。请打开后重试。",
  "extension.link.error.failed": "Ravenpass 无法关联此扩展。请重试。",
  "extension.link.error.reference": "版本 {version} · 参考码 {code}",
  "extension.link.error.clipboard":
    "Ravenpass 无法读取剪贴板。请直接将密钥粘贴到输入框中。",
  "extension.linked.title": "已关联到 Ravenpass",
  "extension.linked.description": "Ravenpass 将在登录框中建议密码。",
  "extension.linked.desktop": "此电脑上的 Ravenpass",
  "extension.linked.desktop.unlocked": "已解锁",
  "extension.linked.desktop.locked": "已锁定。解锁 Ravenpass 以填充密码。",
  "extension.linked.desktop.not-open": "未打开。打开 Ravenpass 以填充密码。",
  "extension.linked.since": "关联时间",
  "extension.chrome-autofill": "关闭 Chrome 的自动填充",
  "extension.chrome-autofill.detail":
    "避免 Chrome 自带的建议覆盖 Ravenpass 的建议。",
  "extension.chrome-autofill.error": "Ravenpass 无法更改此设置。请重试。",
  "extension.unlink.action": "解除关联",
  "extension.unlink.title": "解除此扩展的关联？",
  "extension.unlink.detail":
    "Ravenpass 将停止在此浏览器中建议密码。如需重新关联，您需要从 Ravenpass 获取新密钥。",
  "extension.unlink.cancel": "取消",
  "extension.unlink.error": "Ravenpass 无法解除此扩展的关联。请重试。",
  "extension.unlinked":
    "此扩展已在 Ravenpass 中被解除关联。如需使用请重新关联。",
  "extension.menu.label": "{site} 的密码",
  "extension.menu.codes.label": "{site} 的一次性代码",
  "extension.menu.locked.title": "Ravenpass 已锁定",
  "extension.menu.locked.detail": "解锁以填充密码。",
  "extension.menu.locked.action": "解锁",
  "extension.menu.unlock.error":
    "Ravenpass 无法解锁。请打开 Ravenpass 应用后重试。",
  "extension.menu.not-open.title": "Ravenpass 未打开",
  "extension.menu.not-open.detail": "在此电脑上打开 Ravenpass 以填充密码。",
  "extension.menu.empty.title": "无此网站的密码",
  "extension.menu.empty.detail":
    "您在 Ravenpass 中为此网站保存的密码会显示在此处。",
  "extension.menu.codes.empty.title": "无此网站的代码",
  "extension.menu.codes.empty.detail":
    "您在 Ravenpass 中为此网站设置的一次性代码会显示在此处。",
  "extension.menu.cards.label": "Ravenpass 中的卡片",
  "extension.menu.cards.locked.detail": "解锁以填充卡片。",
  "extension.menu.cards.not-open.detail": "在此电脑上打开以填充卡片。",
  "extension.menu.cards.empty.title": "Ravenpass 中没有卡片",
  "extension.menu.cards.empty.detail":
    "您在 Ravenpass 中添加的卡片会显示在此处。",
  "extension.menu.cards.error": "Ravenpass 无法填充此卡片。请重试。",
  "extension.context.passwords": "显示密码",
  "extension.context.codes": "显示一次性代码",
  "extension.context.cards": "显示卡片",
  "extension.menu.code.waiting": "下一个代码将在 {seconds} 秒后填充",
  "extension.menu.passkey.detail": "通行密钥 · {account}",
  "extension.menu.hint.show": "显示",
  "extension.menu.hint.fill": "填充",
  "extension.menu.hint.open": "打开",
  "extension.menu.hint.upload": "上传",
  "extension.card.sign-in.title": "登录到 {site}",
  "extension.card.code.title": "{site} 的代码",
  "extension.card.sign-in-as": "作为 {account} 登录",
  "extension.card.close": "关闭",
  "extension.confirm.password.title": "在此填充该密码？",
  "extension.confirm.code.title": "在此填充该代码？",
  "extension.confirm.page": "当前页面",
  "extension.confirm.saved": "保存于",
  "extension.confirm.insecure-page":
    "此页面未采用安全连接。仅在您信任它时才进行填充。",
  "extension.confirm.other-site":
    "此页面不是您为其保存凭据的网站。仅在您信任它时才进行填充。",
  "extension.confirm.remember": "为此项目记住此网站",
  "extension.confirm.cancel": "取消",
  "extension.confirm.fill": "填充",
  "extension.fill.confirm-on-device.title": "在 Mac 上确认",
  "extension.fill.confirm-on-device.detail":
    "使用触控 ID 或 Mac 密码批准填充。",
  "extension.fill.confirm-in-ravenpass.title": "在 Ravenpass 中确认",
  "extension.fill.confirm-in-ravenpass.detail":
    "在 Mac 上的 Ravenpass 窗口中输入您的 PIN 码。",
  "extension.fill.declined": "已在 Mac 上取消填充。",
  "extension.fill.unverifiable":
    "在 Ravenpass 中开启生物识别或设置 PIN 码以确认填充。",
  "extension.offer.locked.detail": "解锁以保存此密码。",
  "extension.offer.card.locked.detail": "解锁以保存此卡片。",
  "extension.offer.not-open.detail": "在此电脑上打开以保存密码。",
  "extension.passkey.sign-in.title": "使用通行密钥登录",
  "extension.passkey.save.title": "保存通行密钥",
  "extension.passkey.list.label": "{site} 的通行密钥",
  "extension.passkey.target.add": "添加到 “{label}”",
  "extension.passkey.save": "保存",
  "extension.passkey.elsewhere": "其他选项",
  "extension.passkey.excluded.title": "Ravenpass 已有此账户的通行密钥",
  "extension.passkey.excluded.detail": "请改用已有通行密钥登录。",
  "extension.passkey.locked.detail": "解锁以使用通行密钥。",
  "extension.passkey.not-open.detail": "在此电脑上打开以使用通行密钥。",
  "extension.passkey.confirm-on-device.title": "在 Mac 上确认",
  "extension.passkey.confirm-on-device.detail":
    "使用触控 ID 或 Mac 密码批准请求。",
  "extension.passkey.confirm-in-ravenpass.title": "在 Ravenpass 中确认",
  "extension.passkey.confirm-in-ravenpass.detail":
    "在 Ravenpass 窗口中输入您的 PIN 码。",
  "extension.passkey.declined": "无法确认您的身份。请重试或使用其他设备。",
  "extension.passkey.unverifiable":
    "在 Ravenpass 中开启生物识别或设置 PIN 码以使用通行密钥。",
  "extension.passkey.full":
    "此项目容纳的通行密钥数量已达上限。请将该通行密钥保存为新项目。",
  "extension.passkey.sign-in.error":
    "Ravenpass 无法使用此通行密钥登录。请重试。",
  "extension.passkey.save.error": "Ravenpass 无法保存此通行密钥。请重试。",
  "extension.upload.menu": "上传身份文件…",
  "extension.upload.identities.label": "{site} 的身份文件",
  "extension.upload.files.label": "{identity} 的文件",
  "extension.upload.back": "所有身份",
  "extension.upload.photo": "照片",
  "extension.upload.file.detail": "{name} · {format} 文件",
  "extension.upload.identity.empty": "无照片或扫描件",
  "extension.upload.empty.title": "暂无身份信息",
  "extension.upload.empty.detail":
    "您在 Ravenpass 中添加的身份信息的照片和扫描件将显示在此处。",
  "extension.upload.no-destination.title": "无法在此处上传文件",
  "extension.upload.no-destination.detail": "请右键单击文件输入框或拖拽区域。",
  "extension.upload.locked.detail": "解锁以上传文件。",
  "extension.upload.not-open.detail": "在此电脑上打开以上传文件。",
  "extension.upload.unlinked.title": "此扩展未关联",
  "extension.upload.unlinked.detail":
    "请使用工具栏中的扩展按钮将其关联到 Ravenpass。",
  "extension.upload.confirm-on-device.title": "在 Mac 上确认",
  "extension.upload.confirm-on-device.detail":
    "使用触控 ID 或 Mac 密码批准请求。",
  "extension.upload.confirm-in-ravenpass.title": "在 Ravenpass 中确认",
  "extension.upload.confirm-in-ravenpass.detail":
    "在 Ravenpass 窗口中输入您的 PIN 码。",
  "extension.upload.declined.title": "共享已被拒绝",
  "extension.upload.declined.detail": "未共享任何内容。请重新选择文件重试。",
  "extension.upload.unverifiable.title": "Ravenpass 无法确认您的身份",
  "extension.upload.unverifiable.detail":
    "在 Ravenpass 中开启生物识别或设置 PIN 码以上传文件。",
  "extension.upload.not-taken.title": "页面未接受该文件",
  "extension.upload.not-taken.detail": "请右键单击文件输入字段本身重试。",
  "extension.upload.identities.error":
    "Ravenpass 无法列出您的身份信息。请重试。",
  "extension.upload.error": "Ravenpass 无法上传此文件。请重试。",
};
