use crate::backend::{Backend, ListItem};
use ratatui::widgets::ListState;
use std::time::{Duration, Instant};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum Mode {
    /// Browsing aggregated feed items from the server.
    Aggregate,
    /// Ad-hoc preview of a CLI-supplied URL. No subscribe-on-add behavior unless `s` pressed.
    Preview {
        // Source URL — used by `s` keybind to subscribe.
        // Stored on App rather than enum so events.rs can read it.
    },
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum Focus {
    List,
    AddFeedUrl,
    AddFeedCategory,
    DiscoverInput,
    DiscoverPicker,
}

pub(super) struct App {
    pub(super) items: Vec<ListItem>,
    pub(super) list_state: ListState,
    pub(super) should_quit: bool,
    pub(super) status_message: Option<(String, Instant)>,
    pub(super) focus: Focus,
    pub(super) mode: Mode,
    pub(super) preview_source_url: Option<String>,
    pub(super) show_help: bool,

    // Add-feed form
    pub(super) add_url: String,
    pub(super) add_category: String,

    // Discover
    pub(super) discover_input: String,
    pub(super) discover_results: Vec<String>,
    pub(super) discover_state: ListState,

    #[allow(dead_code)]
    pub(super) remote_url: String,
    pub(super) has_key: bool,
}

impl App {
    pub(super) fn new(items: Vec<ListItem>, remote_url: String, has_key: bool) -> Self {
        let mut list_state = ListState::default();
        if !items.is_empty() {
            list_state.select(Some(0));
        }
        Self {
            items,
            list_state,
            should_quit: false,
            status_message: None,
            focus: Focus::List,
            mode: Mode::Aggregate,
            preview_source_url: None,
            show_help: false,
            add_url: String::new(),
            add_category: String::new(),
            discover_input: String::new(),
            discover_results: Vec::new(),
            discover_state: ListState::default(),
            remote_url,
            has_key,
        }
    }

    pub(super) fn into_preview(mut self, source_url: String) -> Self {
        self.mode = Mode::Preview {};
        self.preview_source_url = Some(source_url);
        self
    }

    pub(super) fn set_status(&mut self, msg: impl Into<String>) {
        self.status_message = Some((msg.into(), Instant::now()));
    }

    pub(super) fn clear_expired_status(&mut self) {
        if let Some((_, t)) = &self.status_message {
            if t.elapsed() > Duration::from_secs(2) {
                self.status_message = None;
            }
        }
    }

    pub(super) fn selected_item(&self) -> Option<&ListItem> {
        self.list_state.selected().and_then(|i| self.items.get(i))
    }

    pub(super) fn move_up(&mut self) {
        if self.items.is_empty() {
            return;
        }
        let i = match self.list_state.selected() {
            Some(i) if i > 0 => i - 1,
            Some(_) => self.items.len() - 1,
            None => 0,
        };
        self.list_state.select(Some(i));
    }

    pub(super) fn move_down(&mut self) {
        if self.items.is_empty() {
            return;
        }
        let i = match self.list_state.selected() {
            Some(i) if i + 1 < self.items.len() => i + 1,
            Some(_) => 0,
            None => 0,
        };
        self.list_state.select(Some(i));
    }

    pub(super) fn open_selected(&mut self) {
        if let Some(item) = self.selected_item() {
            let link = item.link.clone();
            if let Err(e) = open::that(&link) {
                self.set_status(format!("Failed to open: {e}"));
            } else {
                self.set_status("Opened in browser");
            }
        }
    }

    pub(super) fn refresh(&mut self, backend: &Backend) {
        match self.mode {
            Mode::Aggregate => match backend.list_items(100) {
                Ok(items) => {
                    self.items = items;
                    if self.items.is_empty() {
                        self.list_state.select(None);
                    } else {
                        self.list_state.select(Some(0));
                    }
                    self.set_status("Refreshed");
                }
                Err(e) => self.set_status(e.to_string()),
            },
            Mode::Preview {} => {
                if let Some(url) = self.preview_source_url.clone() {
                    match backend.preview(&url) {
                        Ok(items) => {
                            self.items = items;
                            if self.items.is_empty() {
                                self.list_state.select(None);
                            } else {
                                self.list_state.select(Some(0));
                            }
                            self.set_status("Refreshed");
                        }
                        Err(e) => self.set_status(e.to_string()),
                    }
                }
            }
        }
    }

    pub(super) fn start_add_feed(&mut self, prefill: Option<String>) {
        if !self.has_key {
            self.set_status("API key required (run: feeds auth)");
            return;
        }
        self.add_url = prefill.unwrap_or_default();
        self.add_category.clear();
        self.focus = Focus::AddFeedUrl;
    }

    pub(super) fn submit_add_feed(&mut self, backend: &Backend) {
        let url = self.add_url.trim().to_string();
        if url.is_empty() {
            self.set_status("URL required");
            return;
        }
        let cat = self.add_category.trim();
        let cat_opt = if cat.is_empty() { None } else { Some(cat) };
        match backend.add_subscription(&url, cat_opt) {
            Ok(()) => {
                self.set_status("Subscribed");
                self.add_url.clear();
                self.add_category.clear();
                self.focus = Focus::List;
                if self.mode == Mode::Aggregate {
                    self.refresh(backend);
                }
            }
            Err(e) => self.set_status(e.to_string()),
        }
    }

    pub(super) fn cancel_add_feed(&mut self) {
        self.add_url.clear();
        self.add_category.clear();
        self.focus = Focus::List;
    }

    pub(super) fn start_discover(&mut self) {
        if !self.has_key {
            self.set_status("API key required (run: feeds auth)");
            return;
        }
        self.discover_input.clear();
        self.discover_results.clear();
        self.discover_state.select(None);
        self.focus = Focus::DiscoverInput;
    }

    pub(super) fn submit_discover(&mut self, backend: &Backend) {
        let url = self.discover_input.trim().to_string();
        if url.is_empty() {
            self.set_status("URL required");
            return;
        }
        match backend.discover(&url) {
            Ok(feeds) if feeds.is_empty() => {
                self.set_status("No feeds found");
            }
            Ok(feeds) => {
                self.discover_results = feeds;
                self.discover_state.select(Some(0));
                self.focus = Focus::DiscoverPicker;
            }
            Err(e) => self.set_status(e.to_string()),
        }
    }

    pub(super) fn cancel_discover(&mut self) {
        self.discover_input.clear();
        self.discover_results.clear();
        self.discover_state.select(None);
        self.focus = Focus::List;
    }

    pub(super) fn discover_picker_up(&mut self) {
        let n = self.discover_results.len();
        if n == 0 {
            return;
        }
        let i = match self.discover_state.selected() {
            Some(i) if i > 0 => i - 1,
            Some(_) => n - 1,
            None => 0,
        };
        self.discover_state.select(Some(i));
    }

    pub(super) fn discover_picker_down(&mut self) {
        let n = self.discover_results.len();
        if n == 0 {
            return;
        }
        let i = match self.discover_state.selected() {
            Some(i) if i + 1 < n => i + 1,
            Some(_) => 0,
            None => 0,
        };
        self.discover_state.select(Some(i));
    }

    pub(super) fn discover_pick(&mut self) {
        if let Some(i) = self.discover_state.selected() {
            if let Some(url) = self.discover_results.get(i).cloned() {
                self.discover_results.clear();
                self.discover_state.select(None);
                self.discover_input.clear();
                self.start_add_feed(Some(url));
            }
        }
    }

    pub(super) fn subscribe_preview_source(&mut self, backend: &Backend) {
        if !self.has_key {
            self.set_status("API key required (run: feeds auth)");
            return;
        }
        let Some(url) = self.preview_source_url.clone() else {
            self.set_status("No source URL");
            return;
        };
        match backend.add_subscription(&url, None) {
            Ok(()) => self.set_status("Subscribed"),
            Err(e) => self.set_status(e.to_string()),
        }
    }
}
