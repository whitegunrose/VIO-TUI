package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

const configFileName = "config.json"

// CanvasConfig for runtime
type CanvasConfig struct {
	Domain      string
	Token       string
	PollMinutes int
}

// Verify credentials (for Canvas sync)
func (c CanvasConfig) HasCredentials() bool {
	return strings.TrimSpace(c.Domain) != "" && strings.TrimSpace(c.Token) != ""
}

func (c CanvasConfig) PollInterval() time.Duration {
	if c.PollMinutes <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.PollMinutes) * time.Minute
}

type storedConfig struct {
	CanvasDomain string `json:"canvas_domain"`
	TokenEnc     string `json:"canvas_token_enc"`
	PollMinutes  int    `json:"poll_minutes"`
}

// Load and decrypt Canvas config
func LoadCanvasConfig() (CanvasConfig, bool, error) {
	path, err := configFilePath()
	if err != nil {
		return CanvasConfig{}, false, err
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CanvasConfig{}, false, nil
		}
		return CanvasConfig{}, false, fmt.Errorf("read config file: %w", err)
	}

	var stored storedConfig
	if err := json.Unmarshal(bytes, &stored); err != nil {
		return CanvasConfig{}, false, fmt.Errorf("decode config file: %w", err)
	}

	token, err := decryptString(stored.TokenEnc)
	if err != nil {
		return CanvasConfig{}, true, fmt.Errorf("decrypt canvas token: %w", err)
	}

	cfg := CanvasConfig{
		Domain:      strings.TrimSpace(stored.CanvasDomain),
		Token:       token,
		PollMinutes: stored.PollMinutes,
	}
	if cfg.PollMinutes <= 0 {
		cfg.PollMinutes = 5
	}

	return cfg, true, nil
}

// Encrypt and save Canvas config
func SaveCanvasConfig(cfg CanvasConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create app config directory: %w", err)
	}

	encrypted, err := encryptString(strings.TrimSpace(cfg.Token))
	if err != nil {
		return fmt.Errorf("encrypt canvas token: %w", err)
	}

	stored := storedConfig{
		CanvasDomain: strings.TrimSpace(cfg.Domain),
		TokenEnc:     encrypted,
		PollMinutes:  cfg.PollMinutes,
	}
	if stored.PollMinutes <= 0 {
		stored.PollMinutes = 5
	}

	bytes, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}

	if err := writeFileAtomic(path, bytes, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func configFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config dir: %w", err)
	}

	return filepath.Join(configDir, appFolderName, configFileName), nil
}

// Get machine key
func machineSecret() ([]byte, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("read hostname: %w", err)
	}

	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("read current user: %w", err)
	}

	sum := sha256.Sum256([]byte("vio-canvas-config|" + usr.Username + "|" + host))
	return sum[:], nil
}

func encryptString(plain string) (string, error) {
	key, err := machineSecret()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptString(enc string) (string, error) {
	if strings.TrimSpace(enc) == "" {
		return "", nil
	}

	key, err := machineSecret()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	bytes, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}

	if len(bytes) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := bytes[:gcm.NonceSize()]
	ciphertext := bytes[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}
