import type { generator as english } from "../en/generator.ts";

export const generator: typeof english = {
  "generator.open": "生成密码",
  "generator.title": "新密码",
  "generator.again": "再生成一个",
  "generator.strength": "{strength} · {bits} 位",
  "generator.strength.weak": "弱",
  "generator.strength.fair": "一般",
  "generator.strength.strong": "强",
  "generator.strength.very-strong": "非常强",
  "generator.mode": "密码类型",
  "generator.mode.words": "单词",
  "generator.mode.characters": "字符",
  "generator.words": "{count, plural, other {# 个单词}}",
  "generator.separator": "单词之间",
  "generator.separator.hyphen": "连字符",
  "generator.separator.dot": "点",
  "generator.separator.underscore": "下划线",
  "generator.separator.space": "空格",
  "generator.capitalize": "首字母大写",
  "generator.digit": "加一个数字",
  "generator.length": "{count, plural, other {# 个字符}}",
  "generator.uppercase": "大写字母",
  "generator.digits": "数字",
  "generator.symbols": "符号",
  "generator.digits.fewest": "{count, plural, other {至少 # 个数字}}",
  "generator.symbols.fewest": "{count, plural, other {至少 # 个符号}}",
  "generator.look-alikes": "不使用 I、O、l、0 和 1",
  "generator.cancel": "取消",
  "generator.use": "使用此密码",
  "generator.copy": "复制密码",
  "generator.copied": "密码已复制。",
  "generator.history": "已复制和已使用",
  "generator.history.empty":
    "你在这里复制或在项目中使用的密码会在列表中保留 {count, plural, other {# 天}}。",
  "generator.history.off":
    "在“隐私”设置中打开“密码历史”，即可列出你复制或使用的密码。",
  "generator.history.show": "显示密码",
  "generator.history.hide": "隐藏密码",
  "generator.history.clear": "清除列表…",
  "generator.history.clear.title":
    "{count, plural, other {从列表中清除 # 个密码？}}",
  "generator.history.clear.detail":
    "项目中保存的密码不受影响。此操作无法撤销。",
  "generator.history.clear.confirm": "清除",
  "generator.error.words":
    "Ravenpass 无法加载单词表。请重新打开 Ravenpass 后重试。",
  "generator.error.history":
    "Ravenpass 无法读取已复制密码的列表。请锁定并重新解锁密码库后重试。",
  "generator.error.record": "Ravenpass 无法将此密码加入已复制密码的列表。",
  "generator.error.copy": "Ravenpass 无法复制此密码。请重试。",
  "generator.error.clear": "Ravenpass 无法清除列表。请重试。",

  "settings.password-history": "密码历史",
  "settings.password-history.detail":
    "列出你复制或使用的生成密码，仅保存在这台设备上。",
  "settings.password-history.period": "保留",
  "settings.password-history.days": "{count, plural, other {# 天}}",
  "settings.password-history.off.title":
    "{count, plural, other {关闭历史会删除 # 个密码。}}",
  "settings.password-history.off.confirm": "关闭并删除",
  "settings.password-history.shorten.title":
    "{count, plural, other {缩短期限会立即删除 # 个密码。}}",
  "settings.password-history.shorten.confirm": "删除并缩短",
  "settings.password-history.delete.detail": "项目中保存的密码不受影响。",
  "settings.password-history.error": "Ravenpass 无法更改密码历史。请重试。",
};
