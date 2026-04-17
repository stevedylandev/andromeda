use super::app::{App, Focus};
use crate::backend::Backend;
use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};

pub(super) fn handle_key(
    app: &mut App,
    backend: &Backend,
    key: KeyEvent,
    content_line_count: u16,
) {
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
                        KeyCode::Enter | KeyCode::Tab => app.focus = Focus::CreateContent,
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
                        KeyCode::Enter | KeyCode::Tab => app.focus = Focus::EditContent,
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
