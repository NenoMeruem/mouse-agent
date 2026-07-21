// prompt.rs — Template variable extraction, prompt building, param injection,
// and default recipe seed data.
// Mirrors Go's internal/prompt/ package.

use crate::models::Prompt;
use chrono::Utc;
use regex::Regex;
use std::collections::HashMap;

// ---------------------------------------------------------------------------
// Variable extraction
// ---------------------------------------------------------------------------

/// Finds all `{{variable}}` placeholders in a template string.
/// Returns a deduplicated list in order of first occurrence.
/// Mirrors Go's `ExtractVariables`.
pub fn extract_variables(template: &str) -> Vec<String> {
    let re = Regex::new(r"\{\{(\w+)\}\}").expect("invalid regex");
    let mut seen = std::collections::HashSet::new();
    let mut vars = Vec::new();
    for cap in re.captures_iter(template) {
        let name = cap[1].to_string();
        if seen.insert(name.clone()) {
            vars.push(name);
        }
    }
    vars
}

// ---------------------------------------------------------------------------
// Prompt builder
// ---------------------------------------------------------------------------

/// Replaces `{{key}}` placeholders in `template` with values from `data`.
/// Unknown placeholders are left untouched.
/// Mirrors Go's `SimpleBuilder.Build`.
pub fn build_prompt(template: &str, data: &HashMap<String, String>) -> String {
    let mut result = template.to_string();
    for (key, value) in data {
        let placeholder = format!("{{{{{}}}}}", key);
        result = result.replace(&placeholder, value);
    }
    result
}

// ---------------------------------------------------------------------------
// Param injection
// ---------------------------------------------------------------------------

/// Suffix instructions appended to the prompt for each parameter value.
/// Mirrors Go's `paramSuffixes` map.
fn param_suffix(param: &str, value: &str) -> Option<&'static str> {
    match (param, value) {
        ("tone", "professional") => Some("Reply in a professional tone."),
        ("tone", "casual") => Some("Reply in a casual, friendly tone."),
        ("tone", "concise") => Some("Be very concise and direct."),
        ("length", "short") => Some("Keep the response under 3 sentences."),
        ("length", "medium") => Some("Aim for 1-2 paragraphs."),
        ("length", "long") => Some("Provide a detailed, comprehensive response."),
        ("complexity", "simple") => Some("Explain like I'm 5 years old. Use simple words."),
        ("complexity", "normal") => None,
        ("complexity", "technical") => Some("Use technical terms. Assume expert-level knowledge."),
        _ => None,
    }
}

/// Appends tone / length / complexity instruction suffixes to the prompt.
/// Params are applied in deterministic order: tone → length → complexity.
/// Mirrors Go's `InjectParams`.
pub fn inject_params(prompt: &str, param_values: &HashMap<String, String>) -> String {
    let mut suffixes: Vec<&'static str> = Vec::new();
    for key in &["tone", "length", "complexity"] {
        if let Some(val) = param_values.get(*key) {
            if let Some(suffix) = param_suffix(key, val) {
                suffixes.push(suffix);
            }
        }
    }
    if suffixes.is_empty() {
        return prompt.to_string();
    }
    format!("{}\n\n{}", prompt, suffixes.join(" "))
}

// ---------------------------------------------------------------------------
// Default recipes (seeded on fresh database)
// ---------------------------------------------------------------------------

