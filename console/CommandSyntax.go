package console

import (
	"encoding/hex"
	"fmt"
	"net"
	"slices"
	"strconv"
)

// コマンドの構文を定義する

type TokenType int

const (
	TokenWord  TokenType = iota // 一般的な単語( '.' も含む)
	TokenColon                  // ':'
	TokenEOF                    // 末尾。末尾に空白がある場合、それがStringに入る
)

type Token struct {
	Type   TokenType
	Pos    int // トークンの開始位置(Rune単位)
	String string
}

type NodeId int

const (
	NodeOr NodeId = iota
	NodeSeq
	NodeOption
	NodeRepeat

	NodeWord
	NodeColon

	NodeLiteral
	NodeCommand
	NodeIPAddress
	NodeClassCode
	NodeInstanceCode
	NodeEOJ
	NodeDeviceAlias
	NodeDeviceSpecifier
	NodeEPC
	NodePropertyValue
	NodePropertyValueAlias
	NodePropertyAlias
	NodeProperty
	NodeFilterCriteria

	NodeGetOptions
	NodeOnOff
	NodeDeleteOption
	NodeListOptions

	NodeLast
)

func (i NodeId) String() string {
	switch i {
	case NodeOr:
		return "Or"
	case NodeSeq:
		return "Seq"
	case NodeOption:
		return "Option"
	case NodeRepeat:
		return "Repeat"
	case NodeWord:
		return "Word"
	case NodeColon:
		return "Colon"
	case NodeLiteral:
		return "Literal"
	case NodeCommand:
		return "Command"
	case NodeIPAddress:
		return "IPAddress"
	case NodeClassCode:
		return "ClassCode"
	case NodeInstanceCode:
		return "InstanceCode"
	case NodeEOJ:
		return "EOJ"
	case NodeDeviceAlias:
		return "DeviceAlias"
	case NodeDeviceSpecifier:
		return "DeviceSpecifier"
	case NodeEPC:
		return "EPC"
	case NodePropertyValue:
		return "PropertyValue"
	case NodePropertyValueAlias:
		return "PropertyValueAlias"
	case NodePropertyAlias:
		return "PropertyAlias"
	case NodeProperty:
		return "Property"
	case NodeFilterCriteria:
		return "FilterCriteria"
	case NodeGetOptions:
		return "GetOptions"
	case NodeOnOff:
		return "OnOff"
	case NodeDeleteOption:
		return "DeleteOption"
	case NodeListOptions:
		return "ListOptions"

	case NodeLast:
		return "Last"
	}
	return fmt.Sprintf("NodeId(%d)", i)
}

type MatchResult interface {
}

type Traversable interface {
	TraverseInside(func(Node) error) error
}

func Traverse(node Node, f func(Node) error) error {
	err := f(node)
	if err != nil {
		return err
	}
	if t, ok := node.(Traversable); ok {
		return t.TraverseInside(f)
	}
	return nil
}

// Node は、コマンドの構文を表す非終端文字列を表すインターフェース
type Node interface {
	Id() NodeId

	// Match は、 入力トークン列の先頭からこのノードにマッチするときに、マッチ結果(MatchResult)と、このトークンにマッチする部分のトークン数を返す
	Match(input []Token) (MatchResult, int)
}

type CandidatesNode interface {
	// 現在の入力トークン状況から、最後の単語についての候補列を返す
	Candidates(input []Token) (int, []NodeId)
}

func Candidates(node Node, input []Token) (int, []NodeId) {
	inner := node
	if comp, ok := node.(CompositeNode); ok { // TODO
		inner = comp.Node
	}

	if c, ok := inner.(CandidatesNode); ok {
		return c.Candidates(input)
	}

	r, i := node.Match(input)
	if r != nil {
		return input[i].Pos, []NodeId{}
	}
	return 0, []NodeId{node.Id()}
}

// Or は Node の集合を表す。Or にマッチするとき、その中のどれか一つにマッチする
type Or []Node

func (b Or) Id() NodeId {
	return NodeOr
}

func (b Or) Match(input []Token) (MatchResult, int) {
	for _, node := range b {
		result, len := node.Match(input)
		if result != nil {
			return result, len
		}
	}
	return nil, 0
}

