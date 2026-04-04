use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use serde::{Deserialize, Serialize};

#[derive(Serialize)]
struct ClaudeRequest {
    model: String,
    max_tokens: u32,
    messages: Vec<ClaudeMessage>,
}

#[derive(Serialize)]
struct ClaudeMessage {
    role: String,
    content: Vec<ClaudeContent>,
}

#[derive(Serialize)]
#[serde(tag = "type")]
enum ClaudeContent {
    #[serde(rename = "image")]
    Image { source: ImageSource },
    #[serde(rename = "text")]
    Text { text: String },
}

#[derive(Serialize)]
struct ImageSource {
    #[serde(rename = "type")]
    source_type: String,
    media_type: String,
    data: String,
}

#[derive(Deserialize, Serialize)]
pub struct AnalyzeResult {
    pub name: String,
    pub origin: String,
    pub grape: String,
    pub background: String,
}

#[derive(Deserialize)]
struct ClaudeResponse {
    content: Vec<ContentBlock>,
}

#[derive(Deserialize)]
struct ContentBlock {
    text: Option<String>,
}

pub async fn analyze_wine_image(
    api_key: &str,
    image_bytes: &[u8],
    media_type: &str,
) -> Result<AnalyzeResult, String> {
    let encoded = STANDARD.encode(image_bytes);

    let request = ClaudeRequest {
        model: "claude-sonnet-4-20250514".to_string(),
        max_tokens: 1024,
        messages: vec![ClaudeMessage {
            role: "user".to_string(),
            content: vec![
                ClaudeContent::Image {
                    source: ImageSource {
                        source_type: "base64".to_string(),
                        media_type: media_type.to_string(),
                        data: encoded,
                    },
                },
                ClaudeContent::Text {
                    text: "Look at this wine bottle label. Return a JSON object with exactly these fields: {\"name\": \"the full wine name\", \"origin\": \"region and/or country\", \"grape\": \"grape variety or blend\", \"background\": \"brief background about the wine and the winery, including any notable history or interesting facts\"}. If you cannot determine a field, use an empty string. Respond with ONLY the JSON, no other text.".to_string(),
                },
            ],
        }],
    };

    let client = reqwest::Client::new();
    let response = client
        .post("https://api.anthropic.com/v1/messages")
        .header("x-api-key", api_key)
        .header("anthropic-version", "2023-06-01")
        .header("content-type", "application/json")
        .json(&request)
        .send()
        .await
        .map_err(|e| format!("Request failed: {}", e))?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();
        return Err(format!("API error {}: {}", status, body));
    }

    let claude_response: ClaudeResponse = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse response: {}", e))?;

    let text = claude_response
        .content
        .iter()
        .find_map(|block| block.text.as_ref())
        .ok_or_else(|| "No text in response".to_string())?;

    let text = text.trim();
    let json_str = if let Some(start) = text.find('{') {
        if let Some(end) = text.rfind('}') {
            &text[start..=end]
        } else {
            text
        }
    } else {
        text
    };

    serde_json::from_str(json_str).map_err(|e| format!("Failed to parse JSON: {} (raw: {})", e, text))
}
