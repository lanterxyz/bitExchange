package config

import (
    "os"
    "path/filepath"
)

type Paths struct {
    RootDir          string
    ChatFile         string
    ReceivedFilesDir string
}

func EnsureDefaultPaths(root string) (Paths, error) {
    paths := Paths{
        RootDir:          root,
        ChatFile:         filepath.Join(root, "chat.txt"),
        ReceivedFilesDir: filepath.Join(root, "receivedFiles"),
    }

    if err := os.MkdirAll(paths.ReceivedFilesDir, 0o755); err != nil {
        return Paths{}, err
    }

    if err := os.MkdirAll(paths.RootDir, 0o755); err != nil {
        return Paths{}, err
    }

    file, err := os.OpenFile(paths.ChatFile, os.O_CREATE, 0o644)
    if err != nil {
        return Paths{}, err
    }
    _ = file.Close()

    return paths, nil
}
