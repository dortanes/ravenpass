import type { note as english } from "../en/note.ts";

export const note: typeof english = {
  "note.untitled": "未命名笔记",
  "note.search.label": "搜索笔记",
  "note.search.placeholder": "搜索名称和首行",

  "note.list.label": "已保存的笔记",
  "note.list.empty.all": "你添加的笔记会显示在这里。",
  "note.list.empty.recent": "你打开过的笔记会显示在这里。",
  "note.list.empty.pinned": "你添加到收藏的笔记会显示在这里。",
  "note.list.empty.search": "没有与搜索匹配的笔记。",
  "note.preview.hidden": "文本已隐藏",
  "note.preview.empty": "无文本",

  "note.loading": "正在加载笔记…",
  "note.detail.loading": "正在打开笔记…",
  "note.empty.select.title": "选择一条笔记",
  "note.empty.select.detail": "从列表中选择一条笔记来阅读。",
  "note.empty.first.title": "添加第一条笔记",
  "note.empty.first.detail": "你添加的笔记会显示在列表中。",
  "note.empty.first.action": "添加笔记",

  "note.words": "{count, plural, other {# 个词}}",
  "note.body.empty": "这条笔记没有文本。",
  "note.hidden.detail": "文本已隐藏。选择“显示笔记”即可阅读。",
  "note.hidden.show": "显示笔记",
  "note.hidden.hide": "隐藏笔记",

  "note.field.label": "名称",
  "note.field.label.placeholder": "家里的 Wi-Fi",
  "note.field.body": "文本",
  "note.field.hidden": "隐藏文本",

  "note.copy": "复制笔记",
  "note.copied": "笔记已复制。",

  "note.editor.new.title": "新建笔记",
  "note.editor.edit.title": "编辑笔记",
  "note.editor.create": "添加笔记",
  "note.editor.requirement": "名称为必填项",

  "note.error.list": "Ravenpass 无法加载你的笔记。请重新启动应用。",
  "note.error.close": "Ravenpass 无法关闭所选笔记。请锁定密码库。",
  "note.error.open": "无法打开这条笔记。请重新选择，或重新打开 Ravenpass。",
  "note.error.copy": "Ravenpass 无法复制这条笔记。请打开后重试。",
  "note.error.pin": "Ravenpass 无法将这条笔记添加到收藏。请重试。",
  "note.error.unpin": "Ravenpass 无法将这条笔记从收藏中移除。请重试。",
  "note.error.save": "Ravenpass 无法保存这条笔记。请检查输入内容后重试。",
  "note.error.saved-partly": "笔记已保存，但{detail}",
  "note.error.delete": "Ravenpass 无法删除这条笔记。请重试。",
  "note.error.duplicate": "Ravenpass 无法复制出这条笔记的副本。请重试。",
  "note.error.deleted-partly": "笔记已删除，但{detail}",
};
