package intelligence

import "context"

const HeuristicModelVersion = "heuristic-v1"

// ModelVersion is the default label when no LLM is configured (tests + meta fallback).
const ModelVersion = HeuristicModelVersion

// ThreadPostInput is one post fed to an LLM summarizer.
type ThreadPostInput struct {
	Author    string
	IsOP      bool
	BodyPlain string
}

// ThreadSummaryInput is everything needed to summarize a thread.
type ThreadSummaryInput struct {
	Title string
	Posts []ThreadPostInput
}

// Summarizer generates a thread summary. Implementations call external LLM APIs.
type Summarizer interface {
	ModelVersion() string
	SummarizeThread(ctx context.Context, in ThreadSummaryInput) (string, error)
}