package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateLocationAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		// 既存の有効なケース
		{"valid alias", "#2F寝室", false},
		{"valid alias with number", "#room1", false},
		{"valid alias english", "#living", false},

		// 既存の無効なケース
		{"empty string", "", true},
		{"no prefix", "2F寝室", true},
		{"only prefix", "#", true},
		{"wrong prefix @", "@living", true},
		{"wrong prefix ~", "~living", true},

		// 新規: 二文字目以降の#禁止
		{"alias with hash in middle", "#foo#bar", true},
		{"alias with multiple hashes", "#a#b#c", true},

		// 新規: 長さ制限 (最大32文字、文字数ベース)
		{"alias at max length", "#1234567890123456789012345678901", false},          // #含めて32文字
		{"alias too long", "#12345678901234567890123456789012", true},               // #含めて33文字
		{"japanese alias at max length", "#あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほま", false}, // #含めて32文字
		{"japanese alias too long", "#あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみ", true},      // #含めて33文字

		// 新規: 空白文字の禁止
		{"alias with space", "#foo bar", true},
		{"alias with tab", "#foo\tbar", true},
		{"alias with newline", "#foo\nbar", true},

		// 新規: シェル特殊文字の禁止
		{"alias with dollar", "#foo$bar", true},
		{"alias with backtick", "#foo`bar", true},
		{"alias with pipe", "#foo|bar", true},
		{"alias with semicolon", "#foo;bar", true},
		{"alias with ampersand", "#foo&bar", true},
		{"alias with less than", "#foo<bar", true},
		{"alias with greater than", "#foo>bar", true},

		// 新規: 区切り記号の禁止
		{"alias with double quote", "#foo\"bar", true},
		{"alias with single quote", "#foo'bar", true},
		{"alias with comma", "#foo,bar", true},
		{"alias with slash", "#foo/bar", true},
		{"alias with backslash", "#foo\\bar", true},
		{"alias with bracket", "#foo[bar", true},
		{"alias with brace", "#foo{bar", true},
		{"alias with paren", "#foo(bar", true},

		// 新規: その他の禁止文字
		{"alias with exclamation", "#foo!bar", true},
		{"alias with at sign", "#foo@bar", true},
		{"alias with asterisk", "#foo*bar", true},
		{"alias with question", "#foo?bar", true},
		{"alias with equals", "#foo=bar", true},
		{"alias with caret", "#foo^bar", true},
		{"alias with tilde", "#foo~bar", true},
		{"alias with percent", "#foo%bar", true},

		// 新規: 許可される記号
		{"alias with hyphen", "#room-1", false},
		{"alias with underscore", "#room_1", false},
		{"alias with dot", "#room.1", false},
		{"alias with colon", "#room:1", false},
		{"alias with allowed symbols combined", "#room-1_a.b:c", false},

		// 新規: 日本語との組み合わせ
		{"japanese with allowed symbols", "#2F-寝室_A", false},
		{"mixed japanese english", "#2FRoom寝室", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocationAlias(tt.alias)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLocationAlias(%q) error = %v, wantErr %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

func TestLocationAliases_Add(t *testing.T) {
	la := NewLocationAliases()

	// 正常な追加
	err := la.Add("#2F寝室", "room2")
	if err != nil {
		t.Errorf("Add() error = %v, want nil", err)
	}

	// 重複追加
	err = la.Add("#2F寝室", "room3")
	if err == nil {
		t.Error("Add() should return error for duplicate alias")
	}
	if _, ok := err.(*LocationAliasAlreadyExistsError); !ok {
		t.Errorf("Add() error type = %T, want *LocationAliasAlreadyExistsError", err)
	}

	// 無効なエイリアス
	err = la.Add("invalid", "room4")
	if err == nil {
		t.Error("Add() should return error for invalid alias")
	}
}

func TestLocationAliases_Update(t *testing.T) {
	la := NewLocationAliases()
	_ = la.Add("#2F寝室", "room2")

	// 正常な更新
	err := la.Update("#2F寝室", "room3")
	if err != nil {
		t.Errorf("Update() error = %v, want nil", err)
	}

	// 存在しないエイリアスの更新
	err = la.Update("#存在しない", "room4")
	if err == nil {
		t.Error("Update() should return error for non-existent alias")
	}
	if _, ok := err.(*LocationAliasNotFoundError); !ok {
		t.Errorf("Update() error type = %T, want *LocationAliasNotFoundError", err)
	}
}

func TestLocationAliases_Delete(t *testing.T) {
	la := NewLocationAliases()
	_ = la.Add("#2F寝室", "room2")

	// 正常な削除
	err := la.Delete("#2F寝室")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}

	// 存在しないエイリアスの削除
	err = la.Delete("#2F寝室")
	if err == nil {
		t.Error("Delete() should return error for non-existent alias")
	}
}

func TestLocationAliases_FindByAlias(t *testing.T) {
	la := NewLocationAliases()
	_ = la.Add("#2F寝室", "room2")

	value, ok := la.FindByAlias("#2F寝室")
	if !ok {
		t.Error("FindByAlias() should find existing alias")
	}
	if value != "room2" {
		t.Errorf("FindByAlias() = %v, want room2", value)
	}

	_, ok = la.FindByAlias("#存在しない")
	if ok {
		t.Error("FindByAlias() should not find non-existent alias")
	}
}

