package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yeruyi1024/novagrid-node/internal/mocknode"
	nodev1 "github.com/yeruyi1024/novagrid-node/protocol/node/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type summary struct {
	Scenario         string `json:"scenario"`
	OfferDecision    string `json:"offer_decision,omitempty"`
	Result           string `json:"result"`
	PromptTokens     uint32 `json:"prompt_tokens,omitempty"`
	CompletionTokens uint32 `json:"completion_tokens,omitempty"`
}

func main() {
	scenarioName := flag.String("scenario", string(mocknode.ScenarioSuccess), "模拟场景：success、reject、timeout、disconnect 或 late")
	flag.Parse()

	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	node := mocknode.New(mocknode.Scenario(*scenarioName), nil, func() time.Time { return now })
	offer := exampleOffer(now)
	reply, err := node.HandleOffer(offer)
	if err != nil {
		write(summary{Scenario: *scenarioName, Result: err.Error()})
		return
	}
	if reply.GetDecision() != nodev1.OfferDecision_OFFER_DECISION_ACCEPTED {
		write(summary{Scenario: *scenarioName, OfferDecision: reply.GetDecision().String(), Result: reply.GetRejectionCode()})
		return
	}
	grant := &nodev1.LeaseGranted{
		OfferId: offer.GetOfferId(), JobId: offer.GetJobId(), AttemptId: offer.GetAttemptId(),
		LeaseId: "lease-demo", LeaseEpoch: 1, ExpiresAt: timestamppb.New(now.Add(time.Minute)),
	}
	if err := node.GrantLease(grant); err != nil {
		write(summary{Scenario: *scenarioName, Result: err.Error()})
		return
	}
	result, err := node.Complete(context.Background())
	if err != nil {
		write(summary{Scenario: *scenarioName, OfferDecision: reply.GetDecision().String(), Result: err.Error()})
		return
	}
	write(summary{
		Scenario: *scenarioName, OfferDecision: reply.GetDecision().String(), Result: "completed",
		PromptTokens: result.GetUsage().GetPromptTokens(), CompletionTokens: result.GetUsage().GetCompletionTokens(),
	})
}

func exampleOffer(now time.Time) *nodev1.TaskOffer {
	return &nodev1.TaskOffer{
		OfferId: "offer-demo", JobId: "job-demo", AttemptId: "attempt-demo", ModelReleaseId: "model-demo",
		DeadlineAt: timestamppb.New(now.Add(2 * time.Minute)), MaxOutputTokens: 64,
		Parameters:       &nodev1.ChatParameters{Temperature: 0.7, TopP: 0.9, MaxTokens: 32},
		MessagesJsonUtf8: []byte(`[{"role":"user","content":"mock input"}]`), PayloadDigest: "sha256:demo",
	}
}

func write(value summary) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "输出模拟摘要失败")
		os.Exit(1)
	}
}
