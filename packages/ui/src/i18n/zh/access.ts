import type { access as english } from "../en/access.ts";

export const access: typeof english = {
  "wizard.steps.label": "设置进度",
  "wizard.steps.storage": "保险库文件",
  "wizard.steps.phrase": "恢复密钥",
  "wizard.steps.confirm": "核对密钥",
  "wizard.steps.unlock": "解锁方式",
  "wizard.storage.title": "选择保存保险库的位置",
  "wizard.storage.title-fixed": "您的保险库将保存在此设备上",
  "wizard.storage.description": "您的 Ravenpass 数据保存在一个加密文件中。",
  "wizard.storage.type": "存储类型",
  "wizard.storage.location": "位置",
  "wizard.storage.file-empty": "正在检查…",
  "wizard.storage.change": "更改…",
  "wizard.storage.change-busy": "正在选择…",
  "wizard.storage.fact-encrypted": "写入前在此设备上完成加密。",
  "wizard.storage.fact-saved": "每次更改均会在操作时立即保存。",
  "wizard.storage.fact-backup": "您可以在设置中导出单独的加密副本。",
  "wizard.storage.unrestricted": "此位置无法限制谁可以读取该文件。",
  "wizard.phrase.title": "保存恢复密钥",
  "wizard.phrase.description": "恢复保险库需要此密钥及一份加密副本。",
  "wizard.phrase.list": "24 词恢复密钥",
  "wizard.phrase.copy": "复制密钥",
  "wizard.phrase.copy-busy": "正在复制…",
  "wizard.phrase.copy-done": "已复制。存放密钥后请清空剪贴板。",
  "wizard.phrase.save": "保存到文件…",
  "wizard.phrase.save-busy": "正在保存…",
  "wizard.phrase.save-done": "已保存。请将此密钥文件与您的加密保险库分开存放。",
  "wizard.phrase.print": "打印…",
  "wizard.phrase.print-busy": "正在打印…",
  "wizard.phrase.note":
    "保存的密钥文件未加密。任何拥有该密钥和加密副本的人都能打开您的保险库。",
  "wizard.phrase.note-words":
    "任何拥有此密钥和加密副本的人都能打开您的保险库。",
  "wizard.phrase.acknowledgment": "我已将恢复密钥妥善存放在安全的地方。",
  "wizard.confirm.title": "核对恢复密钥",
  "wizard.confirm.description": "请输入您刚才保存的密钥中的三个单词。",
  "wizard.confirm.words": "恢复密钥中的单词",
  "wizard.confirm.hint": "输入空格以前往下一个单词。",
  "phrase.word": "第 {number} 个词",
  "wizard.unlock.title": "选择打开此保险库的方式",
  "wizard.unlock.description": "这仅适用于此设备。您稍后可以在设置中更改。",
  "wizard.unlock.create": "创建保险库",
  "wizard.unlock.creating": "正在创建保险库",
  "wizard.actions.back": "返回",
  "wizard.actions.cancel": "取消",
  "wizard.actions.continue": "继续",
  "wizard.actions.preparing": "正在准备…",
  "wizard.errors.storage-unreadable":
    "Ravenpass 无法读取保险库保存位置。请重启应用。",
  "wizard.errors.location-rejected":
    "Ravenpass 无法使用该位置。请选择其他文件夹重试。",
  "wizard.errors.begin-failed": "Ravenpass 无法启动设置。请重试。",
  "wizard.errors.cancel-failed": "Ravenpass 无法退出设置。请重启应用。",
  "wizard.errors.leave-failed": "Ravenpass 无法返回到另一个保险库。请重试。",
  "wizard.errors.copy-failed": "Ravenpass 无法复制密钥。请重试或保存到文件。",
  "wizard.errors.copy-failed-no-file":
    "Ravenpass 无法复制密钥。请重试或将单词抄写下来。",
  "wizard.errors.save-failed": "Ravenpass 无法保存密钥。请选择其他位置重试。",
  "wizard.errors.print-failed":
    "Ravenpass 无法打印密钥。请检查打印机后重试，或将单词抄写下来。",
  "wizard.errors.words-mismatch": "输入的单词不匹配。请核对恢复密钥后重试。",
  "wizard.errors.finish-failed": "Ravenpass 无法完成设置。请重试。",

  "unlock-methods.biometry.title": "生物识别",
  "unlock-methods.biometry.description": "设备的生物识别或锁屏密码。",
  "unlock-methods.biometry.unavailable": "此设备不支持生物识别。",
  "unlock-methods.biometry.creating":
    "正在设备的安全硬件中创建密钥，这可能需要约 20 秒。",
  "unlock-methods.biometry.last":
    "这是在此设备上唯一的解锁方式。设置 PIN 码后可关闭生物识别。",
  "unlock-methods.pin.last":
    "这是在此设备上唯一的解锁方式。开启生物识别后可关闭 PIN 码。",
  "unlock-methods.pin.only": "PIN 码是此设备唯一支持的解锁方式。",
  "unlock-methods.pin.title": "PIN 码",
  "unlock-methods.pin.description":
    "{min} 至 {max, plural, other {# 位数字}}。",
  "unlock-methods.pin.weaker": "避免使用容易被猜到的 PIN 码。",
  "unlock-methods.pin.set": "设置 PIN 码",
  "unlock-methods.pin.change": "更改 PIN 码",
  "unlock-methods.pin.current": "当前 PIN 码",
  "unlock-methods.pin.new": "新 PIN 码",
  "unlock-methods.pin.repeat": "重复 PIN 码",
  "unlock-methods.pin.save": "保存 PIN 码",
  "unlock-methods.pin.saving": "正在保存…",
  "unlock-methods.pin.cancel": "取消",
  "unlock-methods.pin.mismatch": "两次输入的 PIN 码不一致。",
  "unlock-methods.pin.saved": "PIN 码已保存。",
  "unlock-methods.pin.removed": "PIN 码已移除。",
  "unlock-methods.biometry.on": "生物识别已开启。",
  "unlock-methods.biometry.off": "生物识别已关闭。",
  "unlock-methods.confirm.title": "输入您的 PIN 码",
  "unlock-methods.confirm.description":
    "请确认您的身份以更改此保险库的解锁方式。",
  "unlock-methods.confirm.action": "继续",
  "unlock-methods.confirm-key.title": "输入恢复密钥",
  "unlock-methods.confirm-key.description":
    "此设备无法使用 PIN 码或生物识别确认身份。请输入保险库的恢复密钥以更改解锁方式。",
  "unlock-methods.errors.save-failed": "Ravenpass 无法保存该更改。请重试。",
  "unlock-methods.errors.unreadable":
    "Ravenpass 无法读取此保险库的解锁方式。请重启应用。",
  "unlock-methods.choose": "至少选择一种方式以继续。",

  "unlock.title": "解锁保险库",
  "unlock.action": "解锁保险库",
  "unlock.action-busy": "正在解锁…",
  "unlock.biometry-action": "使用生物识别",
  "unlock.back": "返回开始",
  "unlock.restore.title": "恢复此保险库的访问权限",
  "unlock.restore.reason": "此设备没有可用于解锁此保险库的方式。",
  "unlock.restore.next": "输入您的恢复密钥并设置新的解锁方式。",
  "unlock.missing.title": "未找到保险库文件",
  "unlock.missing.reason":
    "保险库文件已不在 {location}。它可能已在其他设备上被移动。",
  "unlock.missing.next": "从新位置打开该文件以在此处解锁。",
  "unlock.missing.action": "打开保险库文件…",
  "unlock.pin.label": "PIN 码",
  "unlock.pin.placeholder": "您的 PIN 码",
  "unlock.pin.action": "解锁",
  "unlock.pin.busy": "正在检查…",
  "unlock.pin.attempts":
    "{count, plural, other {还剩 # 次尝试机会，之后 PIN 码将被移除。}}",
  "unlock.none.title": "在此设备上无法打开此保险库",
  "unlock.none.description": "使用恢复密钥以设置新的解锁方式。",
  "unlock.recovery.title": "恢复访问权限",
  "unlock.recovery.description": "输入 24 词恢复密钥以打开此保险库。",
  "unlock.recovery.action": "使用恢复密钥",
  "unlock.errors.failed":
    "Ravenpass 无法解锁此保险库。请重试，或使用恢复密钥。",
  "unlock.errors.switch-failed":
    "Ravenpass 无法打开该保险库。请选择其他保险库并重试。",
  "unlock.errors.back-failed": "Ravenpass 无法关闭此保险库。请重试。",
  "unlock.errors.adopt-failed":
    "Ravenpass 无法打开该版本。请解锁保险库后重试。",
  "unlock.diverged.title": "在其他设备上发生了更改",
  "unlock.diverged.description":
    "此保险库在其他设备上同时发生了更改。此设备上的最新修改已被该版本替换。",
  "unlock.diverged.confirm": "打开此版本",
  "unlock.diverged.cancel": "暂不",

  "recovery.phrase.label": "恢复密钥",
  "recovery.phrase.show": "显示恢复密钥",
  "recovery.phrase.hide": "隐藏恢复密钥",
  "recovery.preview.older.title": "此副本可能较旧",
  "recovery.preview.older.description":
    "恢复此副本可能会丢失在此副本保存之后添加的项目。",
  "recovery.preview.accept-loss": "我了解较新的项目可能会丢失。",
  "recovery.preview.replaced-key.title": "此副本使用了已被替换的恢复密钥",
  "recovery.preview.replaced-key.description":
    "此保险库的恢复密钥此前已被替换。任何持有旧密钥的人都可能写入此副本。仅当由您本人恢复时才建议打开。",
  "recovery.preview.accept-replaced-key": "这是我亲自恢复的副本。",
  "recovery.actions.back": "返回",
  "recovery.actions.continue": "继续",
  "recovery.actions.checking": "正在检查…",
  "recovery.unlock.title": "选择打开此保险库的方式",
  "recovery.unlock.description":
    "选择在此设备上的解锁方式。您稍后可在设置中更改。",
  "recovery.actions.restore": "恢复访问",
  "recovery.restoring": "正在恢复对保险库的访问",
  "recovery.errors.phrase-missing": "请先输入您的 24 词恢复密钥。",
  "recovery.errors.local-failed": "Ravenpass 无法打开保险库。请重试。",
  "recovery.errors.confirm-failed":
    "Ravenpass 无法完成恢复。请重启应用后重试。",
  "recovery.errors.cancel-failed": "Ravenpass 无法退出恢复流程。请重启应用。",

  "storage-unavailable.title": "保险库存储不可用",
  "storage-unavailable.description-folder":
    "Ravenpass 无法访问 {place}。请重新连接，或选择您的保险库文件。",
  "storage-unavailable.description": "Ravenpass 无法访问存放保险库的文件夹。",
  "storage-unavailable.note":
    "Ravenpass 在此设备上没有第二份副本。请重新连接包含保险库文件的存储，或选择文件当前所在的位置。",
  "storage-unavailable.actions.locate": "选择保险库文件…",
  "storage-unavailable.actions.retry": "重试",
  "storage-unavailable.actions.retry-busy": "正在检查…",
  "storage-unavailable.errors.unreadable":
    "Ravenpass 无法读取保险库保存位置。请重启应用。",
  "storage-unavailable.errors.unreachable":
    "Ravenpass 仍无法访问该位置。请重新连接，或选择您的保险库文件。",
  "storage-unavailable.errors.switch-failed":
    "Ravenpass 无法打开该保险库。请选择其他保险库重试。",
  "storage-unavailable.errors.create-failed":
    "Ravenpass 无法新建保险库。请重试。",
  "storage-unavailable.errors.open-failed":
    "无法打开该文件。请选择 Ravenpass 保险库文件。",
};
