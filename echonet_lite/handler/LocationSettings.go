package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	LocationSettingsFileName = "location_settings.json"
	LocationAliasPrefix      = "#"
	// MaxLocationAliasLength は#を含めたエイリアスの最大長
	MaxLocationAliasLength = 32
)

// 禁止文字のパターン（#プレフィックス後に適用）
// 空白系: スペース、タブ、改行
// シェル特殊文字: $, `, |, ;, &, <, >
// 区切り記号: ", ', ,, /, \, [, ], {, }, (, )
// その他: !, @, *, ?, =, ^, ~, %
var locationAliasProhibitedChars = regexp.MustCompile("[\t\n\r \"!$%&'()*,/;<=>?@\\[\\\\\\]^`{|}~]")

// LocationAliasAlreadyExistsError はロケーションエイリアスが既に存在する場合のエラーです
type LocationAliasAlreadyExistsError struct {
	Alias string
}

func (e LocationAliasAlreadyExistsError) Error() string {
	return fmt.Sprintf("location alias %s already exists", e.Alias)
}

// InvalidLocationAliasError はロケーションエイリアスが無効な場合のエラーです
type InvalidLocationAliasError struct {
	Alias  string
	Reason string
}

func (e InvalidLocationAliasError) Error() string {
	return fmt.Sprintf("invalid location alias %s: %s", e.Alias, e.Reason)
}

// LocationAliasNotFoundError はロケーションエイリアスが見つからない場合のエラーです
type LocationAliasNotFoundError struct {
	Alias string
}

func (e LocationAliasNotFoundError) Error() string {
	return fmt.Sprintf("location alias %s not found", e.Alias)
}

// ValidateLocationAlias はロケーションエイリアスが有効かどうかを検証します
// ロケーションエイリアスは # プレフィックスで始まる必要があります
func ValidateLocationAlias(alias string) error {
	if alias == "" {
		return &InvalidLocationAliasError{Alias: alias, Reason: "alias cannot be empty"}
	}

	if !strings.HasPrefix(alias, LocationAliasPrefix) {
		return &InvalidLocationAliasError{Alias: alias, Reason: fmt.Sprintf("location alias must start with '%s'", LocationAliasPrefix)}
	}

	// プレフィックス後に少なくとも1文字必要
	if len(alias) <= len(LocationAliasPrefix) {
		return &InvalidLocationAliasError{Alias: alias, Reason: fmt.Sprintf("location alias must have at least one character after '%s'", LocationAliasPrefix)}
	}

	// 長さ制限チェック
	if len(alias) > MaxLocationAliasLength {
		return &InvalidLocationAliasError{Alias: alias, Reason: fmt.Sprintf("location alias must be %d characters or less", MaxLocationAliasLength)}
	}

	// プレフィックス以降の部分を取得
	afterPrefix := alias[len(LocationAliasPrefix):]

	// 二文字目以降に#が含まれていないかチェック
	if strings.Contains(afterPrefix, LocationAliasPrefix) {
		return &InvalidLocationAliasError{Alias: alias, Reason: fmt.Sprintf("location alias cannot contain '%s' after the first character", LocationAliasPrefix)}
	}

	// 禁止文字チェック
	if locationAliasProhibitedChars.MatchString(afterPrefix) {
		return &InvalidLocationAliasError{Alias: alias, Reason: "location alias contains prohibited characters"}
	}

	return nil
}

// LocationAliasValuePair はロケーションエイリアスと値のペアを表します
type LocationAliasValuePair struct {
	Alias string // #プレフィックス付き
	Value string // 生のロケーション値（例: "living", "room2"）
}

// LocationAliases はロケーションエイリアスを管理する構造体です
type LocationAliases struct {
	mu      sync.RWMutex
	aliases map[string]string // "#alias" -> "rawLocationValue"
}

// NewLocationAliases は新しい LocationAliases インスタンスを作成します
func NewLocationAliases() *LocationAliases {
	return &LocationAliases{
		aliases: make(map[string]string),
	}
}

