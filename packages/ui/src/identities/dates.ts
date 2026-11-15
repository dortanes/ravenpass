import type { Locale } from "date-fns";
import { differenceInCalendarDays } from "date-fns/differenceInCalendarDays";
import { differenceInMonths } from "date-fns/differenceInMonths";
import { differenceInYears } from "date-fns/differenceInYears";
import { format } from "date-fns/format";
import { formatDuration } from "date-fns/formatDuration";
import { intervalToDuration } from "date-fns/intervalToDuration";
import { isValid } from "date-fns/isValid";
import { enUS } from "date-fns/locale/en-US";
import { ru } from "date-fns/locale/ru";
import { zhCN } from "date-fns/locale/zh-CN";
import { parse } from "date-fns/parse";
import { startOfDay } from "date-fns/startOfDay";

const storedPattern = "yyyy-MM-dd";
const storedForm = /^\d{4}-\d{2}-\d{2}$/;

/** Inclusive, counted back from the expiry date. */
export const expiringSoonDays = 90;

/** The earliest year the vault accepts in a date. */
export const earliestYear = 1900;

export function dateLocale(language: string): Locale {
  if (language === "ru") return ru;
  if (language === "zh") return zhCN;
  return enUS;
}

/** `new Date("YYYY-MM-DD")` reads UTC midnight; this reads local midnight. */
export class CalendarDate {
  readonly day: Date;

  private constructor(day: Date) {
    this.day = day;
  }

  /** Null for anything but `YYYY-MM-DD`, and for a day the calendar lacks, such as `2026-02-30`. */
  static parse(value: string): CalendarDate | null {
    if (!storedForm.test(value)) return null;
    const day = parse(value, storedPattern, new Date(0));
    return isValid(day) ? new CalendarDate(day) : null;
  }

  static of(moment: Date): CalendarDate {
    return new CalendarDate(startOfDay(moment));
  }

  stored(): string {
    return format(this.day, storedPattern);
  }

  /** Negative once the day has passed. */
  daysFrom(today: Date): number {
    return differenceInCalendarDays(this.day, today);
  }

  format(language: string): string {
    return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
      this.day,
    );
  }
}

/** How a typed date reads: `YYYY/MM/DD` in Chinese, `DD.MM.YYYY` in Russian, `MM/DD/YYYY` in English. */
export class DateEntryFormat {
  /** The day order, in the terms of Maskito's date mask. */
  readonly mode: "dd/mm/yyyy" | "mm/dd/yyyy" | "yyyy/mm/dd";
  readonly separator: "." | "/";
  private readonly pattern: string;
  private readonly form: RegExp;

  private constructor(mode: "dd/mm/yyyy" | "mm/dd/yyyy" | "yyyy/mm/dd") {
    this.mode = mode;
    if (mode === "yyyy/mm/dd") {
      this.separator = "/";
      this.pattern = "yyyy/MM/dd";
      this.form = /^\d{4}\/\d{2}\/\d{2}$/;
    } else if (mode === "dd/mm/yyyy") {
      this.separator = ".";
      this.pattern = "dd.MM.yyyy";
      this.form = /^\d{2}\.\d{2}\.\d{4}$/;
    } else {
      this.separator = "/";
      this.pattern = "MM/dd/yyyy";
      this.form = /^\d{2}\/\d{2}\/\d{4}$/;
    }
  }

  static for(language: string): DateEntryFormat {
    if (language === "zh") return new DateEntryFormat("yyyy/mm/dd");
    if (language === "ru") return new DateEntryFormat("dd/mm/yyyy");
    return new DateEntryFormat("mm/dd/yyyy");
  }

  /** Empty for an unreadable date. */
  display(stored: string): string {
    const date = CalendarDate.parse(stored);
    return date ? format(date.day, this.pattern) : "";
  }

  /** Empty text is no date; text that is not a whole, real date from `earliestYear` on is null. */
  read(typed: string): string | null {
    if (!typed) return "";
    if (!this.form.test(typed)) return null;
    const day = parse(typed, this.pattern, new Date(0));
    if (!isValid(day) || day.getFullYear() < earliestYear) return null;
    return CalendarDate.of(day).stored();
  }
}

export type ValidityState = "valid" | "expiring" | "expired";

type Unit = "year" | "month" | "day";

/** From the issue date when known until the end of the expiry day. */
export class Validity {
  readonly expires: CalendarDate;
  readonly issued: CalendarDate | null;

  private constructor(expires: CalendarDate, issued: CalendarDate | null) {
    this.expires = expires;
    this.issued = issued;
  }

  /** Null without a readable expiry date. */
  static of(expiresOn: string, issuedOn = ""): Validity | null {
    const expires = CalendarDate.parse(expiresOn);
    return expires ? new Validity(expires, CalendarDate.parse(issuedOn)) : null;
  }

  /** Zero on the expiry day, negative after it. */
  daysLeft(today: Date): number {
    return this.expires.daysFrom(today);
  }

  state(today: Date): ValidityState {
    const left = this.daysLeft(today);
    if (left < 0) return "expired";
    return left <= expiringSoonDays ? "expiring" : "valid";
  }

  /** From 1 to 0; without an issue date the period is the last `expiringSoonDays`. */
  share(today: Date): number {
    const left = this.daysLeft(today);
    const period = this.issued
      ? differenceInCalendarDays(this.expires.day, this.issued.day)
      : expiringSoonDays;
    if (period <= 0) return left >= 0 ? 1 : 0;
    return Math.min(Math.max(left / period, 0), 1);
  }

  /** The largest whole unit, such as `2y` or `5d`; empty once expired. */
  timeLeft(today: Date, language: string): string {
    const left = this.largestUnit(today);
    if (!left) return "";
    return new Intl.NumberFormat(language, {
      style: "unit",
      unit: left.unit,
      unitDisplay: "narrow",
    }).format(left.count);
  }

  /** Years and months, or days under a month, such as `8 years 11 months`; empty once expired. */
  timeLeftInWords(today: Date, language: string): string {
    const start = startOfDay(today);
    if (this.daysLeft(start) < 0) return "";
    const left = intervalToDuration({ start, end: this.expires.day });
    const locale = dateLocale(language);
    if (left.years || left.months) {
      return formatDuration(left, { format: ["years", "months"], locale });
    }
    return formatDuration(
      { days: left.days ?? 0 },
      { format: ["days"], zero: true, locale },
    );
  }

  /** Relative to today, such as `in 2 months` or `tomorrow`. */
  untilExpiry(today: Date, language: string): string {
    const left = this.largestUnit(today) ?? {
      count: this.daysLeft(today),
      unit: "day" as const,
    };
    return new Intl.RelativeTimeFormat(language, { numeric: "auto" }).format(
      left.count,
      left.unit,
    );
  }

  private largestUnit(today: Date): { count: number; unit: Unit } | null {
    const start = startOfDay(today);
    const days = this.daysLeft(start);
    if (days < 0) return null;
    const years = differenceInYears(this.expires.day, start);
    if (years >= 1) return { count: years, unit: "year" };
    const months = differenceInMonths(this.expires.day, start);
    if (months >= 1) return { count: months, unit: "month" };
    return { count: days, unit: "day" };
  }
}
