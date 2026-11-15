import type { extension as english } from "../en/extension.ts";

export const extension: typeof english = {
  "extension.link.title": "关联到 Ravenpass",
  "extension.link.description":
    "在这台电脑的 Ravenpass 中，打开“设置”→“自动填充”，选择“关联扩展”，然后复制密钥。",
  "extension.link.key": "连接密钥",
  "extension.link.paste": "粘贴密钥",
  "extension.link.action": "关联",
  "extension.link.busy": "正在关联…",
  "extension.link.note": "此密钥只能使用一次，5 分钟后失效。",
  "extension.link.error.not-key":
    "这不是 Ravenpass 密钥。请从 Ravenpass 中重新复制。",
  "extension.link.error.expired":
    "此密钥已失效。请在 Ravenpass 中创建新密钥并粘贴到这里。",
  "extension.link.error.not-open":
    "这台电脑上的 Ravenpass 没有打开。请打开后重试。",
  "extension.link.error.failed": "Ravenpass 无法关联此扩展。请重试。",
  "extension.link.error.reference": "版本 {version} · 参考代码 {code}",
  "extension.link.error.clipboard":
    "Ravenpass 无法读取剪贴板。请将密钥粘贴到输入框中。",
  "extension.linked.title": "已关联到 Ravenpass",
  "extension.linked.description": "Ravenpass 会在登录输入框中建议密码。",
  "extension.linked.desktop": "这台电脑上的 Ravenpass",
  "extension.linked.desktop.unlocked": "已解锁",
  "extension.linked.desktop.locked": "已锁定。解锁 Ravenpass 后才能填充密码。",
  "extension.linked.desktop.not-open":
    "未打开。打开 Ravenpass 后才能填充密码。",
  "extension.linked.since": "关联时间",
  "extension.chrome-autofill": "关闭 Chrome 的自动填充",
  "extension.chrome-autofill.detail":
    "这样 Chrome 自己的建议就不会盖住 Ravenpass 的建议。",
  "extension.chrome-autofill.error": "Ravenpass 无法更改此设置。请重试。",
  "extension.unlink.action": "取消关联",
  "extension.unlink.title": "取消关联此扩展？",
  "extension.unlink.detail":
    "Ravenpass 将不再在此浏览器中建议密码。若要重新关联，需要从 Ravenpass 获取新密钥。",
  "extension.unlink.cancel": "取消",
  "extension.unlink.error": "Ravenpass 无法取消关联此扩展。请重试。",
  "extension.unlinked": "此扩展已在 Ravenpass 中取消关联。重新关联后才能使用。",
  "extension.menu.label": "{site} 的密码",
  "extension.menu.codes.label": "{site} 的一次性验证码",
  "extension.menu.locked.title": "Ravenpass 已锁定",
  "extension.menu.locked.detail": "解锁后才能填充密码。",
  "extension.menu.locked.action": "解锁",
  "extension.menu.unlock.error":
    "Ravenpass 无法解锁。请打开 Ravenpass 应用后重试。",
  "extension.menu.not-open.title": "Ravenpass 没有打开",
  "extension.menu.not-open.detail": "在这台电脑上打开它，才能填充密码。",
  "extension.menu.empty.title": "此网站没有密码",
  "extension.menu.empty.detail":
    "你在 Ravenpass 中为此网站保存的密码会显示在这里。",
  "extension.menu.codes.empty.title": "此网站没有验证码",
  "extension.menu.codes.empty.detail":
    "你在 Ravenpass 中为此网站设置的一次性验证码会显示在这里。",
  "extension.menu.cards.label": "Ravenpass 中的卡片",
  "extension.menu.cards.locked.detail": "解锁后才能填充卡片。",
  "extension.menu.cards.not-open.detail": "在这台电脑上打开它，才能填充卡片。",
  "extension.menu.cards.empty.title": "Ravenpass 中没有卡片",
  "extension.menu.cards.empty.detail":
    "你在 Ravenpass 中添加的卡片会显示在这里。",
  "extension.menu.cards.error": "Ravenpass 无法填充这张卡片。请重试。",
  "extension.context.passwords": "显示密码",
  "extension.context.codes": "显示一次性验证码",
  "extension.context.cards": "显示卡片",
  "extension.menu.code.waiting": "下一个验证码将在 {seconds} 秒后填充",
  "extension.menu.passkey.detail": "通行密钥 · {account}",
  "extension.menu.hint.show": "显示",
  "extension.menu.hint.fill": "填充",
  "extension.menu.hint.open": "打开",
  "extension.menu.hint.upload": "上传",
  "extension.card.sign-in.title": "登录 {site}",
  "extension.card.code.title": "{site} 的验证码",
  "extension.card.sign-in-as": "以 {account} 的身份登录",
  "extension.card.close": "关闭",
  "extension.confirm.password.title": "在此处填充这个密码？",
  "extension.confirm.code.title": "在此处填充这个验证码？",
  "extension.confirm.page": "此页面",
  "extension.confirm.saved": "保存用于",
  "extension.confirm.insecure-page":
    "此页面没有使用安全连接。只有在你信任它时才填充。",
  "extension.confirm.other-site":
    "此页面不是你保存时对应的网站。只有在你信任它时才填充。",
  "extension.confirm.remember": "为此项目记住这个网站",
  "extension.confirm.cancel": "取消",
  "extension.confirm.fill": "填充",
  "extension.fill.confirm-on-device.title": "在 Mac 上确认",
  "extension.fill.confirm-on-device.detail":
    "用触控 ID 或 Mac 登录密码批准填充。",
  "extension.fill.confirm-in-ravenpass.title": "在 Ravenpass 中确认",
  "extension.fill.confirm-in-ravenpass.detail":
    "在 Mac 上的 Ravenpass 窗口中输入 PIN 码。",
  "extension.fill.declined": "填充已在 Mac 上取消。",
  "extension.fill.unverifiable":
    "请在 Ravenpass 中打开生物识别或设置 PIN 码，以便确认填充。",
  "extension.offer.locked.detail": "解锁后才能保存此密码。",
  "extension.offer.card.locked.detail": "解锁后才能保存这张卡片。",
  "extension.offer.not-open.detail": "在这台电脑上打开它，才能保存密码。",
  "extension.passkey.sign-in.title": "用通行密钥登录",
  "extension.passkey.save.title": "保存通行密钥",
  "extension.passkey.list.label": "{site} 的通行密钥",
  "extension.passkey.target.add": "添加到“{label}”",
  "extension.passkey.save": "保存",
  "extension.passkey.elsewhere": "其他选项",
  "extension.passkey.excluded.title": "Ravenpass 已有此账户的通行密钥",
  "extension.passkey.excluded.detail": "请改用它登录。",
  "extension.passkey.locked.detail": "解锁后才能使用通行密钥。",
  "extension.passkey.not-open.detail": "在这台电脑上打开它，才能使用通行密钥。",
  "extension.passkey.confirm-on-device.title": "在 Mac 上确认",
  "extension.passkey.confirm-on-device.detail":
    "用触控 ID 或 Mac 登录密码批准此请求。",
  "extension.passkey.confirm-in-ravenpass.title": "在 Ravenpass 中确认",
  "extension.passkey.confirm-in-ravenpass.detail":
    "在 Ravenpass 窗口中输入 PIN 码。",
  "extension.passkey.declined": "无法确认是你本人。请重试或使用其他设备。",
  "extension.passkey.unverifiable":
    "请在 Ravenpass 中打开生物识别或设置 PIN 码，以便使用通行密钥。",
  "extension.passkey.full":
    "此项目存放的通行密钥已达上限。请将通行密钥另存为新项目。",
  "extension.passkey.sign-in.error": "Ravenpass 无法用此通行密钥登录。请重试。",
  "extension.passkey.save.error": "Ravenpass 无法保存此通行密钥。请重试。",
  "extension.upload.menu": "上传身份文件…",
  "extension.upload.identities.label": "{site} 的身份文件",
  "extension.upload.files.label": "{identity} 的文件",
  "extension.upload.back": "所有身份",
  "extension.upload.photo": "照片",
  "extension.upload.file.detail": "{name} · {format} 文件",
  "extension.upload.identity.empty": "没有照片或扫描件",
  "extension.upload.empty.title": "还没有身份",
  "extension.upload.empty.detail":
    "你添加到 Ravenpass 的身份中的照片和证件扫描件会显示在这里。",
  "extension.upload.no-destination.title": "无法在这里上传文件",
  "extension.upload.no-destination.detail": "请右键点按文件输入框或拖放区域。",
  "extension.upload.locked.detail": "解锁后才能上传文件。",
  "extension.upload.not-open.detail": "在这台电脑上打开它，才能上传文件。",
  "extension.upload.unlinked.title": "此扩展未关联",
  "extension.upload.unlinked.detail":
    "使用工具栏中的扩展按钮将它关联到 Ravenpass。",
  "extension.upload.confirm-on-device.title": "在 Mac 上确认",
  "extension.upload.confirm-on-device.detail":
    "用触控 ID 或 Mac 登录密码批准此请求。",
  "extension.upload.confirm-in-ravenpass.title": "在 Ravenpass 中确认",
  "extension.upload.confirm-in-ravenpass.detail":
    "在 Ravenpass 窗口中输入 PIN 码。",
  "extension.upload.declined.title": "共享已被拒绝",
  "extension.upload.declined.detail":
    "没有共享任何内容。重新选择文件即可再试。",
  "extension.upload.unverifiable.title": "Ravenpass 无法确认是你本人",
  "extension.upload.unverifiable.detail":
    "请在 Ravenpass 中打开生物识别或设置 PIN 码，以便上传文件。",
  "extension.upload.not-taken.title": "页面没有接受该文件",
  "extension.upload.not-taken.detail": "请右键点按文件输入框本身后重试。",
  "extension.upload.identities.error": "Ravenpass 无法列出你的身份。请重试。",
  "extension.upload.error": "Ravenpass 无法上传此文件。请重试。",
};
