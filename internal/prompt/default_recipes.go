package prompt

import (
	"time"

	"github.com/meruem/promptly/pkg/models"
)

// DefaultRecipes returns the built-in recipe definitions seeded from the
// owner's actual recipe collection. These are written into a fresh database
// on first `prompt-agent init`.
// CreatedAt is fixed so ordering is stable and deterministic.
func DefaultRecipes() []models.Prompt {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []models.Prompt{
		{
			ID:          "ci_thin_on_vn_bng_ting_vit",
			Name:        "Cải thiện đoạn văn bằng tiếng Việt",
			Description: "Rà soát và cải thiện đoạn văn bản tiếng Việt về chính tả, ngữ pháp và sự mạch lạc.",
			Engine:      "gemini",
			Template: `Hãy đóng vai một biên tập viên ngôn ngữ chuyên nghiệp. Hãy rà soát và cải thiện đoạn văn bản tiếng Việt sau đây. Nhiệm vụ của bạn là:

Sửa lỗi chính tả và ngữ pháp.

Làm cho câu văn trôi chảy, tự nhiên và mạch lạc hơn.

Giữ nguyên ý nghĩa gốc và văn phong của tác giả (trang trọng hoặc gần gũi).

Văn bản cần chỉnh sửa: {{selection}}`,
			Variables: []string{"selection"},
			Params:    []string{},
			Icon:      "✨",
			SortOrder: 0,
			CreatedAt: seed,
			UpdatedAt: seed,
		},
		{
			ID:          "dch_vn_bn_sang_ting_anh",
			Name:        "Dịch văn bản sang tiếng Anh",
			Description: "Dịch sang tiếng Anh, đảm bảo chính xác từ ngữ và phù hợp văn hóa.",
			Engine:      "gemini",
			Template: `Hãy dịch đoạn văn bản sau sang tiếng Anh. Đảm bảo bản dịch không chỉ chính xác về mặt từ ngữ mà còn phù hợp với ngữ cảnh văn hóa và sắc thái biểu cảm của bản gốc. Tránh dịch word-by-word quá cứng nhắc.

Văn bản gốc: {{selection}}`,
			Variables: []string{"selection"},
			Params:    []string{},
			Icon:      "🇬🇧",
			SortOrder: 1,
			CreatedAt: seed,
			UpdatedAt: seed,
		},
		{
			ID:          "dch_vn_bn_sang_ting_vit",
			Name:        "Dịch văn bản sang tiếng Việt",
			Description: "Dịch sang tiếng Việt, diễn đạt tự nhiên theo cách nói của người Việt.",
			Engine:      "gemini",
			Template: `Hãy dịch đoạn văn bản sau sang tiếng Việt. Yêu cầu bản dịch diễn đạt tự nhiên theo cách nói của người Việt, câu văn gãy gọn, dễ hiểu và truyền tải đầy đủ thông tin từ bản gốc.

Văn bản gốc: {{selection}}`,
			Variables: []string{"selection"},
			Params:    []string{},
			Icon:      "🇻🇳",
			SortOrder: 2,
			CreatedAt: seed,
			UpdatedAt: seed,
		},
		{
			ID:          "dch_vn_bn_k_thut",
			Name:        "Dịch văn bản kỹ thuật",
			Description: "Dịch tài liệu kỹ thuật sang tiếng Việt, giữ nguyên thuật ngữ lập trình phổ biến.",
			Engine:      "gemini",
			Template: `Bạn là một Software Engineer dày dạn kinh nghiệm. Hãy dịch đoạn văn bản kỹ thuật sau đây sang tiếng Việt.
Yêu cầu cụ thể:

Sử dụng ngôn ngữ đơn giản, bình dân, tránh dùng từ Hán Việt quá trang trọng nếu không cần thiết.

Giữ nguyên các thuật ngữ chuyên ngành phổ biến trong giới lập trình (ví dụ: 'build', 'debug', 'instance', 'deploy') nếu việc dịch chúng sang tiếng Việt khiến câu văn trở nên khó hiểu hoặc xa lạ.

Giải thích ngắn gọn trong ngoặc nếu thuật ngữ đó quá trừu tượng.

Ưu tiên sự rõ ràng và tính ứng dụng cao.

Văn bản kỹ thuật: {{selection}}`,
			Variables: []string{"selection"},
			Params:    []string{},
			Icon:      "⚙️",
			SortOrder: 3,
			CreatedAt: seed,
			UpdatedAt: seed,
		},
	}
}
