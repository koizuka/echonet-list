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

// SessionTimeoutType はセッションタイムアウトの種類を表す型
type SessionTimeoutType int

const (
	SessionTimeoutMaxRetries SessionTimeoutType = iota // 最大再送回数に達した
)

// SessionTimeoutEvent はセッションタイムアウトに関するイベントを表す構造体
type SessionTimeoutEvent struct {
	Device IPAndEOJ           // タイムアウトが発生したデバイス
	Type   SessionTimeoutType // タイムアウトの種類
	Error  error              // エラー情報
}

// ErrMaxRetriesReached は最大再送回数に達したことを示すエラー
type ErrMaxRetriesReached struct {
	MaxRetries int
	Device     IPAndEOJ
}

func (e ErrMaxRetriesReached) Error() string {
	return fmt.Sprintf("maximum retries reached (%d) for device %v", e.MaxRetries, e.Device)
}

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

type Session struct {
	mu              sync.RWMutex
	dispatchTable   DispatchTable
	receiveCallback PersistentCallbackFunc
	infCallback     PersistentCallbackFunc
	tid             TIDType
	eoj             EOJ
	conn            *network.UDPConnection
	Debug           bool
	ctx             context.Context          // コンテキスト
	cancel          context.CancelFunc       // コンテキストのキャンセル関数
	MaxRetries      int                      // 最大再送回数
	RetryInterval   time.Duration            // 再送間隔
	TimeoutCh       chan SessionTimeoutEvent // タイムアウト通知用チャンネル
}

// SetTimeoutChannel はタイムアウト通知用チャンネルを設定する
func (s *Session) SetTimeoutChannel(ch chan SessionTimeoutEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TimeoutCh = ch
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
		MaxRetries:    3,               // デフォルトの最大再送回数
		RetryInterval: 3 * time.Second, // デフォルトの再送間隔
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
							s.unregisterCallback(key)
						}
						break
					}
				}
			}
			if err != nil && logger != nil {
				logger.Log("ディスパッチエラー: %v", err)
			}
		case ESVINF, ESVINFC:
			// Get the callback while holding the lock
			s.mu.RLock()
			callback := s.infCallback
			s.mu.RUnlock()

			// Execute callback outside the lock
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil && logger != nil {
					logger.Log("Infコールバックエラー: %v", err)
				}
			}
		case ESVGet, ESVSetC, ESVSetI, ESVINF_REQ:
			s.mu.RLock()
			callback := s.receiveCallback
			s.mu.RUnlock()
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil && logger != nil {
					logger.Log("%v: ReceiveCallbackエラー: %v", msg.DEOJ, err)
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
		DEOJ:       NodeProfileObject,
		ESV:        ESV,
		Properties: property,
	}
	return s.sendMessage(BroadcastIP, msg)
}

// GetPropertiesCallbackFunc はプロパティ取得のコールバック関数の型。
type GetPropertiesCallbackFunc func(device IPAndEOJ, success bool, properties Properties, FailedEPCs []EPCType) (CallbackCompleteStatus, error)

func (s *Session) CreateGetPropertyMessage(device IPAndEOJ, EPCs []EPCType) *ECHONETLiteMessage {
	props := make([]Property, 0, len(EPCs))
	for _, epc := range EPCs {
		props = append(props, *epc.PropertyForGet())
	}
	return &ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        ESVGet,
		Properties: props,
	}
}

