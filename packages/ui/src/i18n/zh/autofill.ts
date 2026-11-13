import type { autofill as english } from "../en/autofill.ts";

export const autofill: typeof english = {
  "autofill.entry.unlock": "解锁",
  "autofill.entry.search": "搜索 Ravenpass",
  "autofill.provider.subtitle": "密码、通行密钥和验证码",
  "autofill.unlock.description": "解锁您的保险库以填充密码。",
  "autofill.verify.description": "输入您的 PIN 码以确认身份。",
  "autofill.search.title": "选择密码",
  "autofill.search.code-title": "选择一次性代码",
  "autofill.search.passkey-title": "选择通行密钥或密码",
  "autofill.search.passkey-only-title": "选择通行密钥",
  "autofill.search.no-passkeys": "您为此网站保存的通行密钥会显示在此处。",
  "autofill.search.passkeys": "通行密钥",
  "autofill.search.for-site": "此网站可用",
  "autofill.search.for-app": "此应用可用",
  "autofill.search.everything": "所有密码",
  "autofill.search.error": "Ravenpass 无法搜索您的密码。请重试。",
  "autofill.add-site.title": "将 {site} 添加到 “{label}”？",
  "autofill.add-site.detail": "此后您可以在 {site} 上填充此登录信息。",
  "autofill.add-site.confirm": "添加并填充",
  "autofill.add-site.error": "Ravenpass 无法将此网站添加到该项目。请重试。",
  "autofill.link.title": "在此应用中使用？",
  "autofill.link.detail": "Ravenpass 今后将在此应用中推荐 “{label}”。",
  "autofill.link.confirm": "使用",
  "autofill.link.error": "Ravenpass 无法将此应用关联到该项目。请重试。",
  "autofill.no-code": "该项目没有一次性代码。请选择其他项目。",
  "autofill.cancel": "取消",
  "autofill.close": "关闭",
  "autofill.wait.opening": "正在打开 Ravenpass…",
  "autofill.wait.unlock":
    "在 Ravenpass 中通过触控 ID、Mac 密码或 PIN 码确认您的身份。",
  "autofill.wait.verify.title": "确认身份",
  "autofill.wait.verify":
    "使用触控 ID、Mac 密码，或在 Ravenpass 中输入您的 PIN 码。",
  "autofill.failed.title": "出现问题",
  "autofill.failed.unreachable": "Ravenpass 未响应。请打开 Ravenpass 并重试。",
  "autofill.failed.outdated":
    "Ravenpass 在运行期间已更新。请重启 Ravenpass 并重试。",
  "autofill.failed.not-found": "该项目已不在您的保险库中。",
  "autofill.failed.sign-in": "Ravenpass 无法使用此通行密钥登录。请重试。",
  "autofill.failed.save-passkey": "Ravenpass 无法保存此通行密钥。请重试。",
  "autofill.failed.unverifiable":
    "该网站要求您确认身份。请在 Ravenpass 中设置 PIN 码或开启生物识别，然后重试。",
  "autofill.failed.unsupported":
    "Ravenpass 无法创建该网站所需的通行密钥。请使用其他登录方式。",
  "autofill.failed.other": "Ravenpass 无法完成此请求。请重试。",
};
