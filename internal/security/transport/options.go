package transport

import "net/http"

// RequestOption 定义一个函数类型，用于修改 http.Request
type RequestOption func(*http.Request)

// ===========================
// 常用选项定义
// ===========================

// WithHeader 添加或覆盖自定义 Header
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithoutHeader 强制移除某个 Header (如果默认逻辑里加了，你想去掉)
func WithoutHeader(key string) RequestOption {
	return func(req *http.Request) {
		req.Header.Del(key)
	}
}

// WithContentType 快捷设置 Content-Type
func WithContentType(contentType string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Content-Type", contentType)
	}
}

// WithGzipRequest 标记这个请求体是 Gzip 压缩的
// 注意：这只是设置 Header，实际压缩逻辑需要在 Client 内部配合处理
// 这里我们做一个标记 Header，Client 内部读到这个标记再执行压缩
func WithGzipRequest() RequestOption {
	return func(req *http.Request) {
		req.Header.Set("X-Client-Action", "gzip")  // 内部标记
		req.Header.Set("Content-Encoding", "gzip") // 真实的 HTTP 头
	}
}
