import type { codes as english } from "../en/codes.ts";

export const codes: typeof english = {
  "code-setup.title": "添加一次性代码",
  "code-setup.detail.issuer-account": "选择 {issuer} 上 {account} 的密码。",
  "code-setup.detail.issuer": "选择 {issuer} 的密码。",
  "code-setup.detail.account": "选择 {account} 的密码。",
  "code-setup.detail.unnamed": "选择此代码所属的密码。",
  "code-setup.empty":
    "没有与此代码匹配的已保存密码。请搜索已有密码或添加新密码。",
  "code-setup.new": "新密码",
  "code-setup.cancel": "取消",
  "code-setup.replace.title": "替换一次性代码？",
  "code-setup.replace.detail":
    "“{label}” 已包含一次性代码。新代码将替换原有代码。",
  "code-setup.replace.confirm": "替换",
  "code-setup.added": "一次性代码已添加。",
  "code-setup.error": "Ravenpass 无法添加一次性代码。请重试。",
  "code-setup.error.dismiss":
    "Ravenpass 无法丢弃该一次性代码。系统将在五分钟内将其遗忘。",
};
