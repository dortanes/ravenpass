import type { failures as english } from "../en/failures.ts";

/** One message per failure code the desktop service reports. */
export const failures: typeof english = {
  "failure.general":
    "Ravenpass 无法完成此操作。请重试；如果问题仍然存在，请重新打开密码库。",
  "failure.window-unavailable":
    "Ravenpass 无法访问自己的窗口。请重新打开应用后重试。",
  "failure.authentication-canceled": "身份验证已取消。准备好后请重试。",
  "failure.authentication-failed": "无法验证你的身份。请重试。",
  "failure.unlock-unavailable":
    "设备的安全解锁不可用。请检查设备的安全设置后重试。",
  "failure.unlock-key-missing":
    "生物识别已无法在这台设备上解锁此密码库。请使用 PIN 码或恢复密钥，然后在“设置”中重新打开生物识别。",
  "failure.way-in-not-set":
    "Ravenpass 无法在这台设备上设置这种解锁方式。请重试，或选择其他方式。",
  "failure.vault-locked": "密码库已锁定。请解锁后重试。",
  "failure.vault-exists": "该位置已有一个密码库。请为新密码库选择其他文件。",
  "failure.setup-in-progress": "设置已在进行中。请先完成或取消它。",
  "failure.setup-not-active": "密码库设置已失效。请重新开始创建密码库。",
  "failure.no-local-vault":
    "该位置没有找到密码库。请选择一份加密副本来恢复它。",
  "failure.recovery-replaced-key":
    "此副本使用的恢复密钥已被替换。打开前，请确认副本是你本人恢复的。",
  "failure.key-change-unfinished":
    "恢复密钥已更改，但 Ravenpass 无法在这台设备上完成保存。从现在起请使用新的恢复密钥，并重新解锁密码库。",
  "failure.key-change-uncertain":
    "Ravenpass 无法确认恢复密钥是否已更改。在密码库重新打开之前，请同时保留旧的和新的恢复密钥。",
  "failure.recovery-needs-confirmation":
    "此副本可能比你最新的密码库旧。恢复之前，请确认你接受可能丢失的更改。",
  "failure.recovery-changed": "恢复期间密码库发生了变化。请重新开始恢复。",
  "failure.recovery-phrase-invalid": "该恢复密钥不正确。请检查单词后重试。",
  "failure.recovery-key-mismatch":
    "该恢复密钥与此密码库不符。请检查单词后重试。",
  "failure.recovery-key-not-saved":
    "没有存储任何文件。请选择存储恢复密钥的位置。",
  "failure.recovery-key-save-failed":
    "无法存储恢复密钥。请选择其他位置后重试。",
  "failure.recovery-key-copy-failed":
    "无法复制恢复密钥。请将它存储到私密文件中，或抄写下来。",
  "failure.recovery-key-print-failed":
    "无法打印恢复密钥。请检查打印机后重试，或把这些单词抄写下来。",
  "failure.vault-changed":
    "密码库在打开之后发生了变化。请锁定 Ravenpass 并重新打开后再编辑。",
  "failure.vault-missing":
    "密码库文件已不在 Ravenpass 打开它的位置，可能已在另一台设备上被移动。请从新的位置打开密码库文件。",
  "failure.save-unconfirmed":
    "无法确认是否已保存。请重新打开 Ravenpass 后再进行更改。",
  "failure.authentication-invalid":
    "无法打开密码库。请检查恢复密钥，或使用一份确认完好的副本。",
  "failure.unsupported-format":
    "此密码库文件的格式无法被当前版本的 Ravenpass 打开。请用恢复密钥从加密副本恢复，或新建一个密码库。",
  "failure.malformed-vault":
    "此密码库文件无法读取。请用恢复密钥从加密副本恢复。",
  "failure.invalid-item": "部分字段无效。请检查后重试。",
  "failure.invalid-code":
    "这个一次性验证码设置无法生成验证码。请粘贴网站显示的设置密钥，或它的一次性验证码链接。",
  "failure.no-code": "此密码没有设置一次性验证码。请先在项目中添加，然后重试。",
  "failure.code-setup-expired":
    "这个一次性验证码设置已过期。请重新打开它的链接或扫描它的二维码。",
  "failure.qr-code-missing":
    "没有找到二维码。请选择或复制一张显示完整二维码的图片。",
  "failure.qr-code-not-setup":
    "此二维码不是用于设置一次性验证码的。请使用网站为身份验证器应用显示的二维码。",
  "failure.qr-code-ambiguous":
    "图片中有多个设置码。请使用只有一个设置码的图片。",
  "failure.resource-limit": "此项目或密码库超出了大小限制。请缩短项目后重试。",
  "failure.passkeys-full":
    "这些密码合起来的通行密钥超出了一个密码能存放的数量。请先从其中一个移除通行密钥，再合并。",
  "failure.breach-checks-off": "泄露检查已关闭。请在“隐私”设置中打开。",
  "failure.breach-check-unreachable":
    "Ravenpass 无法连接泄露检查服务。请检查网络连接后重试。",
  "failure.permission-denied":
    "Ravenpass 无法使用该文件或文件夹。请选择其他位置后重试。",
  "failure.file-size-invalid":
    "该文件为空或过大。请选择有效的 Ravenpass 文件。",
  "failure.file-unverified": "无法验证该文件。请选择其他位置后重试。",
  "failure.file-not-selected": "没有选择文件。请选择一个文件以继续。",
  "failure.location-not-selected": "无法选择该位置。请重试。",
  "failure.item-unreadable": "无法打开此项目。请从列表中重新选择。",
  "failure.photo-unsupported":
    "无法读取该图片。请选择 JPEG、PNG 或 WebP 照片。",
  "failure.photo-too-large":
    "该图片过大。请选择小于 32 MB 且少于 5000 万像素的图片。",
  "failure.scan-unsupported":
    "无法将该文件读取为扫描件。请选择 JPEG、PNG 或 WebP 图片，或者 PDF。",
  "failure.scan-too-large":
    "该扫描件过大，无法保存。请选择更小的图片，或小于 960 KB 的 PDF。",
  "failure.scan-limit": "此证件已有 4 份扫描件。请先移除一份再附加新的。",
  "failure.field-not-copyable": "该字段无法复制。请选择其他字段。",
  "failure.field-empty": "该字段为空。请先填写内容再复制。",
  "failure.copy-failed": "无法复制该值。请打开项目后手动复制。",
  "failure.website-missing":
    "此密码没有保存网站。请在项目中添加网站地址，然后重试。",
  "failure.website-unsupported":
    "该网站地址不是 Ravenpass 能打开的网址。请改成以 https:// 开头的地址，或复制后自行打开。",
  "failure.website-open-failed": "无法打开该网站。请复制地址后自行打开。",
  "failure.export-canceled": "导出已取消。请选择存储副本的文件位置。",
  "failure.export-stale":
    "导出前密码库发生了变化。请重新打开密码库并导出新副本。",
  "failure.export-unconfirmed":
    "已存储加密副本，但 Ravenpass 无法确认它是最新的。请重新打开密码库后再次导出。",
  "failure.export-unrecorded":
    "已存储加密副本，但 Ravenpass 无法记录它。请再次导出，以便“设置”显示最新的副本。",
  "failure.storage-unavailable":
    "Ravenpass 无法访问存放密码库的文件夹。请重新连接它，或选择你的密码库文件。",
  "failure.storage-selection-invalid":
    "Ravenpass 无法读取密码库的存放位置。请选择你的密码库文件以继续。",
  "failure.storage-occupied": "该位置已有文件。请选择其他名称或文件夹。",
  "failure.storage-same-location":
    "你的密码库已经存放在那里。请选择其他文件夹来移动它。",
  "failure.storage-kind-unsupported":
    "此版本将密码库保存为文件。请选择一个文件位置。",
  "failure.storage-path-invalid":
    "该位置无法存放密码库文件。请选择其他名称或文件夹。",
  "failure.move-unverified": "无法在新位置验证密码库，它仍保留在原处。",
  "failure.second-copy":
    "该文件是你已有密码库的另一份副本。请将其中一个文件作为密码库，另一个留作加密备份。",
  "failure.older-copy":
    "此文件比这台设备使用的密码库旧。请等待最新版本同步完成，或切换回你之前使用的密码库文件。",
  "failure.vault-diverged":
    "此密码库同时在另一台设备上被更改。请打开 Ravenpass 并在那里解锁，以继续使用那个版本。",
  "failure.vault-key-replaced":
    "此密码库的恢复密钥已在另一台设备上更改。请用新的恢复密钥解锁一次。",
  "failure.witness-missing": "这台设备没有该密码库的记录。请用恢复密钥恢复它。",
  "failure.vault-unknown": "Ravenpass 不认识该密码库。请重新打开应用后重试。",
  "failure.no-other-vault":
    "没有可以返回的其他密码库。请完成创建当前密码库，或打开一个密码库文件。",
  "failure.group-unknown": "Ravenpass 不认识该分组。请重新打开密码库后重试。",
  "failure.group-name-invalid":
    "该分组名称为空、过长或已被使用。请换一个名称。",
  "failure.language-unsupported": "Ravenpass 不提供该语言。请从列表中选择。",
  "failure.clear-delay-unsupported": "Ravenpass 不提供该时长。请从列表中选择。",
  "failure.appearance-unsupported": "Ravenpass 不提供该外观。请从列表中选择。",
  "failure.sign-in-style-unsupported":
    "Ravenpass 不提供该登录方式。请从列表中选择。",
  "failure.auto-lock-delay-unsupported":
    "Ravenpass 不提供该时长。请从列表中选择。",
  "failure.backup-choice-unsupported":
    "Ravenpass 不提供该备份频率或数量。请从列表中选择。",
  "failure.backup-folder-missing":
    "尚未选择备份文件夹。请先选择文件夹，再打开自动备份。",
  "failure.shortcut-invalid":
    "Ravenpass 无法将这些按键用作快捷键。请选择其他按键。",
  "failure.pin-invalid": "PIN 码为 6 到 12 位数字。请输入后重试。",
  "failure.pin-wrong": "PIN 码错误。请稍候再试。",
  "failure.pin-too-soon": "连续输错 PIN 码次数过多。请等几秒后再试。",
  "failure.pin-removed":
    "PIN 码输错次数过多，已被移除。请用其他方式解锁，或用恢复密钥恢复访问权限。",
  "failure.pin-missing":
    "这台设备上的此密码库没有 PIN 码。请用其他方式解锁，或在“设置”中设置 PIN 码。",
  "failure.biometry-unavailable":
    "这台设备无法验证你的身份。请在设备设置中打开锁屏，或使用 PIN 码。",
  "failure.biometry-disabled":
    "此密码库的生物识别已关闭。请使用 PIN 码，或在“设置”中重新打开。",
  "failure.unlock-method-required":
    "请为此密码库保留一种解锁方式。先设置好另一种方式。",
  "failure.unlock-record-unreadable":
    "Ravenpass 无法读取此密码库在这台设备上的解锁方式。请用恢复密钥恢复它。",
  "failure.owner-unverified": "未能确认你的身份，因此没有做任何更改。请重试。",
  "failure.import-source-unknown":
    "Ravenpass 无法从此应用导入。请选择列表中的应用。",
  "failure.import-unrecognized":
    "Ravenpass 无法将此文件读取为 {source} 的导出文件。请选择 {source} 存储的文件。",
  "failure.import-too-large":
    "此文件过大，无法导入。请少导出一些项目，生成更小的文件后重试。",
  "failure.import-account-bound":
    "此文件只能在 {source} 中用你的账户打开。请重新导出，并设置文件密码或不加密。",
  "failure.import-password-wrong":
    "该密码无法打开此文件。请输入导出时设置的密码。",
  "failure.import-encryption-unsupported":
    "此文件的加密方式 Ravenpass 无法打开。请从 {source} 重新导出。",
  "failure.import-not-active": "请重新选择文件以导入。",
  "failure.import-empty":
    "没有可导入的内容：所有项目都已在 Ravenpass 中，或者无法导入。",
  "failure.import-limit":
    "你的密码库放不下此文件中的全部内容。请关闭按文件夹分组，或从导出文件中删除部分项目后重试。",
  "failure.import-trash-failed": "Ravenpass 无法将文件移到废纸篓。请自行删除。",
  "failure.import-reveal-failed":
    "Ravenpass 无法打开存放该文件的文件夹。请找到文件并删除。",
  "failure.link-expired": "此连接密钥已失效。请创建新密钥来关联扩展。",
  "failure.link-canceled": "关联已取消。请创建新密钥来关联扩展。",
  "failure.link-unavailable":
    "Ravenpass 无法为扩展打开连接，可能有其他应用正在使用。请稍后重试。",
  "failure.extension-not-found":
    "该扩展已取消关联。请重新打开“设置”查看仍在关联的扩展。",
  "failure.extension-name-invalid": "该名称为空或过长。最多可用 64 个字符。",
  "failure.confirmation-ended":
    "此请求已结束。请从浏览器或发起请求的应用重新开始。",
};
