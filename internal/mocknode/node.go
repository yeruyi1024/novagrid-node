package mocknode

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	nodev1 "github.com/yeruyi1024/novagrid-node/protocol/node/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ProtocolVersion = "v1"

// Scenario 控制模拟节点的确定性故障行为。
type Scenario string

const (
	ScenarioSuccess    Scenario = "success"
	ScenarioReject     Scenario = "reject"
	ScenarioTimeout    Scenario = "timeout"
	ScenarioDisconnect Scenario = "disconnect"
	ScenarioLate       Scenario = "late"
)

type offerRecord struct {
	digest [32]byte
	reply  *nodev1.OfferReply
	offer  *nodev1.TaskOffer
}

// Node 在内存中模拟单 GPU、单活动 Lease 的 NodeChannel 消费者。
type Node struct {
	mu          sync.Mutex
	now         func() time.Time
	runtime     Runtime
	scenario    Scenario
	offers      map[string]offerRecord
	activeLease *nodev1.LeaseGranted
}

// New 创建无持久化、无网络和无 GPU 访问的确定性模拟节点。
func New(scenario Scenario, runtime Runtime, now func() time.Time) *Node {
	if runtime == nil {
		runtime = DeterministicRuntime{}
	}
	if now == nil {
		now = time.Now
	}
	return &Node{scenario: scenario, runtime: runtime, now: now, offers: make(map[string]offerRecord)}
}

// HandleOffer 校验 TaskOffer，并保证相同 offer_id 的重复决定一致。
func (n *Node) HandleOffer(offer *nodev1.TaskOffer) (*nodev1.OfferReply, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if err := validateOffer(offer, n.now()); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(mustMarshal(offer))
	if previous, ok := n.offers[offer.GetOfferId()]; ok {
		if previous.digest != digest {
			return nil, ErrDuplicateConflict
		}
		return proto.Clone(previous.reply).(*nodev1.OfferReply), nil
	}

	switch n.scenario {
	case ScenarioTimeout:
		return nil, ErrOfferTimeout
	case ScenarioDisconnect:
		return nil, ErrDisconnected
	}

	decision := nodev1.OfferDecision_OFFER_DECISION_ACCEPTED
	rejectionCode := ""
	if n.scenario == ScenarioReject || n.activeLease != nil {
		decision = nodev1.OfferDecision_OFFER_DECISION_REJECTED
		rejectionCode = "GPU_BUSY"
	}
	reply := &nodev1.OfferReply{
		OfferId:       offer.GetOfferId(),
		Decision:      decision,
		RejectionCode: rejectionCode,
		DecidedAt:     timestamppb.New(n.now()),
	}
	n.offers[offer.GetOfferId()] = offerRecord{digest: digest, reply: reply, offer: proto.Clone(offer).(*nodev1.TaskOffer)}
	return proto.Clone(reply).(*nodev1.OfferReply), nil
}

// GrantLease 接受 Control 的唯一执行授权，并拒绝未接受、过期或并发 Lease。
func (n *Node) GrantLease(grant *nodev1.LeaseGranted) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	record, ok := n.offers[grant.GetOfferId()]
	if !ok || record.reply.GetDecision() != nodev1.OfferDecision_OFFER_DECISION_ACCEPTED {
		return ErrLeaseInvalid
	}
	if grant.GetLeaseId() == "" || grant.GetLeaseEpoch() == 0 || grant.GetJobId() != record.offer.GetJobId() || grant.GetAttemptId() != record.offer.GetAttemptId() {
		return ErrLeaseInvalid
	}
	if grant.GetExpiresAt() == nil || !grant.GetExpiresAt().AsTime().After(n.now()) || grant.GetExpiresAt().AsTime().After(record.offer.GetDeadlineAt().AsTime()) {
		return ErrLeaseInvalid
	}
	if n.activeLease != nil {
		if proto.Equal(n.activeLease, grant) {
			return nil
		}
		return ErrLeaseInvalid
	}
	n.activeLease = proto.Clone(grant).(*nodev1.LeaseGranted)
	return nil
}

// RenewLease 生成不包含正文的进度消息，且不能复活过期 Lease。
func (n *Node) RenewLease(generatedTokens uint32) (*nodev1.LeaseRenewal, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.activeLease == nil || !n.activeLease.GetExpiresAt().AsTime().After(n.now()) {
		return nil, ErrLeaseInvalid
	}
	return &nodev1.LeaseRenewal{
		LeaseId:     n.activeLease.GetLeaseId(),
		LeaseEpoch:  n.activeLease.GetLeaseEpoch(),
		RequestedAt: timestamppb.New(n.now()),
		Progress: &nodev1.RuntimeProgress{
			GeneratedTokens:     generatedTokens,
			ElapsedMilliseconds: 30,
		},
	}, nil
}

// Complete 执行当前 Lease，并丢弃过期、撤销或迟到结果。
func (n *Node) Complete(ctx context.Context) (*nodev1.ChatResult, error) {
	n.mu.Lock()
	if n.activeLease == nil {
		n.mu.Unlock()
		return nil, ErrLeaseInvalid
	}
	lease := proto.Clone(n.activeLease).(*nodev1.LeaseGranted)
	record := n.offers[lease.GetOfferId()]
	n.mu.Unlock()

	if n.scenario == ScenarioDisconnect {
		return nil, ErrDisconnected
	}
	if n.scenario == ScenarioTimeout {
		return nil, context.DeadlineExceeded
	}
	result, err := n.runtime.Run(ctx, record.offer)
	if err != nil {
		return nil, err
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.activeLease == nil || !proto.Equal(n.activeLease, lease) {
		return nil, ErrLeaseInvalid
	}
	if n.scenario == ScenarioLate || !lease.GetExpiresAt().AsTime().After(n.now()) || !record.offer.GetDeadlineAt().AsTime().After(n.now()) {
		n.activeLease = nil
		return nil, ErrLateResult
	}
	n.activeLease = nil
	return &nodev1.ChatResult{
		JobId:          lease.GetJobId(),
		AttemptId:      lease.GetAttemptId(),
		LeaseId:        lease.GetLeaseId(),
		LeaseEpoch:     lease.GetLeaseEpoch(),
		ModelReleaseId: record.offer.GetModelReleaseId(),
		FinishReason:   nodev1.FinishReason_FINISH_REASON_STOP,
		Content:        result.Content,
		Usage:          result.Usage,
		Runtime:        result.Runtime,
		CompletedAt:    timestamppb.New(n.now()),
	}, nil
}

func validateOffer(offer *nodev1.TaskOffer, now time.Time) error {
	if offer == nil || offer.GetOfferId() == "" || offer.GetJobId() == "" || offer.GetAttemptId() == "" || offer.GetModelReleaseId() == "" {
		return ErrInvalidOffer
	}
	if offer.GetDeadlineAt() == nil || !offer.GetDeadlineAt().AsTime().After(now) || offer.GetMaxOutputTokens() == 0 || offer.GetParameters() == nil {
		return ErrInvalidOffer
	}
	if len(offer.GetMessagesJsonUtf8()) == 0 || !json.Valid(offer.GetMessagesJsonUtf8()) || offer.GetPayloadDigest() == "" {
		return ErrInvalidOffer
	}
	if offer.GetParameters().GetMaxTokens() == 0 || offer.GetParameters().GetMaxTokens() > offer.GetMaxOutputTokens() {
		return ErrInvalidOffer
	}
	return nil
}

func mustMarshal(message proto.Message) []byte {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		panic(fmt.Sprintf("已校验协议消息无法序列化: %v", err))
	}
	return data
}
