package mocknode

import (
	"context"
	"errors"
	"testing"
	"time"

	nodev1 "github.com/yeruyi1024/novagrid-node/protocol/node/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNode_Success(t *testing.T) {
	now := fixedTime()
	node := New(ScenarioSuccess, nil, func() time.Time { return now })
	offer := validOffer(now)
	reply, err := node.HandleOffer(offer)
	if err != nil || reply.GetDecision() != nodev1.OfferDecision_OFFER_DECISION_ACCEPTED {
		t.Fatalf("接受 Offer 失败: reply=%v err=%v", reply, err)
	}
	if err := node.GrantLease(validLease(now, offer)); err != nil {
		t.Fatalf("授予 Lease 失败: %v", err)
	}
	if _, err := node.RenewLease(2); err != nil {
		t.Fatalf("续租失败: %v", err)
	}
	first, err := node.Complete(context.Background())
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	secondNode := New(ScenarioSuccess, nil, func() time.Time { return now })
	_, _ = secondNode.HandleOffer(proto.Clone(offer).(*nodev1.TaskOffer))
	_ = secondNode.GrantLease(validLease(now, offer))
	second, _ := secondNode.Complete(context.Background())
	if !proto.Equal(first.GetUsage(), second.GetUsage()) || first.GetContent() != second.GetContent() {
		t.Fatal("相同输入未生成确定性结果")
	}
}

func TestNode_Reject(t *testing.T) {
	node := New(ScenarioReject, nil, fixedClock())
	reply, err := node.HandleOffer(validOffer(fixedTime()))
	if err != nil || reply.GetDecision() != nodev1.OfferDecision_OFFER_DECISION_REJECTED || reply.GetRejectionCode() == "" {
		t.Fatalf("拒绝场景不符合预期: reply=%v err=%v", reply, err)
	}
}

func TestNode_Timeout(t *testing.T) {
	node := New(ScenarioTimeout, nil, fixedClock())
	if _, err := node.HandleOffer(validOffer(fixedTime())); !errors.Is(err, ErrOfferTimeout) {
		t.Fatalf("期望 Offer 超时，实际为 %v", err)
	}
}

func TestNode_Disconnect(t *testing.T) {
	node := New(ScenarioDisconnect, nil, fixedClock())
	if _, err := node.HandleOffer(validOffer(fixedTime())); !errors.Is(err, ErrDisconnected) {
		t.Fatalf("期望断线，实际为 %v", err)
	}
}

func TestNode_Duplicate(t *testing.T) {
	node := New(ScenarioSuccess, nil, fixedClock())
	offer := validOffer(fixedTime())
	first, _ := node.HandleOffer(offer)
	second, err := node.HandleOffer(proto.Clone(offer).(*nodev1.TaskOffer))
	if err != nil || !proto.Equal(first, second) {
		t.Fatalf("重复 Offer 未返回相同决定: err=%v", err)
	}
	conflict := proto.Clone(offer).(*nodev1.TaskOffer)
	conflict.JobId = "other-job"
	if _, err := node.HandleOffer(conflict); !errors.Is(err, ErrDuplicateConflict) {
		t.Fatalf("期望重复冲突，实际为 %v", err)
	}
}

func TestNode_LateResult(t *testing.T) {
	now := fixedTime()
	node := New(ScenarioLate, nil, func() time.Time { return now })
	offer := validOffer(now)
	_, _ = node.HandleOffer(offer)
	_ = node.GrantLease(validLease(now, offer))
	if _, err := node.Complete(context.Background()); !errors.Is(err, ErrLateResult) {
		t.Fatalf("期望迟到结果被丢弃，实际为 %v", err)
	}
}

func TestNode_InvalidOffer(t *testing.T) {
	node := New(ScenarioSuccess, nil, fixedClock())
	invalidJSON := validOffer(fixedTime())
	invalidJSON.MessagesJsonUtf8 = []byte(`{"broken"`)
	cases := []*nodev1.TaskOffer{nil, {}, validOffer(fixedTime().Add(-3 * time.Minute)), invalidJSON}
	for _, offer := range cases {
		if _, err := node.HandleOffer(offer); !errors.Is(err, ErrInvalidOffer) {
			t.Fatalf("期望拒绝非法 Offer，实际为 %v", err)
		}
	}
}

func TestNode_UnknownFields(t *testing.T) {
	offer := validOffer(fixedTime())
	encoded, err := proto.Marshal(offer)
	if err != nil {
		t.Fatal(err)
	}
	encoded = protowire.AppendTag(encoded, 999, protowire.VarintType)
	encoded = protowire.AppendVarint(encoded, 1)
	decoded := &nodev1.TaskOffer{}
	if err := proto.Unmarshal(encoded, decoded); err != nil {
		t.Fatalf("未知可选字段应保持兼容: %v", err)
	}
	node := New(ScenarioSuccess, nil, fixedClock())
	if _, err := node.HandleOffer(decoded); err != nil {
		t.Fatalf("带未知可选字段的 Offer 应被接受: %v", err)
	}
}

func validOffer(now time.Time) *nodev1.TaskOffer {
	return &nodev1.TaskOffer{
		OfferId: "offer-1", JobId: "job-1", AttemptId: "attempt-1", ModelReleaseId: "model-1",
		DeadlineAt: timestamppb.New(now.Add(2 * time.Minute)), MaxOutputTokens: 64,
		Parameters:       &nodev1.ChatParameters{Temperature: 0.7, TopP: 0.9, MaxTokens: 32},
		MessagesJsonUtf8: []byte(`[{"role":"user","content":"unique privacy probe"}]`), PayloadDigest: "sha256:test",
	}
}

func validLease(now time.Time, offer *nodev1.TaskOffer) *nodev1.LeaseGranted {
	return &nodev1.LeaseGranted{
		OfferId: offer.GetOfferId(), JobId: offer.GetJobId(), AttemptId: offer.GetAttemptId(),
		LeaseId: "lease-1", LeaseEpoch: 1, ExpiresAt: timestamppb.New(now.Add(time.Minute)),
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
}

func fixedClock() func() time.Time {
	return func() time.Time { return fixedTime() }
}
