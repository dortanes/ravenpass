import type { note as english } from "../en/note.ts";

export const note: typeof english = {
  "note.untitled": "Заметка без названия",
  "note.search.label": "Поиск заметок",
  "note.search.placeholder": "Поиск по названиям и первым строкам",

  "note.list.label": "Сохранённые заметки",
  "note.list.empty.all": "Добавленные заметки появятся здесь.",
  "note.list.empty.recent": "Открытые заметки появятся здесь.",
  "note.list.empty.pinned": "Заметки, добавленные в избранное, появятся здесь.",
  "note.list.empty.search": "Подходящих заметок нет.",
  "note.preview.hidden": "Текст скрыт",
  "note.preview.empty": "Без текста",

  "note.loading": "Загрузка заметок…",
  "note.detail.loading": "Открываем заметку…",
  "note.empty.select.title": "Выберите заметку",
  "note.empty.select.detail": "Выберите заметку в списке, чтобы прочитать её.",
  "note.empty.first.title": "Добавьте первую заметку",
  "note.empty.first.detail": "Добавленные заметки появятся в списке.",
  "note.empty.first.action": "Добавить заметку",

  "note.words":
    "{count, plural, one {# слово} few {# слова} many {# слов} other {# слова}}",
  "note.body.empty": "В этой заметке нет текста.",
  "note.hidden.detail":
    "Текст скрыт. Нажмите «Показать заметку», чтобы прочитать его.",
  "note.hidden.show": "Показать заметку",
  "note.hidden.hide": "Скрыть заметку",

  "note.field.label": "Название",
  "note.field.label.placeholder": "Домашний Wi-Fi",
  "note.field.body": "Текст",
  "note.field.hidden": "Скрывать текст",

  "note.copy": "Скопировать заметку",
  "note.copied": "Заметка скопирована.",

  "note.editor.new.title": "Новая заметка",
  "note.editor.edit.title": "Изменить заметку",
  "note.editor.create": "Добавить заметку",
  "note.editor.requirement": "Укажите название",

  "note.error.list":
    "Не удалось загрузить ваши заметки. Перезапустите приложение.",
  "note.error.close":
    "Не удалось закрыть выбранную заметку. Заблокируйте хранилище.",
  "note.error.open":
    "Не удалось открыть эту заметку. Выберите её снова или перезапустите Ravenpass.",
  "note.error.copy":
    "Не удалось скопировать эту заметку. Откройте её и повторите.",
  "note.error.pin":
    "Не удалось добавить эту заметку в избранное. Повторите попытку.",
  "note.error.unpin":
    "Не удалось убрать эту заметку из избранного. Повторите попытку.",
  "note.error.save":
    "Не удалось сохранить эту заметку. Проверьте введённые данные и повторите.",
  "note.error.saved-partly": "Заметка сохранена, но {detail}",
  "note.error.delete": "Не удалось удалить эту заметку. Повторите попытку.",
  "note.error.duplicate":
    "Не удалось создать копию этой заметки. Повторите попытку.",
  "note.error.deleted-partly": "Заметка удалена, но {detail}",
};
