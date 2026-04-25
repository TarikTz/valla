package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// validLabel matches safe DNS label characters only (letters, digits, hyphens).
// Rejects dots, slashes, newlines, and any other character that could inject
// content into dnsmasq config files or construct path-traversal filenames.
var validLabel = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)

// validateLabel returns an error if s is not a safe DNS label.
func validateLabel(kind, s string) error {
	if !validLabel.MatchString(s) {
		return fmt.Errorf("invalid %s %q: must contain only letters, digits, and hyphens (no dots, slashes, or whitespace)", kind, s)
	}
	return nil
}

// SetupDNS configures wildcard DNS resolution for *.{tld} → 127.0.0.1.
//
//   - macOS:   Homebrew dnsmasq + /etc/resolver/{tld}
//   - Linux:   NetworkManager dnsmasq plugin, or systemd-resolved + dnsmasq:5353
//   - Windows: prints manual instructions (recommend --domain lvh.me)
func SetupDNS(tld string) error {
	if err := validateLabel("domain/tld", tld); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return setupDNSDarwin(tld)
	case "linux":
		return setupDNSLinux(tld)
	default:
		printManualDNS(tld)
		return nil
	}
}

func setupDNSDarwin(tld string) error {
	if err := ensureDnsmasq(); err != nil {
		printManualDNS(tld)
		return err
	}
	if err := configureDnsmasq(tld); err != nil {
		return err
	}
	if err := writeResolver(tld); err != nil {
		return err
	}
	return restartDnsmasq()
}