func (b Or) Candidates(input []Token) (int, []NodeId) {
	resultCandidates := make([]NodeId, 0, len(b))
	resultPos := 0 // トークンの位置が最も遠いものを選ぶ
	for _, node := range b {
		pos, candidates := Candidates(node, input)
		if pos > resultPos {
			resultCandidates = candidates
			resultPos = pos
		} else if pos == resultPos {
			resultCandidates = append(resultCandidates, candidates...)
		}
	}
	return resultPos, resultCandidates
}

func (b Or) TraverseInside(f func(Node) error) error {
	for _, node := range b {
		if err := Traverse(node, f); err != nil {
			return err
		}
	}
	return nil
}

// Option は、Node がマッチしなくてもよいことを表すNode
type Option struct {
	Node Node
}

func (o Option) Id() NodeId {
	return NodeOption
}

func (o Option) TraverseInside(f func(Node) error) error {
	return Traverse(o.Node, f)
}

// OptionResult は、OptionNode にマッチしなかったときの結果。処理するときはスキップする
type OptionResult struct{}

func (o Option) Match(input []Token) (MatchResult, int) {
	if result, len := o.Node.Match(input); result != nil {
		return result, len
	}
	return OptionResult{}, 0
}

func (o Option) Candidates(input []Token) (int, []NodeId) {
	return Candidates(o.Node, input)
}

// Seq は、Node のリストを表す。
type Seq []Node

func (l Seq) Id() NodeId {
	return NodeSeq
}

type SeqResult []MatchResult

func (l Seq) Match(input []Token) (MatchResult, int) {
	result := SeqResult{}
	len := 0
	for _, node := range l {
		r, l := node.Match(input[len:])
		if r == nil {
			return nil, 0
		}
		result = append(result, r)
		len += l
	}
	return result, len
}

// Candidates は、入力トークン列の先頭からこのノードにマッチするときに、次に続くトークンの候補を返す
// この場合、最初のノードにマッチするトークンの位置と、次に続くノードの候補を返す
func (l Seq) Candidates(input []Token) (int, []NodeId) {
	var candidates []NodeId

	i := 0 // index の要素番号(トークンの位置)
	n := 0 // l の要素番号(ノードの位置)

	// 先頭から　Match 成功する分は Matchの返したtoken数分スキップする
	for i < len(input) && n < len(l) {
		r, m := l[n].Match(input[i:])
		_, c := Candidates(l[n], input[i:])
		if r == nil {
			candidates = append(candidates, c...)
			break
		}
		if m == 0 {
			// マッチしたのに長さが0なら、Optionでマッチしなかったケースなので、候補に加えて続行する
			candidates = append(candidates, c...)
		} else {
			candidates = []NodeId{}
		}
		i += m
		n++
	}

	return input[i].Pos, candidates
}

func (l Seq) TraverseInside(f func(Node) error) error {
	for _, node := range l {
		if err := Traverse(node, f); err != nil {
			return err
		}
	}
	return nil
}

// Repeat は、1個以上のNode の列にマッチする。
type Repeat struct {
	Node Node
}

func (a Repeat) Id() NodeId {
	return NodeRepeat
}

type RepeatResult []MatchResult

func (a Repeat) Match(input []Token) (MatchResult, int) {
	results := RepeatResult{}
	len := 0
	for {
		result, l := a.Node.Match(input[len:])
		if result == nil {
			break
		}
		results = append(results, result)
		len += l
		if l == 0 {
			// 長さ0でマッチしたら無限ループになるので終了する
			break
		}
	}
	if len == 0 {
		return nil, 0
	}
	return results, len
}

func (a Repeat) Candidates(input []Token) (int, []NodeId) {
	return Candidates(a.Node, input)
}

func (a Repeat) TraverseInside(f func(Node) error) error {
	return Traverse(a.Node, f)
}

type CompositeNode struct {
	NodeId      NodeId
	Node        Node
	BuildResult func(MatchResult) MatchResult
	String      string
}

func (h CompositeNode) Id() NodeId {
	return h.NodeId
}

func (h CompositeNode) Match(input []Token) (MatchResult, int) {
	if r, n := h.Node.Match(input); r != nil {
		if result := h.BuildResult(r); result != nil {
			return result, n
		}
	}
	return nil, 0
}

// CompositeNode は TraverseInside に対して内側を隠蔽する
func (h CompositeNode) TraverseInside(func(Node) error) error {
	return nil
}

