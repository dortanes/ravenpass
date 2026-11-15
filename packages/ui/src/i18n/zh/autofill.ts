import type { autofill as english } from "../en/autofill.ts";

export const autofill: typeof english = {
  "autofill.entry.unlock": "解锁",
  "autofill.entry.search": "搜索 Ravenpass",
  "autofill.provider.subtitle": "密码、通行密钥、验证码",
  "autofill.unlock.description": "解锁密码库以填充密码。",
  "autofill.verify.description": "输入 PIN 码以确认是你本人。",
  "autofill.search.title": "选择密码",
  "autofill.search.code-title": "选择一次性验证码",
  "autofill.search.passkey-title": "选择通行密钥或密码",
  "autofill.search.passkey-only-title": "选择通行密钥",
  "autofill.search.no-passkeys": "你为此网站保存的通行密钥会显示在这里。",
  "autofill.search.passkeys": "通行密钥",
  "autofill.search.for-site": "适用于此网站",
  "autofill.search.for-app": "适用于此应用",
  "autofill.search.everything": "所有密码",
  "autofill.search.error": "Ravenpass 无法搜索你的密码。请重试。",
  "autofill.add-site.title": "将 {site} 添加到“{label}”？",
  "autofill.add-site.detail": "此后你可以在 {site} 上填充此登录信息。",
  "autofill.add-site.confirm": "添加并填充",
  "autofill.add-site.error": "Ravenpass 无法将此网站添加到该项目。请重试。",
  "autofill.link.title": "用于此应用？",
  "autofill.link.detail": "此后 Ravenpass 会在此应用中建议“{label}”。",
  "autofill.link.confirm": "使用",
  "autofill.link.error": "Ravenpass 无法将此应用关联到该项目。请重试。",
  "autofill.no-code": "此项目没有一次性验证码。请选择其他项目。",
  "autofill.cancel": "取消",
  "autofill.close": "关闭",
  "autofill.wait.opening": "正在打开 Ravenpass…",
  "autofill.wait.unlock":
    "在 Ravenpass 中使用触控 ID、Mac 登录密码或 PIN 码确认是你本人。",
  "autofill.wait.verify.title": "确认是你本人",
  "autofill.wait.verify":
    "使用触控 ID 或 Mac 登录密码，或在 Ravenpass 中输入 PIN 码。",
  "autofill.failed.title": "出了点问题",
  "autofill.failed.unreachable":
    "Ravenpass 没有响应。请打开 Ravenpass 后重试。",
  "autofill.failed.outdated":
    "Ravenpass 在运行时已更新。请重新启动 Ravenpass 后重试。",
  "autofill.failed.not-found": "此项目已不在你的密码库中。",
  "autofill.failed.sign-in": "Ravenpass 无法用此通行密钥登录。请重试。",
  "autofill.failed.save-passkey": "Ravenpass 无法保存此通行密钥。请重试。",
  "autofill.failed.unverifiable":
    "此网站要求确认是你本人。请在 Ravenpass 中设置 PIN 码或打开生物识别，然后重试。",
  "autofill.failed.unsupported":
    "Ravenpass 无法创建此网站要求的通行密钥。请使用其他登录方式。",
  "autofill.failed.other": "Ravenpass 无法完成此请求。请重试。",
};
