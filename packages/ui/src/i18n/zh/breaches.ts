import type { breaches as english } from "../en/breaches.ts";

export const breaches: typeof english = {
  "settings.privacy.passwords": "密码",
  "settings.breach-checks": "检查已泄露的密码",
  "settings.breach-checks.detail":
    "Ravenpass 会向 Have I Been Pwned 发送每个密码 SHA-1 哈希值的前 5 个字符（绝不会发送密码本身），并标记在已知泄露事件中发现的密码。",
  "settings.breach-checks.error": "Ravenpass 无法更改泄露检查设置。请重试。",
  "breach.mark": "已泄露",
  "breach.list.checking": "正在检查密码泄露情况…",
  "breach.list.found":
    "{count, plural, other {在已检查的 {checked} 个密码中，发现 # 个曾出现在数据泄露中。建议优先修改。}}",
  "breach.list.clean":
    "{count, plural, other {已检查 # 个密码，未在已知泄露中发现。}}",
  "breach.list.failed":
    "Ravenpass 无法检查密码泄露情况。保险库更改或解锁时将再次尝试。",
  "breach.detail":
    "{count, plural, other {此密码曾在已知泄露事件中出现 # 次。请先在网站上修改，然后在此处更新。}}",
  "breach.unreachable":
    "Ravenpass 无法检查此密码的泄露情况。保险库更改或解锁时将再次尝试。",
  "breach.editor":
    "{count, plural, other {此密码曾在已知泄露事件中出现 # 次。请选择其他密码。}}",
};
