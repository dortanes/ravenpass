import type { extension as english } from "../en/extension.ts";

export const extension: typeof english = {
  "extension.link.title": "Привязка к Ravenpass",
  "extension.link.description":
    "В Ravenpass на этом компьютере откройте «Настройки» → «Автозаполнение», нажмите «Привязать расширение» и скопируйте ключ.",
  "extension.link.key": "Ключ подключения",
  "extension.link.paste": "Вставить ключ",
  "extension.link.action": "Привязать",
  "extension.link.busy": "Привязываем…",
  "extension.link.note": "Ключ можно использовать один раз в течение 5 минут.",
  "extension.link.error.not-key":
    "Это не ключ Ravenpass. Скопируйте его в Ravenpass ещё раз.",
  "extension.link.error.expired":
    "Срок действия ключа истёк. Создайте новый ключ в Ravenpass и вставьте его сюда.",
  "extension.link.error.not-open":
    "Ravenpass не открыт на этом компьютере. Откройте его и повторите попытку.",
  "extension.link.error.failed":
    "Не удалось привязать расширение. Повторите попытку.",
  "extension.link.error.clipboard":
    "Не удалось прочитать буфер обмена. Вставьте ключ в поле вручную.",
  "extension.linked.title": "Привязано к Ravenpass",
  "extension.linked.description": "Ravenpass предлагает пароли в полях входа.",
  "extension.linked.desktop": "Ravenpass на этом компьютере",
  "extension.linked.desktop.unlocked": "Разблокирован",
  "extension.linked.desktop.locked":
    "Заблокирован. Разблокируйте Ravenpass, чтобы подставлять пароли.",
  "extension.linked.desktop.not-open":
    "Не открыт. Откройте Ravenpass, чтобы подставлять пароли.",
  "extension.linked.since": "Привязано",
  "extension.chrome-autofill": "Отключить автозаполнение Chrome",
  "extension.chrome-autofill.detail":
    "Подсказки Chrome не будут перекрывать подсказки Ravenpass.",
  "extension.chrome-autofill.error":
    "Не удалось изменить настройку. Повторите попытку.",
  "extension.unlink.action": "Отвязать",
  "extension.unlink.title": "Отвязать расширение?",
  "extension.unlink.detail":
    "Ravenpass перестанет предлагать пароли в этом браузере. Чтобы привязать расширение снова, понадобится новый ключ из Ravenpass.",
  "extension.unlink.cancel": "Отмена",
  "extension.unlink.error":
    "Не удалось отвязать расширение. Повторите попытку.",
  "extension.unlinked":
    "Расширение отвязано в Ravenpass. Привяжите его снова, чтобы им пользоваться.",
  "extension.menu.label": "Пароли для {site}",
  "extension.menu.codes.label": "Одноразовые коды для {site}",
  "extension.menu.locked.title": "Ravenpass заблокирован",
  "extension.menu.locked.detail":
    "Разблокируйте его, чтобы подставлять пароли.",
  "extension.menu.locked.action": "Разблокировать",
  "extension.menu.unlock.error":
    "Не удалось разблокировать Ravenpass. Откройте приложение Ravenpass и повторите попытку.",
  "extension.menu.not-open.title": "Ravenpass не открыт",
  "extension.menu.not-open.detail":
    "Откройте его на этом компьютере, чтобы подставлять пароли.",
  "extension.menu.empty.title": "Для этого сайта нет паролей",
  "extension.menu.empty.detail":
    "Здесь появятся пароли, сохранённые в Ravenpass для этого сайта.",
  "extension.menu.codes.empty.title": "Для этого сайта нет кодов",
  "extension.menu.codes.empty.detail":
    "Здесь появятся одноразовые коды, настроенные в Ravenpass для этого сайта.",
  "extension.menu.cards.label": "Карты в Ravenpass",
  "extension.menu.cards.locked.detail":
    "Разблокируйте его, чтобы подставлять карты.",
  "extension.menu.cards.not-open.detail":
    "Откройте его на этом компьютере, чтобы подставлять карты.",
  "extension.menu.cards.empty.title": "В Ravenpass нет карт",
  "extension.menu.cards.empty.detail":
    "Здесь появятся карты, добавленные в Ravenpass.",
  "extension.menu.cards.error":
    "Не удалось подставить карту. Повторите попытку.",
  "extension.context.passwords": "Показать пароли",
  "extension.context.codes": "Показать одноразовые коды",
  "extension.context.cards": "Показать карты",
  "extension.menu.code.waiting": "Следующий код подставится через {seconds} с",
  "extension.menu.passkey.detail": "Ключ доступа · {account}",
  "extension.menu.hint.show": "показать",
  "extension.menu.hint.fill": "подставить",
  "extension.menu.hint.open": "открыть",
  "extension.menu.hint.upload": "загрузить",
  "extension.card.sign-in.title": "Войти на {site}",
  "extension.card.code.title": "Код для {site}",
  "extension.card.sign-in-as": "Войти как {account}",
  "extension.card.close": "Закрыть",
  "extension.confirm.password.title": "Подставить пароль на этой странице?",
  "extension.confirm.code.title": "Подставить код на этой странице?",
  "extension.confirm.page": "Эта страница",
  "extension.confirm.saved": "Сохранено для",
  "extension.confirm.insecure-page":
    "Страница открыта без защищённого соединения. Подставляйте, только если доверяете ей.",
  "extension.confirm.other-site":
    "Это не тот сайт, для которого вы сохранили данные. Подставляйте, только если доверяете этой странице.",
  "extension.confirm.remember": "Запомнить этот сайт для записи",
  "extension.confirm.cancel": "Отмена",
  "extension.confirm.fill": "Подставить",
  "extension.fill.confirm-on-device.title": "Подтвердите на Mac",
  "extension.fill.confirm-on-device.detail":
    "Подтвердите подстановку с помощью Touch ID или пароля от Mac.",
  "extension.fill.confirm-in-ravenpass.title": "Подтвердите в Ravenpass",
  "extension.fill.confirm-in-ravenpass.detail":
    "Введите пин-код в окне Ravenpass на Mac.",
  "extension.fill.declined": "Подстановка отменена на Mac.",
  "extension.fill.unverifiable":
    "Включите биометрию или задайте пин-код в Ravenpass, чтобы подтверждать подстановку.",
  "extension.offer.locked.detail": "Разблокируйте его, чтобы сохранить пароль.",
  "extension.offer.card.locked.detail":
    "Разблокируйте его, чтобы сохранить карту.",
  "extension.offer.not-open.detail":
    "Откройте его на этом компьютере, чтобы сохранять пароли.",
  "extension.passkey.sign-in.title": "Войти с ключом доступа",
  "extension.passkey.save.title": "Сохранить ключ доступа",
  "extension.passkey.list.label": "Ключи доступа для {site}",
  "extension.passkey.target.add": "Добавить в «{label}»",
  "extension.passkey.save": "Сохранить",
  "extension.passkey.elsewhere": "Другие способы",
  "extension.passkey.excluded.title":
    "В Ravenpass уже есть ключ доступа для этого аккаунта",
  "extension.passkey.excluded.detail": "Войдите с его помощью.",
  "extension.passkey.locked.detail":
    "Разблокируйте его, чтобы использовать ключи доступа.",
  "extension.passkey.not-open.detail":
    "Откройте его на этом компьютере, чтобы использовать ключи доступа.",
  "extension.passkey.confirm-on-device.title": "Подтвердите на Mac",
  "extension.passkey.confirm-on-device.detail":
    "Подтвердите запрос с помощью Touch ID или пароля от Mac.",
  "extension.passkey.confirm-in-ravenpass.title": "Подтвердите в Ravenpass",
  "extension.passkey.confirm-in-ravenpass.detail":
    "Введите пин-код в окне Ravenpass.",
  "extension.passkey.declined":
    "Не удалось подтвердить, что это вы. Повторите попытку или используйте другое устройство.",
  "extension.passkey.unverifiable":
    "Включите биометрию или задайте пин-код в Ravenpass, чтобы использовать ключи доступа.",
  "extension.passkey.full":
    "В этой записи уже больше нельзя хранить ключи доступа. Сохраните ключ в новую запись.",
  "extension.passkey.sign-in.error":
    "Не удалось войти с этим ключом доступа. Повторите попытку.",
  "extension.passkey.save.error":
    "Не удалось сохранить ключ доступа. Повторите попытку.",
  "extension.upload.menu": "Загрузить файл из профиля…",
  "extension.upload.identities.label": "Файлы профилей для {site}",
  "extension.upload.files.label": "Файлы профиля {identity}",
  "extension.upload.back": "Все профили",
  "extension.upload.photo": "Фото",
  "extension.upload.file.detail": "{name} · файл {format}",
  "extension.upload.identity.empty": "Нет фото и сканов",
  "extension.upload.empty.title": "Профилей пока нет",
  "extension.upload.empty.detail":
    "Здесь появятся фотографии и сканы документов из профилей Ravenpass.",
  "extension.upload.no-destination.title": "Здесь нельзя загрузить файл",
  "extension.upload.no-destination.detail":
    "Щёлкните правой кнопкой по полю файла или области для перетаскивания.",
  "extension.upload.locked.detail": "Разблокируйте его, чтобы загружать файлы.",
  "extension.upload.not-open.detail":
    "Откройте его на этом компьютере, чтобы загружать файлы.",
  "extension.upload.unlinked.title": "Расширение не привязано",
  "extension.upload.unlinked.detail":
    "Нажмите кнопку расширения на панели инструментов, чтобы привязать его к Ravenpass.",
  "extension.upload.confirm-on-device.title": "Подтвердите на Mac",
  "extension.upload.confirm-on-device.detail":
    "Подтвердите запрос с помощью Touch ID или пароля от Mac.",
  "extension.upload.confirm-in-ravenpass.title": "Подтвердите в Ravenpass",
  "extension.upload.confirm-in-ravenpass.detail":
    "Введите пин-код в окне Ravenpass.",
  "extension.upload.declined.title": "Передача файла отклонена",
  "extension.upload.declined.detail":
    "Файл не передан. Выберите его снова, чтобы повторить.",
  "extension.upload.unverifiable.title":
    "Ravenpass не может подтвердить, что это вы",
  "extension.upload.unverifiable.detail":
    "Включите биометрию или задайте пин-код в Ravenpass, чтобы загружать файлы.",
  "extension.upload.not-taken.title": "Страница не приняла файл",
  "extension.upload.not-taken.detail":
    "Щёлкните правой кнопкой по самому полю файла и повторите попытку.",
  "extension.upload.identities.error":
    "Не удалось получить список профилей. Повторите попытку.",
  "extension.upload.error": "Не удалось загрузить файл. Повторите попытку.",
};
