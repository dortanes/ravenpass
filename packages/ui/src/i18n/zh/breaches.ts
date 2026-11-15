import type { breaches as english } from "../en/breaches.ts";

export const breaches: typeof english = {
  "settings.privacy.passwords": "密码",
  "settings.breach-checks": "检查已泄露的密码",
  "settings.breach-checks.detail":
    "Ravenpass 只会向 Have I Been Pwned 发送每个密码 SHA-1 哈希值的前 5 个字符，绝不发送密码本身，并标记出现在已知泄露事件中的密码。",
  "settings.breach-checks.error": "Ravenpass 无法更改泄露检查。请重试。",
  "breach.mark": "已泄露",
  "breach.list.checking": "正在检查密码是否泄露…",
  "breach.list.found":
    "{count, plural, other {在 {checked} 个密码中，有 # 个出现在泄露事件中。请优先更改。}}",
  "breach.list.clean":
    "{count, plural, other {已检查 # 个密码，均未出现在泄露事件中。}}",
  "breach.list.failed":
    "Ravenpass 无法检查密码是否泄露。密码库发生变化或解锁时会再次尝试。",
  "breach.detail":
    "{count, plural, other {此密码在已知泄露事件中出现过 # 次。请先在网站上更改，再在这里更改。}}",
  "breach.unreachable":
    "Ravenpass 无法检查此密码是否泄露。密码库发生变化或解锁时会再次尝试。",
  "breach.editor":
    "{count, plural, other {此密码在已知泄露事件中出现过 # 次。请换一个。}}",
};
