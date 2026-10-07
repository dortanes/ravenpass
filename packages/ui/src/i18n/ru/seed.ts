import type { seed as english } from "../en/seed.ts";

export const seed: typeof english = {
  "seed.untitled": "Сид без названия",
  "seed.search.label": "Поиск сидов",
  "seed.search.placeholder": "Поиск по названиям и кошелькам",

  "seed.list.label": "Сохранённые сиды",
  "seed.list.empty.all": "Добавленные сиды появятся здесь.",
  "seed.list.empty.recent": "Открытые сиды появятся здесь.",
  "seed.list.empty.pinned": "Сиды, добавленные в избранное, появятся здесь.",
  "seed.list.empty.search": "Подходящих сидов нет.",

  "seed.loading": "Загрузка сидов…",
  "seed.detail.loading": "Открываем сид…",
  "seed.empty.select.title": "Выберите сид",
  "seed.empty.select.detail":
    "Выберите сид в списке, чтобы увидеть подробности.",
  "seed.empty.first.title": "Добавьте первый сид",
  "seed.empty.first.detail":
    "Добавленные сид-фразы, закрытые ключи и резервные коды появятся в списке.",
  "seed.empty.first.action": "Добавить сид",

  "seed.summary.words":
    "{count, plural, one {# слово} few {# слова} many {# слов} other {# слова}}",
  "seed.summary.codes":
    "{left, plural, one {Остался # код} few {Осталось # кода} many {Осталось # кодов} other {Осталось # кода}} из {total}",
  "seed.summary.key": "Закрытый ключ",
  "seed.subtitle.phrase":
    "Сид-фраза из {count, plural, one {# слова} few {# слов} many {# слов} other {# слова}}",
  "seed.subtitle.codes": "Резервные коды",

  "seed.format.label": "Формат",
  "seed.format.phrase": "Сид-фраза",
  "seed.format.key": "Закрытый ключ",
  "seed.format.codes": "Резервные коды",

  "seed.checksum.valid": "Контрольная сумма верна",
  "seed.checksum.invalid": "Контрольная сумма не сходится",
  "seed.checksum.unknown": "Не фраза BIP-39",
  "seed.checked.on": "Проверено {date}",
  "seed.checked.never": "Копия ещё не проверялась",

  "seed.block.phrase": "Фраза",
  "seed.block.key": "Ключ",
  "seed.block.codes": "Коды",
  "seed.block.wallet": "Кошелёк",
  "seed.block.addresses": "Адреса",

  "seed.phrase.reveal": "Показать все слова на 30 секунд",
  "seed.phrase.conceal": "Скрыть слова",
  "seed.phrase.hold": "Удерживайте, чтобы показать слово {number}",
  "seed.phrase.left":
    "{seconds, plural, one {Осталась # секунда} few {Осталось # секунды} many {Осталось # секунд} other {Осталось # секунды}}",

  "seed.field.label": "Название",
  "seed.field.label.placeholder": "Основной кошелёк",
  "seed.field.phrase": "Фраза",
  "seed.field.passphrase": "Дополнительный пароль",
  "seed.field.path": "Путь деривации",
  "seed.field.path.note":
    "Путь деривации определяет, какие счета кошелёк получит из сид-фразы. Оставьте поле пустым, если кошелёк не показывал путь.",
  "seed.field.key": "Ключ",
  "seed.field.codes": "Коды",
  "seed.field.wallet": "Кошелёк",
  "seed.field.wallet.placeholder": "Аппаратный кошелёк",
  "seed.field.address-label": "Подпись",
  "seed.field.address": "Адрес",
  "seed.address.untitled": "Адрес {number}",

  "seed.passphrase.reveal": "Показать дополнительный пароль",
  "seed.passphrase.conceal": "Скрыть дополнительный пароль",
  "seed.key.reveal": "Показать ключ",
  "seed.key.conceal": "Скрыть ключ",

  "seed.copy.phrase": "Скопировать фразу",
  "seed.copy.passphrase": "Скопировать дополнительный пароль",
  "seed.copy.path": "Скопировать путь",
  "seed.copy.key": "Скопировать ключ",
  "seed.copy.address": "Скопировать адрес",
  "seed.copy.next-code": "Скопировать следующий неиспользованный код",
  "seed.copied.phrase": "Сид-фраза скопирована.",
  "seed.copied.passphrase": "Дополнительный пароль скопирован.",
  "seed.copied.path": "Путь скопирован.",
  "seed.copied.key": "Закрытый ключ скопирован.",
  "seed.copied.address": "Адрес скопирован.",
  "seed.copied.code": "Код скопирован и отмечен как использованный.",

  "seed.copy-phrase.title": "Скопировать всю фразу?",
  "seed.copy-phrase.detail":
    "Любой, кто узнает эту фразу, сможет получить доступ к вашим средствам. Другие приложения могут прочитать содержимое буфера обмена, пока он не очистится.",
  "seed.copy-phrase.confirm": "Скопировать фразу",

  "seed.code.use": "Скопировать код {number} и отметить его как использованный",
  "seed.code.used": "Использован",

  "seed.check.open": "Проверить копию",
  "seed.check.detail": "Введите из своей копии слова с указанных позиций.",
  "seed.check.word": "Слово {number}",
  "seed.check.empty": "Введите это слово.",
  "seed.check.mismatch":
    "Слова не совпадают. Сверьте записанную копию с фразой и повторите.",
  "seed.check.submit": "Проверить",
  "seed.check.done": "Проверка копии завершена. Дата проверки сохранена.",

  "seed.editor.new.title": "Новый сид",
  "seed.editor.edit.title": "Изменить сид",
  "seed.editor.create": "Добавить сид",
  "seed.editor.requirement.phrase": "Укажите название и фразу",
  "seed.editor.requirement.key": "Укажите название и ключ",
  "seed.editor.requirement.codes": "Укажите название и хотя бы один код",
  "seed.editor.unfinished": "Исправьте выделенные значения, чтобы сохранить.",
  "seed.editor.phrase.placeholder": "Вставьте или введите слова через пробел",
  "seed.editor.phrase.unknown": "Слова не из списка BIP-39: {positions}",
  "seed.editor.phrase.too-many":
    "Во фразе может быть не больше {limit, plural, one {# слова} few {# слов} many {# слов} other {# слова}}.",
  "seed.editor.phrase.too-long":
    "В слове может быть не больше {limit, plural, one {# символа} few {# символов} many {# символов} other {# символа}}.",
  "seed.editor.phrase.suggestions": "Подходящие слова",
  "seed.editor.codes.placeholder": "По одному коду в строке",
  "seed.editor.codes.count":
    "{count, plural, one {# код} few {# кода} many {# кодов} other {# кода}}",
  "seed.editor.codes.too-many":
    "Можно добавить не больше {limit, plural, one {# кода} few {# кодов} many {# кодов} other {# кода}}.",
  "seed.editor.codes.too-long":
    "В коде может быть не больше {limit, plural, one {# символа} few {# символов} many {# символов} other {# символа}}.",
  "seed.editor.codes.used": "Использованные коды",
  "seed.editor.codes.used.detail":
    "Отметьте коды, которые вы уже использовали.",
  "seed.editor.address.add": "Адрес",
  "seed.editor.address.remove": "Удалить адрес",
  "seed.editor.address.missing": "Введите каждый адрес или удалите его строку.",

  "seed.error.list":
    "Не удалось загрузить ваши сиды. Перезапустите приложение.",
  "seed.error.close":
    "Не удалось закрыть выбранный сид. Заблокируйте хранилище.",
  "seed.error.open":
    "Не удалось открыть этот сид. Выберите его снова или перезапустите Ravenpass.",
  "seed.error.copy":
    "Не удалось скопировать это значение. Откройте сид и повторите.",
  "seed.error.pin":
    "Не удалось добавить этот сид в избранное. Повторите попытку.",
  "seed.error.unpin":
    "Не удалось убрать этот сид из избранного. Повторите попытку.",
  "seed.error.save":
    "Не удалось сохранить этот сид. Проверьте введённые данные и повторите.",
  "seed.error.saved-partly": "Сид сохранён, но {detail}",
  "seed.error.delete": "Не удалось удалить этот сид. Повторите попытку.",
  "seed.error.duplicate":
    "Не удалось создать копию этого сида. Повторите попытку.",
  "seed.error.deleted-partly": "Сид удалён, но {detail}",
  "seed.error.use-code":
    "Не удалось отметить этот код использованным. Откройте сид и повторите.",
  "seed.error.check":
    "Не удалось сохранить результат проверки. Откройте сид и повторите.",
  "seed.error.wordlist":
    "Не удалось загрузить список слов BIP-39. Вводите каждое слово полностью.",
};
