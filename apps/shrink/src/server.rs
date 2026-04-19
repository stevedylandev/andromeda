use askama::Template;
use axum::{
    Router,
    extract::Multipart,
    http::{StatusCode, header},
    response::{Html, IntoResponse, Response},
    routing::{get, post},
};
use axum::extract::DefaultBodyLimit;
use img_parts::ImageEXIF;
use img_parts::jpeg::Jpeg;
use tower_http::services::ServeDir;

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate;

pub async fn run(host: String, port: u16) {
    let app = Router::new()
        .route("/", get(get_index))
        .route("/compress", post(post_compress))
        .layer(DefaultBodyLimit::max(20 * 1024 * 1024))
        .nest_service("/static", ServeDir::new("static"))
        .merge(andromeda_darkmatter_css::router::<()>());

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on {}", addr);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn get_index() -> impl IntoResponse {
    let html = IndexTemplate.render().unwrap();
    Html(html)
}

async fn post_compress(mut multipart: Multipart) -> Result<Response, (StatusCode, String)> {
    let mut file_data: Option<Vec<u8>> = None;
    let mut quality: u8 = 80;
    let mut width: u32 = 0;
    let mut original_filename: String = "image".to_string();

    while let Ok(Some(field)) = multipart.next_field().await {
        let name = field.name().unwrap_or("").to_string();
        match name.as_str() {
            "file" => {
                if let Some(fname) = field.file_name() {
                    original_filename = fname.to_string();
                }
                let bytes = field
                    .bytes()
                    .await
                    .map_err(|e| (StatusCode::BAD_REQUEST, format!("Failed to read file: {}", e)))?;
                file_data = Some(bytes.to_vec());
            }
            "quality" => {
                let text = field
                    .text()
                    .await
                    .map_err(|e| (StatusCode::BAD_REQUEST, format!("Failed to read quality: {}", e)))?;
                quality = text.parse::<u8>().unwrap_or(80).clamp(1, 100);
            }
            "width" => {
                let text = field
                    .text()
                    .await
                    .map_err(|e| (StatusCode::BAD_REQUEST, format!("Failed to read width: {}", e)))?;
                width = text.parse::<u32>().unwrap_or(0);
            }
_ => {}
        }
    }

    let file_data = file_data.ok_or((StatusCode::BAD_REQUEST, "No file provided".to_string()))?;

    let result =
        tokio::task::spawn_blocking(move || compress_image(&file_data, quality, width))
            .await
            .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, format!("Task failed: {}", e)))?
            .map_err(|e| {
                (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("Compression failed: {}", e),
                )
            })?;

    let download_name = build_download_filename(&original_filename, "jpg");

    Ok((
        StatusCode::OK,
        [
            (header::CONTENT_TYPE, "image/jpeg".to_string()),
            (
                header::CONTENT_DISPOSITION,
                format!("attachment; filename=\"{}\"", download_name),
            ),
        ],
        result,
    )
        .into_response())
}

fn compress_image(data: &[u8], quality: u8, width: u32) -> Result<Vec<u8>, String> {
    // Extract EXIF from original before re-encoding destroys it
    let original_exif = Jpeg::from_bytes(data.to_vec().into())
        .ok()
        .and_then(|j| j.exif().map(|e| e.to_vec()));

    let img =
        image::load_from_memory(data).map_err(|e| format!("Failed to decode image: {}", e))?;

    let img = if width > 0 && width != img.width() {
        let aspect = img.height() as f64 / img.width() as f64;
        let height = (width as f64 * aspect).round() as u32;
        img.resize(width, height, image::imageops::FilterType::Lanczos3)
    } else {
        img
    };

    let mut output = Vec::new();
    let encoder = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut output, quality);
    img.write_with_encoder(encoder)
        .map_err(|e| format!("JPEG encoding failed: {}", e))?;

    // Re-inject EXIF into the compressed output (always strip GPS data)
    if let Some(exif_bytes) = original_exif {
        let exif = strip_gps_from_exif(&exif_bytes);

        let mut out_jpeg = Jpeg::from_bytes(output.into())
            .map_err(|e| format!("Failed to parse compressed JPEG: {}", e))?;
        out_jpeg.set_exif(Some(exif.into()));
        let mut final_output = Vec::new();
        out_jpeg
            .encoder()
            .write_to(&mut final_output)
            .map_err(|e| format!("Failed to write JPEG with EXIF: {}", e))?;
        Ok(final_output)
    } else {
        Ok(output)
    }
}

