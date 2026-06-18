package transfer

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

func SaveIncomingFile(dir string, fileName string, reader io.Reader) (string, error) {
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return "", err
    }

    targetPath := filepath.Join(dir, fileName)
    targetPath = uniquePath(targetPath)

    file, err := os.Create(targetPath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    if _, err := io.Copy(file, reader); err != nil {
        return "", err
    }

    return targetPath, nil
}

func uniquePath(path string) string {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return path
    }

    ext := filepath.Ext(path)
    base := strings.TrimSuffix(filepath.Base(path), ext)
    dir := filepath.Dir(path)

    for i := 1; ; i++ {
        candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
        if _, err := os.Stat(candidate); os.IsNotExist(err) {
            return candidate
        }
    }
}
