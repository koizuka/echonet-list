package main

import (
	"context"
	"echonet-list/echonet_lite"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var NodeProfileObject1 = echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 1)

// ブロードキャストアドレスの設定
var BroadcastIP = getIPv4BroadcastIP()

// var BroadcastIP = net.ParseIP("ff02::1")

type Key struct {
	TID echonet_lite.TIDType
}

func MakeKey(msg *echonet_lite.ECHONETLiteMessage) Key {
	return Key{msg.TID}
}

type CallbackFunc func(net.IP, *echonet_lite.ECHONETLiteMessage) error

type Entry struct {
	ESVs     []echonet_lite.ESVType
	Callback CallbackFunc
}

type DispatchTable map[Key]Entry

func (dt DispatchTable) Register(key Key, ESVs []echonet_lite.ESVType, callback CallbackFunc) {
	dt[key] = Entry{ESVs, callback}
}

func (dt DispatchTable) Unregister(key Key) {
	delete(dt, key)
}

func (dt DispatchTable) Dispatch(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	key := MakeKey(msg)
	entry, ok := dt[key]
	if ok {
		for _, esv := range entry.ESVs {
			if esv == msg.ESV {
				return entry.Callback(ip, msg)
			}
		}
	}
	return nil
}

type Session struct {
	mu              sync.RWMutex
	DispatchTable   DispatchTable
	ReceiveCallback CallbackFunc
	InfCallback     CallbackFunc
	TID             echonet_lite.TIDType
	EOJ             echonet_lite.EOJ
	Conn            *UDPConnection
	Debug           bool
	ctx             context.Context    // コンテキスト
	cancel          context.CancelFunc // コンテキストのキャンセル関数
}

type CallbackFinished struct {
}

func (c CallbackFinished) Error() string {
	return "callback finished"
}

func CreateSession(ctx context.Context, ip net.IP, EOJ echonet_lite.EOJ, debug bool) (*Session, error) {
	// タイムアウトなしのコンテキストを作成（キャンセルのみ可能）
	sessionCtx, cancel := context.WithCancel(ctx)

	conn, err := CreateUDPConnection(sessionCtx, ip, UDPConnectionOptions{DefaultTimeout: 30 * time.Second})
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}
	return &Session{
		DispatchTable: make(DispatchTable),
		TID:           echonet_lite.TIDType(1),
		EOJ:           EOJ,
		Conn:          conn,
		Debug:         debug,
		ctx:           sessionCtx,
		cancel:        cancel,
	}, nil
}

func (s *Session) OnInf(callback CallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InfCallback = callback
}

func (s *Session) OnReceive(callback CallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReceiveCallback = callback
}

func (s *Session) MainLoop() {
	for {
		// DispatchTableがnilかどうかをロックして確認
		s.mu.RLock()
		dt := s.DispatchTable
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
		data, addr, err := s.Conn.Receive(receiveCtx)
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

		msg, err := echonet_lite.ParseECHONETLiteMessage(data)
		if err != nil {
			fmt.Println("パケット解析エラー:", err)
			continue
		}

		if s.Debug {
			fmt.Printf("応答を受信: %s から --- %v\n", addr, msg)
		}

		switch msg.ESV {
		case echonet_lite.ESVSet_Res, echonet_lite.ESVSetI_SNA, echonet_lite.ESVSetC_SNA,
			echonet_lite.ESVGet_Res, echonet_lite.ESVGet_SNA,
			echonet_lite.ESVINFC_Res,
			echonet_lite.ESVINF_REQ_SNA,
			echonet_lite.ESVSetGet_Res, echonet_lite.ESVSetGet_SNA:
			// Get the callback while holding the lock
			s.mu.RLock()
			key := MakeKey(msg)
			entry, ok := s.DispatchTable[key]
			s.mu.RUnlock()

			// Execute callback outside the lock
			if ok {
				for _, esv := range entry.ESVs {
					if esv == msg.ESV {
						err = entry.Callback(addr.IP, msg)
						break
					}
				}
			}
			if err != nil {
				var cbErr CallbackFinished
				if errors.As(err, &cbErr) {
					s.mu.Lock()
					delete(s.DispatchTable, key)
					s.mu.Unlock()
				} else {
					fmt.Println("ディスパッチエラー:", err)
				}
			}
		case echonet_lite.ESVINF, echonet_lite.ESVINFC:
			// Get the callback while holding the lock
			s.mu.RLock()
			callback := s.InfCallback
			s.mu.RUnlock()

			// Execute callback outside the lock
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					var cbErr CallbackFinished
					if errors.As(err, &cbErr) {
						s.mu.Lock()
						s.InfCallback = nil
						s.mu.Unlock()
					} else {
						fmt.Println("Infコールバックエラー:", err)
					}
				}
			}
		case echonet_lite.ESVGet, echonet_lite.ESVSetC, echonet_lite.ESVSetI, echonet_lite.ESVINF_REQ:
			s.mu.RLock()
			callback := s.ReceiveCallback
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
	s.DispatchTable = nil // まずディスパッチテーブルをクリアして新しい処理を停止

	// コンテキストをキャンセル
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	// コネクションを閉じてエラーを返す
	if err := s.Conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

func (s *Session) SendTo(ip net.IP, SEOJ echonet_lite.EOJ, DEOJ echonet_lite.EOJ, ESV echonet_lite.ESVType, property echonet_lite.Properties, callback CallbackFunc) error {
	// 送信用のコンテキストを作成（タイムアウト5秒）
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	s.mu.Lock()
	TID := s.TID
	s.TID++
	s.mu.Unlock()
	msg := &echonet_lite.ECHONETLiteMessage{
		TID:        TID,
		SEOJ:       SEOJ,
		DEOJ:       DEOJ,
		ESV:        ESV,
		Properties: property,
	}
	_, err := s.Conn.SendTo(ctx, ip, msg.Encode())
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
		s.DispatchTable[key] = Entry{msg.ESV.ResponseESVs(), callback}
		s.mu.Unlock()
	}
	return nil
}

