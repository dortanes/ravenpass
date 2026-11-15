import type { access as english } from "../en/access.ts";

export const access: typeof english = {
  "wizard.steps.label": "设置进度",
  "wizard.steps.storage": "密码库文件",
  "wizard.steps.phrase": "恢复密钥",
  "wizard.steps.confirm": "核对密钥",
  "wizard.steps.unlock": "解锁",
  "wizard.storage.title": "选择密码库的保存位置",
  "wizard.storage.title-fixed": "你的密码库保存在这台设备上",
  "wizard.storage.description": "你的 Ravenpass 数据保存在一个加密文件中。",
  "wizard.storage.type": "存储类型",
  "wizard.storage.location": "位置",
  "wizard.storage.file-empty": "正在检查…",
  "wizard.storage.change": "更改…",
  "wizard.storage.change-busy": "正在选择…",
  "wizard.storage.fact-encrypted": "写入之前，所有内容都会在这台设备上加密。",
  "wizard.storage.fact-saved": "每次更改都会立即保存。",
  "wizard.storage.fact-backup": "你可以在“设置”中导出一份单独的加密副本。",
  "wizard.storage.unrestricted": "此位置无法限制谁能读取该文件。",
  "wizard.phrase.title": "保存你的恢复密钥",
  "wizard.phrase.description": "要恢复密码库，你需要这个密钥和一份加密副本。",
  "wizard.phrase.list": "24 个单词的恢复密钥",
  "wizard.phrase.copy": "复制密钥",
  "wizard.phrase.copy-busy": "正在复制…",
  "wizard.phrase.copy-done": "已复制。保存好密钥后，请清除剪贴板。",
  "wizard.phrase.save": "存储为文件…",
  "wizard.phrase.save-busy": "正在存储…",
  "wizard.phrase.save-done": "已存储。请将这个密钥文件与加密的密码库分开存放。",
  "wizard.phrase.print": "打印…",
  "wizard.phrase.print-busy": "正在打印…",
  "wizard.phrase.note":
    "存储的密钥文件没有加密。任何拿到密钥和加密副本的人都能打开你的密码库。",
  "wizard.phrase.note-words":
    "任何拿到这个密钥和加密副本的人都能打开你的密码库。",
  "wizard.phrase.acknowledgment": "我已将恢复密钥存放在安全的地方。",
  "wizard.confirm.title": "核对你的恢复密钥",
  "wizard.confirm.description": "输入刚才保存的密钥中的三个单词。",
  "wizard.confirm.words": "恢复密钥中的单词",
  "wizard.confirm.hint": "输入空格即可跳到下一个单词。",
  "phrase.word": "第 {number} 个单词",
  "wizard.unlock.title": "选择打开此密码库的方式",
  "wizard.unlock.description": "此设置仅适用于这台设备，可在“设置”中更改。",
  "wizard.unlock.create": "创建密码库",
  "wizard.unlock.creating": "正在创建你的密码库",
  "wizard.actions.back": "返回",
  "wizard.actions.cancel": "取消",
  "wizard.actions.continue": "继续",
  "wizard.actions.preparing": "正在准备…",
  "wizard.errors.storage-unreadable":
    "Ravenpass 无法读取密码库的保存位置。请重新启动应用。",
  "wizard.errors.location-rejected":
    "Ravenpass 无法使用该位置。请选择其他文件夹后重试。",
  "wizard.errors.begin-failed": "Ravenpass 无法开始设置。请重试。",
  "wizard.errors.cancel-failed": "Ravenpass 无法关闭设置。请重新启动应用。",
  "wizard.errors.leave-failed": "Ravenpass 无法返回你的另一个密码库。请重试。",
  "wizard.errors.copy-failed": "Ravenpass 无法复制密钥。请重试，或存储为文件。",
  "wizard.errors.copy-failed-no-file":
    "Ravenpass 无法复制密钥。请重试，或把这些单词抄写下来。",
  "wizard.errors.save-failed": "Ravenpass 无法存储密钥。请选择其他位置后重试。",
  "wizard.errors.print-failed":
    "Ravenpass 无法打印密钥。请重试，或把这些单词抄写下来。",
  "wizard.errors.words-mismatch": "这些单词不一致。请检查恢复密钥后重试。",
  "wizard.errors.finish-failed": "Ravenpass 无法完成设置。请重试。",

  "unlock-methods.biometry.title": "生物识别",
  "unlock-methods.biometry.description": "设备的生物识别或设备密码。",
  "unlock-methods.biometry.unavailable": "这台设备无法使用生物识别。",
  "unlock-methods.biometry.creating":
    "正在设备的安全硬件中创建密钥，最多可能需要 20 秒。",
  "unlock-methods.biometry.last":
    "这是这台设备上唯一的解锁方式。设置 PIN 码后才能关闭生物识别。",
  "unlock-methods.pin.last":
    "这是这台设备上唯一的解锁方式。打开生物识别后才能关闭 PIN 码。",
  "unlock-methods.pin.only": "PIN 码是这台设备上唯一可用的解锁方式。",
  "unlock-methods.pin.title": "PIN 码",
  "unlock-methods.pin.description":
    "{min} 到 {max, plural, other {# 位数字}}。",
  "unlock-methods.pin.weaker": "请避免使用容易被猜到的 PIN 码。",
  "unlock-methods.pin.set": "设置 PIN 码",
  "unlock-methods.pin.change": "更改 PIN 码",
  "unlock-methods.pin.current": "当前 PIN 码",
  "unlock-methods.pin.new": "新 PIN 码",
  "unlock-methods.pin.repeat": "再次输入 PIN 码",
  "unlock-methods.pin.save": "存储 PIN 码",
  "unlock-methods.pin.saving": "正在存储…",
  "unlock-methods.pin.cancel": "取消",
  "unlock-methods.pin.mismatch": "两次输入的 PIN 码不一致。",
  "unlock-methods.pin.saved": "PIN 码已存储。",
  "unlock-methods.pin.removed": "PIN 码已移除。",
  "unlock-methods.biometry.on": "生物识别已打开。",
  "unlock-methods.biometry.off": "生物识别已关闭。",
  "unlock-methods.confirm.title": "输入 PIN 码",
  "unlock-methods.confirm.description":
    "确认是你本人，以便更改此密码库的解锁方式。",
  "unlock-methods.confirm.action": "继续",
  "unlock-methods.confirm-key.title": "输入恢复密钥",
  "unlock-methods.confirm-key.description":
    "这台设备无法通过 PIN 码或生物识别确认你的身份。请输入密码库的恢复密钥以更改解锁方式。",
  "unlock-methods.errors.save-failed": "Ravenpass 无法存储此更改。请重试。",
  "unlock-methods.errors.unreadable":
    "Ravenpass 无法读取此密码库的打开方式。请重新启动应用。",
  "unlock-methods.choose": "请至少选择一种方式才能继续。",

  "unlock.title": "解锁你的密码库",
  "unlock.action": "解锁密码库",
  "unlock.action-busy": "正在解锁…",
  "unlock.biometry-action": "使用生物识别",
  "unlock.back": "回到开始",
  "unlock.restore.title": "恢复此密码库的访问权限",
  "unlock.restore.reason": "这台设备没有解锁此密码库的方式。",
  "unlock.restore.next": "输入恢复密钥并设置新的解锁方式。",
  "unlock.missing.title": "找不到密码库文件",
  "unlock.missing.reason":
    "密码库文件已不在 {location}，可能已在另一台设备上被移动。",
  "unlock.missing.next": "从新的位置打开它，即可在这里解锁。",
  "unlock.missing.action": "打开密码库文件…",
  "unlock.pin.label": "PIN 码",
  "unlock.pin.placeholder": "你的 PIN 码",
  "unlock.pin.action": "解锁",
  "unlock.pin.busy": "正在检查…",
  "unlock.pin.attempts":
    "{count, plural, other {再输错 # 次，PIN 码将被移除。}}",
  "unlock.none.title": "这台设备上没有打开此密码库的方式",
  "unlock.none.description": "使用恢复密钥设置新的解锁方式。",
  "unlock.recovery.title": "恢复访问权限",
  "unlock.recovery.description": "输入 24 个单词的恢复密钥以打开此密码库。",
  "unlock.recovery.action": "使用恢复密钥",
  "unlock.errors.failed":
    "Ravenpass 无法解锁此密码库。请重试，或使用恢复密钥。",
  "unlock.errors.switch-failed":
    "Ravenpass 无法打开该密码库。请选择其他密码库后重试。",
  "unlock.errors.back-failed": "Ravenpass 无法关闭此密码库。请重试。",
  "unlock.errors.adopt-failed":
    "Ravenpass 无法打开该版本。请解锁密码库后重试。",
  "unlock.diverged.title": "已在另一台设备上更改",
  "unlock.diverged.description":
    "此密码库同时在另一台设备上被更改。这台设备上最近的更改已被那个版本替换。",
  "unlock.diverged.confirm": "打开此版本",
  "unlock.diverged.cancel": "以后再说",

  "recovery.phrase.label": "恢复密钥",
  "recovery.phrase.show": "显示恢复密钥",
  "recovery.phrase.hide": "隐藏恢复密钥",
  "recovery.preview.older.title": "此副本可能较旧",
  "recovery.preview.older.description":
    "恢复它可能会移除此副本保存之后添加的项目。",
  "recovery.preview.accept-loss": "我了解较新的项目可能会丢失。",
  "recovery.preview.replaced-key.title": "此副本使用的是已被替换的恢复密钥",
  "recovery.preview.replaced-key.description":
    "此密码库的恢复密钥已被替换。任何持有旧密钥的人都可能写入了此副本。只有在副本是你本人恢复的情况下才打开它。",
  "recovery.preview.accept-replaced-key": "此副本是我本人恢复的。",
  "recovery.actions.back": "返回",
  "recovery.actions.continue": "继续",
  "recovery.actions.checking": "正在检查…",
  "recovery.unlock.title": "选择打开此密码库的方式",
  "recovery.unlock.description":
    "选择在这台设备上的解锁方式，之后可在“设置”中更改。",
  "recovery.actions.restore": "恢复访问权限",
  "recovery.restoring": "正在恢复对密码库的访问",
  "recovery.errors.phrase-missing": "请先输入 24 个单词的恢复密钥。",
  "recovery.errors.local-failed": "Ravenpass 无法打开密码库。请重试。",
  "recovery.errors.confirm-failed":
    "Ravenpass 无法完成恢复。请重新启动应用后再试。",
  "recovery.errors.cancel-failed": "Ravenpass 无法关闭恢复。请重新启动应用。",

  "storage-unavailable.title": "无法访问密码库的存储位置",
  "storage-unavailable.description-folder":
    "Ravenpass 无法访问 {place}。请重新连接它，或选择你的密码库文件。",
  "storage-unavailable.description": "Ravenpass 无法访问存放密码库的文件夹。",
  "storage-unavailable.note":
    "Ravenpass 在这台设备上没有第二份副本。请重新连接存放密码库文件的存储设备，或在文件现在所在的位置选择它。",
  "storage-unavailable.actions.locate": "选择密码库文件…",
  "storage-unavailable.actions.retry": "重试",
  "storage-unavailable.actions.retry-busy": "正在检查…",
  "storage-unavailable.errors.unreadable":
    "Ravenpass 无法读取密码库的存放位置。请重新启动应用。",
  "storage-unavailable.errors.unreachable":
    "Ravenpass 仍然无法访问该位置。请重新连接它，或选择你的密码库文件。",
  "storage-unavailable.errors.switch-failed":
    "Ravenpass 无法打开该密码库。请选择其他密码库后重试。",
  "storage-unavailable.errors.create-failed":
    "Ravenpass 无法开始创建新的密码库。请重试。",
  "storage-unavailable.errors.open-failed":
    "无法打开该文件。请选择一个 Ravenpass 密码库文件。",
};
