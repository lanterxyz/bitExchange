package history

import "os"

func AppendMessage(path string, line string) error {
    file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString(line + "\n")
    return err
}
