package pairing

type CodeRecord struct {
    Code        string `json:"code"`
    DeviceID    string `json:"device_id"`
    DeviceName  string `json:"device_name"`
    Fingerprint string `json:"fingerprint"`
    ListenAddr  string `json:"listen_addr"`
}
