use arboard::Clipboard;
use crossterm::event::{self, Event, KeyCode, KeyModifiers};
use ratatui::{
    DefaultTerminal,
    layout::{Alignment, Constraint, Layout},
    style::{Color, Modifier, Style},
    text::{Line, Span, Text},
    widgets::{Block, Borders, Clear, List, ListItem, ListState, Paragraph, Widget, Wrap},
};
use crate::backend::Backend;
use crate::config;
use crate::db::Snippet;
use std::io::Cursor;
use std::path::PathBuf;
use std::time::{Duration, Instant};
use syntect::easy::HighlightLines;
use syntect::highlighting::Theme;
use syntect::parsing::SyntaxSet;
use syntect::util::LinesWithEndings;

enum Focus {
    List,
    Content,
    CreateName,
    CreateContent,
    EditName,
    EditContent,
    Search,
}

struct App {
    snippets: Vec<Snippet>,
    list_state: ListState,
    should_quit: bool,
    status_message: Option<(String, Instant)>,
    focus: Focus,
    content_scroll: u16,
    show_help: bool,
    confirm_delete: bool,
    syntax_set: SyntaxSet,
    theme: Theme,
    create_name: String,
    create_content: String,
    edit_short_id: Option<String>,
    search_query: String,
    filtered_indices: Option<Vec<usize>>,
    is_remote: bool,
    remote_url: Option<String>,
    wrap_content: bool,
    edit_scroll: u16,
}

