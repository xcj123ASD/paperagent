package browse

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeTransport 是内存实现的 rpcTransport，用于测试。
//
// 行为：
//   - Send 把消息记录到 sent，并交给 handler 处理
//   - handler 返回的响应放进 recvQ，供 Recv 取出
//   - Close 后 Recv 返回错误
type fakeTransport struct {
	mu      sync.Mutex
	sent    [][]byte
	recvQ   chan []byte
	handler func(req rpcRequest) (rpcResponse, bool)
	closed  bool
}

func newFakeTransport(handler func(req rpcRequest) (rpcResponse, bool)) *fakeTransport {
	return &fakeTransport{
		recvQ:   make(chan []byte, 10),
		handler: handler,
	}
}

func (f *fakeTransport) Send(msg []byte) error {
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	closed := f.closed
	f.mu.Unlock()
	if closed {
		return errors.New("transport closed")
	}

	if f.handler == nil {
		return nil
	}
	var req rpcRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	resp, ok := f.handler(req)
	if !ok {
		return nil
	}
	b, _ := json.Marshal(resp)
	f.recvQ <- b
	return nil
}

func (f *fakeTransport) Recv() ([]byte, error) {
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if closed {
		return nil, errors.New("transport closed")
	}
	msg, ok := <-f.recvQ
	if !ok {
		return nil, errors.New("transport closed")
	}
	return msg, nil
}

func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.recvQ)
	}
	return nil
}

// ---- 测试 ----

func TestMCPClient_CallSuccess(t *testing.T) {
	ft := newFakeTransport(func(req rpcRequest) (rpcResponse, bool) {
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"ok":true}`),
		}, true
	})
	c := newMCPClient(ft, 5*time.Second)
	defer c.Close()

	res, err := c.Call(context.Background(), "test.method", map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("Call 失败: %v", err)
	}
	if string(res) != `{"ok":true}` {
		t.Errorf("result 应为 {\"ok\":true}，实际: %s", res)
	}
}

func TestMCPClient_CallErrorResponse(t *testing.T) {
	ft := newFakeTransport(func(req rpcRequest) (rpcResponse, bool) {
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "Method not found"},
		}, true
	})
	c := newMCPClient(ft, 5*time.Second)
	defer c.Close()

	_, err := c.Call(context.Background(), "bad.method", nil)
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !contains(err.Error(), "Method not found") {
		t.Errorf("错误信息应含 'Method not found'，实际: %v", err)
	}
}

func TestMCPClient_CallTimeout(t *testing.T) {
	// handler 不返回响应，模拟超时
	ft := newFakeTransport(nil)
	c := newMCPClient(ft, 100*time.Millisecond)
	defer c.Close()

	_, err := c.Call(context.Background(), "slow.method", nil)
	if err == nil {
		t.Fatal("应超时")
	}
	if !contains(err.Error(), "超时") {
		t.Errorf("错误信息应含'超时'，实际: %v", err)
	}
}

func TestMCPClient_CallCanceled(t *testing.T) {
	ft := newFakeTransport(nil)
	c := newMCPClient(ft, 5*time.Second)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := c.Call(ctx, "any.method", nil)
	if err == nil {
		t.Fatal("应返回取消错误")
	}
	if !errors.Is(err, context.Canceled) && !contains(err.Error(), "取消") {
		t.Errorf("错误信息应含'取消'或 context.Canceled，实际: %v", err)
	}
}

func TestMCPClient_ClosedCallFails(t *testing.T) {
	ft := newFakeTransport(nil)
	c := newMCPClient(ft, 5*time.Second)
	_ = c.Close()

	_, err := c.Call(context.Background(), "any.method", nil)
	if err == nil {
		t.Fatal("关闭后 Call 应报错")
	}
}

func TestMCPClient_CloseIdempotent(t *testing.T) {
	ft := newFakeTransport(nil)
	c := newMCPClient(ft, 5*time.Second)
	if err := c.Close(); err != nil {
		t.Fatalf("首次 Close 失败: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("重复 Close 应幂等，实际: %v", err)
	}
}

func TestMCPClient_ConcurrentCalls(t *testing.T) {
	ft := newFakeTransport(func(req rpcRequest) (rpcResponse, bool) {
		// 按 id 回显，便于验证请求-响应匹配
		result, _ := json.Marshal(map[string]int64{"id": req.ID})
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}, true
	})
	c := newMCPClient(ft, 5*time.Second)
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := c.Call(context.Background(), "echo", nil)
			if err != nil {
				t.Errorf("并发 Call 失败: %v", err)
				return
			}
			var got map[string]int64
			if err := json.Unmarshal(res, &got); err != nil {
				t.Errorf("反序列化失败: %v", err)
				return
			}
			if got["id"] <= 0 {
				t.Errorf("响应 id 应 > 0，实际: %v", got["id"])
			}
		}()
	}
	wg.Wait()
}

// contains 是 strings.Contains 的别名，避免额外 import。
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
