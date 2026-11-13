import type { generator as english } from "../en/generator.ts";

export const generator: typeof english = {
  "generator.open": "生成密码",
  "generator.title": "新密码",
  "generator.again": "重新生成",
  "generator.value": "生成的密码",
  "generator.strength": "{strength} · {bits} 位",
  "generator.strength.weak": "弱",
  "generator.strength.fair": "一般",
  "generator.strength.strong": "强",
  "generator.strength.very-strong": "非常强",
  "generator.mode": "密码类型",
  "generator.mode.words": "单词",
  "generator.mode.characters": "字符",
  "generator.words": "{count, plural, other {# 个单词}}",
  "generator.separator": "单词分隔符",
  "generator.separator.hyphen": "连字符",
  "generator.separator.dot": "句点",
  "generator.separator.underscore": "下划线",
  "generator.separator.space": "空格",
  "generator.capitalize": "大写字母",
  "generator.digit": "包含数字",
  "generator.length": "{count, plural, other {# 个字符}}",
  "generator.uppercase": "大写字母",
  "generator.digits": "数字",
  "generator.digits.minimum": "{count, plural, other {至少 # 个数字}}",
  "generator.symbols": "符号",
  "generator.symbols.minimum": "{count, plural, other {至少 # 个符号}}",
  "generator.avoid-ambiguous": "避免易混淆字符",
  "generator.avoid-ambiguous.detail": "排除 I、O、l、0 和 1。",
  "generator.cancel": "取消",
  "generator.use": "使用密码",
  "generator.copy": "复制",
  "generator.copied": "密码已复制。",
  "generator.history": "历史记录",
  "generator.history.empty":
    "您复制或使用的密码会显示在此处，并保留 {count, plural, other {# 天}}。",
  "generator.history.off":
    "历史记录已关闭。请在隐私设置中开启，以保留您复制或使用的密码。",
  "generator.history.more": "显示更多 ({count, number})",
  "generator.history.clear": "清除历史记录…",
  "generator.history.clear.title":
    "{count, plural, other {从历史记录中删除 # 个密码？}}",
  "generator.history.clear.detail":
    "此操作无法撤销。已保存在项目中的密码不受影响。",
  "generator.history.clear.confirm": "清除历史记录",
  "generator.history.cleared": "历史记录已清除。",
  "generator.error.words":
    "Ravenpass 无法加载词汇表。请重新打开 Ravenpass 并重试。",
  "generator.error.history":
    "Ravenpass 无法打开生成器历史记录。请重新打开保险库并重试。",
  "generator.error.record": "Ravenpass 无法将此密码添加到历史记录。",
  "generator.error.copy": "Ravenpass 无法复制密码。请重试。",
  "generator.error.clear": "Ravenpass 无法清除历史记录。请重试。",

  "settings.generator-history": "保留已生成的密码",
  "settings.generator-history.detail":
    "仅在此设备上保存您在生成器中复制或使用的密码。",
  "settings.generator-history.period": "删除时间",
  "settings.generator-history.days": "{count, plural, other {# 天}}",
  "settings.generator-history.off.title":
    "{count, plural, other {关闭此功能将从历史记录中删除 # 个密码。是否继续？}}",
  "settings.generator-history.off.confirm": "删除并关闭",
  "settings.generator-history.shorten.title":
    "{count, plural, other {缩短保留时间将立即从历史记录中删除 # 个密码。是否继续？}}",
  "settings.generator-history.shorten.confirm": "删除并更改",
  "settings.generator-history.cancel": "取消",
  "settings.generator-history.error":
    "Ravenpass 无法更改生成器历史记录设置。请重试。",
};
