import type { merge as english } from "../en/merge.ts";

export const merge: typeof english = {
  "workspace.detail.merge": "与另一个密码合并",
  "merge.title": "合并密码",
  "merge.choose.detail":
    "选择要合并到 {name} 的密码。合并后该密码将移至废纸篓。",
  "merge.resolve.detail":
    "在 {kept} 与 {other} 存在冲突之处选择要保留的内容。其余内容将被合并，{other} 将移至废纸篓。",
  "merge.resolve.none": "{kept} 将包含 {other} 的内容，{other} 将移至废纸篓。",
  "merge.reveal": "全部显示",
  "merge.conceal": "全部隐藏",
  "merge.both": "两者均保留",
  "merge.both.detail": "依次保留两条备注",
  "merge.passkeys": "{count, plural, other {# 个通行密钥将移动到此密码。}}",
  "merge.dropped.websites":
    "{count, plural, other {# 个网站超出数量限制已被忽略。}}",
  "merge.dropped.tags":
    "{count, plural, other {# 个标签超出数量限制已被忽略。}}",
  "merge.back": "返回",
  "merge.cancel": "取消",
  "merge.confirm": "合并",
  "merge.busy": "正在合并…",
  "merge.done": "密码已合并",
  "merge.error": "Ravenpass 无法合并这些密码。请重试。",
  "merge.error.read": "Ravenpass 无法打开该密码。请重新选择。",
};
