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
