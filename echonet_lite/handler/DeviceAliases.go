package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
)

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

// 16進数の正規表現パターン
var hexPattern = regexp.MustCompile(`^[0-9A-Fa-f]+$`)

// 先頭文字が記号の場合にマッチする正規表現パターン
var invalidFirstChar = regexp.MustCompile(`^[!"#$%&'()*+,./:;<=>?@\[\\\]^_{|}~\-]`)

// ValidateDeviceAlias はエイリアスが有効かどうかを検証します
func ValidateDeviceAlias(alias string) error {
	// 空文字列は禁止
	if alias == "" {
		return &InvalidAliasError{Alias: alias, Reason: "empty alias is not allowed"}
	}

	// 2桁の倍数の16進数として読み取れる値は禁止
	if len(alias)%2 == 0 && len(alias) > 0 && hexPattern.MatchString(alias) {
		return &InvalidAliasError{Alias: alias, Reason: "alias that can be read as hexadecimal with even number of digits is not allowed"}
	}

	// 記号で始まるエイリアスは禁止
	if invalidFirstChar.MatchString(alias) {
		return &InvalidAliasError{Alias: alias, Reason: "alias that starts with a symbol is not allowed"}
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

// AliasIDStringPair はエイリアスと IDString のペアを表します
type AliasIDStringPair struct {
	Alias string
	ID    IDString
}

// IDNotFoundError は IDString が見つからない場合のエラーです
type IDNotFoundError struct {
	ID IDString
}

func (e IDNotFoundError) Error() string {
	return fmt.Sprintf("ID %v is not registered", e.ID)
}

// 注: AliasNotFoundError, AliasAlreadyExistsError, InvalidAliasError は
// DeviceAliases.go で定義されているものを使用します

// DeviceAliases は IDString に対してエイリアスを紐づけるための構造体です
type DeviceAliases struct {
	mu      sync.RWMutex
	aliases map[string]IDString // エイリアスから IDString へのマップ
}

// NewDeviceAliases は新しい RawDeviceAliases インスタンスを作成します
func NewDeviceAliases() *DeviceAliases {
	return &DeviceAliases{
		aliases: make(map[string]IDString),
	}
}

// Register はエイリアスと IDString を登録します
func (da *DeviceAliases) Register(alias string, idString IDString) error {
	// エイリアスのバリデーション
	if err := ValidateDeviceAlias(alias); err != nil {
		return err
	}

	da.mu.Lock()
	defer da.mu.Unlock()

	// 既存のエイリアスが別の IDString に関連付けられている場合、エラー
	if _, ok := da.aliases[alias]; ok {
		return &AliasAlreadyExistsError{Alias: alias}
	}

	// 新しいエイリアスを設定
	da.aliases[alias] = idString

	return nil
}

// FindByAlias はエイリアスから IDString を検索します
func (da *DeviceAliases) FindByAlias(alias string) (IDString, bool) {
	da.mu.RLock()
	defer da.mu.RUnlock()

	id, ok := da.aliases[alias]
	return id, ok
}

// FindAliasesByIDString は IDString からエイリアスを検索します
// 複数のエイリアスが同じ IDString に関連付けられている場合、すべてのエイリアスを返します
func (da *DeviceAliases) FindAliasesByIDString(idString IDString) []string {
	da.mu.RLock()
	defer da.mu.RUnlock()

	var aliases []string
	for alias, registeredID := range da.aliases {
		if registeredID == idString {
			aliases = append(aliases, alias)
		}
	}

	return aliases
}

// DeleteByAlias はエイリアスを削除します
func (da *DeviceAliases) DeleteByAlias(alias string) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	if _, ok := da.aliases[alias]; !ok {
		return &AliasNotFoundError{Alias: alias}
	}

	delete(da.aliases, alias)
	return nil
}

// DeleteByIDString は IDString に関連付けられたすべてのエイリアスを削除します
func (da *DeviceAliases) DeleteByIDString(idString IDString) error {
	da.mu.Lock()
	defer da.mu.Unlock()

	var found bool
	for alias, registeredID := range da.aliases {
		if registeredID == idString {
			delete(da.aliases, alias)
			found = true
		}
	}

	if !found {
		return &IDNotFoundError{ID: idString}
	}

	return nil
}

// List はすべてのエイリアスと IDString のペアを返します
// 結果はエイリアス名でソートされます
func (da *DeviceAliases) List() []AliasIDStringPair {
	da.mu.RLock()
	defer da.mu.RUnlock()

	result := make([]AliasIDStringPair, 0, len(da.aliases))
	for alias, id := range da.aliases {
		result = append(result, AliasIDStringPair{
			Alias: alias,
			ID:    id,
		})
	}

	// エイリアス名でソート
	sort.Slice(result, func(i, j int) bool {
		return result[i].Alias < result[j].Alias
	})

	return result
}

// SaveToFile はエイリアスと IDString の対応表をJSONファイルに保存します
func (da *DeviceAliases) SaveToFile(filename string) error {
	da.mu.RLock()
	defer da.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("ファイルを閉じる際にエラーが発生しました: %v\n", err)
		}
	}()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(da.aliases); err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	return nil
}

// LoadFromFile はJSONファイルからエイリアスと IDString の対応表を読み込みます
func (da *DeviceAliases) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// ファイルが存在しない場合は何もしない
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("ファイルを閉じる際にエラーが発生しました: %v\n", err)
		}
	}()

	da.mu.Lock()
	defer da.mu.Unlock()

	// マップをクリア
	da.aliases = make(map[string]IDString)

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&da.aliases); err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	return nil
}

// Count はエイリアスの総数を返す
func (da *DeviceAliases) Count() int {
	da.mu.RLock()
	defer da.mu.RUnlock()
	return len(da.aliases)
}
