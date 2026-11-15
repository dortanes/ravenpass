import type { trash as english } from "../en/trash.ts";

export const trash: typeof english = {
  "workspace.trash.moved": "已移到废纸篓",
  "workspace.trash.undo": "撤销",
  "workspace.error.restore":
    "Ravenpass 无法恢复此项目。请在“密码库”设置的废纸篓中恢复。",

  "settings.trash.heading": "废纸篓",
  "settings.trash.summary": "恢复已删除的项目，或将其永久删除。",
  "settings.trash.retention": "永久删除项目的期限",
  "settings.trash.days": "{count, plural, other {# 天}}",
  "settings.trash.empty": "你删除的项目会显示在这里。",
  "settings.trash.left": "{count, plural, other {# 天后永久删除}}",
  "settings.trash.left.today": "今天永久删除",
  "settings.trash.detail": "{detail} · {left}",
  "settings.trash.restore": "恢复",
  "settings.trash.delete": "删除…",
  "settings.trash.delete.title": "永久删除 {name}？",
  "settings.trash.delete.detail":
    "此操作无法撤销。之前保存的加密副本和备份中仍有它。",
  "settings.trash.delete.confirm": "永久删除",
  "settings.trash.empty-all": "清空废纸篓…",
  "settings.trash.empty-all.title":
    "{count, plural, other {永久删除 # 个项目？}}",
  "settings.trash.empty-all.confirm": "清空废纸篓",
  "settings.trash.shorten.title":
    "{count, plural, other {缩短期限会立即永久删除 # 个项目。要继续吗？}}",
  "settings.trash.shorten.confirm": "删除并更改",
  "settings.trash.error.read":
    "Ravenpass 无法打开废纸篓。请重新打开密码库后重试。",
  "settings.trash.error.restore": "Ravenpass 无法恢复此项目。请重试。",
  "settings.trash.error.delete": "Ravenpass 无法从废纸篓中删除。请重试。",
  "settings.trash.error.retention":
    "Ravenpass 无法更改废纸篓保留项目的期限。请重试。",
};