// CollectStrings は、指定したNodeId にマッチするCompositeNode のStringを収集する
func CollectStrings(root Node, id NodeId) ([]string, error) {
	strings := []string{}
	err := Traverse(root, func(node Node) error {
		if node.Id() == id {
			switch v := node.(type) {
			case CompositeNode:
				if v.String != "" {
					strings = append(strings, v.String)
				}
			default:
				return fmt.Errorf("unexpected node type: %T", node)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return strings, nil
}

type WordNode struct{}

func (n WordNode) Id() NodeId {
	return NodeWord
}

type WordResult string

func (n WordNode) Match(input []Token) (MatchResult, int) {
	if len(input) > 0 && input[0].Type == TokenWord {
		return WordResult(input[0].String), 1
	}
	return nil, 0
}

type ColonNode struct{}

func (n ColonNode) Id() NodeId {
	return NodeColon
}

type ColonResult struct{}

func (c ColonNode) Match(input []Token) (MatchResult, int) {
	if len(input) > 0 && input[0].Type == TokenColon {
		return ColonResult{}, 1
	}
	return nil, 0
}

// SimpleNodeWithFunc は、WordNode にマッチした文字列を関数に渡して、その結果を返すNode
func SimpleNodeWithFunc(id NodeId, s string, f func(string) MatchResult) Node {
	return CompositeNode{
		NodeId: id,
		Node:   WordNode{},
		String: s,
		BuildResult: func(r MatchResult) MatchResult {
			return f(string(r.(WordResult)))
		},
	}
}

func SimpleNode[T interface {
	~string
	MatchResult
}](id NodeId, s string) Node {
	return SimpleNodeWithFunc(id, s, func(t string) MatchResult {
		if t == s {
			return T(s)
		}
		return nil
	})
}

// NameNode は、WordNode にマッチした文字列をそのまま返すNode。名前を表す
func NameNode[T interface {
	~string
	MatchResult
}](id NodeId) Node {
	return SimpleNodeWithFunc(id, "", func(s string) MatchResult {
		return T(s)
	})
}

type LiteralResult string

func LiteralNode(s string) Node {
	return SimpleNode[LiteralResult](NodeLiteral, s)
}

type CommandResult string

func CommandNode(command string) Node {
	return SimpleNodeWithFunc(NodeCommand, command, func(s string) MatchResult {
		if command == "" {
			return nil
		}
		if s == command {
			return CommandResult(s)
		}
		return nil
	})
}

type IPAddressResult net.IP

var IPAddressNode = SimpleNodeWithFunc(
	NodeIPAddress,
	"",
	func(s string) MatchResult {
		if ip := net.ParseIP(s); ip != nil {
			return IPAddressResult(ip)
		}
		return nil
	},
)

type ClassCodeResult struct {
	ClassCode uint16
}

var ClassCodeNode = SimpleNodeWithFunc(
	NodeClassCode,
	"",
	func(s string) MatchResult {
		if len(s) != 4 {
			return nil
		}
		code, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			return nil
		}
		return ClassCodeResult{ClassCode: uint16(code)}
	},
)

type InstanceCodeNode struct{}

type InstanceCodeResult struct {
	InstanceCode uint8
}

func (ic InstanceCodeNode) Id() NodeId {
	return NodeInstanceCode
}

func (ic InstanceCodeNode) Match(input []Token) (MatchResult, int) {
	if len(input) < 2 || input[0].Type != TokenColon || input[1].Type != TokenWord {
		return nil, 0
	}
	// 10進数1桁
	if len(input[0].String) == 1 {
		code, err := strconv.ParseUint(string(input[0].String), 10, 8)
		if err == nil {
			return InstanceCodeResult{InstanceCode: uint8(code)}, 1
		}
	}
	return nil, 0
}

type EOJResult struct {
	ClassCode    uint16
	InstanceCode uint8
}

// EOJNode は、EOJを表すNode。ClassCode:InstanceCode の形式にマッチする
var EOJNode = CompositeNode{
	NodeId: NodeEOJ,
	Node: Seq{
		ClassCodeNode,
		Option{InstanceCodeNode{}},
	},
	BuildResult: func(r MatchResult) MatchResult {
		classCode := r.(SeqResult)[0].(ClassCodeResult).ClassCode
		var instanceCode uint8
		switch v := r.(SeqResult)[1].(type) {
		case InstanceCodeResult:
			instanceCode = v.InstanceCode
		}
		return EOJResult{
			ClassCode:    classCode,
			InstanceCode: instanceCode,
		}
	},
}

type DeviceAliasResult string

var DeviceAliasNode = NameNode[DeviceAliasResult](NodeDeviceAlias)

// DeviceSpecifierNode は、デバイスを特定するためのNode。 EOJNode または AliasNode にマッチする
var DeviceSpecifierNode = CompositeNode{
	NodeId: NodeDeviceSpecifier,
	Node: Or{
		Seq{
			Option{IPAddressNode},
			Option{EOJNode},
		},
		DeviceAliasNode,
	},
	BuildResult: func(r MatchResult) MatchResult {
		result := DeviceSpecifierResult{}
		switch v := r.(type) {
		case SeqResult:
			switch ip := v[0].(type) {
			case IPAddressResult:
				result.IP = &ip
			}
			switch eoj := v[1].(type) {
			case EOJResult:
				result.EOJ = &eoj
			}
		case DeviceAliasResult:
			result.Alias = &v
		}
		return result
	},
}

type DeviceSpecifierResult struct {
	IP    *IPAddressResult
	EOJ   *EOJResult
	Alias *DeviceAliasResult
}

// 16進数2桁
type EPCNode struct{}

func (EPCNode) Id() NodeId {
	return NodeEPC
}

type EPCResult byte

func (EPCNode) Match(input []Token) (MatchResult, int) {
	if len(input) > 0 && input[0].Type == TokenWord && len(input[0].String) == 2 {
		code, err := strconv.ParseUint(string(input[0].String), 16, 8)
		if err == nil {
			return EPCResult(byte(code)), 1
		}
	}
	return nil, 0
}

// PropertyValueNode は、値を表すNode。16進数で2桁の倍数にマッチする
type PropertyValueNode struct{}

func (PropertyValueNode) Id() NodeId {
	return NodePropertyValue
}

type PropertyValueResult []byte

func (PropertyValueNode) Match(input []Token) (MatchResult, int) {
	// 16進数の2桁の倍数
	if len(input) < 2 || input[0].Type != TokenColon || input[1].Type != TokenWord {
		return nil, 0
	}
	if len(input[1].String)%2 == 0 {
		bytes, err := hex.DecodeString(string(input[1].String))
		if err == nil {
			return PropertyValueResult(bytes), 2
		}
	}
	return nil, 0
}

// PropertyValueAliasNode は、値のエイリアスを表すNode
type PropertyValueAliasNode struct{}

func (PropertyValueAliasNode) Id() NodeId {
	return NodePropertyValueAlias
}

type PropertyValueAliasResult string

func (PropertyValueAliasNode) Match(input []Token) (MatchResult, int) {
	if len(input) < 2 || input[0].Type != TokenColon || input[1].Type != TokenWord {
		return nil, 0
	}
	return PropertyValueAliasResult(input[1].String), 1
}

type PropertyAliasResult string

// PropertyAliasNode は、プロパティのエイリアスを表すNode
var PropertyAliasNode = NameNode[PropertyAliasResult](NodePropertyAlias)

type PropertyResult struct {
	EPC EPCResult
	// PropertyValueResult | PropertyValueAliasResult
	Value interface{}
}

// PropertyNode は、プロパティを表すNode。 EPCNode:(PropertyValueNode | PropertyValueAliasNode) または PropertyAlias にマッチする
var PropertyNode = CompositeNode{
	NodeId: NodeProperty,
	Node: Or{
		PropertyAliasNode,
		Seq{
			EPCNode{},
			Or{
				PropertyValueNode{},
				PropertyValueAliasNode{},
			},
		},
	},
	BuildResult: func(r MatchResult) MatchResult {
		result := PropertyResult{}
		switch v := r.(type) {
		case PropertyAliasResult:
			result.Value = v
		case SeqResult:
			result.EPC = v[0].(EPCResult)
			switch v := v[1].(type) {
			case PropertyValueResult:
				result.Value = v
			case PropertyValueAliasResult:
				result.Value = v
			}
		}
		return result
	},
}

var PropertiesNode = Repeat{PropertyNode}

// FilterCriteriaNode は、フィルタ条件を表すNode
type FilterCriteriaResult struct {
	DeviceSpecifier DeviceSpecifierResult
	Properties      []PropertyResult
}

var FilterCriteriaNode = CompositeNode{
	NodeId: NodeFilterCriteria,
	Node: Seq{
		Option{DeviceSpecifierNode},
		Option{PropertiesNode},
	},
	BuildResult: func(r MatchResult) MatchResult {
		result := FilterCriteriaResult{}
		for _, m := range r.(SeqResult) {
			switch v := m.(type) {
			case DeviceSpecifierResult:
				result.DeviceSpecifier = v
			case RepeatResult:
				p := make([]PropertyResult, 0, len(v))
				for _, a := range v {
					p = append(p, a.(PropertyResult))
				}
				result.Properties = p
			}
		}
		return result
	},
}

type GetOptionsResult string

func GetOptionsNode(s string) Node {
	return SimpleNode[GetOptionsResult](
		NodeGetOptions,
		s,
	)
}

type OnOffResult string

func OnOffNode(s string) Node {
	return SimpleNode[OnOffResult](NodeOnOff, s)
}

type DeleteOptionResult string

func DeleteOptionNode(s string) Node {
	return SimpleNode[DeleteOptionResult](NodeDeleteOption, s)
}

type ListOptionsResult string

func ListOptionsNode(s string) Node {
	return SimpleNode[ListOptionsResult](NodeListOptions, s)
}

// Tokenize は、入力文字列をトークン列に変換する。':' だけは前後に空白がなくてもトークンになるが、それ以外は空白で区切る
func Tokenize(line string) []Token {
	tokens := []Token{}
	word := ""
	space := ""
	start := 0
	for i, rune := range line {
		if rune == ' ' || rune == '\t' {
			if word != "" {
				tokens = append(tokens, Token{Type: TokenWord, Pos: start, String: word})
				word = ""
			}
			space += string(rune)
			continue
		}
		space = ""
		if rune == ':' {
			if word != "" {
				tokens = append(tokens, Token{Type: TokenWord, Pos: start, String: word})
				word = ""
			}
			tokens = append(tokens, Token{Type: TokenColon, Pos: i, String: ":"})
			continue
		}
		if word == "" {
			start = i
		}
		word += string(rune)
	}
	if word != "" {
		tokens = append(tokens, Token{Type: TokenWord, Pos: start, String: word})
	}
	tokens = append(tokens, Token{Type: TokenEOF, Pos: len(line), String: space})
	return tokens
}

// --------------------------------

var CommandSyntax = Or{
	Seq{CommandNode("discover")},
	Seq{
		Or{
			CommandNode("list"),
			CommandNode("devices"),
		},
		Option{
			Repeat{
				Or{
					FilterCriteriaNode,
					ListOptionsNode("-all"),
					ListOptionsNode("-props"),
					EPCNode{},
				},
			},
		},
	},
	Seq{CommandNode("set"), DeviceSpecifierNode, PropertiesNode},
	Seq{CommandNode("get"), DeviceSpecifierNode, Repeat{
		Or{
			GetOptionsNode("-skip-validation"),
			EPCNode{},
		},
	}},
	Seq{CommandNode("update"), FilterCriteriaNode},
	Seq{
		CommandNode("alias"), Option{
			Or{
				Seq{DeviceAliasNode, Option{DeviceSpecifierNode}},
				Seq{DeleteOptionNode("-delete"), DeviceAliasNode},
			},
		},
	},
	Seq{
		CommandNode("debug"),
		Option{
			Or{
				OnOffNode("on"),
				OnOffNode("off"),
			},
		},
	},
	Seq{CommandNode("help"), Option{CommandNode("")}},
	Seq{CommandNode("quit")},
}

func ParseCommand(input string) (MatchResult, int) {
	tokens := Tokenize(input)
	result, len := CommandSyntax.Match(tokens)
	return result, len
}

func SplitLastWord(tokens []Token) ([]Token, string) {
	if len(tokens) > 0 {
		i := len(tokens) - 1
		if tokens[i].Type != TokenEOF {
			fmt.Printf("unexpected token: %v\n", tokens[i])
			return tokens, ""
		}
		if tokens[i].String != "" {
			t := slices.Clone(tokens)
			t[i].String = ""
			return t, ""
		}
		if i > 0 {
			eof := Token{Type: TokenEOF, Pos: tokens[i].Pos, String: ""}
			t := slices.Clone(tokens[:i-1])
			return append(t, eof), tokens[i-1].String
		}
	}
	return tokens, ""
}