impl App {
    fn new(snippets: Vec<Snippet>, is_remote: bool, remote_url: Option<String>) -> Self {
        let mut list_state = ListState::default();
        if !snippets.is_empty() {
            list_state.select(Some(0));
        }
        let syntax_set = SyntaxSet::load_defaults_newlines();
        let theme_data = include_bytes!("ansi.tmTheme");
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

    fn selected_snippet(&self) -> Option<&Snippet> {
        self.list_state.selected().and_then(|i| {
            if let Some(indices) = &self.filtered_indices {
                indices.get(i).and_then(|&real| self.snippets.get(real))
            } else {
                self.snippets.get(i)
            }
        })
    }

    fn visible_count(&self) -> usize {
        match &self.filtered_indices {
            Some(indices) => indices.len(),
            None => self.snippets.len(),
        }
    }

    fn move_up(&mut self) {
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

    fn move_down(&mut self) {
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

    fn scroll_up(&mut self) {
        self.content_scroll = self.content_scroll.saturating_sub(1);
    }

    fn scroll_down(&mut self, max_lines: u16) {
        if self.content_scroll < max_lines {
            self.content_scroll += 1;
        }
    }

    fn copy_selected(&mut self) {
        if let Some(snippet) = self.selected_snippet() {
            if let Ok(mut clipboard) = Clipboard::new() {
                let _ = clipboard.set_text(&snippet.content);
                self.status_message = Some(("Copied!".to_string(), Instant::now()));
            }
        }
    }

    fn copy_link(&mut self) {
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

    fn open_in_browser(&mut self) {
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

    fn delete_selected(&mut self, backend: &Backend) {
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

    fn refresh(&mut self, backend: &Backend) {
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

    fn cursor_position_wrapped(&self, width: u16) -> (u16, u16) {
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
                // cursor is at end of this last line
                let cursor_col = if text.ends_with('\n') { 0 } else { line_len };
                let extra_rows = cursor_col / w;
                let col = cursor_col % w;
                visual_row += extra_rows;
                return (col as u16, visual_row as u16);
            }
        }
        (0, visual_row as u16)
    }

    fn auto_scroll_edit(&mut self, cursor_visual_row: u16, visible_height: u16) {
        if visible_height == 0 {
            return;
        }
        if cursor_visual_row < self.edit_scroll {
            self.edit_scroll = cursor_visual_row;
        } else if cursor_visual_row >= self.edit_scroll + visible_height {
            self.edit_scroll = cursor_visual_row - visible_height + 1;
        }
    }

    fn start_create(&mut self) {
        self.create_name.clear();
        self.create_content.clear();
        self.edit_scroll = 0;
        self.focus = Focus::CreateName;
    }

    fn save_create(&mut self, backend: &Backend) {
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

    fn cancel_create(&mut self) {
        self.create_name.clear();
        self.create_content.clear();
        self.focus = Focus::List;
    }

    fn start_edit(&mut self) {
        let data = self.selected_snippet().map(|s| {
            (s.name.clone(), s.content.clone(), s.short_id.clone())
        });
        if let Some((name, content, short_id)) = data {
            self.create_name = name;
            self.create_content = content;
            self.edit_short_id = Some(short_id);
            self.edit_scroll = 0;
            self.focus = Focus::EditName;
        }
    }

    fn save_edit(&mut self, backend: &Backend) {
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

    fn cancel_edit(&mut self) {
        self.create_name.clear();
        self.create_content.clear();
        self.edit_short_id = None;
        self.focus = Focus::List;
    }

    fn start_search(&mut self) {
        self.search_query.clear();
        self.filtered_indices = Some((0..self.snippets.len()).collect());
        self.focus = Focus::Search;
        self.list_state.select(if self.snippets.is_empty() { None } else { Some(0) });
    }

    fn update_search_filter(&mut self) {
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

    fn cancel_search(&mut self) {
        self.filtered_indices = None;
        self.search_query.clear();
        self.focus = Focus::List;
    }

    fn confirm_search(&mut self) {
        let real_index = self.list_state.selected().and_then(|i| {
            self.filtered_indices.as_ref().and_then(|indices| indices.get(i).copied())
        });
        self.filtered_indices = None;
        self.search_query.clear();
        self.focus = Focus::List;
        if let Some(ri) = real_index {
            self.list_state.select(Some(ri));
        }
    }

    fn clear_expired_status(&mut self) {
        if let Some((_, time)) = &self.status_message {
            if time.elapsed() > Duration::from_secs(2) {
                self.status_message = None;
            }
        }
    }

    fn highlight_content(&self, name: &str, content: &str) -> Text<'static> {
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

fn resolve_backend(remote: Option<String>, api_key: Option<String>) -> Result<(Backend, bool, Option<String>), Box<dyn std::error::Error>> {
    if let Some(url) = remote {
        return Ok((
            Backend::remote(url.clone(), api_key),
            true,
            Some(url),
        ));
    }

    if !std::path::Path::new(&crate::db::db_path()).exists() {
        let cfg = config::load_config();
        let url = cfg.remote_url.unwrap_or_else(|| "http://localhost:3000".to_string());
        let api_key = api_key.or(cfg.api_key);
        return Ok((Backend::remote(url.clone(), api_key), true, Some(url)));
    }

    Ok((Backend::local()?, false, Some("http://localhost:3000".to_string())))
}

pub fn run_auth() -> Result<(), Box<dyn std::error::Error>> {
    use std::io::{self, Write};

    print!("Remote URL: ");
    io::stdout().flush()?;
    let mut remote_url = String::new();
    io::stdin().read_line(&mut remote_url)?;
    let remote_url = remote_url.trim().to_string();

    print!("API Key: ");
    io::stdout().flush()?;
    let api_key = rpassword::read_password()?;
    let api_key = api_key.trim().to_string();

    let cfg = config::Config {
        remote_url: if remote_url.is_empty() {
            None
        } else {
            Some(remote_url)
        },
        api_key: if api_key.is_empty() {
            None
        } else {
            Some(api_key)
        },
    };

    config::save_config(&cfg)?;
    println!("Config saved to {}", config::config_path().display());
    Ok(())
}

pub fn run_interactive(remote: Option<String>, api_key: Option<String>) -> Result<(), Box<dyn std::error::Error>> {
    let (backend, is_remote, remote_url) = resolve_backend(remote, api_key)?;

    let snippets = match backend.list_snippets() {
        Ok(s) => s,
        Err(e) => {
            eprintln!("Failed to load snippets: {}", e);
            Vec::new()
        }
    };

    ratatui::run(|terminal| run_app(terminal, App::new(snippets, is_remote, remote_url), &backend))
}

pub fn run_file_upload(remote: Option<String>, api_key: Option<String>, file: PathBuf) -> Result<(), Box<dyn std::error::Error>> {
    let (backend, _, remote_url) = resolve_backend(remote, api_key)?;

    let name = file
        .file_name()
        .ok_or("Invalid file path")?
        .to_string_lossy()
        .to_string();
    let content = std::fs::read_to_string(&file)
        .map_err(|e| format!("Failed to read file: {}", e))?;
    let snippet = backend
        .create_snippet(&name, &content)
        .map_err(|e| format!("{}", e))?;
    let link = match &remote_url {
        Some(url) => format!("{}/s/{}", url.trim_end_matches('/'), snippet.short_id),
        None => snippet.short_id.clone(),
    };
    println!("{}", link);
    if let Ok(mut clipboard) = Clipboard::new() {
        let _ = clipboard.set_text(&link);
        println!("\u{2714} Copied to clipboard!");
    }
    Ok(())
}

fn run_app(
    terminal: &mut DefaultTerminal,
    mut app: App,
    backend: &Backend,
) -> Result<(), Box<dyn std::error::Error>> {
    while !app.should_quit {
        app.clear_expired_status();

        let content_line_count = app
            .selected_snippet()
            .map(|s| s.content.lines().count() as u16)
            .unwrap_or(0);

        terminal.draw(|frame| {
            let outer = Layout::vertical([Constraint::Min(1), Constraint::Length(1)])
                .split(frame.area());

            let chunks = Layout::horizontal([
                Constraint::Percentage(30),
                Constraint::Percentage(70),
            ])
            .split(outer[0]);

            let items: Vec<ListItem> = if let Some(indices) = &app.filtered_indices {
                indices
                    .iter()
                    .filter_map(|&i| app.snippets.get(i))
                    .map(|s| ListItem::new(s.name.as_str()))
                    .collect()
            } else {
                app.snippets
                    .iter()
                    .map(|s| ListItem::new(s.name.as_str()))
                    .collect()
            };

            let list_border_style = match app.focus {
                Focus::List | Focus::Search => Style::default().fg(Color::Yellow),
                _ => Style::default().fg(Color::DarkGray),
            };
            let content_border_style = match app.focus {
                Focus::Content => Style::default().fg(Color::Yellow),
                _ => Style::default().fg(Color::DarkGray),
            };

            let list = List::new(items)
                .block(
                    Block::default()
                        .title(" Snippets ")
                        .borders(Borders::ALL)
                        .border_style(list_border_style),
                )
                .highlight_style(
                    Style::default()
                        .fg(Color::Yellow)
                        .add_modifier(Modifier::BOLD),
                )
                .highlight_symbol("▶ ");

            if matches!(app.focus, Focus::Search) {
                let search_split = Layout::vertical([
                    Constraint::Min(1),
                    Constraint::Length(3),
                ])
                .split(chunks[0]);

                let search_items: Vec<ListItem> = if let Some(indices) = &app.filtered_indices {
                    indices
                        .iter()
                        .filter_map(|&i| app.snippets.get(i))
                        .map(|s| ListItem::new(s.name.as_str()))
                        .collect()
                } else {
                    app.snippets.iter().map(|s| ListItem::new(s.name.as_str())).collect()
                };
                let search_list = List::new(search_items)
                .block(
                    Block::default()
                        .title(" Snippets ")
                        .borders(Borders::ALL)
                        .border_style(list_border_style),
                )
                .highlight_style(
                    Style::default()
                        .fg(Color::Yellow)
                        .add_modifier(Modifier::BOLD),
                )
                .highlight_symbol("▶ ");
                frame.render_stateful_widget(search_list, search_split[0], &mut app.list_state);

                let search_input = Paragraph::new(app.search_query.as_str()).block(
                    Block::default()
                        .title(" Search ")
                        .borders(Borders::ALL)
                        .border_style(Style::default().fg(Color::Yellow)),
                );
                frame.render_widget(search_input, search_split[1]);

                let x = search_split[1].x + 1 + app.search_query.len() as u16;
                let y = search_split[1].y + 1;
                frame.set_cursor_position((x, y));
            } else {
                frame.render_stateful_widget(list, chunks[0], &mut app.list_state);
            }

            match app.focus {
                Focus::CreateName | Focus::CreateContent | Focus::EditName | Focus::EditContent => {
                    let form_title = match app.focus {
                        Focus::EditName | Focus::EditContent => " Edit Snippet ",
                        _ => " New Snippet ",
                    };
                    let create_block = Block::default()
                        .title(form_title)
                        .borders(Borders::ALL)
                        .border_style(Style::default().fg(Color::Yellow));

                    let inner = create_block.inner(chunks[1]);
                    frame.render_widget(create_block, chunks[1]);

                    let form_layout = Layout::vertical([
                        Constraint::Length(3),
                        Constraint::Min(1),
                    ])
                    .split(inner);

                    let name_style = match app.focus {
                        Focus::CreateName | Focus::EditName => Style::default().fg(Color::Yellow),
                        _ => Style::default().fg(Color::DarkGray),
                    };
                    let name_input = Paragraph::new(app.create_name.as_str()).block(
                        Block::default()
                            .title(" Name ")
                            .borders(Borders::ALL)
                            .border_style(name_style),
                    );
                    frame.render_widget(name_input, form_layout[0]);

                    let content_style = match app.focus {
                        Focus::CreateContent | Focus::EditContent => Style::default().fg(Color::Yellow),
                        _ => Style::default().fg(Color::DarkGray),
                    };
                    let mut content_input = Paragraph::new(app.create_content.as_str()).block(
                        Block::default()
                            .title(" Content ")
                            .borders(Borders::ALL)
                            .border_style(content_style),
                    );
                    if app.wrap_content {
                        content_input = content_input.wrap(Wrap { trim: false });
                    }
                    content_input = content_input.scroll((app.edit_scroll, 0));
                    frame.render_widget(content_input, form_layout[1]);

                    let content_inner = Block::default()
                        .borders(Borders::ALL)
                        .inner(form_layout[1]);
                    let inner_width = content_inner.width;
                    let inner_height = content_inner.height;

                    match app.focus {
                        Focus::CreateName | Focus::EditName => {
                            let x = form_layout[0].x + 1 + app.create_name.len() as u16;
                            let y = form_layout[0].y + 1;
                            frame.set_cursor_position((x, y));
                        }
                        Focus::CreateContent | Focus::EditContent => {
                            let (cx, cy) = if app.wrap_content {
                                app.cursor_position_wrapped(inner_width)
                            } else {
                                let last_line = app.create_content.lines().last().unwrap_or("");
                                let line_count = app.create_content.lines().count()
                                    + if app.create_content.ends_with('\n') { 1 } else { 0 };
                                let y_offset = if line_count == 0 { 0 } else { line_count - 1 };
                                let col = if app.create_content.ends_with('\n') {
                                    0
                                } else {
                                    last_line.len() as u16
                                };
                                (col, y_offset as u16)
                            };
                            app.auto_scroll_edit(cy, inner_height);
                            let screen_y = cy.saturating_sub(app.edit_scroll);
                            let x = content_inner.x + cx;
                            let y = content_inner.y + screen_y;
                            frame.set_cursor_position((x, y));
                        }
                        _ => {}
                    }

                }
                _ => {
                    let highlighted = match app.selected_snippet() {
                        Some(s) => app.highlight_content(&s.name, &s.content),
                        None => Text::raw(""),
                    };

                    let paragraph = Paragraph::new(highlighted)
                        .block(
                            Block::default()
                                .title(" Content ")
                                .borders(Borders::ALL)
                                .border_style(content_border_style),
                        )
                        .scroll((app.content_scroll, 0));

                    frame.render_widget(paragraph, chunks[1]);
                }
            }

            let hints = match app.focus {
                Focus::List => Line::from(vec![
                    Span::styled("j/k", Style::default().fg(Color::Yellow)),
                    Span::raw(": Navigate  "),
                    Span::styled("Enter", Style::default().fg(Color::Yellow)),
                    Span::raw(": View  "),
                    Span::styled("y", Style::default().fg(Color::Yellow)),
                    Span::raw(": Copy  "),
                    Span::styled("e", Style::default().fg(Color::Yellow)),
                    Span::raw(": Edit  "),
                    Span::styled("d", Style::default().fg(Color::Yellow)),
                    Span::raw(": Delete  "),
                    Span::styled("c", Style::default().fg(Color::Yellow)),
                    Span::raw(": Create  "),
                    Span::styled("/", Style::default().fg(Color::Yellow)),
                    Span::raw(": Search  "),
                    Span::styled("?", Style::default().fg(Color::Yellow)),
                    Span::raw(": Help  "),
                    Span::styled("q", Style::default().fg(Color::Yellow)),
                    Span::raw(": Quit"),
                ]),
                Focus::Content => Line::from(vec![
                    Span::styled("j/k", Style::default().fg(Color::Yellow)),
                    Span::raw(": Scroll  "),
                    Span::styled("y", Style::default().fg(Color::Yellow)),
                    Span::raw(": Copy  "),
                    Span::styled("e", Style::default().fg(Color::Yellow)),
                    Span::raw(": Edit  "),
                    Span::styled("Esc", Style::default().fg(Color::Yellow)),
                    Span::raw(": Back  "),
                    Span::styled("?", Style::default().fg(Color::Yellow)),
                    Span::raw(": Help"),
                ]),
                Focus::CreateName | Focus::CreateContent
                | Focus::EditName | Focus::EditContent => Line::from(vec![
                    Span::styled("Tab", Style::default().fg(Color::Yellow)),
                    Span::raw(": Switch field  "),
                    Span::styled("Ctrl+S", Style::default().fg(Color::Yellow)),
                    Span::raw(": Save  "),
                    Span::styled("Ctrl+W", Style::default().fg(Color::Yellow)),
                    Span::raw(": Wrap  "),
                    Span::styled("Esc", Style::default().fg(Color::Yellow)),
                    Span::raw(": Cancel"),
                ]),
                Focus::Search => Line::from(vec![
                    Span::styled("Type", Style::default().fg(Color::Yellow)),
                    Span::raw(": Filter  "),
                    Span::styled("Enter", Style::default().fg(Color::Yellow)),
                    Span::raw(": Select  "),
                    Span::styled("Esc", Style::default().fg(Color::Yellow)),
                    Span::raw(": Cancel"),
                ]),
            };
            frame.render_widget(Paragraph::new(hints), outer[1]);

            if let Some((msg, _)) = &app.status_message {
                let area = frame.area();
                let msg_width = (msg.len() as u16 + 4).max(20).min(area.width.saturating_sub(4));
                let popup_area = ratatui::layout::Rect {
                    x: (area.width.saturating_sub(msg_width)) / 2,
                    y: (area.height.saturating_sub(3)) / 2,
                    width: msg_width,
                    height: 3,
                };
                Clear.render(popup_area, frame.buffer_mut());
                let status_popup = Paragraph::new(Line::from(msg.as_str()))
                    .style(Style::default().fg(Color::Green).add_modifier(Modifier::BOLD))
                    .alignment(Alignment::Center)
                    .block(
                        Block::default()
                            .borders(Borders::ALL)
                            .border_style(Style::default().fg(Color::Green)),
                    );
                frame.render_widget(status_popup, popup_area);
            }

            if app.confirm_delete {
                let delete_msg = match app.selected_snippet() {
                    Some(s) => format!("Delete {}? (y/n)", s.name),
                    None => "Delete snippet? (y/n)".to_string(),
                };
                let area = frame.area();
                let msg_width = (delete_msg.len() as u16 + 4).max(24).min(area.width.saturating_sub(4));
                let popup_area = ratatui::layout::Rect {
                    x: (area.width.saturating_sub(msg_width)) / 2,
                    y: (area.height.saturating_sub(3)) / 2,
                    width: msg_width,
                    height: 3,
                };
                Clear.render(popup_area, frame.buffer_mut());
                let confirm_popup = Paragraph::new(Line::from(delete_msg))
                    .style(Style::default().fg(Color::Red).add_modifier(Modifier::BOLD))
                    .alignment(Alignment::Center)
                    .block(
                        Block::default()
                            .borders(Borders::ALL)
                            .border_style(Style::default().fg(Color::Red)),
                    );
                frame.render_widget(confirm_popup, popup_area);
            }

            if app.show_help {
                let area = frame.area();
                let popup_width = 34u16.min(area.width.saturating_sub(4));
                let popup_height = 21u16.min(area.height.saturating_sub(4));
                let popup_area = ratatui::layout::Rect {
                    x: (area.width.saturating_sub(popup_width)) / 2,
                    y: (area.height.saturating_sub(popup_height)) / 2,
                    width: popup_width,
                    height: popup_height,
                };

                let mut help_lines = vec![
                    Line::from(""),
                    Line::from(vec![
                        Span::styled(
                            "  j/↓  ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Move down / Scroll down"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  k/↑  ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Move up / Scroll up"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  Enter",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("  Focus content pane"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  Esc  ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Back / Quit"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  y    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Copy snippet"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  Y    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Copy link"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  o    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Open in browser"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  d    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Delete snippet"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  c    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Create snippet"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  e    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Edit snippet"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  /    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Search snippets"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  ^W   ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Toggle word wrap (edit)"),
                    ]),
                ];

                if app.is_remote {
                    help_lines.push(Line::from(vec![
                        Span::styled(
                            "  r    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Refresh snippets"),
                    ]));
                }

                help_lines.extend([
                    Line::from(vec![
                        Span::styled(
                            "  q    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Quit"),
                    ]),
                    Line::from(vec![
                        Span::styled(
                            "  ?    ",
                            Style::default()
                                .fg(Color::Yellow)
                                .add_modifier(Modifier::BOLD),
                        ),
                        Span::raw("Toggle this help"),
                    ]),
                    Line::from(""),
                    Line::from(Span::styled(
                        "  Press any key to close",
                        Style::default().fg(Color::DarkGray),
                    )),
                ]);

                let help_text = Text::from(help_lines);

                Clear.render(popup_area, frame.buffer_mut());
                let help = Paragraph::new(help_text).block(
                    Block::default()
                        .title(" Keybindings ")
                        .borders(Borders::ALL)
                        .border_style(Style::default().fg(Color::Yellow)),
                );
                frame.render_widget(help, popup_area);
            }
        })?;

        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if app.show_help {
                    app.show_help = false;
                } else if app.status_message.is_some() {
                    app.status_message = None;
                } else if app.confirm_delete {
                    if key.code == KeyCode::Char('y') {
                        app.delete_selected(backend);
                    }
                    app.confirm_delete = false;
                } else {
                    match app.focus {
                        Focus::List => match key.code {
                            KeyCode::Char('q') | KeyCode::Esc => app.should_quit = true,
                            KeyCode::Char('j') | KeyCode::Down => app.move_down(),
                            KeyCode::Char('k') | KeyCode::Up => app.move_up(),
                            KeyCode::Char('y') => app.copy_selected(),
                            KeyCode::Char('Y') => app.copy_link(),
                            KeyCode::Char('d') => app.confirm_delete = true,
                            KeyCode::Char('c') => app.start_create(),
                            KeyCode::Char('e') => app.start_edit(),
                            KeyCode::Char('/') => app.start_search(),
                            KeyCode::Char('o') => app.open_in_browser(),
                            KeyCode::Char('r') if app.is_remote => app.refresh(backend),
                            KeyCode::Char('?') => app.show_help = true,
                            KeyCode::Enter | KeyCode::Char('l') => {
                                if app.selected_snippet().is_some() {
                                    app.focus = Focus::Content;
                                }
                            }
                            _ => {}
                        },
                        Focus::Content => match key.code {
                          KeyCode::Char(' ') | KeyCode::Esc | KeyCode::Char('q') | KeyCode::Char('h') => {
                                app.focus = Focus::List;
                            }
                            KeyCode::Char('j') | KeyCode::Down => {
                                app.scroll_down(content_line_count);
                            }
                            KeyCode::Char('k') | KeyCode::Up => app.scroll_up(),
                            KeyCode::Char('y') => app.copy_selected(),
                            KeyCode::Char('Y') => app.copy_link(),
                            KeyCode::Char('e') => app.start_edit(),
                            KeyCode::Char('o') => app.open_in_browser(),
                            KeyCode::Char('?') => app.show_help = true,
                            _ => {}
                        },
                        Focus::CreateName => {
                            if key.modifiers.contains(KeyModifiers::CONTROL)
                                && key.code == KeyCode::Char('s')
                            {
                                app.save_create(backend);
                            } else {
                                match key.code {
                                    KeyCode::Esc => app.cancel_create(),
                                    KeyCode::Enter | KeyCode::Tab => {
                                        app.focus = Focus::CreateContent
                                    }
                                    KeyCode::Backspace => {
                                        app.create_name.pop();
                                    }
                                    KeyCode::Char(c) => app.create_name.push(c),
                                    _ => {}
                                }
                            }
                        }
                        Focus::CreateContent => {
                            if key.modifiers.contains(KeyModifiers::CONTROL) {
                                match key.code {
                                    KeyCode::Char('s') => app.save_create(backend),
                                    KeyCode::Char('w') => {
                                        app.wrap_content = !app.wrap_content;
                                        app.edit_scroll = 0;
                                    }
                                    _ => {}
                                }
                            } else {
                                match key.code {
                                    KeyCode::Esc => app.cancel_create(),
                                    KeyCode::Tab => app.focus = Focus::CreateName,
                                    KeyCode::Enter => app.create_content.push('\n'),
                                    KeyCode::Backspace => {
                                        app.create_content.pop();
                                    }
                                    KeyCode::Char(c) => app.create_content.push(c),
                                    _ => {}
                                }
                            }
                        }
                        Focus::EditName => {
                            if key.modifiers.contains(KeyModifiers::CONTROL)
                                && key.code == KeyCode::Char('s')
                            {
                                app.save_edit(backend);
                            } else {
                                match key.code {
                                    KeyCode::Esc => app.cancel_edit(),
                                    KeyCode::Enter | KeyCode::Tab => {
                                        app.focus = Focus::EditContent
                                    }
                                    KeyCode::Backspace => {
                                        app.create_name.pop();
                                    }
                                    KeyCode::Char(c) => app.create_name.push(c),
                                    _ => {}
                                }
                            }
                        }
                        Focus::EditContent => {
                            if key.modifiers.contains(KeyModifiers::CONTROL) {
                                match key.code {
                                    KeyCode::Char('s') => app.save_edit(backend),
                                    KeyCode::Char('w') => {
                                        app.wrap_content = !app.wrap_content;
                                        app.edit_scroll = 0;
                                    }
                                    _ => {}
                                }
                            } else {
                                match key.code {
                                    KeyCode::Esc => app.cancel_edit(),
                                    KeyCode::Tab => app.focus = Focus::EditName,
                                    KeyCode::Enter => app.create_content.push('\n'),
                                    KeyCode::Backspace => {
                                        app.create_content.pop();
                                    }
                                    KeyCode::Char(c) => app.create_content.push(c),
                                    _ => {}
                                }
                            }
                        }
                        Focus::Search => match key.code {
                            KeyCode::Esc => app.cancel_search(),
                            KeyCode::Enter => app.confirm_search(),
                            KeyCode::Backspace => {
                                app.search_query.pop();
                                app.update_search_filter();
                            }
                            KeyCode::Char(c) => {
                                app.search_query.push(c);
                                app.update_search_filter();
                            }
                            _ => {}
                        },
                    }
                }
            }
        }
    }

    Ok(())
}
