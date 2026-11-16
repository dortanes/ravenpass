import type { access as english } from "../en/access.ts";

export const access: typeof english = {
  "wizard.steps.label": "Ход настройки",
  "wizard.steps.storage": "Файл хранилища",
  "wizard.steps.phrase": "Ключ восстановления",
  "wizard.steps.confirm": "Проверка ключа",
  "wizard.steps.unlock": "Способ входа",
  "wizard.storage.title": "Выберите место для хранилища",
  "wizard.storage.title-fixed": "Хранилище остаётся на этом устройстве",
  "wizard.storage.description":
    "Все данные Ravenpass хранятся в одном зашифрованном файле.",
  "wizard.storage.type": "Тип хранилища",
  "wizard.storage.location": "Расположение",
  "wizard.storage.file-empty": "Проверяем…",
  "wizard.storage.change": "Изменить…",
  "wizard.storage.change-busy": "Выбираем…",
  "wizard.storage.fact-encrypted":
    "Шифруется на этом устройстве ещё до записи на диск.",
  "wizard.storage.fact-saved": "Каждое изменение сохраняется сразу.",
  "wizard.storage.fact-backup":
    "Отдельную зашифрованную копию можно сохранить в настройках.",
  "wizard.storage.unrestricted":
    "В этом расположении нельзя ограничить доступ к файлу.",
  "wizard.phrase.title": "Сохраните ключ восстановления",
  "wizard.phrase.description":
    "Для восстановления хранилища понадобятся этот ключ и зашифрованная копия.",
  "wizard.phrase.list": "Ключ восстановления из 24 слов",
  "wizard.phrase.copy": "Скопировать ключ",
  "wizard.phrase.copy-busy": "Копируем…",
  "wizard.phrase.copy-done":
    "Скопировано. Сохранив ключ, очистите буфер обмена.",
  "wizard.phrase.save": "Сохранить в файл…",
  "wizard.phrase.save-busy": "Сохраняем…",
  "wizard.phrase.save-done":
    "Сохранено. Храните файл с ключом отдельно от зашифрованного хранилища.",
  "wizard.phrase.print": "Распечатать…",
  "wizard.phrase.print-busy": "Печатаем…",
  "wizard.phrase.note":
    "Файл с ключом не зашифрован. Любой, у кого есть ключ и зашифрованная копия, сможет открыть ваше хранилище.",
  "wizard.phrase.note-words":
    "Любой, у кого есть этот ключ и зашифрованная копия, сможет открыть ваше хранилище.",
  "wizard.phrase.acknowledgment":
    "Ключ восстановления сохранён в надёжном месте.",
  "wizard.confirm.title": "Проверьте ключ восстановления",
  "wizard.confirm.description":
    "Введите три слова из ключа, который вы только что сохранили.",
  "wizard.confirm.words": "Слова из ключа восстановления",
  "wizard.confirm.hint": "Нажмите пробел, чтобы перейти к следующему слову.",
  "phrase.word": "Слово {number}",
  "wizard.unlock.title": "Выберите, как открывать это хранилище",
  "wizard.unlock.description":
    "Действует только на этом устройстве. Изменить можно в настройках.",
  "wizard.unlock.create": "Создать хранилище",
  "wizard.unlock.creating": "Создаём хранилище",
  "wizard.actions.back": "Назад",
  "wizard.actions.cancel": "Отмена",
  "wizard.actions.continue": "Продолжить",
  "wizard.actions.preparing": "Готовим…",
  "wizard.errors.storage-unreadable":
    "Не удалось определить, где будет находиться хранилище. Перезапустите приложение.",
  "wizard.errors.location-rejected":
    "Это расположение не подходит. Выберите другую папку и повторите.",
  "wizard.errors.begin-failed":
    "Не удалось начать настройку. Повторите попытку.",
  "wizard.errors.cancel-failed":
    "Не удалось закрыть настройку. Перезапустите приложение.",
  "wizard.errors.leave-failed":
    "Не удалось вернуться к другому хранилищу. Повторите попытку.",
  "wizard.errors.copy-failed":
    "Не удалось скопировать ключ. Повторите или сохраните его в файл.",
  "wizard.errors.copy-failed-no-file":
    "Не удалось скопировать ключ. Повторите или запишите слова.",
  "wizard.errors.save-failed":
    "Не удалось сохранить ключ. Выберите другое расположение и повторите.",
  "wizard.errors.print-failed":
    "Не удалось распечатать ключ. Повторите или запишите слова.",
  "wizard.errors.words-mismatch":
    "Эти слова не совпадают. Проверьте ключ восстановления и повторите.",
  "wizard.errors.finish-failed":
    "Не удалось завершить настройку. Повторите попытку.",

  "unlock-methods.biometry.title": "Биометрия",
  "unlock-methods.biometry.description": "Биометрия устройства или его пароль.",
  "unlock-methods.biometry.unavailable":
    "Биометрия недоступна на этом устройстве.",
  "unlock-methods.biometry.creating":
    "Создаём ключ в защищённом модуле устройства. Это может занять до 20 секунд.",
  "unlock-methods.biometry.last":
    "Это единственный способ разблокировки на этом устройстве. Задайте пин-код, чтобы отключить биометрию.",
  "unlock-methods.pin.last":
    "Это единственный способ разблокировки на этом устройстве. Включите биометрию, чтобы отключить пин-код.",
  "unlock-methods.pin.only":
    "Пин-код — единственный способ разблокировки, который работает на этом устройстве.",
  "unlock-methods.pin.title": "Пин-код",
  "unlock-methods.pin.description":
    "От {min} до {max, plural, one {# буквы или цифры} few {# букв или цифр} many {# букв или цифр} other {# буквы или цифры}}.",
  "unlock-methods.pin.weaker": "Не используйте пин-код, который легко угадать.",
  "unlock-methods.pin.set": "Задать пин-код",
  "unlock-methods.pin.change": "Сменить пин-код",
  "unlock-methods.pin.current": "Текущий пин-код",
  "unlock-methods.pin.new": "Новый пин-код",
  "unlock-methods.pin.repeat": "Повторите пин-код",
  "unlock-methods.pin.save": "Сохранить пин-код",
  "unlock-methods.pin.saving": "Сохраняем…",
  "unlock-methods.pin.cancel": "Отмена",
  "unlock-methods.pin.mismatch": "Пин-коды не совпадают.",
  "unlock-methods.pin.saved": "Пин-код сохранён.",
  "unlock-methods.pin.removed": "Пин-код удалён.",
  "unlock-methods.biometry.on": "Биометрия включена.",
  "unlock-methods.biometry.off": "Биометрия выключена.",
  "unlock-methods.confirm.title": "Введите пин-код",
  "unlock-methods.confirm.description":
    "Подтвердите, что это вы, чтобы изменить способ разблокировки хранилища.",
  "unlock-methods.confirm.action": "Продолжить",
  "unlock-methods.confirm-key.title": "Введите ключ восстановления",
  "unlock-methods.confirm-key.description":
    "На этом устройстве нельзя подтвердить вашу личность пин-кодом или биометрией. Введите ключ восстановления хранилища, чтобы изменить способ разблокировки.",
  "unlock-methods.errors.save-failed":
    "Не удалось сохранить изменение. Повторите попытку.",
  "unlock-methods.errors.unreadable":
    "Не удалось определить способ открытия этого хранилища. Перезапустите приложение.",
  "unlock-methods.choose": "Выберите хотя бы один способ, чтобы продолжить.",

  "unlock.title": "Разблокируйте хранилище",
  "unlock.action": "Разблокировать",
  "unlock.action-busy": "Разблокируем…",
  "unlock.biometry-action": "Использовать биометрию",
  "unlock.back": "Вернуться к началу",
  "unlock.restore.title": "Восстановите доступ к хранилищу",
  "unlock.restore.reason":
    "На этом устройстве не настроен способ разблокировки.",
  "unlock.restore.next":
    "Введите ключ восстановления и настройте новый способ разблокировки.",
  "unlock.missing.title": "Файл хранилища не найден",
  "unlock.missing.reason":
    "Файла хранилища больше нет по адресу {location}. Возможно, его перенесли на другом устройстве.",
  "unlock.missing.next":
    "Откройте его из нового места, чтобы разблокировать здесь.",
  "unlock.missing.action": "Открыть файл хранилища…",
  "unlock.pin.label": "Пин-код",
  "unlock.pin.placeholder": "Ваш пин-код",
  "unlock.pin.action": "Разблокировать",
  "unlock.pin.busy": "Проверяем…",
  "unlock.pin.attempts":
    "{count, plural, one {Осталась # попытка} few {Осталось # попытки} many {Осталось # попыток} other {Осталось # попытки}} до удаления пин-кода.",
  "unlock.none.title": "На этом устройстве нет способа открыть хранилище",
  "unlock.none.description":
    "Используйте ключ восстановления, чтобы настроить новый способ разблокировки.",
  "unlock.recovery.title": "Восстановите доступ",
  "unlock.recovery.description":
    "Введите ключ восстановления из 24 слов, чтобы открыть хранилище.",
  "unlock.recovery.action": "Ввести ключ восстановления",
  "unlock.errors.failed":
    "Не удалось разблокировать хранилище. Повторите или воспользуйтесь ключом восстановления.",
  "unlock.errors.switch-failed":
    "Не удалось открыть это хранилище. Выберите другое и повторите.",
  "unlock.errors.back-failed":
    "Не удалось закрыть это хранилище. Повторите попытку.",
  "unlock.errors.adopt-failed":
    "Не удалось открыть эту версию. Разблокируйте хранилище и повторите.",
  "unlock.diverged.title": "Изменено на другом устройстве",
  "unlock.diverged.description":
    "Это хранилище одновременно изменили на другом устройстве. Последние изменения на этом устройстве заменены той версией.",
  "unlock.diverged.confirm": "Открыть эту версию",
  "unlock.diverged.cancel": "Не сейчас",

  "recovery.phrase.label": "Ключ восстановления",
  "recovery.phrase.show": "Показать ключ восстановления",
  "recovery.phrase.hide": "Скрыть ключ восстановления",
  "recovery.preview.older.title": "Эта копия может быть устаревшей",
  "recovery.preview.older.description":
    "После восстановления могут пропасть записи, добавленные позже этой копии.",
  "recovery.preview.accept-loss":
    "Я понимаю, что новые записи могут быть потеряны.",
  "recovery.preview.replaced-key.title":
    "Эта копия использует заменённый ключ восстановления",
  "recovery.preview.replaced-key.description":
    "Ключ восстановления этого хранилища с тех пор заменили. Эту копию мог записать любой, у кого есть старый ключ. Открывайте её, только если восстановили её сами.",
  "recovery.preview.accept-replaced-key": "Эта копия восстановлена мной.",
  "recovery.actions.back": "Назад",
  "recovery.actions.continue": "Продолжить",
  "recovery.actions.checking": "Проверяем…",
  "recovery.unlock.title": "Выберите, как открывать это хранилище",
  "recovery.unlock.description":
    "Выберите способ разблокировки для этого устройства. Позже его можно изменить в настройках.",
  "recovery.actions.restore": "Восстановить доступ",
  "recovery.restoring": "Восстанавливаем доступ к хранилищу",
  "recovery.errors.phrase-missing":
    "Сначала введите ключ восстановления из 24 слов.",
  "recovery.errors.local-failed":
    "Не удалось открыть хранилище. Повторите попытку.",
  "recovery.errors.confirm-failed":
    "Не удалось завершить восстановление. Перезапустите приложение и повторите.",
  "recovery.errors.cancel-failed":
    "Не удалось закрыть восстановление. Перезапустите приложение.",

  "storage-unavailable.title": "Место хранения недоступно",
  "storage-unavailable.description-folder":
    "Нет доступа к {place}. Восстановите доступ или выберите файл хранилища.",
  "storage-unavailable.description": "Нет доступа к папке с вашим хранилищем.",
  "storage-unavailable.note":
    "На этом устройстве нет второй копии. Восстановите доступ к месту, где лежит файл хранилища, или укажите, где он находится сейчас.",
  "storage-unavailable.actions.locate": "Выбрать файл хранилища…",
  "storage-unavailable.actions.retry": "Повторить",
  "storage-unavailable.actions.retry-busy": "Проверяем…",
  "storage-unavailable.errors.unreadable":
    "Не удалось определить, где находится хранилище. Перезапустите приложение.",
  "storage-unavailable.errors.unreachable":
    "Расположение по-прежнему недоступно. Восстановите доступ или выберите файл хранилища.",
  "storage-unavailable.errors.switch-failed":
    "Не удалось открыть это хранилище. Выберите другое и повторите.",
  "storage-unavailable.errors.create-failed":
    "Не удалось начать создание хранилища. Повторите попытку.",
  "storage-unavailable.errors.open-failed":
    "Не удалось открыть файл. Выберите файл хранилища Ravenpass.",
};
