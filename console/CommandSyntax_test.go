package console

import (
	"net"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func word(pos int, s string) Token {
	return Token{Type: TokenWord, Pos: pos, String: s}
}
func colon(pos int) Token {
	return Token{Type: TokenColon, Pos: pos, String: ":"}
}
func eof(pos int, s string) Token {
	return Token{Type: TokenEOF, Pos: pos, String: s}
}

func TestTokenize(t *testing.T) {
	// Tokenizer は、入力文字列をトークン列に変換する。':' だけは前後に空白がなくてもトークンになるが、それ以外は空白で区切る

	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:     "通常の入力",
			input:    "abc def",
			expected: []Token{word(0, "abc"), word(4, "def"), eof(7, "")},
		},
		{
			name:     "末尾に空白がある入力",
			input:    "abc def ",
			expected: []Token{word(0, "abc"), word(4, "def"), eof(8, " ")},
		},
		{
			name:     "複数の空白を含む入力",
			input:    "  abc  def  ",
			expected: []Token{word(2, "abc"), word(7, "def"), eof(12, "  ")},
		},
		{
			name:     "コロンで区切られた入力",
			input:    "abc:def",
			expected: []Token{word(0, "abc"), colon(3), word(4, "def"), eof(7, "")},
		},
		{
			name:     "空文字列の入力",
			input:    "",
			expected: []Token{eof(0, "")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := Tokenize(tt.input)
			if !cmp.Equal(tokens, tt.expected) {
				t.Errorf("Tokenizer(%q) = %v, want %v, diff %v", tt.input, tokens, tt.expected, cmp.Diff(tokens, tt.expected))
			}
		})
	}
}

func TestLiteralNode_Match(t *testing.T) {
	target := LiteralNode("abc")

	tests := []struct {
		name           string
		input          Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "リテラル文字列と一致",
			input:          word(0, "abc"),
			expectedResult: LiteralResult("abc"),
			expectedPos:    1,
		},
		{
			name:           "リテラル文字列と不一致",
			input:          word(0, "def"),
			expectedResult: nil,
			expectedPos:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match([]Token{tt.input})
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v, diff %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestSeq_Match(t *testing.T) {
	target := Seq{LiteralNode("abc"), LiteralNode("def")}

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "シーケンスと一致",
			input:          Tokenize("abc def"),
			expectedResult: SeqResult{LiteralResult("abc"), LiteralResult("def")},
			expectedPos:    2,
		},
		{
			name:           "シーケンスと不一致",
			input:          Tokenize("abc ghi"),
			expectedResult: nil,
			expectedPos:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v diff %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestSeq_Candidates(t *testing.T) {
	target := Seq{LiteralNode("abc"), LiteralNode("def")}

	tests := []struct {
		name           string
		input          []Token
		expectedPos    int
		expectedResult []NodeId
	}{
		{
			name:           "シーケンスと一致",
			input:          Tokenize("abc def"),
			expectedPos:    7,
			expectedResult: []NodeId{},
		},
		{
			name:           "シーケンスと不一致",
			input:          Tokenize("abc ghi"),
			expectedPos:    4,
			expectedResult: []NodeId{NodeLiteral},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodeIds := Candidates(target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			if !cmp.Equal(nodeIds, tt.expectedResult) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, nodeIds, tt.expectedResult, cmp.Diff(nodeIds, tt.expectedResult))
			}
		})
	}
}

func TestOption_Match(t *testing.T) {
	target := Option{LiteralNode("abc")}

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "オプションあり",
			input:          Tokenize("abc"),
			expectedResult: LiteralResult("abc"),
			expectedPos:    1,
		},
		{
			name:           "オプションなし",
			input:          Tokenize("def"),
			expectedResult: OptionResult{},
			expectedPos:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v, diff %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestOption_Candidates(t *testing.T) {
	target := Option{LiteralNode("abc")}

	tests := []struct {
		name           string
		input          []Token
		expectedPos    int
		expectedResult []NodeId
	}{
		{
			name:           "オプションあり",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []NodeId{},
		},
		{
			name:           "オプションなし",
			input:          []Token{word(0, "def")},
			expectedPos:    0,
			expectedResult: []NodeId{NodeLiteral},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodeIds := Candidates(target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			if !cmp.Equal(nodeIds, tt.expectedResult) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, nodeIds, tt.expectedResult, cmp.Diff(nodeIds, tt.expectedResult))
			}
		})
	}
}

