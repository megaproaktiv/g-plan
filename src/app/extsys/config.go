// Package extsys connects pt-plan to external time-tracking systems. It holds
// the per-user system configuration (credentials, stored outside any project in
// ~/.pt-pm/config.yaml) and the drivers that fetch single-line bookings.
package extsys

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigDirName / ConfigFileName locate the user configuration:
// ~/.pt-pm/config.yaml.
const (
	ConfigDirName  = ".pt-pm"
	ConfigFileName = "config.yaml"
)

// AerionConfig holds the connection settings for the Aerion API. The yaml keys
// are identical to the aerion: section of the reporting tool's config.yaml.
type AerionConfig struct {
	BaseURL      string `yaml:"base_url"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// Complete returns an error naming the settings that are still missing.
func (c AerionConfig) Complete() error {
	var missing []string
	for _, f := range []struct{ name, val string }{
		{"base URL", c.BaseURL}, {"username", c.Username}, {"password", c.Password},
		{"client ID", c.ClientID}, {"client secret", c.ClientSecret},
	} {
		if strings.TrimSpace(f.val) == "" {
			missing = append(missing, f.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("Aerion is not configured (missing: %s) - use Configure System", strings.Join(missing, ", "))
	}
	return ValidateBaseURL(c.BaseURL)
}

// ValidateBaseURL requires an https URL with a host. Plain http is accepted
// only for localhost (development), because credentials are sent to this URL.
func ValidateBaseURL(s string) error {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u.Host == "" {
		return fmt.Errorf("base URL %q is not a valid URL", s)
	}
	host := u.Hostname()
	local := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return fmt.Errorf("base URL must use https")
	}
	return nil
}

// DefaultConfigPath returns ~/.pt-pm/config.yaml.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDirName, ConfigFileName), nil
}

// LoadAerion reads the aerion: section. A missing file or section yields an
// empty config without error.
func LoadAerion(path string) (AerionConfig, error) {
	var doc struct {
		Aerion AerionConfig `yaml:"aerion"`
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AerionConfig{}, nil
		}
		return AerionConfig{}, err
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return AerionConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return doc.Aerion, nil
}

// SaveAerion writes the aerion: section into the config file, keeping every
// other top-level key of an existing file. The directory is created with mode
// 0700 and the file written atomically with mode 0600 (it holds credentials).
func SaveAerion(path string, c AerionConfig) error {
	doc := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if doc == nil {
			doc = map[string]any{}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	doc["aerion"] = c
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
