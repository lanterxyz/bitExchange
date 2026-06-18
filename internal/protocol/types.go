package protocol

type Envelope struct {
    Kind string `json:"kind"`
}

type TextMessage struct {
    Kind       string `json:"kind"`
    FromDevice string `json:"from_device"`
    Body       string `json:"body"`
}

type FileHeader struct {
    Kind       string `json:"kind"`
    FromDevice string `json:"from_device"`
    FileName   string `json:"file_name"`
    FileSize   int64  `json:"file_size"`
}
