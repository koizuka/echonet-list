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

type TestNode struct {
	Id     NodeId
	String string
}

func toTestNode(node Node) TestNode {
	switch v := node.(type) {
	case CompositeNode:
		return TestNode{node.Id(), v.String}
	default:
		return TestNode{node.Id(), ""}
	}
}

func toTestNodes(nodes []Node) []TestNode {
	var testNodes []TestNode
	for _, node := range nodes {
		testNodes = append(testNodes, toTestNode(node))
	}
	return testNodes
}

// DummyCompleter は、テスト用のダミーコンプリータ。 implements CompleterInterface
type DummyCompleter struct {
}

func (d DummyCompleter) getDeviceCandidates() []string {
	return []string{"device1", "device2"}
}
func (d DummyCompleter) getDeviceAliasCandidates() []string {
	return []string{"alias1", "alias2"}
}
func (d DummyCompleter) getPropertyAliasCandidates() []string {
	return []string{"property1", "property2"}
}
func (d DummyCompleter) getCommandCandidates() []string {
	return []string{"command1", "command2"}
}

func TestSeq_Candidates(t *testing.T) {
	target := Seq{LiteralNode("abc"), LiteralNode("def")}

	tests := []struct {
		name           string
		input          []Token
		expectedPos    int
		expectedResult []Node
	}{
		{
			name:           "シーケンスと一致",
			input:          Tokenize("abc def"),
			expectedPos:    7,
			expectedResult: []Node{},
		},
		{
			name:           "シーケンスと不一致",
			input:          Tokenize("abc ghi"),
			expectedPos:    4,
			expectedResult: []Node{LiteralNode("def")},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
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
		expectedResult []Node
	}{
		{
			name:           "オプションあり",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []Node{},
		},
		{
			name:           "オプションなし",
			input:          Tokenize("def"),
			expectedPos:    0,
			expectedResult: []Node{LiteralNode("abc")},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
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
		expectedResult []Node
	}{
		{
			name:           "選択肢1",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []Node{},
		},
		{
			name:           "選択肢2",
			input:          Tokenize("def"),
			expectedPos:    3,
			expectedResult: []Node{},
		},
		{
			name:           "不一致",
			input:          Tokenize("ghi"),
			expectedPos:    0,
			expectedResult: []Node{LiteralNode("abc"), LiteralNode("def")},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
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
		expectedResult []Node
	}{
		{
			name:           "最後までマッチする場合",
			input:          Tokenize("xyz uvw"),
			expectedInt:    7,
			expectedResult: []Node{},
		},
		{
			name:           "Optionがマッチしない場合",
			input:          Tokenize("uvw"),
			expectedInt:    3,
			expectedResult: []Node{},
		},
		{
			name:           "オプションだけマッチした場合",
			input:          Tokenize("xyz"),
			expectedInt:    3,
			expectedResult: []Node{LiteralNode("uvw"), ColonNode{}},
		},
		{
			name:           "入力が空の場合",
			input:          Tokenize(""),
			expectedInt:    0,
			expectedResult: []Node{LiteralNode("xyz"), LiteralNode("uvw"), ColonNode{}},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedInt {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedInt)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("CandidatesResults() = %v, want %v, diff %v", testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
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
		expectedCandidates    []Node
		expectedCandidatesPos int
	}{
		{
			name:                  "マッチする場合",
			input:                 Tokenize("abc"),
			expectedMatchResult:   LiteralResult("abc"),
			expectedMatchPos:      1,
			expectedCandidates:    []Node{},
			expectedCandidatesPos: 3,
		},
		{
			name:                  "マッチしない場合",
			input:                 Tokenize("def"),
			expectedMatchResult:   nil,
			expectedMatchPos:      0,
			expectedCandidates:    []Node{LiteralNode("abc")},
			expectedCandidatesPos: 0,
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pos := target.Match(tt.input)
			if !cmp.Equal(result, tt.expectedMatchResult) {
				t.Errorf("Match(%v) = %v, want %v, diff = %v", tt.input, result, tt.expectedMatchResult, cmp.Diff(result, tt.expectedMatchResult))
			}
			if pos != tt.expectedMatchPos {
				t.Errorf("Match(%v).Pos() = %v, want %v", tt.input, pos, tt.expectedMatchPos)
			}

			candidatesPos, candidates := Candidates(dc, target, tt.input)
			if candidatesPos != tt.expectedCandidatesPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, candidatesPos, tt.expectedCandidatesPos)
			}
			testCandidates := toTestNodes(candidates)
			expectedTestCandidates := toTestNodes(tt.expectedCandidates)
			if !cmp.Equal(testCandidates, expectedTestCandidates) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, testCandidates, expectedTestCandidates, cmp.Diff(testCandidates, expectedTestCandidates))
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

func TestDeviceAliasNode(t *testing.T) {
	target := DeviceAliasNode

	tests := []struct {
		name           string
		input          []Token
		expectedResult MatchResult
		expectedPos    int
	}{
		{
			name:           "デバイスエイリアス",
			input:          Tokenize("abc"),
			expectedResult: DeviceAliasResult("abc"),
			expectedPos:    1,
		},
		{
			name:           "空文字列",
			input:          Tokenize(""),
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

func TestDeviceAliasNode_Candidates(t *testing.T) {
	target := DeviceAliasNode

	tests := []struct {
		name           string
		input          []Token
		expectedPos    int
		expectedResult []Node
	}{
		{
			name:           "デバイスエイリアス",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []Node{},
		},
		{
			name:           "空文字列",
			input:          Tokenize(""),
			expectedPos:    0,
			expectedResult: []Node{DeviceAliasNode},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
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
		expectedResult []Node
	}{
		{
			name:           "コマンド名が一致",
			command:        "abc",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []Node{},
		},
		{
			name:           "コマンド名が不一致",
			command:        "abc",
			input:          Tokenize("def"),
			expectedPos:    0,
			expectedResult: []Node{CommandNode("abc")},
		},
		{
			name:           "コマンド名が空文字列のときは常に候補とする",
			command:        "",
			input:          Tokenize("abc"),
			expectedPos:    0,
			expectedResult: []Node{CommandNode("")},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := CommandNode(tt.command)
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("Candidates(%v) = %v, want %v, diff %v", tt.input, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
			}
		})
	}
}

func TestSeqOption_Candidates(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		input    string
		expected []Node
	}{
		{
			name:     "Option(CommandNode)",
			node:     Option{CommandNode("")}, // 任意のコマンド(オプション)
			input:    "",
			expected: []Node{CommandNode("")},
		},
		{
			name:     "LiteralNode",
			node:     LiteralNode("test"),
			input:    "",
			expected: []Node{LiteralNode("test")},
		},
		{
			name: "Seq{Option(CommandNode), LiteralNode}",
			node: Seq{
				Option{CommandNode("")}, // 任意のコマンド(オプション)
				LiteralNode("test"),
			},
			input:    "",
			expected: []Node{CommandNode(""), LiteralNode("test")},
		},
	}

	dc := DummyCompleter{}

	for _, tt := range tests {
		input := Tokenize(tt.input)
		t.Run(tt.name, func(t *testing.T) {
			_, nodes := Candidates(dc, tt.node, input)
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expected)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("SeqOption_Candidates(%v) = %v, want %v, diff %v", tt.name, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
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
		expectedResult []Node
	}{
		{
			name:           "デバイスエイリアス",
			input:          Tokenize("abc"),
			expectedPos:    3,
			expectedResult: []Node{},
		},
		{
			name:           "空文字列",
			input:          Tokenize(""),
			expectedPos:    0,
			expectedResult: []Node{DeviceSpecifierNode},
		},
		{
			name:           "IPアドレス",
			input:          Tokenize("192.168.1.1"),
			expectedPos:    11,
			expectedResult: []Node{DeviceSpecifierNode}, // Optionが残ってるからまだ足せる
		},
		{
			name:           "クラスコード",
			input:          Tokenize("0130"),
			expectedPos:    4,
			expectedResult: []Node{},
		},
		{
			name:           "IP クラスコード",
			input:          Tokenize("192.168.1.1 0130"),
			expectedPos:    16,
			expectedResult: []Node{},
		}}

	dc := DummyCompleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, nodes := Candidates(dc, target, tt.input)
			if pos != tt.expectedPos {
				t.Errorf("Candidates pos(%v) = %v, want %v", tt.input, pos, tt.expectedPos)
			}
			testNodes := toTestNodes(nodes)
			expectedTestNodes := toTestNodes(tt.expectedResult)
			if !cmp.Equal(testNodes, expectedTestNodes) {
				t.Errorf("Candidates nodes(%v) = %v, want %v, diff %v", tt.input, testNodes, expectedTestNodes, cmp.Diff(testNodes, expectedTestNodes))
			}
		})
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
