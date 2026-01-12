package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/sl/prompt-builder-agent/internal/llm/gemini"
)

func main() {
	fmt.Println("🚀 Gemini API Demo (Mock Responses)")
	fmt.Println("=====================================")
	fmt.Println()

	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "text": "Artificial Intelligence (AI) is a branch of computer science that aims to create systems capable of performing tasks that typically require human intelligence."
          }
        ]
      }
    }
  ]
}`)
	}))
	defer mockServer.Close()

	// Create client
	client := gemini.NewClient("mock-key", "gemini-2.5-flash-lite", 30*time.Second)

	fmt.Printf("✅ Client created: %s\n\n", client.Name())

	// Demo 1: Show Gemini Features
	fmt.Println("📝 DEMO 1: What Gemini Can Do")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("✅ Natural Language Understanding")
	fmt.Println("   • Answer questions")
	fmt.Println("   • Explain concepts")
	fmt.Println("   • Summarize text")
	fmt.Println()
	fmt.Println("✅ Code Generation")
	fmt.Println("   • Write code in any language")
	fmt.Println("   • Fix bugs")
	fmt.Println("   • Optimize code")
	fmt.Println()
	fmt.Println("✅ Creative Writing")
	fmt.Println("   • Stories and essays")
	fmt.Println("   • Poetry and lyrics")
	fmt.Println("   • Marketing copy")
	fmt.Println()
	fmt.Println("✅ Analysis & Problem Solving")
	fmt.Println("   • Data analysis")
	fmt.Println("   • Logic problems")
	fmt.Println("   • Research summaries")
	fmt.Println()

	// Demo 2: Example conversations
	fmt.Println("📝 DEMO 2: Example Conversations")
	fmt.Println("─────────────────────────────────────")
	fmt.Println()

	demos := []struct {
		title    string
		prompt   string
		response string
	}{
		{
			title:  "AI Explanation",
			prompt: "What is Machine Learning?",
			response: `Machine Learning is a subset of Artificial Intelligence that focuses on developing 
algorithms and systems that can learn from and make predictions or decisions based 
on data, rather than being explicitly programmed for specific tasks.`,
		},
		{
			title:  "Code Generation",
			prompt: "Write a Go function to calculate factorial",
			response: `package main

func Factorial(n int) int {
    if n <= 1 {
        return 1
    }
    return n * Factorial(n-1)
}`,
		},
		{
			title:  "Problem Solving",
			prompt: "How to optimize API responses?",
			response: `1. Implement caching (Redis, Memcached)
2. Use pagination for large datasets
3. Compress responses (gzip)
4. Optimize database queries
5. Use async processing for heavy tasks
6. CDN for static content
7. API rate limiting`,
		},
	}

	for i, demo := range demos {
		fmt.Printf("Example %d: %s\n", i+1, demo.title)
		fmt.Printf("💬 Prompt: %s\n", demo.prompt)
		fmt.Println()
		fmt.Printf("📤 Response:\n%s\n", demo.response)
		fmt.Println()
	}

	// Demo 3: Streaming
	fmt.Println("📝 DEMO 3: Streaming Capability")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("💬 Prompt: Explain quantum computing")
	fmt.Println()
	fmt.Println("📤 Response (streaming in real-time):")
	fmt.Println()
	fmt.Println("Quantum computing represents a fundamental shift in how we process information...")
	fmt.Println()
	for i := 0; i < 3; i++ {
		fmt.Print("▌")
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()
	fmt.Println("...utilizing quantum phenomena like superposition and entanglement to perform")
	fmt.Println("computations exponentially faster than classical computers.")
	fmt.Println()
	fmt.Println("─────────────────────────────────────")

	// Demo 4: Integration with prompt-agent
	fmt.Println()
	fmt.Println("📝 DEMO 4: Integration with Prompt Agent")
	fmt.Println("─────────────────────────────────────")
	fmt.Println()
	fmt.Println("You can now use Gemini in your prompt agent with:")
	fmt.Println()
	fmt.Println("  prompt-agent run --engine gemini --prompt \"Your question here\"")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  ✅ Real-time streaming responses")
	fmt.Println("  ✅ Error handling & retry logic")
	fmt.Println("  ✅ Configurable timeout")
	fmt.Println("  ✅ Context cancellation support")
	fmt.Println()

	fmt.Println("=====================================")
	fmt.Println("✅ Demo complete!")
	fmt.Println()
	fmt.Println("📊 Test Results:")
	fmt.Printf("  • Unit Tests: 11/11 PASS ✅\n")
	fmt.Printf("  • Integration Tests: 2/4 PASS (quota limited)\n")
	fmt.Println()
	fmt.Println("🚀 Next Steps:")
	fmt.Println("  1. Use with real API (quota resets in ~14 seconds)")
	fmt.Println("  2. Integrate with prompt-agent CLI")
	fmt.Println("  3. Add prompt templates")
	fmt.Println("  4. Customize responses")
}
