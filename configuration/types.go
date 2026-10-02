package config

type Configuration struct {
	Config    ConfigStruct
	Worker    Worker
	Shuffling bool
	LogLevel  string
}

type Worker struct {
	Download Download
	Upload   Upload
	Threads  int
	Vpn      bool
}

type ConfigStruct struct {
	LocalPort       int
	AddressPort     string
	UserId          string
	Protocol        string
	WsHeaderHost    string // XHTTP Host (formerly WebSocket)
	WsHeaderPath    string // XHTTP Path (formerly WebSocket)
	Sni             string
	RandomizeSni    bool
	Network         string
	Tls             bool
	Security        string
	XhttpMode       string
	FrontingTimeout float64 // seconds
	NTries          int
	Writer          string
	TestUrl         string
	TestBool
}

type TestBool struct {
	DoUploadTest   bool
	DoFrontingTest bool
}

type Download struct {
	MinDlSpeed   float64 // kilobytes per second
	MaxDlTime    float64 // seconds
	MaxDlLatency float64 // seconds

}

type Upload struct {
	MinUlSpeed   float64 // kilobytes per second
	MaxUlTime    float64 // seconds
	MaxUlLatency float64 // seconds

}
