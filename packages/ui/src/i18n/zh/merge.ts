import type { merge as english } from "../en/merge.ts";

export const merge: typeof english = {
  "workspace.detail.merge": "与另一个密码合并",
  "merge.title": "合并密码",
  "merge.choose.detail": "选择要合并到 {name} 的密码。合并后，它会移到废纸篓。",
  "merge.resolve.detail":
    "在 {kept} 与 {other} 不同的地方，选择 {kept} 保留哪一项。其余内容会合并，{other} 会移到废纸篓。",
  "merge.resolve.none": "{kept} 会接收 {other} 的内容，{other} 会移到废纸篓。",
  "merge.reveal": "全部显示",
  "merge.conceal": "全部隐藏",
  "merge.both": "两者都保留",
  "merge.both.detail": "两条备注都保留，依次排列",
  "merge.passkeys": "{count, plural, other {# 个通行密钥会移到此密码。}}",
  "merge.dropped.websites":
    "{count, plural, other {有 # 个网站放不下，将被省略。}}",
  "merge.dropped.tags":
    "{count, plural, other {有 # 个标签放不下，将被省略。}}",
  "merge.back": "返回",
  "merge.cancel": "取消",
  "merge.confirm": "合并",
  "merge.busy": "正在合并…",
  "merge.done": "密码已合并",
  "merge.error": "Ravenpass 无法合并这些密码。请重试。",
  "merge.error.read": "Ravenpass 无法打开该密码。请重新选择。",
};
