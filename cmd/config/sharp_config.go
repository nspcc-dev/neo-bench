package main

type SharpConfig struct {
	ApplicationConfiguration ApplicationConfiguration `yaml:"ApplicationConfiguration"`
	ProtocolConfiguration    ProtocolConfiguration    `yaml:"ProtocolConfiguration"`
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
	Path   string `yaml:"Path"`
}

type P2P struct {
	Port                     uint16 `yaml:"Port"`
	WsPort                   uint16 `yaml:"WsPort"`
	MaxConnections           int    `yaml:"MaxConnections"`
	MaxConnectionsPerAddress int    `yaml:"MaxConnectionsPerAddress"`
}

type UnlockWallet struct {
	Path     string `yaml:"Path"`
	Password string `yaml:"Password"`
	IsActive bool   `yaml:"IsActive"`
}

type ProtocolConfiguration struct {
	Network                     uint32         `yaml:"Network"`
	MaxTransactionsPerBlock     int32          `yaml:"MaxTransactionsPerBlock"`
	MillisecondsPerBlock        int            `yaml:"MillisecondsPerBlock"`
	MaxValidUntilBlockIncrement int            `yaml:"MaxValidUntilBlockIncrement"`
	ValidatorsCount             int            `yaml:"ValidatorsCount"`
	MemoryPoolMaxTransactions   int            `yaml:"MemoryPoolMaxTransactions"`
	StandbyCommittee            []string       `yaml:"StandbyCommittee"`
	SeedList                    []string       `yaml:"SeedList"`
	Hardforks                   map[string]int `yaml:"Hardforks,omitempty" json:"Hardforks,omitempty"`
}

type SharpTemplate struct {
	ApplicationConfiguration ApplicationConfiguration `yaml:"ApplicationConfiguration"`
	ProtocolConfiguration    ProtocolConfiguration    `yaml:"ProtocolConfiguration"`
}
