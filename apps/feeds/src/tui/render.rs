use super::app::{App, Focus, Mode};
use chrono::DateTime;
use ratatui::{
    Frame,
    layout::{Alignment, Constraint, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span, Text},
    widgets::{Block, Borders, Clear, List, ListItem, Paragraph, Widget},
};

fn fmt_date(ts: i64) -> String {
    DateTime::from_timestamp(ts, 0)
        .map(|dt| dt.format("%b %-d").to_string())
        .unwrap_or_default()
}

pub(super) fn draw(frame: &mut Frame, app: &mut App) {
    let outer = Layout::vertical([Constraint::Min(1), Constraint::Length(1)]).split(frame.area());

    let title = match app.mode {
        Mode::Aggregate => " Feeds ".to_string(),
        Mode::Preview {} => format!(
            " Preview: {} ",
            app.preview_source_url.as_deref().unwrap_or("")
        ),
    };

    let items: Vec<ListItem> = app
        .items
        .iter()
        .map(|i| {
            let date = fmt_date(i.published_at);
            let source = i
                .feed_title
                .clone()
                .or_else(|| i.author.clone())
                .unwrap_or_default();
            let line = if source.is_empty() {
                Line::from(vec![
                    Span::styled(format!("{date:>6}  "), Style::default().fg(Color::DarkGray)),
                    Span::raw(i.title.clone()),
                ])
            } else {
                Line::from(vec![
                    Span::styled(format!("{date:>6}  "), Style::default().fg(Color::DarkGray)),
                    Span::raw(i.title.clone()),
                    Span::styled(format!("  — {source}"), Style::default().fg(Color::DarkGray)),
                ])
            };
            ListItem::new(line)
        })
        .collect();

    let list_border = Style::default().fg(Color::Yellow);
    let list = List::new(items)
        .block(
            Block::default()
                .title(title)
                .borders(Borders::ALL)
                .border_style(list_border),
        )
        .highlight_style(Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD))
        .highlight_symbol("▶ ");

    frame.render_stateful_widget(list, outer[0], &mut app.list_state);

    let hints = match app.mode {
        Mode::Aggregate => Line::from(vec![
            Span::styled("j/k", Style::default().fg(Color::Yellow)),
            Span::raw(": Nav  "),
            Span::styled("o/Enter", Style::default().fg(Color::Yellow)),
            Span::raw(": Open  "),
            Span::styled("a", Style::default().fg(Color::Yellow)),
            Span::raw(": Add  "),
            Span::styled("d", Style::default().fg(Color::Yellow)),
            Span::raw(": Discover  "),
            Span::styled("r", Style::default().fg(Color::Yellow)),
            Span::raw(": Refresh  "),
            Span::styled("?", Style::default().fg(Color::Yellow)),
            Span::raw(": Help  "),
            Span::styled("q", Style::default().fg(Color::Yellow)),
            Span::raw(": Quit"),
        ]),
        Mode::Preview {} => Line::from(vec![
            Span::styled("j/k", Style::default().fg(Color::Yellow)),
            Span::raw(": Nav  "),
            Span::styled("o/Enter", Style::default().fg(Color::Yellow)),
            Span::raw(": Open  "),
            Span::styled("s", Style::default().fg(Color::Yellow)),
            Span::raw(": Subscribe  "),
            Span::styled("r", Style::default().fg(Color::Yellow)),
            Span::raw(": Refresh  "),
            Span::styled("q", Style::default().fg(Color::Yellow)),
            Span::raw(": Quit"),
        ]),
    };
    frame.render_widget(Paragraph::new(hints), outer[1]);

    match app.focus {
        Focus::AddFeedUrl | Focus::AddFeedCategory => draw_add_feed(frame, app),
        Focus::DiscoverInput => draw_discover_input(frame, app),
        Focus::DiscoverPicker => draw_discover_picker(frame, app),
        Focus::List => {}
    }

    if let Some((msg, _)) = &app.status_message {
        draw_popup(frame, msg, Color::Green);
    }

    if app.show_help {
        draw_help(frame, app);
    }
}

fn centered_rect(width: u16, height: u16, area: Rect) -> Rect {
    let w = width.min(area.width.saturating_sub(2));
    let h = height.min(area.height.saturating_sub(2));
    Rect {
        x: area.x + (area.width.saturating_sub(w)) / 2,
        y: area.y + (area.height.saturating_sub(h)) / 2,
        width: w,
        height: h,
    }
}