func TestLocationAliases_FindAliasesByValue(t *testing.T) {
	la := NewLocationAliases()
	_ = la.Add("#2F寝室", "room2")
	_ = la.Add("#子供部屋", "room2")
	_ = la.Add("#リビング", "living")

	aliases := la.FindAliasesByValue("room2")
	if len(aliases) != 2 {
		t.Errorf("FindAliasesByValue() returned %d aliases, want 2", len(aliases))
	}

	aliases = la.FindAliasesByValue("存在しない")
	if len(aliases) != 0 {
		t.Errorf("FindAliasesByValue() returned %d aliases, want 0", len(aliases))
	}
}

func TestLocationAliases_GetAll(t *testing.T) {
	la := NewLocationAliases()
	_ = la.Add("#2F寝室", "room2")
	_ = la.Add("#リビング", "living")

	all := la.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d items, want 2", len(all))
	}
	if all["#2F寝室"] != "room2" {
		t.Errorf("GetAll()[#2F寝室] = %v, want room2", all["#2F寝室"])
	}
}

func TestLocationAliases_List(t *testing.T) {
	la := NewLocationAliases()
	_ = la.Add("#リビング", "living")
	_ = la.Add("#2F寝室", "room2")

	list := la.List()
	if len(list) != 2 {
		t.Errorf("List() returned %d items, want 2", len(list))
	}
	// ソートされているか確認
	if list[0].Alias != "#2F寝室" {
		t.Errorf("List()[0].Alias = %v, want #2F寝室 (sorted)", list[0].Alias)
	}
}

func TestLocationOrder_SetAndGet(t *testing.T) {
	lo := NewLocationOrder()

	order := []string{"living", "room2", "kitchen"}
	lo.Set(order)

	got := lo.Get()
	if len(got) != 3 {
		t.Errorf("Get() returned %d items, want 3", len(got))
	}
	for i, v := range order {
		if got[i] != v {
			t.Errorf("Get()[%d] = %v, want %v", i, got[i], v)
		}
	}
}

func TestLocationOrder_Reset(t *testing.T) {
	lo := NewLocationOrder()
	lo.Set([]string{"living", "room2"})
	lo.Reset()

	got := lo.Get()
	if len(got) != 0 {
		t.Errorf("Get() after Reset() returned %d items, want 0", len(got))
	}
}

func TestLocationOrder_EnsureLocation(t *testing.T) {
	lo := NewLocationOrder()
	lo.Set([]string{"living", "room2"})

	// 新しいロケーションを追加
	added := lo.EnsureLocation("kitchen")
	if !added {
		t.Error("EnsureLocation() should return true for new location")
	}

	got := lo.Get()
	if len(got) != 3 {
		t.Errorf("Get() returned %d items, want 3", len(got))
	}
	if got[2] != "kitchen" {
		t.Errorf("Get()[2] = %v, want kitchen", got[2])
	}

	// 既存のロケーション
	added = lo.EnsureLocation("living")
	if added {
		t.Error("EnsureLocation() should return false for existing location")
	}
}

func TestLocationOrder_ApplyOrder(t *testing.T) {
	lo := NewLocationOrder()
	lo.Set([]string{"room2", "living"})

	locations := []string{"kitchen", "living", "room2", "bathroom"}
	result := lo.ApplyOrder(locations)

	// room2, living が先頭に（順序通り）、残りがアルファベット順
	expected := []string{"room2", "living", "bathroom", "kitchen"}
	if len(result) != len(expected) {
		t.Errorf("ApplyOrder() returned %d items, want %d", len(result), len(expected))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("ApplyOrder()[%d] = %v, want %v", i, result[i], v)
		}
	}
}

func TestLocationOrder_ApplyOrder_EmptyOrder(t *testing.T) {
	lo := NewLocationOrder()

	locations := []string{"kitchen", "living", "bathroom"}
	result := lo.ApplyOrder(locations)

	// 順序未設定の場合はアルファベット順
	expected := []string{"bathroom", "kitchen", "living"}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("ApplyOrder()[%d] = %v, want %v", i, result[i], v)
		}
	}
}

func TestLocationSettings_SaveAndLoad(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_location_settings.json")

	// 設定を作成して保存
	ls := NewLocationSettings()
	_ = ls.Aliases.Add("#2F寝室", "room2")
	_ = ls.Aliases.Add("#リビング", "living")
	ls.Order.Set([]string{"living", "room2"})

	err := ls.SaveToFile(filename)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// ファイルの存在を確認
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("SaveToFile() did not create file")
	}

	// 新しいインスタンスで読み込み
	ls2 := NewLocationSettings()
	err = ls2.LoadFromFile(filename)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	// エイリアスの確認
	if ls2.Aliases.Count() != 2 {
		t.Errorf("Aliases count = %d, want 2", ls2.Aliases.Count())
	}
	value, ok := ls2.Aliases.FindByAlias("#2F寝室")
	if !ok || value != "room2" {
		t.Errorf("Alias #2F寝室 = %v, %v, want room2, true", value, ok)
	}

	// 順序の確認
	order := ls2.Order.Get()
	if len(order) != 2 {
		t.Errorf("Order length = %d, want 2", len(order))
	}
	if order[0] != "living" || order[1] != "room2" {
		t.Errorf("Order = %v, want [living, room2]", order)
	}
}

func TestLocationSettings_LoadFromNonExistentFile(t *testing.T) {
	ls := NewLocationSettings()
	err := ls.LoadFromFile("/nonexistent/path/file.json")
	if err != nil {
		t.Errorf("LoadFromFile() should not return error for non-existent file, got %v", err)
	}
}
