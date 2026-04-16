use super::app::{App, Focus};
use ratatui::{
    Frame,
    layout::{Alignment, Constraint, Layout},
    style::{Color, Modifier, Style},
    text::{Line, Span, Text},
    widgets::{Block, Borders, Clear, List, ListItem, Paragraph, Widget, Wrap},
};

pub(super) fn draw(frame: &mut Frame, app: &mut App) {
    let outer = Layout::vertical([Constraint::Min(1), Constraint::Length(1)]).split(frame.area());

    let chunks =
        Layout::horizontal([Constraint::Percentage(30), Constraint::Percentage(70)]).split(outer[0]);

    let items: Vec<ListItem> = if let Some(indices) = &app.filtered_indices {
        indices
            .iter()
            .filter_map(|&i| app.notes.get(i))
            .map(|n| ListItem::new(n.title.as_str()))
            .collect()
    } else {
        app.notes
            .iter()
            .map(|n| ListItem::new(n.title.as_str()))
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
                .title(" Notes ")
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
        let search_split =
            Layout::vertical([Constraint::Min(1), Constraint::Length(3)]).split(chunks[0]);

        let search_items: Vec<ListItem> = if let Some(indices) = &app.filtered_indices {
            indices
                .iter()
                .filter_map(|&i| app.notes.get(i))
                .map(|n| ListItem::new(n.title.as_str()))
                .collect()
        } else {
            app.notes
                .iter()
                .map(|n| ListItem::new(n.title.as_str()))
                .collect()
        };
        let search_list = List::new(search_items)
            .block(
                Block::default()
                    .title(" Notes ")
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
        Focus::CreateTitle | Focus::CreateContent | Focus::EditTitle | Focus::EditContent => {
            let form_title = match app.focus {
                Focus::EditTitle | Focus::EditContent => " Edit Note ",
                _ => " New Note ",
            };
            let create_block = Block::default()
                .title(form_title)
                .borders(Borders::ALL)
                .border_style(Style::default().fg(Color::Yellow));

            let inner = create_block.inner(chunks[1]);
            frame.render_widget(create_block, chunks[1]);

            let form_layout =
                Layout::vertical([Constraint::Length(3), Constraint::Min(1)]).split(inner);

            let title_style = match app.focus {
                Focus::CreateTitle | Focus::EditTitle => Style::default().fg(Color::Yellow),
                _ => Style::default().fg(Color::DarkGray),
            };
            let title_input = Paragraph::new(app.edit_title.as_str()).block(
                Block::default()
                    .title(" Title ")
                    .borders(Borders::ALL)
                    .border_style(title_style),
            );
            frame.render_widget(title_input, form_layout[0]);

            let content_style = match app.focus {
                Focus::CreateContent | Focus::EditContent => Style::default().fg(Color::Yellow),
                _ => Style::default().fg(Color::DarkGray),
            };
            let mut content_input = Paragraph::new(app.edit_content.as_str()).block(
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

            let content_inner = Block::default().borders(Borders::ALL).inner(form_layout[1]);
            let inner_width = content_inner.width;
            let inner_height = content_inner.height;

            match app.focus {
                Focus::CreateTitle | Focus::EditTitle => {
                    let x = form_layout[0].x + 1 + app.edit_title.len() as u16;
                    let y = form_layout[0].y + 1;
                    frame.set_cursor_position((x, y));
                }
                Focus::CreateContent | Focus::EditContent => {
                    let (cx, cy) = if app.wrap_content {
                        app.cursor_position_wrapped(inner_width)
                    } else {
                        let last_line = app.edit_content.lines().last().unwrap_or("");
                        let line_count = app.edit_content.lines().count()
                            + if app.edit_content.ends_with('\n') { 1 } else { 0 };
                        let y_offset = if line_count == 0 { 0 } else { line_count - 1 };
                        let col = if app.edit_content.ends_with('\n') {
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
            let highlighted = match app.selected_note() {
                Some(n) => app.highlighter.highlight_markdown(&n.content),
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
        Focus::CreateTitle | Focus::CreateContent | Focus::EditTitle | Focus::EditContent => {
            Line::from(vec![
                Span::styled("Tab", Style::default().fg(Color::Yellow)),
                Span::raw(": Switch field  "),
                Span::styled("Ctrl+S", Style::default().fg(Color::Yellow)),
                Span::raw(": Save  "),
                Span::styled("Ctrl+W", Style::default().fg(Color::Yellow)),
                Span::raw(": Wrap  "),
                Span::styled("Esc", Style::default().fg(Color::Yellow)),
                Span::raw(": Cancel"),
            ])
        }
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
        let delete_msg = match app.selected_note() {
            Some(n) => format!("Delete {}? (y/n)", n.title),
            None => "Delete note? (y/n)".to_string(),
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
                Span::styled("  j/↓  ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Move down / Scroll down"),
            ]),
            Line::from(vec![
                Span::styled("  k/↑  ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Move up / Scroll up"),
            ]),
            Line::from(vec![
                Span::styled("  Enter", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("  Focus content pane"),
            ]),
            Line::from(vec![
                Span::styled("  Esc  ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Back / Quit"),
            ]),
            Line::from(vec![
                Span::styled("  y    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Copy note"),
            ]),
            Line::from(vec![
                Span::styled("  Y    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Copy link"),
            ]),
            Line::from(vec![
                Span::styled("  o    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Open in browser"),
            ]),
            Line::from(vec![
                Span::styled("  d    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Delete note"),
            ]),
            Line::from(vec![
                Span::styled("  c    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Create note"),
            ]),
            Line::from(vec![
                Span::styled("  e    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Edit note"),
            ]),
            Line::from(vec![
                Span::styled("  E    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Edit in $EDITOR"),
            ]),
            Line::from(vec![
                Span::styled("  /    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Search notes"),
            ]),
            Line::from(vec![
                Span::styled("  ^W   ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Toggle word wrap (edit)"),
            ]),
        ];

        if app.is_remote {
            help_lines.push(Line::from(vec![
                Span::styled("  r    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Refresh notes"),
            ]));
        }

        help_lines.extend([
            Line::from(vec![
                Span::styled("  q    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
                Span::raw("Quit"),
            ]),
            Line::from(vec![
                Span::styled("  ?    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
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
}
