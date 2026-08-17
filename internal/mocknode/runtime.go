package mocknode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"unicode/utf8"

	nodev1 "github.com/yeruyi1024/novagrid-node/protocol/node/v1"
)

// RuntimeResult 保存模拟执行的短生命周期结果，不负责持久化正文。
type RuntimeResult struct {
	Content string
	Usage   *nodev1.TokenUsage
	Runtime *nodev1.RuntimeUsage
}

// Runtime 定义模拟 Node 唯一允许调用的结构化文本运行边界。
type Runtime interface {
	Run(context.Context, *nodev1.TaskOffer) (*RuntimeResult, error)
}

// DeterministicRuntime 对同一输入始终返回相同结果，且不访问 GPU、网络或文件系统。
type DeterministicRuntime struct{}

// Run 仅在内存中计算摘要和 usage；调用方负责在请求结束后释放正文引用。
func (DeterministicRuntime) Run(ctx context.Context, offer *nodev1.TaskOffer) (*RuntimeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(offer.GetMessagesJsonUtf8())
	content := "novagrid-mock:" + hex.EncodeToString(digest[:8])
	promptTokens := uint32((utf8.RuneCount(offer.GetMessagesJsonUtf8()) + 3) / 4)
	completionTokens := uint32((utf8.RuneCountInString(content) + 3) / 4)
	return &RuntimeResult{
		Content: content,
		Usage: &nodev1.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Runtime: &nodev1.RuntimeUsage{
			PromptEvalMilliseconds: 12,
			GenerationMilliseconds: 48,
			TokensPerSecond:        20,
			GpuMilliseconds:        60,
			PeakVramBytes:          0,
		},
	}, nil
}
