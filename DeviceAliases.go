package main

import (
	"echonet-list/echonet_lite"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"
)

// DeviceNotFoundError はデバイスが見つからない場合のエラーです
type DeviceNotFoundError struct {
	Device IPAndEOJ
}

func (e DeviceNotFoundError) Error() string {
	return fmt.Sprintf("device %v is not found", e.Device)
}

// AliasAlreadyExistsError はエイリアスが既に存在する場合のエラーです
type AliasAlreadyExistsError struct {
	Alias string
}

func (e AliasAlreadyExistsError) Error() string {
	return fmt.Sprintf("alias %s is already used for another device", e.Alias)
}

// InvalidAliasError はエイリアスが無効な場合のエラーです
type InvalidAliasError struct {
	Alias  string
	Reason string
}

func (e InvalidAliasError) Error() string {
	return fmt.Sprintf("invalid alias %s: %s", e.Alias, e.Reason)
}

// DeviceAliases は デバイスを特定する IPAndEOJ に対して、
// 永続的な識別子 echonet_lite.IdentificationNumber と、
// 人間が理解しやすい名前 Alias (string) を紐づけるための構造体です。
type DeviceAliases struct {
	mu                     sync.RWMutex
	deviceToIdentification map[IPAndEOJ]string // IPAndEOJ から IdentificationNumber の文字列表現へのマップ
	identificationToDevice map[string]IPAndEOJ // IdentificationNumber の文字列表現から IPAndEOJ へのマップ
	identificationToAlias  map[string]string   // IdentificationNumber の文字列表現からエイリアスへのマップ
	aliasToIdentification  map[string]string   // エイリアスから IdentificationNumber の文字列表現へのマップ
}

// NewDeviceAliases は新しいDeviceAliasesインスタンスを作成します
func NewDeviceAliases() *DeviceAliases {
	return &DeviceAliases{
		deviceToIdentification: make(map[IPAndEOJ]string),
		identificationToDevice: make(map[string]IPAndEOJ),
		identificationToAlias:  make(map[string]string),
		aliasToIdentification:  make(map[string]string),
	}
}

// RegisterDeviceIdentification はデバイスとIdentificationNumberを関連付けます
func (da *DeviceAliases) RegisterDeviceIdentification(device IPAndEOJ, identificationNumber *echonet_lite.IdentificationNumber) error {
	if identificationNumber == nil {
		return fmt.Errorf("identificationNumber cannot be nil")
	}

	idStr := identificationNumber.String()

	da.mu.Lock()
	defer da.mu.Unlock()

	// 既存のデバイスに関連付けられたIdentificationNumberがある場合、それを削除
	if existingID, ok := da.deviceToIdentification[device]; ok {
		delete(da.identificationToDevice, existingID)
	}

	// 既存のIdentificationNumberに関連付けられたデバイスがある場合、それを削除
	if existingDevice, ok := da.identificationToDevice[idStr]; ok {
		delete(da.deviceToIdentification, existingDevice)
	}

	// 新しい関連付けを設定
	da.deviceToIdentification[device] = idStr
	da.identificationToDevice[idStr] = device

	return nil
}

// 16進数の正規表現パターン
var hexPattern = regexp.MustCompile(`^[0-9A-Fa-f]+$`)

// 先頭文字が数字と記号の場合にマッチする正規表現パターン
var invalidFirstChar = regexp.MustCompile(`^[0-9\!"#\$%&'\(\)\*\+,\./:;<=>\?@\[\\\]\^_\{\|\}~\-]`)

// validateDeviceAlias はエイリアスが有効かどうかを検証します
func validateDeviceAlias(alias string) error {
	// 空文字列は禁止
	if alias == "" {
		return &InvalidAliasError{Alias: alias, Reason: "empty alias is not allowed"}
	}

	// 2桁の倍数の16進数として読み取れる値は禁止
	if len(alias)%2 == 0 && len(alias) > 0 && hexPattern.MatchString(alias) {
		return &InvalidAliasError{Alias: alias, Reason: "alias that can be read as hexadecimal with even number of digits is not allowed"}
	}

	// 数字や記号で始まるエイリアスは禁止
	if invalidFirstChar.MatchString(alias) {
		return &InvalidAliasError{Alias: alias, Reason: "alias that starts with a number or symbol is not allowed"}
	}

	return nil
}

// SetAlias はIdentificationNumberにエイリアスを設定します
func (da *DeviceAliases) SetAlias(device IPAndEOJ, alias string) error {
	// エイリアスのバリデーション
	if err := validateDeviceAlias(alias); err != nil {
		return err
	}

	da.mu.Lock()
	defer da.mu.Unlock()

	// デバイスからIdentificationNumberを取得
	idStr, ok := da.deviceToIdentification[device]
	if !ok {
		return &DeviceNotFoundError{Device: device}
	}

	// 既存のエイリアスが別のIdentificationNumberに関連付けられている場合、エラー
	if existingID, ok := da.aliasToIdentification[alias]; ok && existingID != idStr {
		return &AliasAlreadyExistsError{Alias: alias}
	}

	// 既存のIdentificationNumberに関連付けられたエイリアスがある場合、それを削除
	if existingAlias, ok := da.identificationToAlias[idStr]; ok {
		delete(da.aliasToIdentification, existingAlias)
	}

	// 新しいエイリアスを設定
	da.identificationToAlias[idStr] = alias
	da.aliasToIdentification[alias] = idStr

	return nil
}

