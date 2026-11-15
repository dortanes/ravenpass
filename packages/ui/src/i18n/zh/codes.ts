import type { codes as english } from "../en/codes.ts";

export const codes: typeof english = {
  "code-setup.title": "添加一次性验证码",
  "code-setup.detail.issuer-account": "选择 {issuer} 上 {account} 的密码。",
  "code-setup.detail.issuer": "选择 {issuer} 的密码。",
  "code-setup.detail.account": "选择 {account} 的密码。",
  "code-setup.detail.unnamed": "选择此验证码所属的密码。",
  "code-setup.empty":
    "没有已保存的密码与此验证码匹配。请搜索一个，或添加新密码。",
  "code-setup.new": "新密码",
  "code-setup.cancel": "取消",
  "code-setup.replace.title": "替换一次性验证码？",
  "code-setup.replace.detail": "“{label}”已有一次性验证码，新的会替换它。",
  "code-setup.replace.confirm": "替换",
  "code-setup.added": "已添加一次性验证码。",
  "code-setup.error": "Ravenpass 无法添加一次性验证码。请重试。",
  "code-setup.error.dismiss":
    "Ravenpass 无法丢弃这个一次性验证码。它会在五分钟内被遗忘。",
};
