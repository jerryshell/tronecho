import {
  CalendarDate,
  DateFormatter,
  fromDate,
  getDayOfWeek,
  getLocalTimeZone,
  now,
  toCalendarDate,
} from "@internationalized/date";

type Duration = {
  years?: number;
  months?: number;
  weeks?: number;
  days?: number;
  hours?: number;
  minutes?: number;
  seconds?: number;
};

interface DateRange {
  start: Date;
  end: Date;
}

// zh-CN `month: "short"` renders as "3月" (matching date-fns "M月"), so combined
// with numeric day/year it reproduces the previous date-fns output byte-for-byte.
const timeFmt = new DateFormatter("zh-CN", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});
const monthDayFmt = new DateFormatter("zh-CN", { month: "short", day: "numeric" });
const monthDayTimeFmt = new DateFormatter("zh-CN", {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});
const yearMonthFmt = new DateFormatter("zh-CN", { year: "numeric", month: "short" });

function toDate(date: Date | string): Date {
  return typeof date === "string" ? new Date(date) : date;
}

function toCalendarDateFrom(date: Date): CalendarDate {
  return toCalendarDate(fromDate(date, getLocalTimeZone()));
}

export function formatTime(date: Date | string): string {
  return timeFmt.format(toDate(date));
}

export function formatMonthDay(date: Date | string): string {
  return monthDayFmt.format(toDate(date));
}

export function formatMonthDayTime(date: Date | string): string {
  return monthDayTimeFmt.format(toDate(date));
}

export function formatYearMonth(date: Date | string): string {
  return yearMonthFmt.format(toDate(date));
}

export function isToday(date: Date | string): boolean {
  const today = toCalendarDateFrom(new Date());
  const target = toCalendarDateFrom(toDate(date));
  return today.compare(target) === 0;
}

/**
 * Returns a native `Date` offset backwards from `date` by `duration`.
 * Drop-in for `date-fns/sub`.
 */
export function sub(date: Date, duration: Duration): Date {
  return fromDate(date, getLocalTimeZone()).subtract(duration).toDate();
}

/**
 * Returns each day in the interval [start, end] (inclusive). Drop-in for
 * `date-fns/eachDayOfInterval`.
 */
export function eachDayOfInterval(range: DateRange): Date[] {
  const end = toCalendarDateFrom(range.end);
  const out: Date[] = [];
  for (let c = toCalendarDateFrom(range.start); c.compare(end) <= 0; c = c.add({ days: 1 })) {
    out.push(c.toDate(getLocalTimeZone()));
  }
  return out;
}

/**
 * Returns the start of each week (Sunday) in the interval. Drop-in for
 * `date-fns/eachWeekOfInterval` with the default `weekStartsOn: 0`.
 */
export function eachWeekOfInterval(range: DateRange): Date[] {
  const end = toCalendarDateFrom(range.end);
  const start = toCalendarDateFrom(range.start);
  const out: Date[] = [];
  for (
    let c = start.subtract({ days: getDayOfWeek(start, "en-US") });
    c.compare(end) <= 0;
    c = c.add({ weeks: 1 })
  ) {
    out.push(c.toDate(getLocalTimeZone()));
  }
  return out;
}

/**
 * Returns the first day of each month in the interval. Drop-in for
 * `date-fns/eachMonthOfInterval`.
 */
export function eachMonthOfInterval(range: DateRange): Date[] {
  const start = toCalendarDateFrom(range.start);
  const end = toCalendarDateFrom(range.end);
  const out: Date[] = [];
  for (
    let c = new CalendarDate(start.year, start.month, 1);
    c.compare(end) <= 0;
    c = c.add({ months: 1 })
  ) {
    out.push(c.toDate(getLocalTimeZone()));
  }
  return out;
}

/**
 * Current timestamp as an ISO string, `duration` in the past. Convenience for
 * mock data generators.
 */
export function ago(duration: Duration): string {
  return now(getLocalTimeZone()).subtract(duration).toString();
}