func ensureDnsmasq() error {
	if _, err := exec.LookPath("dnsmasq"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("dnsmasq not found and Homebrew is not available — install dnsmasq manually")
	}
	fmt.Println("Installing dnsmasq via Homebrew…")
	cmd := exec.Command("brew", "install", "dnsmasq")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func configureDnsmasq(tld string) error {
	prefix, err := brewPrefix()
	if err != nil {
		return err
	}
	confPath := prefix + "/etc/dnsmasq.conf"
	rule := fmt.Sprintf("address=/.%s/127.0.0.1", tld)

	data, _ := os.ReadFile(confPath)
	if strings.Contains(string(data), rule) {
		fmt.Printf("✓  dnsmasq already configured for .%s\n", tld)
		return nil
	}

	f, err := os.OpenFile(confPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("writing dnsmasq config (%s): %w", confPath, err)
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n# Added by valla trust\naddress=/.%s/127.0.0.1\n", tld)
	if err != nil {
		return fmt.Errorf("writing dnsmasq rule: %w", err)
	}
	fmt.Printf("✓  Added *.%s → 127.0.0.1 rule to dnsmasq\n", tld)
	return nil
}

func writeResolver(tld string) error {
	resolverDir := "/etc/resolver"
	resolverFile := resolverDir + "/" + tld

	if data, err := os.ReadFile(resolverFile); err == nil && strings.Contains(string(data), "127.0.0.1") {
		fmt.Printf("✓  /etc/resolver/%s already configured\n", tld)
		return nil
	}

	mkdirCmd := exec.Command("sudo", "mkdir", "-p", resolverDir)
	mkdirCmd.Stdout = os.Stdout
	mkdirCmd.Stderr = os.Stderr
	mkdirCmd.Stdin = os.Stdin
	if err := mkdirCmd.Run(); err != nil {
		return fmt.Errorf("creating %s: %w", resolverDir, err)
	}

	teeCmd := exec.Command("sudo", "tee", resolverFile)
	teeCmd.Stdin = strings.NewReader("nameserver 127.0.0.1\n")
	teeCmd.Stdout = os.Stdout
	teeCmd.Stderr = os.Stderr
	if err := teeCmd.Run(); err != nil {
		return fmt.Errorf("writing %s: %w", resolverFile, err)
	}
	fmt.Printf("✓  Created /etc/resolver/%s\n", tld)
	return nil
}

func restartDnsmasq() error {
	// Use launchctl directly instead of "sudo brew services" to avoid running
	// Homebrew (and its arbitrary Ruby/shell hooks) as root.
	cmd := exec.Command("sudo", "launchctl", "kickstart", "-k", "system/homebrew.mxcl.dnsmasq")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		// Service not yet loaded — load the plist first, then kickstart.
		if plist, perr := dnsmasqPlist(); perr == nil {
			_ = exec.Command("sudo", "launchctl", "load", "-w", plist).Run()
			cmd2 := exec.Command("sudo", "launchctl", "kickstart", "-k", "system/homebrew.mxcl.dnsmasq")
			cmd2.Stdout = os.Stdout
			cmd2.Stderr = os.Stderr
			cmd2.Stdin = os.Stdin
			if err2 := cmd2.Run(); err2 != nil {
				return fmt.Errorf("restarting dnsmasq: %w", err2)
			}
		} else {
			return fmt.Errorf("restarting dnsmasq: %w", err)
		}
	}
	fmt.Println("✓  dnsmasq restarted")
	return nil
}

// dnsmasqPlist returns the path to the Homebrew dnsmasq launchd plist.
func dnsmasqPlist() (string, error) {
	out, err := exec.Command("brew", "--prefix", "dnsmasq").Output()
	if err != nil {
		return "", fmt.Errorf("brew --prefix dnsmasq: %w", err)
	}
	return strings.TrimSpace(string(out)) + "/homebrew.mxcl.dnsmasq.plist", nil
}

func brewPrefix() (string, error) {
	out, err := exec.Command("brew", "--prefix").Output()
	if err != nil {
		return "", fmt.Errorf("brew --prefix: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ── Linux ─────────────────────────────────────────────────────────────────────

func setupDNSLinux(tld string) error {
	// Strategy 1: NetworkManager + dnsmasq plugin (Ubuntu, Fedora, Mint desktop)
	nmDir := "/etc/NetworkManager/dnsmasq.d"
	if _, err := os.Stat(nmDir); err == nil {
		if err := setupDNSLinuxNM(tld, nmDir); err == nil {
			return nil
		}
	}
	// Strategy 2: systemd-resolved is active → dnsmasq on :5353 + resolved drop-in
	if isSystemdResolvedActive() {
		if err := setupDNSLinuxResolved(tld); err == nil {
			return nil
		}
	}
	// Strategy 3: plain dnsmasq + prepend to resolv.conf
	if err := setupDNSLinuxPlain(tld); err == nil {
		return nil
	}
	printManualDNS(tld)
	return nil
}

func setupDNSLinuxNM(tld, nmDir string) error {
	confPath := nmDir + "/valla-" + tld + ".conf"
	rule := "address=/." + tld + "/127.0.0.1"
	if data, _ := os.ReadFile(confPath); strings.Contains(string(data), rule) {
		fmt.Printf("✓  NetworkManager dnsmasq already configured for .%s\n", tld)
		return nil
	}
	cmd := exec.Command("sudo", "tee", confPath)
	cmd.Stdin = strings.NewReader("# Added by valla trust\n" + rule + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("writing NM dnsmasq config: %w", err)
	}
	fmt.Printf("✓  Added *.%s → 127.0.0.1 to NetworkManager dnsmasq\n", tld)
	reload := exec.Command("sudo", "systemctl", "reload", "NetworkManager")
	reload.Stdout = os.Stdout
	reload.Stderr = os.Stderr
	if err := reload.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: could not reload NetworkManager: %v\n", err)
	} else {
		fmt.Println("✓  NetworkManager reloaded")
	}
	return nil
}

func isSystemdResolvedActive() bool {
	return exec.Command("systemctl", "is-active", "--quiet", "systemd-resolved").Run() == nil
}

func setupDNSLinuxResolved(tld string) error {
	if err := ensureDnsmasqLinux(); err != nil {
		return err
	}
	// Configure dnsmasq to answer on 127.0.0.1:5353 (avoids conflict with resolved stub on :53)
	dnsmasqConfDir := "/etc/dnsmasq.d"
	if _, err := os.Stat(dnsmasqConfDir); err != nil {
		dnsmasqConfDir = "/etc" // fallback to /etc/dnsmasq.conf
	}
	dnsmasqConf := dnsmasqConfDir + "/valla-" + tld + ".conf"
	if dnsmasqConfDir == "/etc" {
		dnsmasqConf = "/etc/dnsmasq.conf"
	}
	rule := fmt.Sprintf("address=/.%s/127.0.0.1", tld)
	data, _ := os.ReadFile(dnsmasqConf)
	if !strings.Contains(string(data), rule) {
		content := fmt.Sprintf("# Added by valla trust\n%s\nport=5353\nbind-interfaces\nlisten-address=127.0.0.1\n", rule)
		cmd := exec.Command("sudo", "tee", dnsmasqConf)
		cmd.Stdin = strings.NewReader(content)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("writing dnsmasq config: %w", err)
		}
		fmt.Printf("✓  Added *.%s → 127.0.0.1 to dnsmasq (port 5353)\n", tld)
	}
	// systemd-resolved drop-in: route .test queries to our dnsmasq instance
	dropinDir := "/etc/systemd/resolved.conf.d"
	exec.Command("sudo", "mkdir", "-p", dropinDir).Run()
	dropin := fmt.Sprintf("[Resolve]\nDNS=127.0.0.1#5353\nDomains=~%s\n", tld)
	dropinCmd := exec.Command("sudo", "tee", dropinDir+"/valla-"+tld+".conf")
	dropinCmd.Stdin = strings.NewReader(dropin)
	dropinCmd.Stdout = os.Stdout
	dropinCmd.Stderr = os.Stderr
	if err := dropinCmd.Run(); err != nil {
		return fmt.Errorf("writing resolved drop-in: %w", err)
	}
	exec.Command("sudo", "systemctl", "restart", "dnsmasq").Run()
	exec.Command("sudo", "systemctl", "restart", "systemd-resolved").Run()
	fmt.Printf("✓  Configured dnsmasq:5353 + systemd-resolved split-DNS for .%s\n", tld)
	return nil
}

func setupDNSLinuxPlain(tld string) error {
	if err := ensureDnsmasqLinux(); err != nil {
		return err
	}
	rule := fmt.Sprintf("address=/.%s/127.0.0.1", tld)
	data, _ := os.ReadFile("/etc/dnsmasq.conf")
	if !strings.Contains(string(data), rule) {
		cmd := exec.Command("sudo", "tee", "-a", "/etc/dnsmasq.conf")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("\n# Added by valla trust\n%s\n", rule))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("appending to dnsmasq.conf: %w", err)
		}
		fmt.Printf("✓  Added *.%s → 127.0.0.1 to dnsmasq\n", tld)
	}
	exec.Command("sudo", "systemctl", "restart", "dnsmasq").Run()
	return nil
}

