package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	allAnimeKey   []byte
	allAnimeEpoch string
	allAnimeBuild string
	allAnimeLane  string
)

const mkissaRef = "https://mkissa.to"
const mkissaCDN = "https://cdn.mkissa.net/all/mk/_app/immutable"

type httpGetFunc func(url, referer string) ([]byte, error)
type httpGetWithHeadersFunc func(url string, headers map[string]string) ([]byte, error)

func FetchAllAnimeKey(httpGet httpGetFunc, httpGetWithHeaders httpGetWithHeadersFunc) error {
	page, err := httpGet(mkissaRef, mkissaRef)
	if err != nil {
		return fmt.Errorf("fetch mkissa.to: %w", err)
	}
	pageStr := string(page)

	allAnimeBuild = extractBuildID(pageStr, func(u string) ([]byte, error) {
		return httpGet(u, mkissaRef)
	})
	allAnimeLane = "k7"

	epochStr := extractJSONString(pageStr, "epoch")
	if epochStr != "" {
		allAnimeEpoch = epochStr
	} else {
		epoch := generateEpoch()
		allAnimeEpoch = strconv.FormatInt(epoch, 10)
	}

	aaPartB := extractJSONString(pageStr, "partB")

	appURLRe := regexp.MustCompile(mkissaCDN + `/entry/app\.[A-Za-z0-9_.-]+\.js`)
	appURL := appURLRe.FindString(pageStr)

	maskHex := ""
	if appURL != "" {
		appJS, err := httpGet(appURL, mkissaRef)
		if err == nil {
			chunkRe := regexp.MustCompile(`"[.][.]/chunks/[A-Za-z0-9_.-]+\.js"`)
			chunks := chunkRe.FindAllString(string(appJS), -1)
			for _, c := range chunks {
				chunkPath := strings.Trim(c, "\"")
				chunkPath = strings.TrimPrefix(chunkPath, "../")
				chunkURL := mkissaCDN + "/" + chunkPath
				data, chunkErr := httpGet(chunkURL, mkissaRef)
				if chunkErr == nil {
					hexRe := regexp.MustCompile(`[0-9a-f]{64}`)
					if m := hexRe.FindString(string(data)); m != "" {
						maskHex = m
						break
					}
				}
			}
		}
	}

	if maskHex == "" || aaPartB == "" {
		return errors.New("could not extract crypto key from page or JS")
	}

	partB, err := base64.StdEncoding.DecodeString(aaPartB)
	if err != nil {
		return fmt.Errorf("base64 partB: %w", err)
	}

	mask, err := hex.DecodeString(maskHex)
	if err != nil {
		return fmt.Errorf("hex mask: %w", err)
	}

	if len(partB) < 32 || len(mask) < 32 {
		return errors.New("partB or mask too short")
	}

	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		key[i] = mask[i] ^ partB[i]
	}
	allAnimeKey = key
	return nil
}

func extractJSONString(page, key string) string {
	re := regexp.MustCompile(`"` + key + `":"([^"]+)"`)
	if m := re.FindStringSubmatch(page); len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractBuildID(page string, getURL func(string) ([]byte, error)) string {
	re := regexp.MustCompile(`!=="string"\?"(\d+)"`)
	if m := re.FindStringSubmatch(page); len(m) > 1 {
		return m[1]
	}
	if data, err := getURL("https://cdn.mkissa.net/all/mk/_app/version.json"); err == nil {
		var v struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &v) == nil && v.Version != "" {
			return v.Version
		}
	}
	return "72"
}

func generateEpoch() int64 {
	nowMs := time.Now().UnixMilli()
	epoch := nowMs / 259200000
	if nowMs-epoch*259200000 < 86400000 && epoch > 0 {
		return epoch - 1
	}
	return epoch
}

func Epoch() string { return allAnimeEpoch }

func BuildID() string { return allAnimeBuild }

func Lane() string { return allAnimeLane }

func EncryptAaReq(payload string, ivSrc string) (string, error) {
	if len(allAnimeKey) == 0 {
		return "", errors.New("key not initialized")
	}

	h := sha256.Sum256([]byte(ivSrc))
	iv := h[:12]

	block, err := aes.NewCipher(allAnimeKey)
	if err != nil {
		return "", fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	ciphertextWithTag := gcm.Seal(nil, iv, []byte(payload), nil)

	out := make([]byte, 1+12+len(ciphertextWithTag))
	out[0] = 0x01
	copy(out[1:], iv)
	copy(out[13:], ciphertextWithTag)

	return base64.StdEncoding.EncodeToString(out), nil
}

func DecryptAllAnime(encoded string) (string, error) {
	if len(allAnimeKey) == 0 {
		return "", errors.New("key not initialized")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64: %w", err)
	}

	if len(data) < 1+12+16 {
		return "", errors.New("data too short")
	}

	iv := data[1:13]
	ciphertextWithTag := data[13:]

	block, err := aes.NewCipher(allAnimeKey)
	if err != nil {
		return "", fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertextWithTag, nil)
	if err != nil {
		return "", fmt.Errorf("gcm decrypt: %w", err)
	}

	return string(plaintext), nil
}

func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
