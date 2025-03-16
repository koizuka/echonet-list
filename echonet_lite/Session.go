package echonet_lite

import (
	"context"
	"echonet-list/echonet_lite/log"
	"echonet-list/echonet_lite/network"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var NodeProfileObject1 = MakeEOJ(NodeProfile_ClassCode, 1)

// ブロードキャストアドレスの設定
var BroadcastIP = network.GetIPv4BroadcastIP()

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
	conn            *network.UDPConnection
	Debug           bool
	ctx             context.Context    // コンテキスト
	cancel          context.CancelFunc // コンテキストのキャンセル関数
}

func CreateSession(ctx context.Context, ip net.IP, EOJ EOJ, debug bool) (*Session, error) {
	// タイムアウトなしのコンテキストを作成（キャンセルのみ可能）
	sessionCtx, cancel := context.WithCancel(ctx)

	conn, err := network.CreateUDPConnection(sessionCtx, ip, ECHONETLitePort, BroadcastIP, network.UDPConnectionOptions{DefaultTimeout: 30 * time.Second})
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
	logger := log.GetLogger()
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
			if logger != nil {
				logger.Log("コンテキストがキャンセルされました: %v", s.ctx.Err())
			}
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
				if logger != nil {
					logger.Log("受信終了: 接続が閉じられました")
				}
				break
			}

			// その他のネットワークエラー
			// net.Error.Temporary()はdeprecatedなので、特定のエラータイプで判断する
			if errors.Is(err, net.ErrClosed) {
				// 接続が閉じられた場合
				if logger != nil {
					logger.Log("受信終了: 接続が閉じられました")
				}
				break
			}

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
			if logger != nil {
				logger.Log("パケット解析エラー: %v", err)
			}
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
				if logger != nil {
					logger.Log("ディスパッチエラー: %v", err)
				}
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
					if logger != nil {
						logger.Log("Infコールバックエラー: %v", err)
					}
				}
			}
		case ESVGet, ESVSetC, ESVSetI, ESVINF_REQ:
			s.mu.RLock()
			callback := s.receiveCallback
			s.mu.RUnlock()
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					if logger != nil {
						logger.Log("%v: ReceiveCallbackエラー: %v", msg.DEOJ, err)
					}
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

func (s *Session) newTID() TIDType {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tid++
	return s.tid
}

func (s *Session) registerCallback(key Key, ESVs []ESVType, callback CallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatchTable.Register(key, ESVs, callback)
}

func (s *Session) registerCallbackFromMessage(msg *ECHONETLiteMessage, callback CallbackFunc) {
	s.registerCallback(MakeKey(msg), msg.ESV.ResponseESVs(), callback)
}

func (s *Session) sendMessage(ip net.IP, msg *ECHONETLiteMessage) error {
	if _, err := s.conn.SendTo(ip, msg.Encode()); err != nil {
		logger := log.GetLogger()
		if logger != nil {
			logger.Log("パケット送信エラー: %v", err)
		}
		return err
	}
	if s.Debug {
		fmt.Printf("パケットを送信: %v へ --- %v\n", ip, msg)
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
	return s.sendMessage(ip, msgSend)
}

func (s *Session) Broadcast(SEOJ EOJ, ESV ESVType, property Properties) error {
	msg := &ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       SEOJ,
		DEOJ:       NodeProfileObject1,
		ESV:        ESV,
		Properties: property,
	}
	return s.sendMessage(BroadcastIP, msg)
}

// GetPropertiesCallbackFunc はプロパティ取得のコールバック関数の型。
type GetPropertiesCallbackFunc func(device IPAndEOJ, success bool, properties Properties, FailedEPCs []EPCType) (CallbackCompleteStatus, error)

func (s *Session) GetProperties(device IPAndEOJ, EPCs []EPCType, callback GetPropertiesCallbackFunc) error {
	props := make([]Property, 0, len(EPCs))
	for _, epc := range EPCs {
		props = append(props, *epc.PropertyForGet())
	}
	// TODO broadcastでない場合、タイムアウト時に再送する仕組みを追加したい
	msg := &ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        ESVGet,
		Properties: props,
	}
	if err := s.sendMessage(device.IP, msg); err != nil {
		return err
	}
	s.registerCallbackFromMessage(msg, func(ip net.IP, msg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		device := IPAndEOJ{ip, msg.SEOJ}
		if msg.ESV == ESVGet_Res {
			return callback(device, true, msg.Properties, nil)
		}
		// Getは EDT=nilが失敗
		successProperties := make(Properties, 0, len(msg.Properties))
		failedEPCs := make([]EPCType, 0, len(msg.Properties))
		for _, p := range msg.Properties {
			if p.EDT != nil {
				successProperties = append(successProperties, p)
			} else {
				failedEPCs = append(failedEPCs, p.EPC)
			}
		}
		return callback(device, false, successProperties, failedEPCs)
	})
	return nil
}

func (s *Session) SetProperties(device IPAndEOJ, properties Properties, callback GetPropertiesCallbackFunc) error {
	msg := &ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        ESVSetC,
		Properties: properties,
	}
	if err := s.sendMessage(device.IP, msg); err != nil {
		return err
	}
	s.registerCallbackFromMessage(msg, func(ip net.IP, msg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		device := IPAndEOJ{ip, msg.SEOJ}
		if msg.ESV == ESVSet_Res {
			return callback(device, true, msg.Properties, nil)
		}
		successProperties := make(Properties, 0, len(properties))
		failedEPCs := make([]EPCType, 0)

		// Setは EDT == nil が成功
		for i, p := range msg.Properties {
			if p.EDT == nil {
				successProperties = append(successProperties, properties[i])
			} else {
				failedEPCs = append(failedEPCs, p.EPC)
			}
		}
		return callback(device, false, successProperties, failedEPCs)
	})
	return nil
}