fn draw_add_feed(frame: &mut Frame, app: &App) {
    let area = centered_rect(70, 9, frame.area());
    Clear.render(area, frame.buffer_mut());
    let block = Block::default()
        .title(" Add Feed ")
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Yellow));
    let inner = block.inner(area);
    frame.render_widget(block, area);

    let layout = Layout::vertical([Constraint::Length(3), Constraint::Length(3)]).split(inner);

    let url_style = if app.focus == Focus::AddFeedUrl {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::DarkGray)
    };
    let url_input = Paragraph::new(app.add_url.as_str()).block(
        Block::default()
            .title(" Feed URL ")
            .borders(Borders::ALL)
            .border_style(url_style),
    );
    frame.render_widget(url_input, layout[0]);

    let cat_style = if app.focus == Focus::AddFeedCategory {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::DarkGray)
    };
    let cat_input = Paragraph::new(app.add_category.as_str()).block(
        Block::default()
            .title(" Category (optional) ")
            .borders(Borders::ALL)
            .border_style(cat_style),
    );
    frame.render_widget(cat_input, layout[1]);

    match app.focus {
        Focus::AddFeedUrl => {
            let x = layout[0].x + 1 + app.add_url.len() as u16;
            let y = layout[0].y + 1;
            frame.set_cursor_position((x, y));
        }
        Focus::AddFeedCategory => {
            let x = layout[1].x + 1 + app.add_category.len() as u16;
            let y = layout[1].y + 1;
            frame.set_cursor_position((x, y));
        }
        _ => {}
    }
}

fn draw_discover_input(frame: &mut Frame, app: &App) {
    let area = centered_rect(70, 5, frame.area());
    Clear.render(area, frame.buffer_mut());
    let block = Block::default()
        .title(" Discover Feeds (enter site URL) ")
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::Yellow));
    let inner = block.inner(area);
    frame.render_widget(block, area);

    let p = Paragraph::new(app.discover_input.as_str());
    frame.render_widget(p, inner);
    let x = inner.x + app.discover_input.len() as u16;
    let y = inner.y;
    frame.set_cursor_position((x, y));
}

fn draw_discover_picker(frame: &mut Frame, app: &mut App) {
    let area = centered_rect(80, 12, frame.area());
    Clear.render(area, frame.buffer_mut());
    let items: Vec<ListItem> = app
        .discover_results
        .iter()
        .map(|u| ListItem::new(u.as_str()))
        .collect();
    let list = List::new(items)
        .block(
            Block::default()
                .title(" Discovered Feeds — Enter to add ")
                .borders(Borders::ALL)
                .border_style(Style::default().fg(Color::Yellow)),
        )
        .highlight_style(Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD))
        .highlight_symbol("▶ ");
    frame.render_stateful_widget(list, area, &mut app.discover_state);
}

fn draw_popup(frame: &mut Frame, msg: &str, color: Color) {
    let w = (msg.len() as u16 + 4).max(20);
    let area = centered_rect(w, 3, frame.area());
    Clear.render(area, frame.buffer_mut());
    let p = Paragraph::new(Line::from(msg.to_string()))
        .style(Style::default().fg(color).add_modifier(Modifier::BOLD))
        .alignment(Alignment::Center)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .border_style(Style::default().fg(color)),
        );
    frame.render_widget(p, area);
}

fn draw_help(frame: &mut Frame, _app: &App) {
    let area = centered_rect(40, 14, frame.area());
    Clear.render(area, frame.buffer_mut());
    let lines = vec![
        Line::from(""),
        Line::from(vec![
            Span::styled("  j/k    ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Navigate"),
        ]),
        Line::from(vec![
            Span::styled("  o      ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Open in browser"),
        ]),
        Line::from(vec![
            Span::styled("  Enter  ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Open in browser"),
        ]),
        Line::from(vec![
            Span::styled("  a      ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Add feed"),
        ]),
        Line::from(vec![
            Span::styled("  d      ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Discover feeds"),
        ]),
        Line::from(vec![
            Span::styled("  s      ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Subscribe (preview)"),
        ]),
        Line::from(vec![
            Span::styled("  r      ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Refresh"),
        ]),
        Line::from(vec![
            Span::styled("  q      ", Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD)),
            Span::raw("Quit"),
        ]),
        Line::from(""),
        Line::from(Span::styled(
            "  Press any key to close",
            Style::default().fg(Color::DarkGray),
        )),
    ];
    let p = Paragraph::new(Text::from(lines)).block(
        Block::default()
            .title(" Keybindings ")
            .borders(Borders::ALL)
            .border_style(Style::default().fg(Color::Yellow)),
    );
    frame.render_widget(p, area);
}
