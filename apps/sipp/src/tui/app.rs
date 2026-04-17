use crate::backend::Backend;
use crate::db::Snippet;
use arboard::Clipboard;
use ratatui::style::{Color, Style};
use ratatui::text::{Line, Span, Text};
use ratatui::widgets::ListState;
use std::io::Cursor;
use std::time::{Duration, Instant};
use syntect::easy::HighlightLines;
use syntect::highlighting::Theme;
use syntect::parsing::SyntaxSet;
use syntect::util::LinesWithEndings;

pub(super) enum Focus {
    List,
    Content,
    CreateName,
    CreateContent,
    EditName,
    EditContent,
    Search,
}

pub(super) struct App {
    pub(super) snippets: Vec<Snippet>,
    pub(super) list_state: ListState,
    pub(super) should_quit: bool,
    pub(super) status_message: Option<(String, Instant)>,
    pub(super) focus: Focus,
    pub(super) content_scroll: u16,
    pub(super) show_help: bool,
    pub(super) confirm_delete: bool,
    syntax_set: SyntaxSet,
    theme: Theme,
    pub(super) create_name: String,
    pub(super) create_content: String,
    pub(super) edit_short_id: Option<String>,
    pub(super) search_query: String,
    pub(super) filtered_indices: Option<Vec<usize>>,
    pub(super) is_remote: bool,
    pub(super) remote_url: Option<String>,
    pub(super) wrap_content: bool,
    pub(super) edit_scroll: u16,
}

impl App {
    pub(super) fn new(
        snippets: Vec<Snippet>,
        is_remote: bool,
        remote_url: Option<String>,
    ) -> Self {
        let mut list_state = ListState::default();
        if !snippets.is_empty() {
            list_state.select(Some(0));
        }
        let syntax_set = SyntaxSet::load_defaults_newlines();
        let theme_data = include_bytes!("../ansi.tmTheme");
        let theme =
            syntect::highlighting::ThemeSet::load_from_reader(&mut Cursor::new(&theme_data[..]))
                .expect("failed to load base16 theme");
        Self {
            snippets,
            list_state,
            should_quit: false,
            status_message: None,
            focus: Focus::List,
            content_scroll: 0,
            show_help: false,
            confirm_delete: false,
            syntax_set,
            theme,
            create_name: String::new(),
            create_content: String::new(),
            edit_short_id: None,
            search_query: String::new(),
            filtered_indices: None,
            is_remote,
            remote_url,
            wrap_content: true,
            edit_scroll: 0,
        }
    }

    pub(super) fn selected_snippet(&self) -> Option<&Snippet> {
        self.list_state.selected().and_then(|i| {
            if let Some(indices) = &self.filtered_indices {
                indices.get(i).and_then(|&real| self.snippets.get(real))
            } else {
                self.snippets.get(i)
            }
        })
    }

    pub(super) fn visible_count(&self) -> usize {
        match &self.filtered_indices {
            Some(indices) => indices.len(),
            None => self.snippets.len(),
        }
    }

    pub(super) fn move_up(&mut self) {
        let count = self.visible_count();
        if count == 0 {
            return;
        }
        let i = match self.list_state.selected() {
            Some(i) if i > 0 => i - 1,
            Some(_) => count - 1,
            None => 0,
        };
        self.list_state.select(Some(i));
        self.content_scroll = 0;
    }

    pub(super) fn move_down(&mut self) {
        let count = self.visible_count();
        if count == 0 {
            return;
        }
        let i = match self.list_state.selected() {
            Some(i) if i < count - 1 => i + 1,
            Some(_) => 0,
            None => 0,
        };
        self.list_state.select(Some(i));
        self.content_scroll = 0;
    }

    pub(super) fn scroll_up(&mut self) {
        self.content_scroll = self.content_scroll.saturating_sub(1);
    }

    pub(super) fn scroll_down(&mut self, max_lines: u16) {
        if self.content_scroll < max_lines {
            self.content_scroll += 1;
        }
    }

    pub(super) fn copy_selected(&mut self) {
        if let Some(snippet) = self.selected_snippet() {
            if let Ok(mut clipboard) = Clipboard::new() {
                let _ = clipboard.set_text(&snippet.content);
                self.status_message = Some(("Copied!".to_string(), Instant::now()));
            }
        }
    }

