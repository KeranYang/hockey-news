package main

import (
	"context"
	"log"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// anthropicClient is initialized in main() from the ANTHROPIC_API_KEY env var.
var anthropicClient *anthropic.Client

// isRelevant asks Claude whether an article belongs in the digest for a parent
// of a 7-year-old U8 house league hockey player in Oakville.
// On API failure it fails open (returns true) so news is never silently dropped.
func isRelevant(title, summary string) bool {
	prompt := "You are helping a parent of a 7-year-old boy who plays U8 house league hockey in Oakville, Ontario.\n\n" +
		"Is this article relevant to them?\n\n" +
		"Title: " + title + "\n" +
		"Summary: " + summary + "\n\n" +
		"Reply with only \"yes\" or \"no\"."

	msg, err := anthropicClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     "claude-haiku-4-5",
		MaxTokens: 16,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		log.Printf("Warning: relevance check failed for %q: %v — including article", title, err)
		return true
	}
	if len(msg.Content) == 0 {
		return true
	}

	for _, block := range msg.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			return strings.HasPrefix(strings.ToLower(strings.TrimSpace(tb.Text)), "yes")
		}
	}
	return true
}