func (s *Session) prepareStartGetProperties(device IPAndEOJ, EPCs []EPCType, callback GetPropertiesCallbackFunc) (*ECHONETLiteMessage, Key) {
	msg := s.CreateGetPropertyMessage(device, EPCs)
	key := MakeKey(msg)
	s.registerCallback(key, msg.ESV.ResponseESVs(), func(ip net.IP, msg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
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
	return msg, key
}

func (s *Session) StartGetProperties(device IPAndEOJ, EPCs []EPCType, callback GetPropertiesCallbackFunc) (Key, error) {
	msg, key := s.prepareStartGetProperties(device, EPCs, callback)
	if err := s.sendMessage(device.IP, msg); err != nil {
		return Key{}, err
	}
	return key, nil
}

// StartGetPropertiesWithRetry は、プロパティ取得を行い、タイムアウトした場合は go routineで再試行する
func (s *Session) StartGetPropertiesWithRetry(ctx1 context.Context, device IPAndEOJ, EPCs []EPCType, callback GetPropertiesCallbackFunc) error {
	desc := fmt.Sprintf("StartGetPropertiesWithRetry(%v, %v)", device, EPCs)

	ctx, cancel := context.WithCancel(ctx1)

	msg, key := s.prepareStartGetProperties(device, EPCs, func(device IPAndEOJ, success bool, properties Properties, FailedEPCs []EPCType) (CallbackCompleteStatus, error) {
		cancel()
		_, err := callback(device, success, properties, FailedEPCs)
		return CallbackFinished, err
	})

	err := s.sendMessage(device.IP, msg)
	if err != nil {
		cancel()
		s.unregisterCallback(key)
		return err
	}

	go func() {
		defer cancel() // ゴルーチン終了時にキャンセル

		// 再送カウンタ
		retryCount := 0

		// タイマーの作成
		timer := time.NewTimer(s.RetryInterval)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				s.unregisterCallback(key)

				if retryCount > 0 {
					// fmt.Printf("%v: リトライ後に完了\n", desc) // DEBUG
				}
				return

			case <-timer.C:
				// タイムアウトした場合
				retryCount++

				logger := log.GetLogger()
				if retryCount >= s.MaxRetries {
					// 最大再送回数に達した場合
					if logger != nil {
						logger.Log("%v 最大再送回数(%d)に達しました", desc, s.MaxRetries)
					}
					_ = s.notifyDeviceTimeout(device)
					return
				}

				// ログ出力
				if logger != nil {
					logger.Log("%v: リクエストを再送します (試行 %d/%d)", desc, retryCount, s.MaxRetries)
				}
				// fmt.Printf("%v: リクエストを再送します (試行 %d/%d)\n", desc, retryCount, s.MaxRetries) // DEBUG

				// 再送
				if err := s.sendMessage(device.IP, msg); err != nil {
					return
				}

				// タイマーをリセット
				timer.Reset(s.RetryInterval)
			}
		}
	}()
	return nil
}

func (s *Session) notifyDeviceTimeout(device IPAndEOJ) error {
	maxRetriesErr := ErrMaxRetriesReached{
		MaxRetries: s.MaxRetries,
		Device:     device,
	}
	if s.TimeoutCh != nil {
		select {
		case s.TimeoutCh <- SessionTimeoutEvent{
			Device: device,
			Type:   SessionTimeoutMaxRetries,
			Error:  maxRetriesErr,
		}:
			// 送信成功
		default:
			// チャンネルがブロックされている場合は無視
		}
	}
	return maxRetriesErr
}

func (s *Session) CreateSetPropertyMessage(device IPAndEOJ, properties Properties) *ECHONETLiteMessage {
	return &ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        ESVSetC,
		Properties: properties,
	}
}

// コールバックを登録解除する関数
func (s *Session) unregisterCallback(key Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dispatchTable, key)
}

