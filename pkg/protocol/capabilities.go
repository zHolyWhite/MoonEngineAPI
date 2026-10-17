package protocol

const ContractVersion = 1

type AudioCapabilities struct {
	Formats     []string `json:"formats"`
	SampleRates []int    `json:"sampleRates"`
	Channels    []int    `json:"channels"`
}

type Limits struct {
	MaxBodyBytes  int64 `json:"maxBodyBytes"`
	MaxFrameBytes int   `json:"maxFrameBytes"`
}

type Capabilities struct {
	Contract  int               `json:"contract"`
	Endpoints []string          `json:"endpoints"`
	Audio     AudioCapabilities `json:"audio"`
	Limits    Limits            `json:"limits"`
}