/// Strips GPS data from raw EXIF bytes by zeroing the GPS IFD entry count.
/// Preserves all other metadata (camera, lens, settings, etc.) and offsets.
fn strip_gps_from_exif(exif: &[u8]) -> Vec<u8> {
    let mut data = exif.to_vec();

    // img-parts strips the "Exif\0\0" prefix, so bytes start with TIFF header (II/MM)
    let tiff_start = if data.len() >= 14 && &data[0..4] == b"Exif" {
        6
    } else if data.len() >= 8 && (&data[0..2] == b"II" || &data[0..2] == b"MM") {
        0
    } else {
        return data;
    };
    let big_endian = &data[tiff_start..tiff_start + 2] == b"MM";

    let read_u16 = |d: &[u8], off: usize| -> u16 {
        if big_endian {
            u16::from_be_bytes([d[off], d[off + 1]])
        } else {
            u16::from_le_bytes([d[off], d[off + 1]])
        }
    };

    let read_u32 = |d: &[u8], off: usize| -> u32 {
        if big_endian {
            u32::from_be_bytes([d[off], d[off + 1], d[off + 2], d[off + 3]])
        } else {
            u32::from_le_bytes([d[off], d[off + 1], d[off + 2], d[off + 3]])
        }
    };

    // IFD0 offset (relative to TIFF start)
    let ifd0_rel = read_u32(&data, tiff_start + 4) as usize;
    let ifd0_off = tiff_start + ifd0_rel;
    if ifd0_off + 2 > data.len() {
        return data;
    }

    let entry_count = read_u16(&data, ifd0_off) as usize;

    for i in 0..entry_count {
        let entry_off = ifd0_off + 2 + i * 12;
        if entry_off + 12 > data.len() {
            break;
        }
        let tag = read_u16(&data, entry_off);
        if tag == 0x8825 {
            // GPS IFD pointer — read the offset, then zero out the GPS IFD entry count
            let gps_ifd_rel = read_u32(&data, entry_off + 8) as usize;
            let gps_ifd_off = tiff_start + gps_ifd_rel;
            if gps_ifd_off + 2 <= data.len() {
                let zero = if big_endian {
                    0u16.to_be_bytes()
                } else {
                    0u16.to_le_bytes()
                };
                data[gps_ifd_off] = zero[0];
                data[gps_ifd_off + 1] = zero[1];
            }
            break;
        }
    }

    data
}

fn build_download_filename(original: &str, new_ext: &str) -> String {
    let stem = std::path::Path::new(original)
        .file_stem()
        .and_then(|s| s.to_str())
        .unwrap_or("compressed");
    format!("{}_compressed.{}", stem, new_ext)
}

#[cfg(test)]
mod tests {
    use super::*;

    // ── build_download_filename ────────────────────────────────────────

    #[test]
    fn filename_with_extension() {
        assert_eq!(build_download_filename("photo.png", "jpg"), "photo_compressed.jpg");
    }

    #[test]
    fn filename_without_extension() {
        assert_eq!(build_download_filename("photo", "jpg"), "photo_compressed.jpg");
    }

    #[test]
    fn filename_empty_string() {
        assert_eq!(build_download_filename("", "jpg"), "compressed_compressed.jpg");
    }

    #[test]
    fn filename_multiple_dots() {
        assert_eq!(
            build_download_filename("my.cool.photo.png", "jpg"),
            "my.cool.photo_compressed.jpg"
        );
    }

    // ── strip_gps_from_exif ────────────────────────────────────────────

    #[test]
    fn strip_gps_too_short_returns_unchanged() {
        let data = vec![0u8; 4];
        assert_eq!(strip_gps_from_exif(&data), data);
    }

    #[test]
    fn strip_gps_no_tiff_header_returns_unchanged() {
        let data = vec![0u8; 32];
        assert_eq!(strip_gps_from_exif(&data), data);
    }

