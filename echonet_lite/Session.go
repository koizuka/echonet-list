package echonet_lite

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var NodeProfileObject1 = MakeEOJ(NodeProfile_ClassCode, 1)

// ブロードキャストアドレスの設定
var BroadcastIP = getIPv4BroadcastIP()

// var BroadcastIP = net.ParseIP("ff02::1")

type Key struct {
	TID TIDType
}

func MakeKey(msg *ECHONETLiteMessage) Key {
	return Key{msg.TID}
}

type CallbackCompleteStatus int // プロパティコールバック完了ステータス
const (
	CallbackFinished CallbackCompleteStatus = iota
	CallbackContinue
)

type CallbackFunc func(net.IP, *ECHONETLiteMessage) (CallbackCompleteStatus, error)
type PersistentCallbackFunc func(net.IP, *ECHONETLiteMessage) error

type Entry struct {
	ESVs     []ESVType
	Callback CallbackFunc
}

type DispatchTable map[Key]Entry

func (dt DispatchTable) Register(key Key, ESVs []ESVType, callback CallbackFunc) {
	dt[key] = Entry{ESVs, callback}
}

func (dt DispatchTable) Unregister(key Key) {
	delete(dt, key)
}

func (dt DispatchTable) Dispatch(ip net.IP, msg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
	key := MakeKey(msg)
	entry, ok := dt[key]
	if ok {
		for _, esv := range entry.ESVs {
			if esv == msg.ESV {
				return entry.Callback(ip, msg)
			}
		}
	}
	return CallbackContinue, nil
}

type Session struct {
	mu              sync.RWMutex
	dispatchTable   DispatchTable
	receiveCallback PersistentCallbackFunc
	infCallback     PersistentCallbackFunc
	tid             TIDType
	eoj             EOJ
	conn            *UDPConnection
	Debug           bool
	ctx             context.Context    // コンテキスト
	cancel          context.CancelFunc // コンテキストのキャンセル関数
}

func CreateSession(ctx context.Context, ip net.IP, EOJ EOJ, debug bool) (*Session, error) {
	// タイムアウトなしのコンテキストを作成（キャンセルのみ可能）
	sessionCtx, cancel := context.WithCancel(ctx)

	conn, err := CreateUDPConnection(sessionCtx, ip, UDPConnectionOptions{DefaultTimeout: 30 * time.Second})
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}
	return &Session{
		dispatchTable: make(DispatchTable),
		tid:           TIDType(1),
		eoj:           EOJ,
		conn:          conn,
		Debug:         debug,
		ctx:           sessionCtx,
		cancel:        cancel,
	}, nil
}

func (s *Session) OnInf(callback PersistentCallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.infCallback = callback
}

func (s *Session) OnReceive(callback PersistentCallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receiveCallback = callback
}

