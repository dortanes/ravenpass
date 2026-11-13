import type { note as english } from "../en/note.ts";

export const note: typeof english = {
  "note.untitled": "未命名备注",
  "note.search.label": "搜索备注",
  "note.search.placeholder": "搜索名称和首行内容",

  "note.list.label": "已保存的备注",
  "note.list.empty.all": "您添加的备注会显示在此处。",
  "note.list.empty.recent": "您打开过的备注会显示在此处。",
  "note.list.empty.pinned": "您添加到个人收藏的备注会显示在此处。",
  "note.list.empty.search": "没有找到匹配的备注。",
  "note.preview.hidden": "文本已隐藏",
  "note.preview.empty": "无文本",

  "note.loading": "正在加载备注…",
  "note.detail.loading": "正在打开备注…",
  "note.empty.select.title": "选择一条备注",
  "note.empty.select.detail": "从列表中选择一条备注以查看内容。",
  "note.empty.first.title": "添加您的第一条备注",
  "note.empty.first.detail": "添加的备注将显示在列表中。",
  "note.empty.first.action": "添加备注",

  "note.words": "{count, plural, other {# 个单词}}",
  "note.body.empty": "此备注没有文本内容。",
  "note.hidden.detail": "文本已隐藏。选择 “显示备注” 以阅读内容。",
  "note.hidden.show": "显示备注",
  "note.hidden.hide": "隐藏备注",

  "note.field.label": "名称",
  "note.field.label.placeholder": "家中 Wi-Fi",
  "note.field.body": "文本",
  "note.field.hidden": "隐藏文本",

  "note.copy": "复制备注",
  "note.copied": "备注已复制。",

  "note.editor.new.title": "新建备注",
  "note.editor.edit.title": "编辑备注",
  "note.editor.create": "添加备注",
  "note.editor.requirement": "名称为必填项",

  "note.error.list": "Ravenpass 无法加载您的备注。请重启应用。",
  "note.error.close": "Ravenpass 无法关闭选中的备注。请锁定保险库。",
  "note.error.open": "无法打开此备注。请重新选择或重新打开 Ravenpass。",
  "note.error.copy": "Ravenpass 无法复制此备注。请打开该备注后重试。",
  "note.error.pin": "Ravenpass 无法将此备注添加到个人收藏。请重试。",
  "note.error.unpin": "Ravenpass 无法从个人收藏中移除此备注。请重试。",
  "note.error.save": "Ravenpass 无法保存此备注。请检查输入内容后重试。",
  "note.error.saved-partly": "备注已保存，但{detail}",
  "note.error.delete": "Ravenpass 无法删除此备注。请重试。",
  "note.error.duplicate": "Ravenpass 无法创建此备注的副本。请重试。",
  "note.error.deleted-partly": "备注已删除，但{detail}",
};
