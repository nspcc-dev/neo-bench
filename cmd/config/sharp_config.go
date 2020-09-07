package main

type SharpConfig struct {
	ApplicationConfiguration ApplicationConfiguration
}

type ApplicationConfiguration struct {
	Logger       Logger       `yaml:"Logger"`
	Storage      Storage      `yaml:"Storage"`
	P2P          P2P          `yaml:"P2P"`
	UnlockWallet UnlockWallet `yaml:"UnlockWallet"`
	PluginURL    string       `yaml:"PluginURL"`
}

type Logger struct {
	Path          string `yaml:"Path"`
	ConsoleOutput bool   `yaml:"ConsoleOutput"`
	Active        bool   `yaml:"Active"`
}

type Storage struct {
	Engine string `yaml:"Engine"`
}

type P2P struct {
	Port                     uint16 `yaml:"Port"`
	WsPort                   uint16 `yaml:"WsPort"`
	MaxConnections           int    `yaml:"MaxConnections"`
	MaxConnectionsPerAddress int    `yaml:"MaxConnectionsPerAddress"`
}

type UnlockWallet struct {
	Path           string `yaml:"Path"`
	Password       string `yaml:"Password"`
	StartConsensus bool   `yaml:"StartConsensus"`
	IsActive       bool   `yaml:"IsActive"`
}

type SharpProtocol struct {
	ProtocolConfiguration ProtocolConfiguration
}

type ProtocolConfiguration struct {
	Magic                     uint32   `yaml:"Magic"`
	MillisecondsPerBlock      int      `yaml:"MillisecondsPerBlock"`
	ValidatorsCount           int      `yaml:"ValidatorsCount"`
	MemoryPoolMaxTransactions int      `yaml:"MemoryPoolMaxTransactions"`
	StandbyCommittee          []string `yaml:"StandbyCommittee"`
	SeedList                  []string `yaml:"SeedList"`
}

type SharpTemplate struct {
	ApplicationConfiguration ApplicationConfiguration `yaml:"ApplicationConfiguration"`
	ProtocolConfiguration    ProtocolConfiguration    `yaml:"ProtocolConfiguration"`
}