func (s *Session) MainLoop() {
	for {
		// DispatchTableがnilかどうかをロックして確認
		s.mu.RLock()
		dt := s.dispatchTable
		isNil := dt == nil
		s.mu.RUnlock()

		if isNil {
			break
		}

		// コンテキストがキャンセルされていないか確認
		select {
		case <-s.ctx.Done():
			fmt.Println("コンテキストがキャンセルされました:", s.ctx.Err())
			return
		default:
			// 継続
		}

		// タイムアウトなしでコンテキストを作成（キャンセルのみ可能）
		receiveCtx, cancel := context.WithCancel(s.ctx)
		data, addr, err := s.conn.Receive(receiveCtx)
		cancel() // 必ずキャンセルする

		if err != nil {
			// コンテキストのキャンセルまたはタイムアウトの場合
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				// タイムアウトの場合は次の受信を待つ
				continue
			}

			// 接続が閉じられた場合
			if err.Error() == "use of closed network connection" {
				fmt.Println("受信終了: 接続が閉じられました")
				break
			}

			// その他のネットワークエラー
			// net.Error.Temporary()はdeprecatedなので、特定のエラータイプで判断する
			if errors.Is(err, net.ErrClosed) {
				// 接続が閉じられた場合
				fmt.Println("受信終了: 接続が閉じられました")
				break
			}

			// 一時的なエラーとして扱い、少し待ってから再試行
			fmt.Printf("ネットワークエラー: %v - 再試行します\n", err)
			// エラーログを記録
			if logger != nil {
				logger.Log("ERROR: データ受信中にエラーが発生: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if len(data) == 0 {
			// データが空の場合は次の受信を待つ
			continue
		}

		if s.Debug {
			hexDump := hex.EncodeToString(data)
			if logger != nil {
				logger.Log("%v: 受信データ(hex): %s", addr, hexDump)
			}
		}

		msg, err := ParseECHONETLiteMessage(data)
		if err != nil {
			fmt.Println("パケット解析エラー:", err)
			continue
		}

		if s.Debug {
			fmt.Printf("応答を受信: %s から --- %v\n", addr, msg)
		}

		switch msg.ESV {
		case ESVSet_Res, ESVSetI_SNA, ESVSetC_SNA,
			ESVGet_Res, ESVGet_SNA,
			ESVINFC_Res,
			ESVINF_REQ_SNA,
			ESVSetGet_Res, ESVSetGet_SNA:
			// Get the callback while holding the lock
			s.mu.RLock()
			key := MakeKey(msg)
			entry, ok := s.dispatchTable[key]
			s.mu.RUnlock()

			// Execute callback outside the lock
			if ok {
				for _, esv := range entry.ESVs {
					if esv == msg.ESV {
						var complete CallbackCompleteStatus
						complete, err = entry.Callback(addr.IP, msg)
						if complete == CallbackFinished {
							s.mu.Lock()
							delete(s.dispatchTable, key)
							s.mu.Unlock()
						}
						break
					}
				}
			}
			if err != nil {
				fmt.Println("ディスパッチエラー:", err)
			}
		case ESVINF, ESVINFC:
			// Get the callback while holding the lock
			s.mu.RLock()
			callback := s.infCallback
			s.mu.RUnlock()

			// Execute callback outside the lock
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					fmt.Println("Infコールバックエラー:", err)
				}
			}
		case ESVGet, ESVSetC, ESVSetI, ESVINF_REQ:
			s.mu.RLock()
			callback := s.receiveCallback
			s.mu.RUnlock()
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					fmt.Printf("%v: ReceiveCallbackエラー: %v\n", msg.DEOJ, err)
				}
			}
		}
	}
}