func ensureDnsmasqLinux() error {
	if _, err := exec.LookPath("dnsmasq"); err == nil {
		return nil
	}
	for _, pm := range [][]string{
		{"apt-get", "install", "-y", "dnsmasq"},
		{"dnf", "install", "-y", "dnsmasq"},
		{"yum", "install", "-y", "dnsmasq"},
		{"pacman", "-S", "--noconfirm", "dnsmasq"},
	} {
		if _, err := exec.LookPath(pm[0]); err == nil {
			fmt.Printf("Installing dnsmasq via %s…\n", pm[0])
			cmd := exec.Command("sudo", pm...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}
	return fmt.Errorf("dnsmasq not found and no supported package manager detected")
}

// ── Shared ─────────────────────────────────────────────────────────────────────

func printManualDNS(tld string) {
	fmt.Printf(`
Automatic DNS setup is not supported on this platform.
To resolve *.%s → 127.0.0.1 use one of the options below.

macOS (Homebrew):
  brew install dnsmasq
  echo "address=/.%s/127.0.0.1" >> $(brew --prefix)/etc/dnsmasq.conf
  sudo mkdir -p /etc/resolver && echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/%s
  sudo launchctl kickstart -k system/homebrew.mxcl.dnsmasq

Linux — NetworkManager:
  echo "address=/.%s/127.0.0.1" | sudo tee /etc/NetworkManager/dnsmasq.d/valla-%s.conf
  sudo systemctl reload NetworkManager

Linux — systemd-resolved:
  echo "address=/.%s/127.0.0.1\nport=5353" | sudo tee /etc/dnsmasq.d/valla-%s.conf
  printf '[Resolve]\nDNS=127.0.0.1#5353\nDomains=~%s\n' | sudo tee /etc/systemd/resolved.conf.d/valla-%s.conf
  sudo systemctl restart dnsmasq systemd-resolved

Windows — Acrylic DNS Proxy (https://mayakron.altervista.org/wikibase/show.php?id=AcrylicHome):
  1. Install Acrylic and open AcrylicHosts.txt
  2. Add:  127.0.0.1  *.%s
  3. Restart the Acrylic service

Alternative — no local setup required:
  Use --domain lvh.me (public wildcard DNS that always resolves to 127.0.0.1).
  Requires internet access but works on any OS without any configuration.
`, tld, tld, tld, tld, tld, tld, tld, tld, tld, tld)
}
