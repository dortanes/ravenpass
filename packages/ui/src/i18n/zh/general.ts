import type { general as english } from "../en/general.ts";

export const general: typeof english = {
  "app.name": "Ravenpass",
  "language.label": "语言",
  "app.unavailable.title": "Ravenpass 不可用",
  "app.unavailable.detail": "请重新启动应用。",
  "app.opening.title": "正在打开 Ravenpass",
  "app.opening.detail": "正在检查你的密码库…",
  "app.error.title": "无法打开 Ravenpass",
  "app.error.detail": "无法打开你的密码库。请重试。",
  "app.error.retry": "重试",
  "app.error.capabilities":
    "Ravenpass 无法确认这台设备支持哪些功能，因此隐藏了部分功能。重新打开 Ravenpass 即可再试一次。",
  "app.error.setting-read":
    "Ravenpass 无法读取你的某项设置。重新启动 Ravenpass 即可再试一次。",
  "fill.error": "Ravenpass 无法填充此密码。请重试。",
  "fill-code.error": "Ravenpass 无法填充此验证码。请重试。",
};