func TestOr_Match(t *testing.T) {
	target := Or{
		LiteralNode("abc"),
		LiteralNode("def"),
	}

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "選択肢1",
			input:          Tokenize("abc"),
			expectedResult: LiteralResult("abc"),
			expectedPos:    1,
		},
		{
			name:           "選択肢2",
			input:          Tokenize("def"),
			expectedResult: LiteralResult("def"),
			expectedPos:    1,
		},
		{
			name:           "不一致",
			input:          Tokenize("ghi"),
			expectedResult: nil,
			expectedPos:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v, diff %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestOr_Candidates(t *testing.T) {
	target := Or{
		LiteralNode("abc"),
		LiteralNode("def"),
	}

	tests := []struct {
		name           string
		input          []Token
		expectedPos    int
		expectedResult []NodeId
	}{
		{
			name:           "選択肢1",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []NodeId{},
		},
		{
			name:           "選択肢2",
			input:          Tokenize("def"),
			expectedPos:    3,
			expectedResult: []NodeId{},
		},
		{
			name:           "不一致",
			input:          Tokenize("ghi"),
			expectedPos:    0,
			expectedResult: []NodeId{NodeLiteral, NodeLiteral},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodeIds := Candidates(target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			if !cmp.Equal(nodeIds, tt.expectedResult) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, nodeIds, tt.expectedResult, cmp.Diff(nodeIds, tt.expectedResult))
			}
		})
	}
}

func TestRepeat_Match(t *testing.T) {
	target := Repeat{LiteralNode("abc")}

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "1つの要素",
			input:          Tokenize("abc"),
			expectedResult: RepeatResult{LiteralResult("abc")},
			expectedPos:    1,
		},
		{
			name:           "複数の要素",
			input:          Tokenize("abc abc"),
			expectedResult: RepeatResult{LiteralResult("abc"), LiteralResult("abc")},
			expectedPos:    2,
		},
		{
			name:           "要素なし",
			input:          []Token{},
			expectedResult: nil,
			expectedPos:    0,
		},
		{
			name:           "コロンが来ると終端する",
			input:          Tokenize("abc:def"),
			expectedResult: RepeatResult{LiteralResult("abc")},
			expectedPos:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v, diff = %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestRepeatOptionOr_Match(t *testing.T) {
	// Seq 内に Option, Or が含まれる場合のテスト

	target := Seq{
		Option{LiteralNode("abc")},
		Or{
			LiteralNode("def"),
			LiteralNode("ghi"),
		},
	}

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "Optionがマッチする場合",
			input:          Tokenize("abc def"),
			expectedResult: SeqResult{LiteralResult("abc"), LiteralResult("def")},
			expectedPos:    2,
		},
		{
			name:           "Optionがマッチしない場合",
			input:          Tokenize("def"),
			expectedResult: SeqResult{OptionResult{}, LiteralResult("def")},
			expectedPos:    1,
		},
		{
			name:           "Orがマッチしない場合",
			input:          Tokenize("abc"),
			expectedResult: nil,
			expectedPos:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v, diff = %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestCandidates(t *testing.T) {
	target := Seq{
		Option{LiteralNode("xyz")},
		Or{
			LiteralNode("uvw"),
			ColonNode{},
		},
	}

	tests := []struct {
		name           string
		input          []Token
		expectedInt    int
		expectedResult []NodeId
	}{
		{
			name:           "最後までマッチする場合",
			input:          Tokenize("xyz uvw"),
			expectedInt:    7,
			expectedResult: []NodeId{},
		},
		{
			name:           "Optionがマッチしない場合",
			input:          Tokenize("uvw"),
			expectedInt:    3,
			expectedResult: []NodeId{},
		},
		{
			name:           "オプションだけマッチした場合",
			input:          Tokenize("xyz"),
			expectedInt:    3,
			expectedResult: []NodeId{NodeLiteral, NodeColon},
		},
		{
			name:           "入力が空の場合",
			input:          Tokenize(""),
			expectedInt:    0,
			expectedResult: []NodeId{NodeLiteral, NodeLiteral, NodeColon},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodeIds := Candidates(target, tt.input)
			if pos != tt.expectedInt {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedInt)
			}
			if !cmp.Equal(nodeIds, tt.expectedResult) {
				t.Errorf("CandidatesResults() = %v, want %v, diff %v", nodeIds, tt.expectedResult, cmp.Diff(nodeIds, tt.expectedResult))
			}
		})
	}
}

