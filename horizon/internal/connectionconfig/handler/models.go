package handler

type OnboardRequestPayload struct {
	Service               string `json:"service" binding:"required"`
	Default               bool   `json:"default"`
	ConnProtocol          string `json:"conn_protocol" binding:"required"`
	GrpcChannelAlgorithm  string `json:"grpc_channel_algorithm,omitempty"`
	PlainText             bool   `json:"plain_text,omitempty"`
	Deadline              int    `json:"deadline,omitempty"`
	Timeout               int    `json:"timeout,omitempty"`
	ChannelThreadPoolSize int    `json:"channel_thread_pool_size,omitempty"`
	BoundedQueueSize      int    `json:"bounded_queue_size,omitempty"`
	MaxConnectionPerHost  int    `json:"max_connection_per_host,omitempty"`
	IdleConnectionTimeout int    `json:"idle_connection_timeout,omitempty"`
	MaxIdleConnection     int    `json:"max_idle_connection,omitempty"`
	KeepAliveTime         int    `json:"keep_alive_time,omitempty"`
}

type OnboardRequest struct {
	Payload   OnboardRequestPayload `json:"payload" binding:"required"`
	CreatedBy string                `json:"created_by" binding:"required"`
}

type Message struct {
	Message string `json:"message"`
}

type Response struct {
	Error string  `json:"error"`
	Data  Message `json:"data"`
}

type GrpcConfig struct {
	Deadline              int    `json:"deadline"`
	PlainText             bool   `json:"plain_text"`
	GrpcChannelAlgorithm  string `json:"grpc_channel_algorithm"`
	ChannelThreadPoolSize int    `json:"channel_thread_pool_size"`
	BoundedQueueSize      int    `json:"bounded_queue_size"`
}

type HttpConfig struct {
	Timeout               int `json:"timeout"`
	MaxIdleConnection     int `json:"max_idle_connection"`
	MaxConnectionPerHost  int `json:"max_connection_per_host"`
	IdleConnectionTimeout int `json:"idle_connection_timeout"`
	KeepAliveTime         int `json:"keep_alive_time"`
}

type ConnConfig struct {
	GrpcConfig GrpcConfig `json:"grpc_config,omitempty"`
	HttpConfig HttpConfig `json:"http_config,omitempty"`
}

type Configs struct {
	Id           uint       `json:"id"`
	Default      bool       `json:"default"`
	Service      string     `json:"service"`
	ConnProtocol string     `json:"conn_protocol"`
	Config       ConnConfig `json:"config"`
	CreatedBy    string     `json:"created_by"`
	UpdatedBy    string     `json:"updated_by"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

type GetAllResponse struct {
	Data []Configs `json:"data"`
}

type EditRequestPayload struct {
	Id                    uint   `json:"id" binding:"required"`
	Service               string `json:"service" binding:"required"`
	Default               bool   `json:"default"`
	ConnProtocol          string `json:"conn_protocol" binding:"required"`
	GrpcChannelAlgorithm  string `json:"grpc_channel_algorithm,omitempty"`
	PlainText             bool   `json:"plain_text,omitempty"`
	Deadline              int    `json:"deadline,omitempty"`
	Timeout               int    `json:"timeout,omitempty"`
	ChannelThreadPoolSize int    `json:"channel_thread_pool_size,omitempty"`
	BoundedQueueSize      int    `json:"bounded_queue_size,omitempty"`
	MaxConnectionPerHost  int    `json:"max_connection_per_host,omitempty"`
	IdleConnectionTimeout int    `json:"idle_connection_timeout,omitempty"`
	MaxIdleConnection     int    `json:"max_idle_connection,omitempty"`
	KeepAliveTime         int    `json:"keep_alive_time,omitempty"`
}

type EditRequest struct {
	Payload   EditRequestPayload `json:"payload" binding:"required"`
	UpdatedBy string             `json:"updated_by" binding:"required"`
}
