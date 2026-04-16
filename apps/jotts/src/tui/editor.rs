use super::app::App;
use crate::backend::Backend;
use ratatui::DefaultTerminal;
use std::time::Instant;

pub(super) fn edit_in_external_editor(
    terminal: &mut DefaultTerminal,
    app: &mut App,
    backend: &Backend,
) -> Result<(), Box<dyn std::error::Error>> {
    let (short_id, title, content) = match app.selected_note() {
        Some(n) => (n.short_id.clone(), n.title.clone(), n.content.clone()),
        None => return Ok(()),
    };

    let editor = match std::env::var("EDITOR") {
        Ok(e) if !e.trim().is_empty() => e,
        _ => {
            app.status_message = Some(("EDITOR env not set".to_string(), Instant::now()));
            return Ok(());
        }
    };

    let mut path = std::env::temp_dir();
    path.push(format!("jotts-{}.md", short_id));
    std::fs::write(&path, &content)?;

    ratatui::restore();

    let status = std::process::Command::new(&editor).arg(&path).status();

    *terminal = ratatui::init();
    terminal.clear()?;

    match status {
        Ok(s) if s.success() => {
            let new_content = std::fs::read_to_string(&path)?;
            let _ = std::fs::remove_file(&path);
            if new_content == content {
                app.status_message = Some(("No changes".to_string(), Instant::now()));
                return Ok(());
            }
            match backend.update_note(&short_id, &title, &new_content) {
                Ok(Some(updated)) => {
                    if let Some(pos) = app.notes.iter().position(|n| n.short_id == short_id) {
                        app.notes[pos] = updated;
                    }
                    app.status_message = Some(("Updated!".to_string(), Instant::now()));
                }
                Ok(None) => {
                    app.status_message = Some(("Note not found".to_string(), Instant::now()));
                }
                Err(e) => {
                    app.status_message = Some((e.to_string(), Instant::now()));
                }
            }
        }
        Ok(_) => {
            let _ = std::fs::remove_file(&path);
            app.status_message = Some(("Editor exited non-zero".to_string(), Instant::now()));
        }
        Err(e) => {
            let _ = std::fs::remove_file(&path);
            app.status_message =
                Some((format!("Failed to launch editor: {}", e), Instant::now()));
        }
    }
    Ok(())
}