// 共通処理を行う内部関数
func (s *Session) sendRequestWithContext(
	ctx context.Context,
	device IPAndEOJ,
	msg *ECHONETLiteMessage,
) (*ECHONETLiteMessage, error) {
	// 結果を受け取るためのチャネル
	responseCh := make(chan *ECHONETLiteMessage, 1)

	// キーを取得
	key := MakeKey(msg)

	// コールバックを登録
	s.registerCallback(key, msg.ESV.ResponseESVs(), func(ip net.IP, respMsg *ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		// 応答メッセージをチャネルに送信
		select {
		case <-ctx.Done():
			// コンテキストがキャンセルされた場合は何もしない
		default:
			responseCh <- respMsg
		}

		// 必ず登録解除する（ブロードキャストを想定しない）
		s.unregisterCallback(key)

		return CallbackFinished, nil
	})

	// 関数終了時にコールバックを登録解除するための遅延処理
	callbackUnregistered := false
	defer func() {
		if !callbackUnregistered {
			s.unregisterCallback(key)
		}
	}()

	// 最初のリクエスト送信
	if err := s.sendMessage(device.IP, msg); err != nil {
		return nil, err
	}

	// 再送カウンタ
	retryCount := 0

	// タイマーの作成
	timer := time.NewTimer(s.RetryInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// 親コンテキストがキャンセルされた場合
			return nil, ctx.Err()

		case respMsg := <-responseCh:
			// 応答を受信した場合
			callbackUnregistered = true // コールバックは既に登録解除されている
			return respMsg, nil

		case <-timer.C:
			// タイムアウトした場合
			retryCount++

			logger := log.GetLogger()
			if retryCount >= s.MaxRetries {
				// 最大再送回数に達した場合
				if logger != nil {
					logger.Log("最大再送回数(%d)に達しました", s.MaxRetries)
				}

				// タイムアウト通知をチャンネルに送信
				return nil, s.notifyDeviceTimeout(device)
			}

			// ログ出力
			if logger != nil {
				logger.Log("タイムアウト: リクエストを再送します (試行 %d/%d)", retryCount+1, s.MaxRetries)
			}

			// 再送
			if err := s.sendMessage(device.IP, msg); err != nil {
				return nil, err
			}

			// タイマーをリセット
			timer.Reset(s.RetryInterval)
		}
	}
}

// GetProperties - プロパティ取得
func (s *Session) GetProperties(
	ctx context.Context,
	device IPAndEOJ,
	EPCs []EPCType,
) (bool, Properties, []EPCType, error) {
	// メッセージを作成
	msg := s.CreateGetPropertyMessage(device, EPCs)

	// 共通処理を呼び出し
	respMsg, err := s.sendRequestWithContext(ctx, device, msg)

	// エラーチェック
	if err != nil {
		// タイムアウトやコンテキストキャンセルの場合
		return false, nil, EPCs, err
	}

	// 応答を処理
	success := respMsg.ESV == ESVGet_Res

	// 成功/失敗のプロパティを分類
	successProperties := make(Properties, 0, len(respMsg.Properties))
	failedEPCs := make([]EPCType, 0, len(respMsg.Properties))

	for _, p := range respMsg.Properties {
		if p.EDT != nil {
			successProperties = append(successProperties, p)
		} else {
			failedEPCs = append(failedEPCs, p.EPC)
		}
	}

	return success, successProperties, failedEPCs, nil
}

// SetProperties - プロパティ設定
func (s *Session) SetProperties(
	ctx context.Context,
	device IPAndEOJ,
	properties Properties,
) (bool, Properties, []EPCType, error) {
	// メッセージを作成
	msg := s.CreateSetPropertyMessage(device, properties)

	// 共通処理を呼び出し
	respMsg, err := s.sendRequestWithContext(ctx, device, msg)

	// エラーチェック
	if err != nil {
		// タイムアウトやコンテキストキャンセルの場合
		failedEPCs := make([]EPCType, 0, len(properties))
		for _, p := range properties {
			failedEPCs = append(failedEPCs, p.EPC)
		}
		return false, nil, failedEPCs, err
	}

	// 応答を処理
	success := respMsg.ESV == ESVSet_Res

	// 成功/失敗のプロパティを分類
	successProperties := make(Properties, 0, len(properties))
	failedEPCs := make([]EPCType, 0, len(properties))

	// Setは EDT == nil が成功
	for i, p := range respMsg.Properties {
		if p.EDT == nil {
			successProperties = append(successProperties, properties[i])
		} else {
			failedEPCs = append(failedEPCs, p.EPC)
		}
	}

	return success, successProperties, failedEPCs, nil
}
