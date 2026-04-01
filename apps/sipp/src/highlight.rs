use std::io::Cursor;
use syntect::highlighting::{Theme, ThemeSet};
use syntect::html::highlighted_html_for_string;
use syntect::parsing::SyntaxSet;

pub struct Highlighter {
    syntax_set: SyntaxSet,
    theme: Theme,
}

impl Highlighter {
    pub fn new() -> Self {
        let theme_data = include_bytes!("darkmatter.tmTheme");
        let theme = ThemeSet::load_from_reader(&mut Cursor::new(&theme_data[..]))
            .expect("failed to load darkmatter theme");
        Self {
            syntax_set: SyntaxSet::load_defaults_newlines(),
            theme,
        }
    }

    pub fn highlight(&self, name: &str, content: &str) -> String {
        let raw_ext = name.rsplit('.').next().unwrap_or("");
        let ext = match raw_ext {
            "ts" | "tsx" | "jsx" => "js",
            other => other,
        };
        let syntax = self
            .syntax_set
            .find_syntax_by_extension(ext)
            .unwrap_or_else(|| self.syntax_set.find_syntax_plain_text());
        highlighted_html_for_string(content, &self.syntax_set, syntax, &self.theme)
            .unwrap_or_else(|_| {
                let escaped = content
                    .replace('&', "&amp;")
                    .replace('<', "&lt;")
                    .replace('>', "&gt;");
                format!("<pre>{}</pre>", escaped)
            })
    }
}
