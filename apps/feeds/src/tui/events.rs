use super::app::{App, Focus, Mode};
use crate::backend::Backend;
use crossterm::event::{KeyCode, KeyEvent};

pub(super) fn handle_key(app: &mut App, backend: &Backend, key: KeyEvent) {
    if app.show_help {
        app.show_help = false;
        return;
    }
    if app.status_message.is_some() {
        app.status_message = None;
        return;
    }

    match app.focus {
        Focus::List => match (app.mode, key.code) {
            (_, KeyCode::Char('q') | KeyCode::Esc) => app.should_quit = true,
            (_, KeyCode::Char('j') | KeyCode::Down) => app.move_down(),
            (_, KeyCode::Char('k') | KeyCode::Up) => app.move_up(),
            (_, KeyCode::Char('o') | KeyCode::Enter) => app.open_selected(),
            (_, KeyCode::Char('r')) => app.refresh(backend),
            (_, KeyCode::Char('?')) => app.show_help = true,
            (Mode::Aggregate, KeyCode::Char('a')) => app.start_add_feed(None),
            (Mode::Aggregate, KeyCode::Char('d')) => app.start_discover(),
            (Mode::Preview {}, KeyCode::Char('s')) => app.subscribe_preview_source(backend),
            _ => {}
        },
        Focus::AddFeedUrl => match key.code {
            KeyCode::Esc => app.cancel_add_feed(),
            KeyCode::Tab | KeyCode::Enter => app.focus = Focus::AddFeedCategory,
            KeyCode::Backspace => {
                app.add_url.pop();
            }
            KeyCode::Char(c) => app.add_url.push(c),
            _ => {}
        },
        Focus::AddFeedCategory => match key.code {
            KeyCode::Esc => app.cancel_add_feed(),
            KeyCode::Tab => app.focus = Focus::AddFeedUrl,
            KeyCode::Enter => app.submit_add_feed(backend),
            KeyCode::Backspace => {
                app.add_category.pop();
            }
            KeyCode::Char(c) => app.add_category.push(c),
            _ => {}
        },
        Focus::DiscoverInput => match key.code {
            KeyCode::Esc => app.cancel_discover(),
            KeyCode::Enter => app.submit_discover(backend),
            KeyCode::Backspace => {
                app.discover_input.pop();
            }
            KeyCode::Char(c) => app.discover_input.push(c),
            _ => {}
        },
        Focus::DiscoverPicker => match key.code {
            KeyCode::Esc | KeyCode::Char('q') => app.cancel_discover(),
            KeyCode::Char('j') | KeyCode::Down => app.discover_picker_down(),
            KeyCode::Char('k') | KeyCode::Up => app.discover_picker_up(),
            KeyCode::Enter => app.discover_pick(),
            _ => {}
        },
    }
}