    pub(super) fn copy_link(&mut self) {
        match &self.remote_url {
            Some(url) => {
                if let Some(snippet) = self.selected_snippet() {
                    let link = format!("{}/s/{}", url.trim_end_matches('/'), snippet.short_id);
                    if let Ok(mut clipboard) = Clipboard::new() {
                        let _ = clipboard.set_text(&link);
                        self.status_message =
                            Some(("Link copied!".to_string(), Instant::now()));
                    }
                }
            }
            None => {
                self.status_message =
                    Some(("No remote URL configured".to_string(), Instant::now()));
            }
        }
    }

    pub(super) fn open_in_browser(&mut self) {
        match &self.remote_url {
            Some(url) => {
                if let Some(snippet) = self.selected_snippet() {
                    let link = format!("{}/s/{}", url.trim_end_matches('/'), snippet.short_id);
                    if let Err(e) = open::that(&link) {
                        self.status_message =
                            Some((format!("Failed to open browser: {}", e), Instant::now()));
                    } else {
                        self.status_message =
                            Some(("Opened in browser!".to_string(), Instant::now()));
                    }
                }
            }
            None => {
                self.status_message =
                    Some(("No remote URL configured".to_string(), Instant::now()));
            }
        }
    }

    pub(super) fn delete_selected(&mut self, backend: &Backend) {
        if let Some(selected_index) = self.list_state.selected() {
            let real_index = if let Some(indices) = &self.filtered_indices {
                match indices.get(selected_index) {
                    Some(&ri) => ri,
                    None => return,
                }
            } else {
                selected_index
            };
            if let Some(snippet) = self.snippets.get(real_index) {
                let short_id = snippet.short_id.clone();
                match backend.delete_snippet(&short_id) {
                    Ok(true) => {
                        self.snippets.remove(real_index);
                        if self.filtered_indices.is_some() {
                            self.update_search_filter();
                        }
                        let count = self.visible_count();
                        if count == 0 {
                            self.list_state.select(None);
                        } else if selected_index >= count {
                            self.list_state.select(Some(count - 1));
                        } else {
                            self.list_state.select(Some(selected_index));
                        }
                        self.status_message = Some(("Deleted!".to_string(), Instant::now()));
                    }
                    Ok(false) => {
                        self.status_message =
                            Some(("Snippet not found".to_string(), Instant::now()));
                    }
                    Err(e) => {
                        self.status_message = Some((e.to_string(), Instant::now()));
                    }
                }
            }
        }
    }

    pub(super) fn refresh(&mut self, backend: &Backend) {
        match backend.list_snippets() {
            Ok(snippets) => {
                self.snippets = snippets;
                self.filtered_indices = None;
                self.search_query.clear();
                if self.snippets.is_empty() {
                    self.list_state.select(None);
                } else {
                    let idx = self.list_state.selected().unwrap_or(0);
                    if idx >= self.snippets.len() {
                        self.list_state.select(Some(self.snippets.len() - 1));
                    }
                }
                self.status_message = Some(("Refreshed!".to_string(), Instant::now()));
            }
            Err(e) => {
                self.status_message = Some((e.to_string(), Instant::now()));
            }
        }
    }

    pub(super) fn cursor_position_wrapped(&self, width: u16) -> (u16, u16) {
        let w = width as usize;
        if w == 0 {
            return (0, 0);
        }
        let text = &self.create_content;
        let mut visual_row: usize = 0;
        let lines: Vec<&str> = if text.is_empty() {
            vec![""]
        } else if text.ends_with('\n') {
            text.split('\n').collect()
        } else {
            text.split('\n').collect()
        };
        let last_idx = lines.len() - 1;
        for (i, line) in lines.iter().enumerate() {
            let line_len = line.len();
            let wrapped_lines = if line_len == 0 {
                1
            } else {
                (line_len + w - 1) / w
            };
            if i < last_idx {
                visual_row += wrapped_lines;
            } else {
                let cursor_col = if text.ends_with('\n') { 0 } else { line_len };
                let extra_rows = cursor_col / w;
                let col = cursor_col % w;
                visual_row += extra_rows;
                return (col as u16, visual_row as u16);
            }
        }
        (0, visual_row as u16)
    }

    pub(super) fn auto_scroll_edit(&mut self, cursor_visual_row: u16, visible_height: u16) {
        if visible_height == 0 {
            return;
        }
        if cursor_visual_row < self.edit_scroll {
            self.edit_scroll = cursor_visual_row;
        } else if cursor_visual_row >= self.edit_scroll + visible_height {
            self.edit_scroll = cursor_visual_row - visible_height + 1;
        }
    }

    pub(super) fn start_create(&mut self) {
        self.create_name.clear();
        self.create_content.clear();
        self.edit_scroll = 0;
        self.focus = Focus::CreateName;
    }