    #[test]
    fn strip_gps_little_endian_zeroes_gps_ifd() {
        // Build minimal TIFF (little-endian) with one IFD entry: GPS tag 0x8825
        let mut data = Vec::new();
        // TIFF header: "II" + magic 42 + offset to IFD0 (8)
        data.extend_from_slice(b"II");
        data.extend_from_slice(&42u16.to_le_bytes());
        data.extend_from_slice(&8u32.to_le_bytes()); // IFD0 at offset 8

        // IFD0 at offset 8: 1 entry
        data.extend_from_slice(&1u16.to_le_bytes());

        // IFD entry: tag=0x8825 (GPS), type=LONG(4), count=1, value=offset to GPS IFD
        let gps_ifd_offset: u32 = 22; // right after this IFD entry + next IFD pointer
        data.extend_from_slice(&0x8825u16.to_le_bytes());
        data.extend_from_slice(&4u16.to_le_bytes()); // type LONG
        data.extend_from_slice(&1u32.to_le_bytes()); // count
        data.extend_from_slice(&gps_ifd_offset.to_le_bytes()); // GPS IFD offset

        // Next IFD pointer (none)
        // We need padding to get to offset 22
        // Current size = 8 + 2 + 12 = 22, perfect

        // GPS IFD at offset 22: entry count = 5 (nonzero, should be zeroed)
        data.extend_from_slice(&5u16.to_le_bytes());
        // Some dummy GPS entries
        data.extend_from_slice(&[0u8; 24]);

        let result = strip_gps_from_exif(&data);
        // GPS IFD entry count at offset 22 should now be 0
        let gps_count = u16::from_le_bytes([result[22], result[23]]);
        assert_eq!(gps_count, 0);
    }

    #[test]
    fn strip_gps_big_endian_zeroes_gps_ifd() {
        let mut data = Vec::new();
        // TIFF header: "MM" + magic 42 + offset to IFD0 (8)
        data.extend_from_slice(b"MM");
        data.extend_from_slice(&42u16.to_be_bytes());
        data.extend_from_slice(&8u32.to_be_bytes());

        // IFD0: 1 entry
        data.extend_from_slice(&1u16.to_be_bytes());

        // GPS tag entry pointing to GPS IFD at offset 22
        data.extend_from_slice(&0x8825u16.to_be_bytes());
        data.extend_from_slice(&4u16.to_be_bytes());
        data.extend_from_slice(&1u32.to_be_bytes());
        data.extend_from_slice(&22u32.to_be_bytes());

        // GPS IFD at offset 22
        data.extend_from_slice(&3u16.to_be_bytes()); // 3 entries
        data.extend_from_slice(&[0u8; 24]);

        let result = strip_gps_from_exif(&data);
        let gps_count = u16::from_be_bytes([result[22], result[23]]);
        assert_eq!(gps_count, 0);
    }

    #[test]
    fn strip_gps_no_gps_tag_unchanged() {
        let mut data = Vec::new();
        data.extend_from_slice(b"II");
        data.extend_from_slice(&42u16.to_le_bytes());
        data.extend_from_slice(&8u32.to_le_bytes());

        // IFD0: 1 entry, but NOT a GPS tag (use 0x010F = Make)
        data.extend_from_slice(&1u16.to_le_bytes());
        data.extend_from_slice(&0x010Fu16.to_le_bytes());
        data.extend_from_slice(&2u16.to_le_bytes());
        data.extend_from_slice(&1u32.to_le_bytes());
        data.extend_from_slice(&0u32.to_le_bytes());

        let original = data.clone();
        let result = strip_gps_from_exif(&data);
        assert_eq!(result, original);
    }

    // ── compress_image ─────────────────────────────────────────────────

    #[test]
    fn compress_image_invalid_data_returns_error() {
        let result = compress_image(&[0, 1, 2, 3], 80, 0);
        assert!(result.is_err());
    }

    #[test]
    fn compress_image_valid_jpeg() {
        // Create a minimal 2x2 RGB image and encode as JPEG
        let img = image::RgbImage::from_fn(2, 2, |_, _| image::Rgb([255u8, 0, 0]));
        let mut buf = Vec::new();
        let encoder = image::codecs::jpeg::JpegEncoder::new(&mut buf);
        image::DynamicImage::ImageRgb8(img)
            .write_with_encoder(encoder)
            .unwrap();

        let result = compress_image(&buf, 80, 0);
        assert!(result.is_ok());
        assert!(!result.unwrap().is_empty());
    }

    #[test]
    fn compress_image_with_resize() {
        let img = image::RgbImage::from_fn(100, 50, |_, _| image::Rgb([0u8, 128, 255]));
        let mut buf = Vec::new();
        let encoder = image::codecs::jpeg::JpegEncoder::new(&mut buf);
        image::DynamicImage::ImageRgb8(img)
            .write_with_encoder(encoder)
            .unwrap();

        let result = compress_image(&buf, 80, 50).unwrap();
        // Verify output is valid JPEG (starts with FFD8)
        assert!(result.len() >= 2);
        assert_eq!(result[0], 0xFF);
        assert_eq!(result[1], 0xD8);
    }
}
