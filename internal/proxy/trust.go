package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Trust installs the local CA into the OS trust store.
func Trust() error {
	_, created, err := LoadOrCreateCA()
	if err != nil {
		return fmt.Errorf("loading CA: %w", err)
	}

	certPath, err := CertPath()
	if err != nil {
		return err
	}

	if !created {
		fmt.Printf("Valla CA already exists at %s\n", certPath)
	} else {
		fmt.Printf("Generated new Valla CA at %s\n", certPath)
	}

	if err := installTrust(certPath); err != nil {
		return err
	}

	fmt.Println("\nSetting up local DNS for *.test → 127.0.0.1…")
	if err := SetupDNS("test"); err != nil {
		// Non-fatal: print warning and continue.
		fmt.Fprintf(os.Stderr, "WARNING: DNS setup failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "You can set it up manually — see the output above.\n")
	}
	return nil
}

// installTrust delegates to the platform-specific trust store command.
func installTrust(certPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return installDarwin(certPath)
	case "linux":
		return installLinux(certPath)
	case "windows":
		return installWindows(certPath)
	default:
		printManualInstall(certPath)
		return nil
	}
}

func installDarwin(certPath string) error {
	args := []string{
		"security", "add-trusted-cert",
		"-d",
		"-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain",
		certPath,
	}
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		printManualInstall(certPath)
		return fmt.Errorf("trust store install failed (you may need to run with sudo): %w", err)
	}
	fmt.Println("\u2713  Valla CA trusted system-wide. Browsers will show a green padlock.")
	return nil
}

func installLinux(certPath string) error {
	// Try NSS databases (Chrome/Firefox).
	nssDBs := []string{
		os.ExpandEnv("$HOME/.pki/nssdb"),
		os.ExpandEnv("$HOME/snap/chromium/current/.pki/nssdb"),
	}
	added := false
	if _, err := exec.LookPath("certutil"); err == nil {
		for _, db := range nssDBs {
			if _, err := os.Stat(db); err != nil {
				continue
			}
			cmd := exec.Command("certutil", "-A", "-n", "Valla Local CA", "-t", "CT,,", "-i", certPath, "-d", "sql:"+db)
			if err := cmd.Run(); err == nil {
				added = true
			}
		}
	}
	// System-wide (Debian/Ubuntu).
	destDir := "/usr/local/share/ca-certificates"
	if _, err := os.Stat(destDir); err == nil {
		data, rerr := os.ReadFile(certPath)
		if rerr == nil {
			dest := destDir + "/valla-local-ca.crt"
			if werr := os.WriteFile(dest, data, 0o644); werr == nil {
				_ = exec.Command("update-ca-certificates").Run()
				added = true
			}
		}
	}
	if added {
		fmt.Println("\u2713  Valla CA added to system and NSS trust stores.")
		return nil
	}
	printManualInstall(certPath)
	return nil
}

func installWindows(certPath string) error {
	cmd := exec.Command("certmgr", "/add", certPath, "/s", "Root")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printManualInstall(certPath)
		return fmt.Errorf("trust store install failed: %w", err)
	}
	fmt.Println("\u2713  Valla CA trusted system-wide.")
	return nil
}

func printManualInstall(certPath string) {
	fmt.Printf("\nCould not install the CA automatically on this platform.\nTo trust the Valla CA manually, add the following file to your browser or OS trust store:\n\n  %s\n\n", certPath)
}
