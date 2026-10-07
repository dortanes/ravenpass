import type { system as english } from "../en/system.ts";

export const system: typeof english = {
  "system.dialog.choose-vault-location": "Выберите место для хранилища",
  "system.dialog.select-vault-file": "Выберите файл хранилища",
  "system.dialog.save-export": "Сохраните зашифрованную копию хранилища",
  "system.dialog.save-recovery-key": "Сохраните ключ восстановления",
  "system.dialog.choose-photo": "Выберите фотографию",
  "system.dialog.choose-scan": "Выберите скан документа",
  "system.dialog.choose-qr-code": "Выберите изображение с QR-кодом",
  "system.dialog.save-scan": "Сохраните незашифрованную копию скана",
  "system.dialog.select-import": "Выберите файл экспорта",
  "system.dialog.choose-backup-folder":
    "Выберите папку для автоматических копий",
  "system.filter.vault": "Хранилище Ravenpass",
  "system.filter.text": "Текстовый файл",
  "system.filter.photo": "Изображение",
  "system.filter.scan": "Изображение или PDF",
  "system.filter.import": "Экспорт хранилища",
  "system.file.vault": "vault.rpv",
  "system.file.export": "Экспорт Ravenpass.rpv",
  "system.file.recovery-key": "Ключ восстановления Ravenpass.txt",
  "system.reason.share-file":
    "передать сайту {site} файл «{file}» из профиля «{identity}»",
  "system.reason.save-passkey": "сохранить ключ доступа для сайта {site}",
  "system.reason.sign-in-passkey":
    "войти на сайт {site} как «{account}» с ключом доступа",
  "system.reason.fill-sign-in":
    "вставить данные входа «{account}» на сайте {site}",
  "system.reason.fill-card": "вставить карту «{card}» на сайте {site}",
  "system.reason.change-unlock": "изменить способ разблокировки хранилища",
  "system.reason.create-vault": "создать новое хранилище",
  "system.reason.open-vault": "открыть другой файл хранилища",
  "system.reason.delete-vault": "удалить хранилище «{vault}»",
  "system.reason.reveal-item": "показать «{item}»",
  "system.reason.unlock-vault": "разблокировать хранилище",
  "system.prompt.share-file.title": "Передача файла",
  "system.prompt.share-file":
    "Подтвердите, что это вы, чтобы передать сайту {site} файл «{file}» из профиля «{identity}».",
  "system.prompt.save-passkey.title": "Сохранение ключа доступа",
  "system.prompt.save-passkey":
    "Подтвердите, что это вы, чтобы сохранить ключ доступа для сайта {site}.",
  "system.prompt.sign-in-passkey.title": "Вход с ключом доступа",
  "system.prompt.sign-in-passkey":
    "Подтвердите, что это вы, чтобы войти на сайт {site} как «{account}».",
  "system.prompt.fill-sign-in.title": "Вставка данных входа",
  "system.prompt.fill-sign-in":
    "Подтвердите, что это вы, чтобы вставить данные входа «{account}» на сайте {site}.",
  "system.prompt.fill-card.title": "Вставка карты",
  "system.prompt.fill-card":
    "Подтвердите, что это вы, чтобы вставить карту «{card}» на сайте {site}.",
  "system.prompt.change-unlock.title": "Изменение разблокировки",
  "system.prompt.change-unlock":
    "Подтвердите, что это вы, чтобы изменить способ разблокировки хранилища.",
  "system.prompt.create-vault.title": "Создание хранилища",
  "system.prompt.create-vault":
    "Подтвердите, что это вы, чтобы создать новое хранилище.",
  "system.prompt.open-vault.title": "Открытие файла хранилища",
  "system.prompt.open-vault":
    "Подтвердите, что это вы, чтобы открыть другой файл хранилища.",
  "system.prompt.reveal-item.title": "Показ секрета",
  "system.prompt.reveal-item":
    "Подтвердите, что это вы, чтобы показать «{item}».",
  "system.prompt.delete-vault.title": "Удаление хранилища",
  "system.prompt.delete-vault":
    "Подтвердите, что это вы, чтобы удалить хранилище «{vault}».",
  "system.prompt.unlock-vault.title": "Разблокировка Ravenpass",
  "system.prompt.unlock-vault":
    "Подтвердите, что это вы, чтобы разблокировать хранилище.",
  "system.place.on-this-device": "На этом устройстве",
  "system.recovery-file.title": "Ключ восстановления Ravenpass",
  "system.recovery-file.notice":
    "Этот ключ открывает существующее зашифрованное хранилище Ravenpass или его копию. Он не содержит ваши пароли и не является копией хранилища. Этот файл не зашифрован. Храните его в тайне.",
  "system.recovery-sheet.printed": "Напечатано {date}",
  "system.recovery-sheet.keep":
    "Храните этот лист в надёжном месте, недоступном для других.",
  "system.recovery-sheet.warning":
    "Любой, у кого есть эти слова, сможет открыть ваше хранилище.",
  "system.menu.open": "Открыть Ravenpass",
  "system.menu.lock": "Заблокировать хранилище",
  "system.menu.quit": "Выйти из Ravenpass",
};
