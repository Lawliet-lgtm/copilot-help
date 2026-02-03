package commander

import "linuxFileWatcher/internal/model"

// Dispatcher 指令分发器接口
// 它的职责是接收原始指令，解析后路由给具体的 Executor
type Dispatcher interface {
	// Dispatch 分发指令 (非阻塞)
	Dispatch(cmd model.CommandPayload) error
}