func (s *Session) Close() error {
	s.mu.Lock()
	s.dispatchTable = nil // まずディスパッチテーブルをクリアして新しい処理を停止

	// コンテキストをキャンセル
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	// コネクションを閉じてエラーを返す
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

func (s *Session) sendTo(ip net.IP, SEOJ EOJ, DEOJ EOJ, ESV ESVType, property Properties, callback CallbackFunc) error {
	// 送信用のコンテキストを作成（タイムアウト5秒）
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	s.mu.Lock()
	TID := s.tid
	s.tid++
	s.mu.Unlock()
	msg := &ECHONETLiteMessage{
		TID:        TID,
		SEOJ:       SEOJ,
		DEOJ:       DEOJ,
		ESV:        ESV,
		Properties: property,
	}
	_, err := s.conn.SendTo(ctx, ip, msg.Encode())
	if err != nil {
		fmt.Println("パケット送信に失敗:", err)
		return err
	}
	if s.Debug {
		fmt.Printf("パケットを送信: %v へ --- %v\n", ip, msg)
	}
	if callback != nil {
		s.mu.Lock()
		key := MakeKey(msg)
		s.dispatchTable[key] = Entry{msg.ESV.ResponseESVs(), callback}
		s.mu.Unlock()
	}
	return nil
}

func (s *Session) SendResponse(ip net.IP, msg *ECHONETLiteMessage, ESV ESVType, property Properties, setGetProperty Properties) error {
	msgSend := &ECHONETLiteMessage{
		TID:              msg.TID,
		SEOJ:             msg.DEOJ,
		DEOJ:             msg.SEOJ,
		ESV:              ESV,
		Properties:       property,
		SetGetProperties: setGetProperty,
	}

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	_, err := s.conn.SendTo(ctx, ip, msgSend.Encode())
	if err != nil {
		fmt.Println("パケット送信に失敗:", err)
		return err
	}
	if s.Debug {
		fmt.Printf("パケットを送信: %v へ --- %v\n", ip, msgSend)
	}
	return nil
}

func (s *Session) Broadcast(SEOJ EOJ, ESV ESVType, property Properties) error {
	return s.sendTo(BroadcastIP, SEOJ, NodeProfileObject1, ESV, property, nil)
}

func (s *Session) BroadcastNodeList(nodes []EOJ) error {
	list := InstanceListNotification(nodes)
	return s.Broadcast(NodeProfileObject1, ESVINF, Properties{*list.Property()})
}

type GetPropertiesCallbackFunc func(IPAndEOJ, bool, Properties) (CallbackCompleteStatus, error)

func (s *Session) GetProperties(device IPAndEOJ, EPCs []EPCType, callback GetPropertiesCallbackFunc) error {
	forGet := make([]IPropertyForGet, 0, len(EPCs))
	for _, epc := range EPCs {
		forGet = append(forGet, epc)
	}
	// TODO broadcastでない場合、タイムアウト時に再送する仕組みを追加したい
	return s.sendTo(device.IP, s.eoj, device.EOJ, ESVGet, PropertiesForESVGet(forGet...), func(ip net.IP, msg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		device := IPAndEOJ{ip, msg.SEOJ}
		if msg.ESV == ESVGet_Res {
			return callback(device, true, msg.Properties)
		}
		return callback(device, false, msg.Properties)
	})
}

type GetPropertyCallbackFunc func(IPAndEOJ, bool, Property) (CallbackCompleteStatus, error)

func (s *Session) GetProperty(device IPAndEOJ, EPC EPCType, callback GetPropertyCallbackFunc) error {
	return s.GetProperties(device, []EPCType{EPC}, func(device IPAndEOJ, success bool, properties Properties) (CallbackCompleteStatus, error) {
		// propertiesの中から EPC == EPC となるものを探す
		for _, p := range properties {
			if p.EPC == EPC {
				return callback(device, success, p)
			}
		}
		return callback(device, false, Property{})
	})
}

func (s *Session) GetSelfNodeInstanceListS(ip net.IP, callback GetPropertyCallbackFunc) error {
	return s.GetProperty(IPAndEOJ{ip, NodeProfileObject1}, EPC_NPO_SelfNodeInstanceListS, callback)
}

func (s *Session) SetProperties(device IPAndEOJ, properties Properties, callback GetPropertiesCallbackFunc) error {
	return s.sendTo(device.IP, s.eoj, device.EOJ, ESVSetC, properties, func(ip net.IP, msg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		device := IPAndEOJ{ip, msg.SEOJ}
		if msg.ESV == ESVSet_Res {
			return callback(device, true, msg.Properties)
		}
		return callback(device, false, msg.Properties)
	})
}

func (s *Session) SetProperty(device IPAndEOJ, property Property, callback GetPropertyCallbackFunc) error {
	return s.SetProperties(device, Properties{property}, func(device IPAndEOJ, success bool, properties Properties) (CallbackCompleteStatus, error) {
		// propertiesの中から EPC == property.EPC となるものを探す
		for _, p := range properties {
			if p.EPC == property.EPC {
				return callback(device, success, p)
			}
		}
		return callback(device, false, Property{})
	})
}
