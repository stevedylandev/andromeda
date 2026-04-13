use rand::RngCore;
use subtle::ConstantTimeEq;

/// Constant-time password comparison to prevent timing attacks.
/// Pads/truncates both sides to a fixed 256-byte buffer so length
/// differences don't leak via timing.
pub fn verify_password(input: &str, expected: &str) -> bool {
    const LEN: usize = 256;
    let mut a = [0u8; LEN];
    let mut b = [0u8; LEN];
    let ib = input.as_bytes();
    let eb = expected.as_bytes();
    a[..ib.len().min(LEN)].copy_from_slice(&ib[..ib.len().min(LEN)]);
    b[..eb.len().min(LEN)].copy_from_slice(&eb[..eb.len().min(LEN)]);
    let lengths_match = subtle::Choice::from((ib.len() == eb.len()) as u8);
    (lengths_match & a.ct_eq(&b)).into()
}

/// Generate a 32-byte cryptographically random hex token.
pub fn generate_session_token() -> String {
    let mut bytes = [0u8; 32];
    rand::rngs::OsRng.fill_bytes(&mut bytes);
    bytes.iter().map(|b| format!("{:02x}", b)).collect()
}

/// Build a session cookie with HttpOnly, SameSite=Strict, 7-day Max-Age.
pub fn build_session_cookie(token: &str, secure: bool) -> String {
    let mut cookie = format!(
        "session={}; HttpOnly; SameSite=Strict; Path=/; Max-Age=604800",
        token
    );
    if secure {
        cookie.push_str("; Secure");
    }
    cookie
}

/// Build a cookie that clears the session.
pub fn clear_session_cookie() -> String {
    "session=; HttpOnly; SameSite=Strict; Path=/; Max-Age=0".to_string()
}

/// Extract the session token from the Cookie header.
pub fn extract_session_cookie(headers: &axum::http::HeaderMap) -> Option<String> {
    let cookie_header = headers.get("cookie")?.to_str().ok()?;
    for part in cookie_header.split(';') {
        let part = part.trim();
        if let Some(val) = part.strip_prefix("session=") {
            let val = val.trim().to_string();
            if !val.is_empty() {
                return Some(val);
            }
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::http::HeaderMap;

    // ── verify_password ────────────────────────────────────────────────

    #[test]
    fn verify_password_correct() {
        assert!(verify_password("hunter2", "hunter2"));
    }

    #[test]
    fn verify_password_wrong() {
        assert!(!verify_password("hunter2", "hunter3"));
    }

    #[test]
    fn verify_password_empty_both() {
        assert!(verify_password("", ""));
    }

    #[test]
    fn verify_password_empty_vs_nonempty() {
        assert!(!verify_password("", "something"));
        assert!(!verify_password("something", ""));
    }

    #[test]
    fn verify_password_length_mismatch() {
        assert!(!verify_password("short", "longer_password"));
    }

    #[test]
    fn verify_password_over_256_bytes_truncated_to_same() {
        // Both 300 chars, identical first 256 → truncation makes them equal
        let long_a = "a".repeat(300);
        let mut long_b = "a".repeat(256);
        long_b.push_str(&"b".repeat(44));
        // Same length, same first 256 bytes → passes
        assert!(verify_password(&long_a, &long_b));
    }

    #[test]
    fn verify_password_over_256_bytes_different_prefix() {
        let long_a = "a".repeat(300);
        let mut long_b = "a".repeat(300);
        // Differ within first 256 bytes
        unsafe { long_b.as_bytes_mut()[0] = b'z'; }
        assert!(!verify_password(&long_a, &long_b));
    }

    #[test]
    fn verify_password_exactly_256_bytes() {
        let pw = "x".repeat(256);
        assert!(verify_password(&pw, &pw));
    }

    // ── generate_session_token ─────────────────────────────────────────

    #[test]
    fn session_token_is_64_hex_chars() {
        let token = generate_session_token();
        assert_eq!(token.len(), 64);
        assert!(token.chars().all(|c| c.is_ascii_hexdigit()));
    }

    #[test]
    fn session_token_unique_across_calls() {
        let a = generate_session_token();
        let b = generate_session_token();
        assert_ne!(a, b);
    }

    // ── build_session_cookie ───────────────────────────────────────────

    #[test]
    fn build_session_cookie_not_secure() {
        let cookie = build_session_cookie("abc123", false);
        assert!(cookie.contains("session=abc123"));
        assert!(cookie.contains("HttpOnly"));
        assert!(cookie.contains("SameSite=Strict"));
        assert!(cookie.contains("Path=/"));
        assert!(cookie.contains("Max-Age=604800"));
        assert!(!cookie.contains("Secure"));
    }

    #[test]
    fn build_session_cookie_secure() {
        let cookie = build_session_cookie("abc123", true);
        assert!(cookie.contains("Secure"));
    }

    // ── clear_session_cookie ───────────────────────────────────────────

    #[test]
    fn clear_session_cookie_has_zero_max_age() {
        let cookie = clear_session_cookie();
        assert!(cookie.contains("session="));
        assert!(cookie.contains("Max-Age=0"));
        assert!(cookie.contains("HttpOnly"));
    }

    // ── extract_session_cookie ─────────────────────────────────────────

    #[test]
    fn extract_session_cookie_present() {
        let mut headers = HeaderMap::new();
        headers.insert("cookie", "session=tok123".parse().unwrap());
        assert_eq!(extract_session_cookie(&headers), Some("tok123".to_string()));
    }

    #[test]
    fn extract_session_cookie_absent() {
        let headers = HeaderMap::new();
        assert_eq!(extract_session_cookie(&headers), None);
    }

    #[test]
    fn extract_session_cookie_empty_value() {
        let mut headers = HeaderMap::new();
        headers.insert("cookie", "session=".parse().unwrap());
        assert_eq!(extract_session_cookie(&headers), None);
    }

    #[test]
    fn extract_session_cookie_among_multiple() {
        let mut headers = HeaderMap::new();
        headers.insert(
            "cookie",
            "theme=dark; session=abc; lang=en".parse().unwrap(),
        );
        assert_eq!(extract_session_cookie(&headers), Some("abc".to_string()));
    }

    #[test]
    fn extract_session_cookie_no_session_key() {
        let mut headers = HeaderMap::new();
        headers.insert("cookie", "theme=dark; lang=en".parse().unwrap());
        assert_eq!(extract_session_cookie(&headers), None);
    }
}
