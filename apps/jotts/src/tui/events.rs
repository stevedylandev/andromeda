use super::app::{App, Focus};
use super::editor::edit_in_external_editor;
use crate::backend::Backend;
use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
use ratatui::DefaultTerminal;

pub(super) fn handle_key(
    terminal: &mut DefaultTerminal,
    app: &mut App,
    backend: &Backend,
    key: KeyEvent,
    content_line_count: u16,
) -> Result<(), Box<dyn std::error::Error>> {
    if app.show_help {
        app.show_help = false;
        return Ok(());
    }
    if app.status_message.is_some() {
        app.status_message = None;
        return Ok(());
    }
    if app.confirm_delete {
        if key.code == KeyCode::Char('y') {
            app.delete_selected(backend);
        }
        app.confirm_delete = false;
        return Ok(());
    }

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
            KeyCode::Char('E') => edit_in_external_editor(terminal, app, backend)?,
            KeyCode::Char('/') => app.start_search(),
            KeyCode::Char('o') => app.open_in_browser(),
            KeyCode::Char('r') if app.is_remote => app.refresh(backend),
            KeyCode::Char('?') => app.show_help = true,
            KeyCode::Enter | KeyCode::Char('l') => {
                if app.selected_note().is_some() {
                    app.focus = Focus::Content;
                }
            }
            _ => {}
        },
        Focus::Content => match key.code {
            KeyCode::Char(' ')
            | KeyCode::Esc
            | KeyCode::Char('q')
            | KeyCode::Char('h') => {
                app.focus = Focus::List;
            }
            KeyCode::Char('j') | KeyCode::Down => app.scroll_down(content_line_count),
            KeyCode::Char('k') | KeyCode::Up => app.scroll_up(),
            KeyCode::Char('y') => app.copy_selected(),
            KeyCode::Char('Y') => app.copy_link(),
            KeyCode::Char('e') => app.start_edit(),
            KeyCode::Char('E') => edit_in_external_editor(terminal, app, backend)?,
            KeyCode::Char('o') => app.open_in_browser(),
            KeyCode::Char('?') => app.show_help = true,
            _ => {}
        },
        Focus::CreateTitle => {
            if key.modifiers.contains(KeyModifiers::CONTROL) && key.code == KeyCode::Char('s') {
                app.save_create(backend);
            } else {
                match key.code {
                    KeyCode::Esc => app.cancel_create(),
                    KeyCode::Enter | KeyCode::Tab => app.focus = Focus::CreateContent,
                    KeyCode::Backspace => {
                        app.edit_title.pop();
                    }
                    KeyCode::Char(c) => app.edit_title.push(c),
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
                    KeyCode::Tab => app.focus = Focus::CreateTitle,
                    KeyCode::Enter => app.edit_content.push('\n'),
                    KeyCode::Backspace => {
                        app.edit_content.pop();
                    }
                    KeyCode::Char(c) => app.edit_content.push(c),
                    _ => {}
                }
            }
        }
        Focus::EditTitle => {
            if key.modifiers.contains(KeyModifiers::CONTROL) && key.code == KeyCode::Char('s') {
                app.save_edit(backend);
            } else {
                match key.code {
                    KeyCode::Esc => app.cancel_edit(),
                    KeyCode::Enter | KeyCode::Tab => app.focus = Focus::EditContent,
                    KeyCode::Backspace => {
                        app.edit_title.pop();
                    }
                    KeyCode::Char(c) => app.edit_title.push(c),
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
                    KeyCode::Tab => app.focus = Focus::EditTitle,
                    KeyCode::Enter => app.edit_content.push('\n'),
                    KeyCode::Backspace => {
                        app.edit_content.pop();
                    }
                    KeyCode::Char(c) => app.edit_content.push(c),
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
    Ok(())
}
