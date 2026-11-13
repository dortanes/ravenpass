import assert from "node:assert/strict";
import test from "node:test";
import {
  CalendarDate,
  DateEntryFormat,
  expiringSoonDays,
  Validity,
} from "./dates.ts";

const today = new Date(2026, 8, 21, 15, 30);

function inDays(days: number): string {
  const day = new Date(2026, 8, 21 + days);
  const month = String(day.getMonth() + 1).padStart(2, "0");
  const date = String(day.getDate()).padStart(2, "0");
  return `${day.getFullYear()}-${month}-${date}`;
}

function validity(expiresOn: string, issuedOn = ""): Validity {
  const read = Validity.of(expiresOn, issuedOn);
  assert.ok(read, `${expiresOn} was not read`);
  return read;
}

test("a stored date is read as a local calendar day", () => {
  const read = CalendarDate.parse("2026-09-21");
  assert.ok(read);
  assert.equal(read.day.getFullYear(), 2026);
  assert.equal(read.day.getMonth(), 8);
  assert.equal(read.day.getDate(), 21);
  assert.equal(read.day.getHours(), 0);
  assert.equal(read.daysFrom(today), 0);
});

test("only a real day in the stored form is read", () => {
  for (const value of [
    "",
    "2026-02-30",
    "2026-13-01",
    "2026-2-3",
    "21.09.2026",
    "2026-09-21T00:00:00Z",
  ]) {
    assert.equal(CalendarDate.parse(value), null, value);
  }
  assert.ok(CalendarDate.parse("2028-02-29"));
});

test("a date is shown in the interface language", () => {
  const read = CalendarDate.parse("2026-03-05");
  assert.ok(read);
  assert.equal(read.format("en"), "Mar 5, 2026");
});

test("a document without an expiry date has no validity", () => {
  assert.equal(Validity.of(""), null);
  assert.equal(Validity.of("not a date", "2020-01-01"), null);
});

test("expiring soon starts 90 days before the expiry day and lasts through it", () => {
  assert.equal(validity(inDays(expiringSoonDays + 1)).state(today), "valid");
  assert.equal(validity(inDays(expiringSoonDays)).state(today), "expiring");
  assert.equal(validity(inDays(0)).state(today), "expiring");
  assert.equal(validity(inDays(-1)).state(today), "expired");
});

test("days left count calendar days regardless of the time of day", () => {
  assert.equal(validity(inDays(1)).daysLeft(today), 1);
  assert.equal(validity(inDays(1)).daysLeft(new Date(2026, 8, 21, 23, 59)), 1);
  assert.equal(validity(inDays(-3)).daysLeft(today), -3);
});

test("with an issue date the ring shows the share of the period left", () => {
  assert.equal(validity(inDays(50), inDays(-50)).share(today), 0.5);
  assert.equal(validity(inDays(300), inDays(0)).share(today), 1);
  assert.equal(validity(inDays(-1), inDays(-100)).share(today), 0);
  assert.equal(validity(inDays(10), inDays(20)).share(today), 1);
});

test("without an issue date the ring is full until the last 90 days", () => {
  assert.equal(validity(inDays(400)).share(today), 1);
  assert.equal(validity(inDays(expiringSoonDays)).share(today), 1);
  assert.equal(validity(inDays(45)).share(today), 0.5);
  assert.equal(validity(inDays(0)).share(today), 0);
  assert.equal(validity(inDays(-10)).share(today), 0);
});

test("the time left is named in its largest whole unit", () => {
  assert.equal(validity("2029-10-01").timeLeft(today, "en"), "3y");
  assert.equal(validity("2027-09-20").timeLeft(today, "en"), "11m");
  assert.equal(validity(inDays(20)).timeLeft(today, "en"), "20d");
  assert.equal(validity(inDays(0)).timeLeft(today, "en"), "0d");
  assert.equal(validity(inDays(-1)).timeLeft(today, "en"), "");
});

test("the time left is stated in years and months", () => {
  assert.equal(
    validity("2035-08-21").timeLeftInWords(today, "en"),
    "8 years 11 months",
  );
  assert.equal(
    validity("2035-08-21").timeLeftInWords(today, "ru"),
    "8 лет 11 месяцев",
  );
  assert.equal(validity("2031-09-21").timeLeftInWords(today, "en"), "5 years");
  assert.equal(
    validity("2027-09-20").timeLeftInWords(today, "en"),
    "11 months",
  );
  assert.equal(validity("2026-12-25").timeLeftInWords(today, "ru"), "3 месяца");
});

test("under a month the time left is stated in days", () => {
  assert.equal(validity("2026-10-21").timeLeftInWords(today, "en"), "1 month");
  assert.equal(validity("2026-10-20").timeLeftInWords(today, "en"), "29 days");
  assert.equal(validity(inDays(1)).timeLeftInWords(today, "en"), "1 day");
  assert.equal(validity(inDays(1)).timeLeftInWords(today, "ru"), "1 день");
  assert.equal(validity(inDays(0)).timeLeftInWords(today, "en"), "0 days");
  assert.equal(validity(inDays(-1)).timeLeftInWords(today, "en"), "");
});

test("a typed date is read in the order of the interface language", () => {
  const russian = DateEntryFormat.for("ru");
  const english = DateEntryFormat.for("en");
  const chinese = DateEntryFormat.for("zh");
  assert.equal(russian.read("05.03.2026"), "2026-03-05");
  assert.equal(english.read("03/05/2026"), "2026-03-05");
  assert.equal(chinese.read("2026/03/05"), "2026-03-05");
  assert.equal(russian.display("2026-03-05"), "05.03.2026");
  assert.equal(english.display("2026-03-05"), "03/05/2026");
  assert.equal(chinese.display("2026-03-05"), "2026/03/05");
  assert.equal(english.display(""), "");
  assert.equal(russian.read(""), "");
  assert.equal(chinese.read(""), "");
});

test("a typed date that is not whole or not real is incomplete", () => {
  const russian = DateEntryFormat.for("ru");
  const chinese = DateEntryFormat.for("zh");
  for (const typed of [
    "05.03",
    "05.03.20",
    "30.02.2026",
    "05/03/2026",
    "01.01.1899",
  ]) {
    assert.equal(russian.read(typed), null, typed);
  }
  assert.equal(DateEntryFormat.for("en").read("13/01/2026"), null);
  assert.equal(chinese.read("2026/13/01"), null);
  assert.equal(chinese.read("2026/02/30"), null);
  assert.equal(chinese.read("1899/01/01"), null);
});

test("a picked day is stored as the vault keeps it", () => {
  assert.equal(
    CalendarDate.of(new Date(2026, 0, 9, 18, 45)).stored(),
    "2026-01-09",
  );
});

test("the expiry is phrased relative to today", () => {
  assert.equal(validity(inDays(1)).untilExpiry(today, "en"), "tomorrow");
  assert.equal(validity(inDays(20)).untilExpiry(today, "en"), "in 20 days");
  assert.equal(validity("2026-11-25").untilExpiry(today, "en"), "in 2 months");
  assert.equal(
    validity("2026-11-25").untilExpiry(today, "ru"),
    "через 2 месяца",
  );
});
