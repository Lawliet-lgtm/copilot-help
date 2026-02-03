package model

// ==========================================
// 主机唯一编码查询接口 - 数据模型
// ==========================================

// ==========================================
// 请求结构体
// ==========================================

// GetComputerClientIDRequest 主机唯一编码查询请求
type GetComputerClientIDRequest struct {
	// MAC地址，字符串，最长18字节
	MAC string `json:"mac" binding:"required,max=18"`
	// 硬件标识，字符串，最长102字节
	Hdcode string `json:"hdcode" binding:"required,max=1024"`
	// 厂商编码，字符串，固定3位
	Vendor string `json:"vendor" binding:"required,len=3"`
}

// ==========================================
// 响应结构体
// ==========================================

// GetComputerClientIDResponseData 主机唯一编码查询响应数据
type GetComputerClientIDResponseData struct {
	// 设备ID，字符串，最长为128字节
	DeviceID string `json:"device_id"`
}

// GetComputerClientIDResponse 主机唯一编码查询响应
type GetComputerClientIDResponse struct {
	// 响应类型，数值型，取值为0(成功)、1(失败)，最长128字节
	Type int `json:"type"`
	// 返回消息内容，最长为64字节的字符串
	Message string `json:"message" binding:"max=64"`
	// 响应数据，包含设备ID
	Data GetComputerClientIDResponseData `json:"data"`
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewGetComputerClientIDRequest 创建新的主机唯一编码查询请求
func NewGetComputerClientIDRequest(mac, hwidcode, vendor string) *GetComputerClientIDRequest {
	return &GetComputerClientIDRequest{
		MAC:    mac,
		Hdcode: hwidcode,
		Vendor: vendor,
	}
}

// NewGetComputerClientIDResponse 创建新的主机唯一编码查询响应
func NewGetComputerClientIDResponse(success bool, message, deviceID string) *GetComputerClientIDResponse {
	responseType := 0
	if !success {
		responseType = 1
	}
	return &GetComputerClientIDResponse{
		Type:    responseType,
		Message: message,
		Data: GetComputerClientIDResponseData{
			DeviceID: deviceID,
		},
	}
}

// ==========================================
// 注册接口 - 数据模型
// ==========================================

// InterfaceConfig 设备配置信息
type InterfaceConfig struct {
	// IP地址
	IP string `json:"ip"`
	// 子网掩码
	Netmask string `json:"netmask"`
	// 网关地址
	Gateway string `json:"gateway"`
	// MAC地址
	MAC string `json:"mac"`
}

// CPUInfo CPU信息
type CPUInfo struct {
	// 物理CPU ID
	PhysicalID string `json:"physical_id"`
	// CPU核心数
	Core int `json:"core"`

	// CPU频率
	Clock string `json:"clock"`
}

// DiskInfo 磁盘信息
type DiskInfo struct {
	// 磁盘大小
	Size int64 `json:"size"`
	// 磁盘序列号
	Serial string `json:"serial"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	// 软件版本信息，32字节的字符串，前八位为年月日
	SoftVersion string `json:"soft_version" binding:"required,max=32"`
	// 设备配置信息，表示设备网络配置
	Interface []InterfaceConfig `json:"interface"`
	// 内存总数，表示设备的内存大小，MB
	MemTotal int `json:"mem_total" binding:"max=128"`
	// CPU信息，包含物理CPU ID、CPU核心数、CPU型号、CPU频率等
	CPUInfo []CPUInfo `json:"cpu_info"`
	// 磁盘信息，表示磁盘类型、size和model
	DiskInfo []DiskInfo `json:"disk_info"`
	// 部门id，字符串，可选，≤128字节
	OrgID string `json:"org_id" binding:"max=128"`
	// 部门编码，同上
	OrgCode string `json:"org_code" binding:"max=128"`
	// 使用人id，同上
	UserID string `json:"user_id" binding:"max=128"`
	// 使用人编号，同上
	UserCode string `json:"user_code" binding:"max=128"`
	// 人员姓名，同上
	UserName string `json:"user_name" binding:"max=128"`
	// 计算机名称，同上
	HostName string `json:"host_name" binding:"max=128"`
	// 操作系统，同上
	OS string `json:"os" binding:"max=128"`
	// CPU架构，同上
	Arch string `json:"arch" binding:"max=128"`
	// 备注信息，≤128字节的字符串
	Memo string `json:"memo" binding:"max=128"`
	// 扩展字段集合，可选，对象类型
	ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	// 响应类型，数值型，取值为0(成功)、1(失败)，最长128字节
	Type int `json:"type"`
	// 返回消息内容，最长为64字节的字符串
	Message string `json:"message" binding:"max=64"`
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewInterfaceConfig 创建新的设备配置信息
func NewInterfaceConfig(ip, netmask, gateway, mac string, manage bool) *InterfaceConfig {
	return &InterfaceConfig{
		IP:      ip,
		Netmask: netmask,
		Gateway: gateway,
		MAC:     mac,
	}
}

// NewCPUInfo 创建新的CPU信息
func NewCPUInfo(physicalID string, core int, model, clock string) *CPUInfo {
	return &CPUInfo{
		PhysicalID: physicalID,
		Core:       core,
		Clock:      clock,
	}
}

// NewDiskInfo 创建新的磁盘信息
func NewDiskInfo(model string, size int64, serial string) *DiskInfo {
	return &DiskInfo{
		Size:   size,
		Serial: serial,
	}
}

// NewRegisterRequest 创建新的注册请求
func NewRegisterRequest(softVersion string) *RegisterRequest {
	return &RegisterRequest{
		SoftVersion:    softVersion,
		Interface:      make([]InterfaceConfig, 0),
		MemTotal:       0,
		CPUInfo:        make([]CPUInfo, 0),
		DiskInfo:       make([]DiskInfo, 0),
		OrgID:          "",
		OrgCode:        "",
		UserID:         "",
		UserCode:       "",
		UserName:       "",
		HostName:       "",
		OS:             "",
		Arch:           "",
		Memo:           "",
		ExtendedFields: make(map[string]interface{}),
	}
}

// NewRegisterResponse 创建新的注册响应
func NewRegisterResponse(success bool, message string) *RegisterResponse {
	responseType := 0
	if !success {
		responseType = 1
	}
	return &RegisterResponse{
		Type:    responseType,
		Message: message,
	}
}

// ==========================================
// 认证接口 - 数据模型
// ==========================================

// AuthLoginResponse 认证响应
type AuthLoginResponse struct {
	// 认证状态，数值类型，0（成功）、1（失败），最长128字节
	Type int `json:"type"`
	// 返回消息内容，最长128字节的字符串
	Message string `json:"message" binding:"max=128"`
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewAuthLoginResponse 创建新的认证响应
func NewAuthLoginResponse(success bool, message string) *AuthLoginResponse {
	responseType := 0
	if !success {
		responseType = 1
	}
	return &AuthLoginResponse{
		Type:    responseType,
		Message: message,
	}
}

// ==========================================
// 注销接口 - 数据模型
// ==========================================

// RegCancelRequest 注销请求
type RegCancelRequest struct {
	// 注销请求body无需信息，保持空结构体
}

// RegCancelResponse 注销响应
type RegCancelResponse struct {
	// 与认证接口响应参数一致
	Type int `json:"type"`
	// 返回消息内容，最长128字节的字符串
	Message string `json:"message" binding:"max=128"`
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewRegCancelRequest 创建新的注销请求
func NewRegCancelRequest() *RegCancelRequest {
	return &RegCancelRequest{}
}

// NewRegCancelResponse 创建新的注销响应
func NewRegCancelResponse(success bool, message string) *RegCancelResponse {
	responseType := 0
	if !success {
		responseType = 1
	}
	return &RegCancelResponse{
		Type:    responseType,
		Message: message,
	}
}
