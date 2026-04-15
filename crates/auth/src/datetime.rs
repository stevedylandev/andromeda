//! Zero-dep datetime helpers for formatting Unix timestamps as
//! `YYYY-MM-DD HH:MM:SS` strings suitable for SQLite TEXT columns
//! and string-sortable comparisons.

use std::time::{SystemTime, UNIX_EPOCH};

/// Current time as `YYYY-MM-DD HH:MM:SS`.
pub fn now_datetime_string() -> String {
    from_unix_secs(now_secs())
}

/// Time `secs_from_now` seconds in the future as `YYYY-MM-DD HH:MM:SS`.
pub fn expiry_datetime_string(secs_from_now: u64) -> String {
    from_unix_secs(now_secs() + secs_from_now)
}

/// Format an absolute Unix timestamp as `YYYY-MM-DD HH:MM:SS`.
pub fn from_unix_secs(secs: u64) -> String {
    let days = secs / 86400;
    let h = (secs / 3600) % 24;
    let m = (secs / 60) % 60;
    let s = secs % 60;
    format_unix_to_datetime(days, h, m, s)
}

/// Format an already-split unix time into `YYYY-MM-DD HH:MM:SS`.
pub fn format_unix_to_datetime(days: u64, h: u64, m: u64, s: u64) -> String {
    let (y, mo, d) = days_to_ymd(days as i64);
    format!("{:04}-{:02}-{:02} {:02}:{:02}:{:02}", y, mo, d, h, m, s)
}

/// Convert days-since-Unix-epoch into `(year, month, day)`.
/// https://howardhinnant.github.io/date_algorithms.html
pub fn days_to_ymd(mut days: i64) -> (i64, i64, i64) {
    days += 719468;
    let era = if days >= 0 { days } else { days - 146096 } / 146097;
    let doe = (days - era * 146097) as u32;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = if m <= 2 { y + 1 } else { y };
    (y, m as i64, d as i64)
}

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn format_unix_epoch() {
        assert_eq!(format_unix_to_datetime(0, 0, 0, 0), "1970-01-01 00:00:00");
    }

    #[test]
    fn format_unix_known_date() {
        assert_eq!(
            format_unix_to_datetime(19737, 12, 30, 45),
            "2024-01-15 12:30:45"
        );
    }

    #[test]
    fn format_unix_y2k() {
        assert_eq!(
            format_unix_to_datetime(10957, 0, 0, 0),
            "2000-01-01 00:00:00"
        );
    }

    #[test]
    fn format_unix_leap_day() {
        assert_eq!(
            format_unix_to_datetime(19782, 23, 59, 59),
            "2024-02-29 23:59:59"
        );
    }

    #[test]
    fn format_unix_end_of_year() {
        assert_eq!(
            format_unix_to_datetime(19722, 23, 59, 59),
            "2023-12-31 23:59:59"
        );
    }

    #[test]
    fn now_string_valid_format() {
        let s = now_datetime_string();
        assert_eq!(s.len(), 19);
        assert_eq!(&s[4..5], "-");
        assert_eq!(&s[7..8], "-");
        assert_eq!(&s[10..11], " ");
        assert_eq!(&s[13..14], ":");
        assert_eq!(&s[16..17], ":");
    }

    #[test]
    fn expiry_string_in_future() {
        let now = now_datetime_string();
        let exp = expiry_datetime_string(7 * 24 * 3600);
        assert!(exp > now);
    }

    #[test]
    fn from_unix_secs_known() {
        // 2024-01-15 12:30:45 UTC = 1705321845
        assert_eq!(from_unix_secs(1705321845), "2024-01-15 12:30:45");
    }
}