func TestSimpleNode(t *testing.T) {
	target := SimpleNodeWithFunc(NodeLiteral, "abc", func(s string) MatchResult {
		if s == "abc" {
			return LiteralResult(s)
		}
		return nil
	})

	if target.Id() != NodeLiteral {
		t.Errorf("SimpleNode.Id() = %v, want %v", target.Id(), NodeLiteral)
	}

	tests := []struct {
		name                  string
		input                 []Token
		expectedMatchResult   MatchResult
		expectedMatchPos      int
		expectedCandidates    []NodeId
		expectedCandidatesPos int
	}{
		{
			name:                  "マッチする場合",
			input:                 Tokenize("abc"),
			expectedMatchResult:   LiteralResult("abc"),
			expectedMatchPos:      1,
			expectedCandidates:    []NodeId{},
			expectedCandidatesPos: 3,
		},
		{
			name:                  "マッチしない場合",
			input:                 Tokenize("def"),
			expectedMatchResult:   nil,
			expectedMatchPos:      0,
			expectedCandidates:    []NodeId{NodeLiteral},
			expectedCandidatesPos: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedMatchResult) {
				t.Errorf("Match(%v) = %v, want %v, diff = %v", tt.input, result, tt.expectedMatchResult, cmp.Diff(result, tt.expectedMatchResult))
			}
			if pos != tt.expectedMatchPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedMatchPos)
			}

			candidatesPos, candidates := Candidates(target, tt.input)
			if candidatesPos != tt.expectedCandidatesPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, candidatesPos, tt.expectedCandidatesPos)
			}
			if !cmp.Equal(candidates, tt.expectedCandidates) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, candidates, tt.expectedCandidates, cmp.Diff(candidates, tt.expectedCandidates))
			}
		})
	}
}

func TestIPAddressNode(t *testing.T) {
	target := IPAddressNode

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "IPアドレス",
			input:          Tokenize("192.168.0.1"),
			expectedResult: IPAddressResult(net.ParseIP("192.168.0.1")),
			expectedPos:    1,
		},
		{
			name:           "不正なIPアドレス",
			input:          Tokenize("192.168.0.256"),
			expectedResult: nil,
			expectedPos:    0,
		},
		{
			name:           "空文字列",
			input:          Tokenize(""),
			expectedResult: nil,
			expectedPos:    0,
		},
		{
			name:           "無効なIPアドレス形式",
			input:          Tokenize("1.2.3.4.5"),
			expectedResult: nil,
			expectedPos:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedResult) {
				t.Errorf("Match(%v) = %v, want %v, diff = %v", tt.input, result, tt.expectedResult, cmp.Diff(result, tt.expectedResult))
			}
			if pos != tt.expectedPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedPos)
			}
		})
	}
}

func TestCommandNode_Candidates(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		input          []Token
		expectedPos    int
		expectedResult []NodeId
	}{
		{
			name:           "コマンド名が一致",
			command:        "abc",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []NodeId{},
		},
		{
			name:           "コマンド名が不一致",
			command:        "abc",
			input:          Tokenize("def"),
			expectedPos:    0,
			expectedResult: []NodeId{NodeCommand},
		},
		{
			name:           "コマンド名が空文字列のときは常に候補とする",
			command:        "",
			input:          Tokenize("abc"),
			expectedPos:    0,
			expectedResult: []NodeId{NodeCommand},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := CommandNode(tt.command)
			pos, nodeIds := Candidates(target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			if !cmp.Equal(nodeIds, tt.expectedResult) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, nodeIds, tt.expectedResult, cmp.Diff(nodeIds, tt.expectedResult))
			}
		})
	}
}

func TestSeqOption_Candidates(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		input    string
		expected []NodeId
	}{
		{
			name:     "Option(CommandNode)",
			node:     Option{CommandNode("")}, // 任意のコマンド(オプション)
			input:    "",
			expected: []NodeId{NodeCommand},
		},
		{
			name:     "LiteralNode",
			node:     LiteralNode("test"),
			input:    "",
			expected: []NodeId{NodeLiteral},
		},
		{
			name: "Seq{Option(CommandNode), LiteralNode}",
			node: Seq{
				Option{CommandNode("")}, // 任意のコマンド(オプション)
				LiteralNode("test"),
			},
			input:    "",
			expected: []NodeId{NodeCommand, NodeLiteral},
		},
	}

	for _, tt := range tests {
		input := Tokenize(tt.input)
		t.Run(tt.name, func(t *testing.T) {
			_, nodeIds := Candidates(tt.node, input)
			if !cmp.Equal(nodeIds, tt.expected) {
				t.Errorf("SeqOption_Candidates(%v) = %v, want %v diff %v", tt.name, nodeIds, tt.expected, cmp.Diff(nodeIds, tt.expected))
			}
		})
	}
}

