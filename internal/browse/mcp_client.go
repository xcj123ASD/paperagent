package browse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// JSON-RPC 2.0 基础结构。
//
// MCP 协议（Model Context Protocol）建立在 JSON-RPC 2.0 之上。
// 本文件只实现最小可用子集：
//   - 请求/响应/错误结构
//   - 双向流（io.ReadWriteCloser）上的请求-响应匹配
//   - 超时与取消
//
// 不实现：
//   - 通知（notification，无 id 的消息）
//   - 批量请求
//   - 服务端主动发起的请求（MCP server 一般不发）

// rpcRequest 是 JSON-RPC 2.0 请求。
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse 是 JSON-RPC 2.0 响应。
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError 是 JSON-RPC 2.0 错误对象。
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("MCP 错误 %d: %s (%s)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("MCP 错误 %d: %s", e.Code, e.Message)
}

// rpcTransport 是双向流抽象，便于测试时用内存 pipe 替换。
//
// 生产实现：子进程的 stdin（写）/ stdout（读）。
// 测试实现：net.Pipe 或 bytes.Buffer 对。
type rpcTransport interface {
	Send(msg []byte) error
	Recv() ([]byte, error)
	Close() error
}

// mcpClient 是 MCP JSON-RPC 客户端，封装请求-响应匹配。
//
// 并发安全：多个 goroutine 可同时 Call，内部用 mutex + id→channel 映射。
type mcpClient struct {
	transport rpcTransport
	timeout   time.Duration

	mu      sync.Mutex
	pending map[int64]chan rpcResponse
	nextID  atomic.Int64

	closed   atomic.Bool
	closeErr error
}

// newMCPClient 构造客户端，启动接收循环。
//
// 接收循环：持续从 transport.Recv() 读消息，按 id 分发给对应等待者。
// 循环在 transport 关闭或出错时退出。
func newMCPClient(transport rpcTransport, timeout time.Duration) *mcpClient {
	c := &mcpClient{
		transport: transport,
		timeout:   timeout,
		pending:   make(map[int64]chan rpcResponse),
	}
	go c.recvLoop()
	return c
}

// Call 发送 JSON-RPC 请求并等待响应。
func (c *mcpClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("browse: MCP client 已关闭")
	}

	id := c.nextID.Add(1)
	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("browse: 序列化 params 失败: %w", err)
		}
		rawParams = b
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("browse: 序列化请求失败: %w", err)
	}

	respCh := make(chan rpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = respCh
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := c.transport.Send(body); err != nil {
		return nil, fmt.Errorf("browse: 发送请求失败: %w", err)
	}

	timeout := c.timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("browse: 调用 %q 被取消: %w", method, ctx.Err())
	case <-timer.C:
		return nil, fmt.Errorf("browse: 调用 %q 超时（%v）", method, timeout)
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// Close 关闭 transport，停止接收循环。幂等。
func (c *mcpClient) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return c.closeErr
	}
	c.closeErr = c.transport.Close()
	return c.closeErr
}

// recvLoop 是接收循环，持续读消息并按 id 分发。
func (c *mcpClient) recvLoop() {
	for {
		raw, err := c.transport.Recv()
		if err != nil {
			return
		}
		var resp rpcResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue // 跳过无法解析的消息，不阻塞其他请求
		}
		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		c.mu.Unlock()
		if ok {
			select {
			case ch <- resp:
			default:
				// 等待者已放弃（超时/取消），丢弃响应
			}
		}
	}
}