func (s *Session) SendResponse(ip net.IP, msg *echonet_lite.ECHONETLiteMessage, ESV echonet_lite.ESVType, property echonet_lite.Properties, setGetProperty echonet_lite.Properties) error {
	msgSend := &echonet_lite.ECHONETLiteMessage{
		TID:              msg.TID,
		SEOJ:             msg.DEOJ,
		DEOJ:             msg.SEOJ,
		ESV:              ESV,
		Properties:       property,
		SetGetProperties: setGetProperty,
	}

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	_, err := s.Conn.SendTo(ctx, ip, msgSend.Encode())
	if err != nil {
		fmt.Println("パケット送信に失敗:", err)
		return err
	}
	if s.Debug {
		fmt.Printf("パケットを送信: %v へ --- %v\n", ip, msgSend)
	}
	return nil
}

func (s *Session) Broadcast(SEOJ echonet_lite.EOJ, ESV echonet_lite.ESVType, property echonet_lite.Properties) error {
	return s.SendTo(BroadcastIP, SEOJ, NodeProfileObject1, ESV, property, nil)
}

func (s *Session) BroadcastNodeList(nodes []echonet_lite.EOJ) error {
	list := echonet_lite.InstanceListNotification(nodes)
	return s.Broadcast(NodeProfileObject1, echonet_lite.ESVINF, echonet_lite.Properties{*list.Property()})
}

type GetPropertiesCallbackFunc func(net.IP, echonet_lite.EOJ, bool, echonet_lite.Properties) error

func (s *Session) GetProperties(ip net.IP, DEOJ echonet_lite.EOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) error {
	forGet := make([]echonet_lite.IPropertyForGet, 0, len(EPCs))
	for _, epc := range EPCs {
		forGet = append(forGet, epc)
	}
	// TODO broadcastでない場合、タイムアウト時に再送する仕組みを追加したい
	return s.SendTo(ip, s.EOJ, DEOJ, echonet_lite.ESVGet, echonet_lite.PropertiesForESVGet(forGet...), func(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
		if msg.ESV == echonet_lite.ESVGet_Res {
			return callback(ip, msg.SEOJ, true, msg.Properties)
		}
		return callback(ip, msg.SEOJ, false, msg.Properties)
	})
}

type GetPropertyCallbackFunc func(net.IP, echonet_lite.EOJ, bool, echonet_lite.Property) error

func (s *Session) GetProperty(ip net.IP, DEOJ echonet_lite.EOJ, EPC echonet_lite.EPCType, callback GetPropertyCallbackFunc) error {
	return s.GetProperties(ip, DEOJ, []echonet_lite.EPCType{EPC}, func(ip net.IP, eoj echonet_lite.EOJ, success bool, properties echonet_lite.Properties) error {
		// propertiesの中から EPC == EPC となるものを探す
		for _, p := range properties {
			if p.EPC == EPC {
				return callback(ip, eoj, success, p)
			}
		}
		return callback(ip, eoj, false, echonet_lite.Property{})
	})
}

func (s *Session) GetSelfNodeInstanceListS(ip net.IP, callback GetPropertyCallbackFunc) error {
	return s.GetProperty(ip, NodeProfileObject1, echonet_lite.EPC_NPO_SelfNodeInstanceListS, callback)
}

func (s *Session) SetProperties(ip net.IP, DEOJ echonet_lite.EOJ, properties echonet_lite.Properties, callback GetPropertiesCallbackFunc) error {
	return s.SendTo(ip, s.EOJ, DEOJ, echonet_lite.ESVSetC, properties, func(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
		if msg.ESV == echonet_lite.ESVSet_Res {
			return callback(ip, msg.SEOJ, true, msg.Properties)
		}
		return callback(ip, msg.SEOJ, false, msg.Properties)
	})
}

func (s *Session) SetProperty(ip net.IP, DEOJ echonet_lite.EOJ, property echonet_lite.Property, callback GetPropertyCallbackFunc) error {
	return s.SetProperties(ip, DEOJ, echonet_lite.Properties{property}, func(ip net.IP, eoj echonet_lite.EOJ, success bool, properties echonet_lite.Properties) error {
		// propertiesの中から EPC == property.EPC となるものを探す
		for _, p := range properties {
			if p.EPC == property.EPC {
				return callback(ip, eoj, success, p)
			}
		}
		return callback(ip, eoj, false, echonet_lite.Property{})
	})
}