/// Returns the built-in recipe set written into a fresh DB on first launch.
/// Mirrors Go's `DefaultRecipes()`.
pub fn default_recipes() -> Vec<Prompt> {
    let seed = chrono::DateTime::parse_from_rfc3339("2026-01-01T00:00:00Z")
        .unwrap()
        .with_timezone(&Utc);

    vec![
        Prompt {
            id: "ci_thin_on_vn_bng_ting_vit".into(),
            name: "Cải thiện đoạn văn bằng tiếng Việt".into(),
            description: "Rà soát và cải thiện đoạn văn bản tiếng Việt về chính tả, ngữ pháp và sự mạch lạc.".into(),
            engine: "gemini".into(),
            template: "Hãy đóng vai một biên tập viên ngôn ngữ chuyên nghiệp. Hãy rà soát và cải thiện đoạn văn bản tiếng Việt sau đây. Nhiệm vụ của bạn là:\n\nSửa lỗi chính tả và ngữ pháp.\n\nLàm cho câu văn trôi chảy, tự nhiên và mạch lạc hơn.\n\nGiữ nguyên ý nghĩa gốc và văn phong của tác giả (trang trọng hoặc gần gũi).\n\nVăn bản cần chỉnh sửa: {{selection}}".into(),
            variables: vec!["selection".into()],
            params: vec![],
            icon: "✨".into(),
            sort_order: 0,
            created_at: seed,
            updated_at: seed,
        },
        Prompt {
            id: "dch_vn_bn_sang_ting_anh".into(),
            name: "Dịch văn bản sang tiếng Anh".into(),
            description: "Dịch sang tiếng Anh, đảm bảo chính xác từ ngữ và phù hợp văn hóa.".into(),
            engine: "gemini".into(),
            template: "Hãy dịch đoạn văn bản sau sang tiếng Anh. Đảm bảo bản dịch không chỉ chính xác về mặt từ ngữ mà còn phù hợp với ngữ cảnh văn hóa và sắc thái biểu cảm của bản gốc. Tránh dịch word-by-word quá cứng nhắc.\n\nVăn bản gốc: {{selection}}".into(),
            variables: vec!["selection".into()],
            params: vec![],
            icon: "🇬🇧".into(),
            sort_order: 1,
            created_at: seed,
            updated_at: seed,
        },
        Prompt {
            id: "dch_vn_bn_sang_ting_vit".into(),
            name: "Dịch văn bản sang tiếng Việt".into(),
            description: "Dịch sang tiếng Việt, diễn đạt tự nhiên theo cách nói của người Việt.".into(),
            engine: "gemini".into(),
            template: "Hãy dịch đoạn văn bản sau sang tiếng Việt. Yêu cầu bản dịch diễn đạt tự nhiên theo cách nói của người Việt, câu văn gãy gọn, dễ hiểu và truyền tải đầy đủ thông tin từ bản gốc.\n\nVăn bản gốc: {{selection}}".into(),
            variables: vec!["selection".into()],
            params: vec![],
            icon: "🇻🇳".into(),
            sort_order: 2,
            created_at: seed,
            updated_at: seed,
        },
        Prompt {
            id: "dch_vn_bn_k_thut".into(),
            name: "Dịch văn bản kỹ thuật".into(),
            description: "Dịch tài liệu kỹ thuật sang tiếng Việt, giữ nguyên thuật ngữ lập trình phổ biến.".into(),
            engine: "gemini".into(),
            template: "Bạn là một Software Engineer dày dạn kinh nghiệm. Hãy dịch đoạn văn bản kỹ thuật sau đây sang tiếng Việt.\nYêu cầu cụ thể:\n\nSử dụng ngôn ngữ đơn giản, bình dân, tránh dùng từ Hán Việt quá trang trọng nếu không cần thiết.\n\nGiữ nguyên các thuật ngữ chuyên ngành phổ biến trong giới lập trình (ví dụ: 'build', 'debug', 'instance', 'deploy') nếu việc dịch chúng sang tiếng Việt khiến câu văn trở nên khó hiểu hoặc xa lạ.\n\nGiải thích ngắn gọn trong ngoặc nếu thuật ngữ đó quá trừu tượng.\n\nƯu tiên sự rõ ràng và tính ứng dụng cao.\n\nVăn bản kỹ thuật: {{selection}}".into(),
            variables: vec!["selection".into()],
            params: vec![],
            icon: "⚙️".into(),
            sort_order: 3,
            created_at: seed,
            updated_at: seed,
        },
    ]
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    // --- extract_variables ---

    #[test]
    fn extract_variables_single() {
        let vars = extract_variables("Hello {{name}}");
        assert_eq!(vars, vec!["name"]);
    }

    #[test]
    fn extract_variables_multiple_unique() {
        let vars = extract_variables("{{a}} and {{b}} and {{a}} again");
        assert_eq!(vars, vec!["a", "b"]);
    }

    #[test]
    fn extract_variables_empty_template() {
        let vars = extract_variables("No placeholders here");
        assert!(vars.is_empty());
    }

    #[test]
    fn extract_variables_selection_placeholder() {
        let vars = extract_variables("Process this: {{selection}}");
        assert_eq!(vars, vec!["selection"]);
    }

    // --- build_prompt ---

    #[test]
    fn build_prompt_substitutes_known_vars() {
        let data = HashMap::from([("name".into(), "World".into())]);
        assert_eq!(build_prompt("Hello {{name}}", &data), "Hello World");
    }

    #[test]
    fn build_prompt_leaves_unknown_vars_untouched() {
        let data: HashMap<String, String> = HashMap::new();
        assert_eq!(build_prompt("Hello {{name}}", &data), "Hello {{name}}");
    }

    #[test]
    fn build_prompt_multiple_substitutions() {
        let data =
            HashMap::from([("a".into(), "X".into()), ("b".into(), "Y".into())]);
        assert_eq!(build_prompt("{{a}}-{{b}}", &data), "X-Y");
    }

    #[test]
    fn build_prompt_replaces_all_occurrences() {
        let data = HashMap::from([("x".into(), "Z".into())]);
        assert_eq!(build_prompt("{{x}} {{x}}", &data), "Z Z");
    }

    // --- inject_params ---

    #[test]
    fn inject_params_no_params_returns_original() {
        let params: HashMap<String, String> = HashMap::new();
        assert_eq!(inject_params("My prompt", &params), "My prompt");
    }

    #[test]
    fn inject_params_appends_tone_suffix() {
        let params = HashMap::from([("tone".into(), "professional".into())]);
        let result = inject_params("Prompt", &params);
        assert!(result.contains("Reply in a professional tone."));
    }

    #[test]
    fn inject_params_appends_length_suffix() {
        let params = HashMap::from([("length".into(), "short".into())]);
        let result = inject_params("Prompt", &params);
        assert!(result.contains("Keep the response under 3 sentences."));
    }

    #[test]
    fn inject_params_complexity_normal_adds_nothing() {
        let params = HashMap::from([("complexity".into(), "normal".into())]);
        // "normal" maps to None — no suffix should be added
        assert_eq!(inject_params("Prompt", &params), "Prompt");
    }

    #[test]
    fn inject_params_multiple_suffixes_joined() {
        let params = HashMap::from([
            ("tone".into(), "casual".into()),
            ("length".into(), "long".into()),
        ]);
        let result = inject_params("My prompt", &params);
        assert!(result.contains("Reply in a casual, friendly tone."));
        assert!(result.contains("Provide a detailed, comprehensive response."));
    }

    #[test]
    fn inject_params_unknown_values_add_nothing() {
        let params = HashMap::from([("tone".into(), "robot-voice".into())]);
        assert_eq!(inject_params("Prompt", &params), "Prompt");
    }

    // --- default_recipes ---

    #[test]
    fn default_recipes_returns_four_entries() {
        let recipes = default_recipes();
        assert_eq!(recipes.len(), 4);
    }

    #[test]
    fn default_recipes_have_unique_ids() {
        let recipes = default_recipes();
        let ids: std::collections::HashSet<_> = recipes.iter().map(|r| &r.id).collect();
        assert_eq!(ids.len(), recipes.len());
    }

    #[test]
    fn default_recipes_sorted_by_sort_order() {
        let recipes = default_recipes();
        let orders: Vec<i64> = recipes.iter().map(|r| r.sort_order).collect();
        let mut sorted = orders.clone();
        sorted.sort();
        assert_eq!(orders, sorted);
    }

    #[test]
    fn default_recipes_contain_selection_variable() {
        for recipe in default_recipes() {
            assert!(
                recipe.variables.contains(&"selection".to_string()),
                "Recipe '{}' missing {{{{selection}}}} variable",
                recipe.id
            );
        }
    }
}