// GetAlias はデバイスに関連付けられたエイリアスを取得します
func (da *DeviceAliases) GetAlias(device IPAndEOJ) (string, bool) {
	da.mu.RLock()
	defer da.mu.RUnlock()

	idStr, ok := da.deviceToIdentification[device]
	if !ok {
		return "", false
	}

	alias, ok := da.identificationToAlias[idStr]
	return alias, ok
}

// GetDeviceByAlias はエイリアスに関連付けられたデバイスを取得します
func (da *DeviceAliases) GetDeviceByAlias(alias string) (IPAndEOJ, bool) {
	da.mu.RLock()
	defer da.mu.RUnlock()

	idStr, ok := da.aliasToIdentification[alias]
	if !ok {
		return IPAndEOJ{}, false
	}

	device, ok := da.identificationToDevice[idStr]
	return device, ok
}

// GetDeviceByIdentificationNumber はIdentificationNumberに関連付けられたデバイスを取得します
func (da *DeviceAliases) GetDeviceByIdentificationNumber(identificationNumber *echonet_lite.IdentificationNumber) (IPAndEOJ, bool) {
	if identificationNumber == nil {
		return IPAndEOJ{}, false
	}

	idStr := identificationNumber.String()

	da.mu.RLock()
	defer da.mu.RUnlock()

	device, ok := da.identificationToDevice[idStr]
	return device, ok
}

// aliasToIdentificationData は永続化のためのデータ構造です
type aliasToIdentificationData map[string]string

// SaveToFile はエイリアスとIdentificationNumberの対応表をJSONファイルに保存します
func (da *DeviceAliases) SaveToFile(filename string) error {
	da.mu.RLock()
	defer da.mu.RUnlock()

	// aliasToIdentificationのみを保存
	data := aliasToIdentificationData(da.aliasToIdentification)

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	return nil
}

// LoadFromFile はJSONファイルからエイリアスとIdentificationNumberの対応表を読み込みます
func (da *DeviceAliases) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// ファイルが存在しない場合は何もしない
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var data aliasToIdentificationData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	da.mu.Lock()
	defer da.mu.Unlock()

	// マップをクリア
	da.aliasToIdentification = make(map[string]string)
	da.identificationToAlias = make(map[string]string)

	// データを読み込む
	for alias, idStr := range data {
		da.aliasToIdentification[alias] = idStr
		da.identificationToAlias[idStr] = alias
	}

	return nil
}

// AliasNotFoundError はエイリアスが見つからない場合のエラーです
type AliasNotFoundError struct {
	Alias string
}

func (e AliasNotFoundError) Error() string {
	return fmt.Sprintf("alias %s is not registered", e.Alias)
}

// AliasDevicePair はエイリアスとデバイスのペアを表します
type AliasDevicePair struct {
	Alias  string
	Device *IPAndEOJ // デバイスが存在しない場合はnil
}

func (pair AliasDevicePair) String() string {
	if pair.Device == nil {
		return fmt.Sprintf("%s: not found", pair.Alias)
	}
	return fmt.Sprintf("%s: %v", pair.Alias, *pair.Device)
}

// GetAllAliases はすべてのエイリアスとそれに対応するデバイス（存在する場合）の一覧を返します
func (da *DeviceAliases) GetAllAliases() []AliasDevicePair {
	da.mu.RLock()
	defer da.mu.RUnlock()

	result := make([]AliasDevicePair, 0, len(da.aliasToIdentification))
	for alias, idStr := range da.aliasToIdentification {
		var devicePtr *IPAndEOJ
		if device, ok := da.identificationToDevice[idStr]; ok {
			devicePtr = &device
		}
		result = append(result, AliasDevicePair{
			Alias:  alias,
			Device: devicePtr,
		})
	}
	return result
}

// RemoveDevice はデバイスとその関連付けを削除します
// エイリアスとIdentificationNumberの関連付けは維持されます
func (da *DeviceAliases) RemoveDevice(device IPAndEOJ) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	// デバイスからIdentificationNumberを取得
	idStr, ok := da.deviceToIdentification[device]
	if !ok {
		return &DeviceNotFoundError{Device: device}
	}

	// デバイスとIdentificationNumberの関連付けを削除
	delete(da.deviceToIdentification, device)
	delete(da.identificationToDevice, idStr)

	return nil
}

// RemoveAlias はエイリアスとその関連付けを削除します
func (da *DeviceAliases) RemoveAlias(alias string) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	// エイリアスからIdentificationNumberを取得
	idStr, ok := da.aliasToIdentification[alias]
	if !ok {
		return &AliasNotFoundError{Alias: alias}
	}

	// エイリアスとIdentificationNumberの関連付けを削除
	delete(da.aliasToIdentification, alias)
	delete(da.identificationToAlias, idStr)

	return nil
}