    pub(super) fn save_create(&mut self, backend: &Backend) {
        if self.create_name.trim().is_empty() {
            self.status_message = Some(("Name cannot be empty".to_string(), Instant::now()));
            return;
        }
        match backend.create_snippet(&self.create_name, &self.create_content) {
            Ok(snippet) => {
                self.snippets.insert(0, snippet);
                self.list_state.select(Some(0));
                self.filtered_indices = None;
                self.search_query.clear();
                self.status_message = Some(("Created!".to_string(), Instant::now()));
                self.focus = Focus::List;
                self.create_name.clear();
                self.create_content.clear();
            }
            Err(e) => {
                self.status_message = Some((e.to_string(), Instant::now()));
            }
        }
    }

    pub(super) fn cancel_create(&mut self) {
        self.create_name.clear();
        self.create_content.clear();
        self.focus = Focus::List;
    }

    pub(super) fn start_edit(&mut self) {
        let data = self
            .selected_snippet()
            .map(|s| (s.name.clone(), s.content.clone(), s.short_id.clone()));
        if let Some((name, content, short_id)) = data {
            self.create_name = name;
            self.create_content = content;
            self.edit_short_id = Some(short_id);
            self.edit_scroll = 0;
            self.focus = Focus::EditName;
        }
    }

    pub(super) fn save_edit(&mut self, backend: &Backend) {
        if self.create_name.trim().is_empty() {
            self.status_message = Some(("Name cannot be empty".to_string(), Instant::now()));
            return;
        }
        let short_id = match &self.edit_short_id {
            Some(id) => id.clone(),
            None => return,
        };
        match backend.update_snippet(&short_id, &self.create_name, &self.create_content) {
            Ok(Some(updated)) => {
                if let Some(pos) = self.snippets.iter().position(|s| s.short_id == short_id) {
                    self.snippets[pos] = updated;
                }
                self.status_message = Some(("Updated!".to_string(), Instant::now()));
                self.focus = Focus::List;
                self.create_name.clear();
                self.create_content.clear();
                self.edit_short_id = None;
            }
            Ok(None) => {
                self.status_message = Some(("Snippet not found".to_string(), Instant::now()));
            }
            Err(e) => {
                self.status_message = Some((e.to_string(), Instant::now()));
            }
        }
    }

    pub(super) fn cancel_edit(&mut self) {
        self.create_name.clear();
        self.create_content.clear();
        self.edit_short_id = None;
        self.focus = Focus::List;
    }

    pub(super) fn start_search(&mut self) {
        self.search_query.clear();
        self.filtered_indices = Some((0..self.snippets.len()).collect());
        self.focus = Focus::Search;
        self.list_state
            .select(if self.snippets.is_empty() { None } else { Some(0) });
    }

    pub(super) fn update_search_filter(&mut self) {
        let query = self.search_query.to_lowercase();
        let indices: Vec<usize> = self
            .snippets
            .iter()
            .enumerate()
            .filter(|(_, s)| s.name.to_lowercase().contains(&query))
            .map(|(i, _)| i)
            .collect();
        self.filtered_indices = Some(indices);
        if self.visible_count() == 0 {
            self.list_state.select(None);
        } else {
            self.list_state.select(Some(0));
        }
    }

    pub(super) fn cancel_search(&mut self) {
        self.filtered_indices = None;
        self.search_query.clear();
        self.focus = Focus::List;
    }

    pub(super) fn confirm_search(&mut self) {
        let real_index = self.list_state.selected().and_then(|i| {
            self.filtered_indices
                .as_ref()
                .and_then(|indices| indices.get(i).copied())
        });
        self.filtered_indices = None;
        self.search_query.clear();
        self.focus = Focus::List;
        if let Some(ri) = real_index {
            self.list_state.select(Some(ri));
        }
    }

    pub(super) fn clear_expired_status(&mut self) {
        if let Some((_, time)) = &self.status_message {
            if time.elapsed() > Duration::from_secs(2) {
                self.status_message = None;
            }
        }
    }

    pub(super) fn highlight_content(&self, name: &str, content: &str) -> Text<'static> {
        let raw_ext = name.rsplit('.').next().unwrap_or("");
        let ext = match raw_ext {
            "ts" | "tsx" | "jsx" => "js",
            other => other,
        };
        let syntax = self
            .syntax_set
            .find_syntax_by_extension(ext)
            .unwrap_or_else(|| self.syntax_set.find_syntax_plain_text());
        let mut highlighter = HighlightLines::new(syntax, &self.theme);

        let lines: Vec<Line<'static>> = LinesWithEndings::from(content)
            .map(|line| {
                let ranges = highlighter
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
