package app

import (
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"

    "bitExchange/internal/config"
    icrypto "bitExchange/internal/crypto"
)

func Run(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("expected subcommand")
    }

    switch args[0] {
    case "init":
        fs := flag.NewFlagSet("init", flag.ContinueOnError)
        root := fs.String("root", filepath.Join(os.Getenv("HOME"), "bitExchange"), "save root")
        deviceName := fs.String("device-name", "device", "device name")
        if err := fs.Parse(args[1:]); err != nil {
            return err
        }

        paths, err := config.EnsureDefaultPaths(*root)
        if err != nil {
            return err
        }

        identity, err := icrypto.GenerateIdentity(*deviceName)
        if err != nil {
            return err
        }

        return config.SaveDeviceConfig(filepath.Join(paths.RootDir, "device.json"), config.DeviceConfig{Identity: identity})
    case "online":
        fs := flag.NewFlagSet("online", flag.ContinueOnError)
        root := fs.String("root", filepath.Join(os.Getenv("HOME"), "bitExchange"), "save root")
        serverURL := fs.String("server", "http://localhost:8080", "signaling server URL")
        if err := fs.Parse(args[1:]); err != nil {
            return err
        }

        cfg, err := config.LoadClientConfig(filepath.Join(*root, "config.json"))
        if err != nil {
            return fmt.Errorf("load config: %w", err)
        }

        cfg.ServerURL = *serverURL
        if err := config.SaveClientConfig(filepath.Join(*root, "config.json"), cfg); err != nil {
            return fmt.Errorf("save config: %w", err)
        }

        fmt.Printf("online: server=%s\n", *serverURL)
        fmt.Println("signaling client will register with heartbeat (not yet connected)")
        return nil

    case "peers":
        fs := flag.NewFlagSet("peers", flag.ContinueOnError)
        root := fs.String("root", filepath.Join(os.Getenv("HOME"), "bitExchange"), "save root")
        serverURL := fs.String("server", "http://localhost:8080", "server URL")
        if err := fs.Parse(args[1:]); err != nil {
            return err
        }

        cfg, err := config.LoadClientConfig(filepath.Join(*root, "config.json"))
        if err != nil {
            return fmt.Errorf("load config: %w", err)
        }

        resp, err := http.Get(*serverURL + "/signaling/candidates?device_id=" + cfg.ServerURL)
        if err != nil {
            fmt.Printf("peers: cannot reach server %s: %v\n", *serverURL, err)
            return nil
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusNotFound {
            fmt.Println("peers: no online devices found")
            return nil
        }

        body, _ := io.ReadAll(resp.Body)
        fmt.Printf("peers: %s\n", string(body))
        return nil
    default:
        return fmt.Errorf("unknown subcommand: %s", args[0])
    }
}
