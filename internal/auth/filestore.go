package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/oauth2"
)

// Compile-time assertion: FileStore must implement TokenStore.
var _ TokenStore = (*FileStore)(nil)

const (
	credFileName = "credentials.enc"
	nonceSize    = 12 // AES-GCM standard nonce size

	hkdfSalt = "gwx-filestore-v1"
	hkdfInfo = "aes-key"
)

// FileStore implements TokenStore using a single AES-256-GCM encrypted file.
// All credentials are stored as a map[string]string in credentials.enc.
type FileStore struct {
	dir string // storage directory (e.g. ~/.config/gwx/)
	key []byte // 32-byte AES-256 key (HKDF derived)
}

// NewFileStore creates a FileStore rooted at os.UserConfigDir()/gwx.
// The encryption key is derived from the machine ID via HKDF-SHA256.
func NewFileStore() (*FileStore, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get config dir: %w", err)
	}
	dir := filepath.Join(base, "gwx")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	mid, err := machineID()
	if err != nil {
		return nil, fmt.Errorf("get machine id: %w", err)
	}

	key, err := deriveKey([]byte(mid))
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	return &FileStore{dir: dir, key: key}, nil
}

// NewFileStoreWithKey creates a FileStore with an explicitly provided 32-byte key.
// Intended for testing — avoids machineID dependency.
func NewFileStoreWithKey(dir string, key []byte) *FileStore {
	k := make([]byte, 32)
	copy(k, key)
	return &FileStore{dir: dir, key: k}
}

// deriveKey derives a 32-byte AES key from seed using HKDF-SHA256.
func deriveKey(seed []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, seed, []byte(hkdfSalt), []byte(hkdfInfo))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// hostnameID is the shared fallback used by platform-specific machineID implementations.
func hostnameID() (string, error) {
	h, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("all machine-id methods failed: %w", err)
	}
	return h, nil
}

// encPath returns the path to the encrypted credentials file.
func (fs *FileStore) encPath() string {
	return filepath.Join(fs.dir, credFileName)
}

// tmpPath returns the path to the temp file used for atomic writes.
func (fs *FileStore) tmpPath() string {
	return filepath.Join(fs.dir, credFileName+".tmp")
}

// lockPath returns the path to the lock file.
func (fs *FileStore) lockPath() string {
	return filepath.Join(fs.dir, credFileName+".lock")
}

// loadMap reads and decrypts the credentials file into a map[string]string.
// Returns an empty map if the file does not exist.
func (fs *FileStore) loadMap() (map[string]string, error) {
	p := fs.encPath()

	// Fix permissions if file exists but has wrong mode.
	if _, err := os.Stat(p); err == nil {
		ensurePermissions(p)
	}

	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}

	plain, err := fs.decrypt(data)
	if err != nil {
		return nil, err
	}

	var m map[string]string
	if err := json.Unmarshal(plain, &m); err != nil {
		return nil, fmt.Errorf("%w: json decode failed", ErrCredentialCorrupted)
	}
	return m, nil
}

// saveMap encrypts and atomically writes the credentials map to disk.
func (fs *FileStore) saveMap(m map[string]string) error {
	plain, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	ciphertext, err := fs.encrypt(plain)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	tmp := fs.tmpPath()
	if err := os.WriteFile(tmp, ciphertext, 0600); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmp, fs.encPath()); err != nil {
		os.Remove(tmp) //nolint:errcheck
		return fmt.Errorf("rename: %w", err)
	}

	// Enforce 0600 after rename (umask may alter it).
	if err := os.Chmod(fs.encPath(), 0600); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}

// encrypt encrypts plaintext with AES-256-GCM.
// Output: [12-byte nonce][GCM ciphertext+tag]
func (fs *FileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(fs.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt decrypts data produced by encrypt.
func (fs *FileStore) decrypt(data []byte) ([]byte, error) {
	if len(data) < nonceSize {
		return nil, fmt.Errorf("%w: data too short", ErrCredentialCorrupted)
	}
	block, err := aes.NewCipher(fs.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// GCM auth failure = wrong key or corrupted data.
		return nil, fmt.Errorf("%w: %v", ErrKeyMismatch, err)
	}
	return plain, nil
}

// ensurePermissions checks that path has 0600 permissions; fixes if not.
func ensurePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		slog.Warn("insecure permissions, fixing to 0600", "path", path, "permissions", fmt.Sprintf("%04o", perm))
		if err := os.Chmod(path, 0600); err != nil {
			slog.Warn("failed to chmod", "path", path, "err", err)
		}
	}
}

