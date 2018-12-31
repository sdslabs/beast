package api

type HTTPPlainResp struct {
	Message string `json:"message" example:"Messsage in response to your request"`
}

type HTTPAuthorizeResp struct {
	Token   string `json:"token" example:"YOUR_AUTHENTICATION_TOKEN"`
	Message string `json:"message" example:"Response message"`
}

type AuthorizationChallengeResp struct {
	Challenge string `json:"challenge" example:"Challenge String"`
	Message   string `json:"message" example:"Response message"`
}

type AvailableImagesResp struct {
	Message string   `json:"message" example:"Available Base images."`
	Images  []string `json:"images" example:"['ubuntu16.04', 'ubuntu18.04']"`
}

type PortsInUseResp struct {
	MinPortValue uint32   `json:"port_min_value" example:"10000"`
	MaxPortValue uint32   `json:"port_max_value" example:"20000"`
	PortsInUse   []uint32 `json:"ports_in_use" example:"[10000, 100001, 100003, 10010]"`
}