// Add はエイリアスを追加します
func (la *LocationAliases) Add(alias string, value string) error {
	if err := ValidateLocationAlias(alias); err != nil {
		return err
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	if _, exists := la.aliases[alias]; exists {
		return &LocationAliasAlreadyExistsError{Alias: alias}
	}

	la.aliases[alias] = value
	return nil
}

// Update は既存のエイリアスを更新します
func (la *LocationAliases) Update(alias string, value string) error {
	if err := ValidateLocationAlias(alias); err != nil {
		return err
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	if _, exists := la.aliases[alias]; !exists {
		return &LocationAliasNotFoundError{Alias: alias}
	}

	la.aliases[alias] = value
	return nil
}

// Delete はエイリアスを削除します
func (la *LocationAliases) Delete(alias string) error {
	la.mu.Lock()
	defer la.mu.Unlock()

	if _, exists := la.aliases[alias]; !exists {
		return &LocationAliasNotFoundError{Alias: alias}
	}

	delete(la.aliases, alias)
	return nil
}

// FindByAlias はエイリアスから値を検索します
func (la *LocationAliases) FindByAlias(alias string) (string, bool) {
	la.mu.RLock()
	defer la.mu.RUnlock()

	value, ok := la.aliases[alias]
	return value, ok
}

// FindAliasesByValue は値からエイリアスを検索します
// 複数のエイリアスが同じ値に関連付けられている場合、すべてのエイリアスを返します
func (la *LocationAliases) FindAliasesByValue(value string) []string {
	la.mu.RLock()
	defer la.mu.RUnlock()

	var aliases []string
	for alias, v := range la.aliases {
		if v == value {
			aliases = append(aliases, alias)
		}
	}

	sort.Strings(aliases)
	return aliases
}

// GetAll はすべてのエイリアスと値のマップを返します
func (la *LocationAliases) GetAll() map[string]string {
	la.mu.RLock()
	defer la.mu.RUnlock()

	result := make(map[string]string, len(la.aliases))
	for k, v := range la.aliases {
		result[k] = v
	}
	return result
}

// List はすべてのエイリアスと値のペアを返します
// 結果はエイリアス名でソートされます
func (la *LocationAliases) List() []LocationAliasValuePair {
	la.mu.RLock()
	defer la.mu.RUnlock()

	result := make([]LocationAliasValuePair, 0, len(la.aliases))
	for alias, value := range la.aliases {
		result = append(result, LocationAliasValuePair{
			Alias: alias,
			Value: value,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Alias < result[j].Alias
	})

	return result
}

// Count はエイリアスの総数を返します
func (la *LocationAliases) Count() int {
	la.mu.RLock()
	defer la.mu.RUnlock()
	return len(la.aliases)
}

// LocationOrder はロケーションの表示順を管理する構造体です
type LocationOrder struct {
	mu    sync.RWMutex
	order []string
}

// NewLocationOrder は新しい LocationOrder インスタンスを作成します
func NewLocationOrder() *LocationOrder {
	return &LocationOrder{
		order: []string{},
	}
}

// Set は表示順を設定します
func (lo *LocationOrder) Set(order []string) {
	lo.mu.Lock()
	defer lo.mu.Unlock()

	lo.order = make([]string, len(order))
	copy(lo.order, order)
}

// Get は現在の表示順を返します
func (lo *LocationOrder) Get() []string {
	lo.mu.RLock()
	defer lo.mu.RUnlock()

	result := make([]string, len(lo.order))
	copy(result, lo.order)
	return result
}

// Reset は表示順をリセット（空に）します
func (lo *LocationOrder) Reset() {
	lo.mu.Lock()
	defer lo.mu.Unlock()

	lo.order = []string{}
}

// EnsureLocation は指定されたロケーションが順序リストに含まれていることを保証します
// 含まれていない場合は末尾に追加します
func (lo *LocationOrder) EnsureLocation(location string) bool {
	lo.mu.Lock()
	defer lo.mu.Unlock()

	// 既に含まれているか確認
	for _, loc := range lo.order {
		if loc == location {
			return false // 追加不要
		}
	}

	// 末尾に追加
	lo.order = append(lo.order, location)
	return true // 追加した
}

// ApplyOrder は指定されたロケーションリストにカスタム順序を適用します
// orderに含まれるロケーションは先頭に（順序通り）、
// 含まれないロケーションはアルファベット順で末尾に配置されます
func (lo *LocationOrder) ApplyOrder(locations []string) []string {
	lo.mu.RLock()
	defer lo.mu.RUnlock()

	if len(lo.order) == 0 {
		// 順序が設定されていない場合はソートして返す
		sorted := make([]string, len(locations))
		copy(sorted, locations)
		sort.Strings(sorted)
		return sorted
	}

	// ロケーションをセットに変換
	locationSet := make(map[string]bool)
	for _, loc := range locations {
		locationSet[loc] = true
	}

	result := make([]string, 0, len(locations))

	// 順序に従って追加
	for _, loc := range lo.order {
		if locationSet[loc] {
			result = append(result, loc)
			delete(locationSet, loc)
		}
	}

	// 残りをアルファベット順で追加
	remaining := make([]string, 0, len(locationSet))
	for loc := range locationSet {
		remaining = append(remaining, loc)
	}
	sort.Strings(remaining)
	result = append(result, remaining...)

	return result
}

// LocationSettings はロケーションのエイリアスと表示順を統合管理する構造体です
type LocationSettings struct {
	Aliases *LocationAliases
	Order   *LocationOrder
}

// NewLocationSettings は新しい LocationSettings インスタンスを作成します
func NewLocationSettings() *LocationSettings {
	return &LocationSettings{
		Aliases: NewLocationAliases(),
		Order:   NewLocationOrder(),
	}
}

// locationSettingsJSON はJSONファイルの形式を定義します
type locationSettingsJSON struct {
	Aliases map[string]string `json:"aliases"`
	Order   []string          `json:"order"`
}

// SaveToFile は設定をJSONファイルに保存します
func (ls *LocationSettings) SaveToFile(filename string) error {
	data := locationSettingsJSON{
		Aliases: ls.Aliases.GetAll(),
		Order:   ls.Order.Get(),
	}

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
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}

	return nil
}

// LoadFromFile はJSONファイルから設定を読み込みます
func (ls *LocationSettings) LoadFromFile(filename string) error {
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

	var data locationSettingsJSON
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	// エイリアスを読み込み
	for alias, value := range data.Aliases {
		if err := ls.Aliases.Add(alias, value); err != nil {
			// ログに警告を出すが続行
			fmt.Printf("Warning: failed to load alias %s: %v\n", alias, err)
		}
	}

	// 順序を読み込み
	ls.Order.Set(data.Order)

	return nil
}