// --- TokenStore implementation ---

// SaveToken stores an OAuth2 token under key "token:{account}".
func (fs *FileStore) SaveToken(account string, token *oauth2.Token) error {
	return fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		data, err := json.Marshal(token)
		if err != nil {
			return fmt.Errorf("marshal token: %w", err)
		}
		m["token:"+account] = string(data)
		return fs.saveMap(m)
	})
}

// LoadToken retrieves the OAuth2 token for account.
// Returns ErrTokenNotFound if no token is stored.
func (fs *FileStore) LoadToken(account string) (*oauth2.Token, error) {
	var result *oauth2.Token
	err := fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		raw, ok := m["token:"+account]
		if !ok {
			return fmt.Errorf("%w: %s", ErrTokenNotFound, account)
		}
		var tok oauth2.Token
		if err := json.Unmarshal([]byte(raw), &tok); err != nil {
			return fmt.Errorf("unmarshal token: %w", err)
		}
		result = &tok
		return nil
	})
	return result, err
}

// DeleteToken removes the token for account.
func (fs *FileStore) DeleteToken(account string) error {
	return fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		key := "token:" + account
		if _, ok := m[key]; !ok {
			return fmt.Errorf("%w: %s", ErrTokenNotFound, account)
		}
		delete(m, key)
		return fs.saveMap(m)
	})
}

// SaveCredentials stores OAuth client credentials under key "cred:{name}".
func (fs *FileStore) SaveCredentials(name string, creds *OAuthCredentials) error {
	return fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		data, err := json.Marshal(creds)
		if err != nil {
			return fmt.Errorf("marshal credentials: %w", err)
		}
		m["cred:"+name] = string(data)
		return fs.saveMap(m)
	})
}

// LoadCredentials retrieves OAuth client credentials by name.
func (fs *FileStore) LoadCredentials(name string) (*OAuthCredentials, error) {
	var result *OAuthCredentials
	err := fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		raw, ok := m["cred:"+name]
		if !ok {
			return fmt.Errorf("%w: credentials %s not found", ErrTokenNotFound, name)
		}
		var c OAuthCredentials
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			return fmt.Errorf("unmarshal credentials: %w", err)
		}
		result = &c
		return nil
	})
	return result, err
}

// SaveProviderToken stores a raw provider token under "provider:{provider}:{account}".
func (fs *FileStore) SaveProviderToken(provider, account, token string) error {
	return fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		m["provider:"+provider+":"+account] = token
		return fs.saveMap(m)
	})
}

// LoadProviderToken retrieves a raw provider token.
func (fs *FileStore) LoadProviderToken(provider, account string) (string, error) {
	var result string
	err := fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		tok, ok := m["provider:"+provider+":"+account]
		if !ok {
			return fmt.Errorf("%w: %s/%s", ErrTokenNotFound, provider, account)
		}
		result = tok
		return nil
	})
	return result, err
}

// DeleteProviderToken removes a provider token.
func (fs *FileStore) DeleteProviderToken(provider, account string) error {
	return fs.withLock(func() error {
		m, err := fs.loadMap()
		if err != nil {
			return err
		}
		key := "provider:" + provider + ":" + account
		if _, ok := m[key]; !ok {
			return fmt.Errorf("%w: %s/%s", ErrTokenNotFound, provider, account)
		}
		delete(m, key)
		return fs.saveMap(m)
	})
}

// HasProviderToken reports whether a token exists for the given provider and account.
func (fs *FileStore) HasProviderToken(provider, account string) bool {
	_, err := fs.LoadProviderToken(provider, account)
	return err == nil
}
