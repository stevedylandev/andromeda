use ratatui::style::{Color, Style};
use ratatui::text::{Line, Span, Text};
use std::io::Cursor;
use syntect::easy::HighlightLines;
use syntect::highlighting::Theme;
use syntect::parsing::SyntaxSet;
use syntect::util::LinesWithEndings;

pub struct Highlighter {
    syntax_set: SyntaxSet,
    theme: Theme,
}

impl Highlighter {
    pub fn new() -> Self {
        let theme_data = include_bytes!("ansi.tmTheme");
        let theme =
            syntect::highlighting::ThemeSet::load_from_reader(&mut Cursor::new(&theme_data[..]))
                .expect("failed to load ansi theme");
        Self {
            syntax_set: SyntaxSet::load_defaults_newlines(),
            theme,
        }
    }

    pub fn highlight_markdown(&self, content: &str) -> Text<'static> {
        let syntax = self
            .syntax_set
            .find_syntax_by_extension("md")
            .or_else(|| self.syntax_set.find_syntax_by_name("Markdown"))
            .unwrap_or_else(|| self.syntax_set.find_syntax_plain_text());
        let mut h = HighlightLines::new(syntax, &self.theme);

        let lines: Vec<Line<'static>> = LinesWithEndings::from(content)
            .map(|line| {
                let ranges = h
                    .highlight_line(line, &self.syntax_set)
                    .unwrap_or_default();
                let spans: Vec<Span<'static>> = ranges
                    .into_iter()
                    .map(|(style, text)| {
                        let color = to_ratatui_color(style.foreground);
                        Span::styled(text.to_owned(), Style::default().fg(color))
                    })
                    .collect();
                Line::from(spans)
            })
            .collect();

        Text::from(lines)
    }
}

fn to_ratatui_color(color: syntect::highlighting::Color) -> Color {
    if color.a == 0 {
        Color::Indexed(color.r)
    } else {
        Color::Reset
    }
}