func TestDeviceSpecifier_Candidates(t *testing.T) {
	target := DeviceSpecifierNode

	tests := []struct {
		name           string
		input          []Token
		expectedPos    int
		expectedResult []NodeId
	}{
		{
			name:           "デバイス指定子",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []NodeId{},
		},
		{
			name:           "空文字列",
			input:          Tokenize(""),
			expectedPos:    0,
			expectedResult: []NodeId{NodeIPAddress, NodeClassCode, NodeDeviceAlias},
		},
		{
			name:           "IPアドレス",
			input:          Tokenize("192.168.1.1"),
			expectedPos:    11,
			expectedResult: []NodeId{NodeClassCode},
		},
		{
			name:           "クラスコード",
			input:          Tokenize("0130"),
			expectedPos:    4,
			expectedResult: []NodeId{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodeIds := Candidates(target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			if !cmp.Equal(nodeIds, tt.expectedResult) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, nodeIds, tt.expectedResult, cmp.Diff(nodeIds, tt.expectedResult))
			}
		})
	}
}

func TestTraverse(t *testing.T) {
	target := Seq{
		Option{CommandNode("")}, // 任意のコマンド(オプション)
		LiteralNode("test"),
		Or{
			LiteralNode("abc"),
			LiteralNode("def"),
		},
		Repeat{LiteralNode("ghi")},
	}
	var nodeIds []NodeId
	_ = Traverse(target, func(node Node) error {
		nodeIds = append(nodeIds, node.Id())
		return nil
	})

	expected := []NodeId{
		NodeSeq,
		NodeOption,
		NodeCommand,
		NodeLiteral,
		NodeOr,
		NodeLiteral,
		NodeLiteral,
		NodeRepeat,
		NodeLiteral,
	}
	if !cmp.Equal(nodeIds, expected) {
		t.Errorf("Traverse() = %v, want %v, diff %v", nodeIds, expected, cmp.Diff(nodeIds, expected))
	}
}

func TestCollectStrings(t *testing.T) {
	target := Or{
		CommandNode("save"),
		CommandNode("load"),
		ColonNode{},
		CommandNode("list"),
		CommandNode("run"),
	}
	strs, err := CollectStrings(target, NodeCommand)
	if err != nil {
		t.Errorf("CollectStrings() returned an error: %v", err)
	}

	expected := []string{"save", "load", "list", "run"}
	if !cmp.Equal(strs, expected) {
		t.Errorf("CollectStrings() = %v, want %v, diff %v", strs, expected, cmp.Diff(strs, expected))
	}

	_, err = CollectStrings(target, NodeColon)
	if err == nil {
		t.Error("CollectStrings() did not return an error")
	}
}

func TokensToString(tokens []Token) string {
	var s []string
	for _, token := range tokens {
		if token.Type == TokenEOF {
			if token.String != "" {
				s = append(s, "")
			}
			break
		}
		s = append(s, token.String)
	}
	return strings.Join(s, " ")
}

func TestSplitLastWord(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedBefore   string
		expectedLastWord string
	}{
		{
			name:             "通常の入力(1語)",
			input:            "abc",
			expectedBefore:   "",
			expectedLastWord: "abc",
		},
		{
			name:             "通常の入力",
			input:            "abc def",
			expectedBefore:   "abc",
			expectedLastWord: "def",
		},
		{
			name:             "末尾に空白がある入力",
			input:            "abc def ",
			expectedBefore:   "abc def",
			expectedLastWord: "",
		},
		{
			name:             "複数の空白を含む入力",
			input:            "  abc  def  ",
			expectedBefore:   "abc def",
			expectedLastWord: "",
		},
		{
			name:             "空文字列の入力",
			input:            "",
			expectedBefore:   "",
			expectedLastWord: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBefore, gotLastWord := SplitLastWord(Tokenize(tt.input))
			expectedBefore := Tokenize(tt.expectedBefore)
			if TokensToString(gotBefore) != TokensToString(expectedBefore) || gotLastWord != tt.expectedLastWord {
				t.Errorf("SplitLastWord(%q) = (%#v, %#v), want (%#v, %#v)", tt.input, TokensToString(gotBefore), gotLastWord, TokensToString(expectedBefore), tt.expectedLastWord)
			}
		})
	}
}
