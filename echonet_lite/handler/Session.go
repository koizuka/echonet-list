package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/network"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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
	Device echonet_lite.IPAndEOJ // タイムアウトが発生したデバイス
	Type   SessionTimeoutType    // タイムアウトの種類
	Error  error                 // エラー情報
}

// ErrMaxRetriesReached は最大再送回数に達したことを示すエラー
type ErrMaxRetriesReached struct {
	MaxRetries int
	Device     echonet_lite.IPAndEOJ
}

func (e ErrMaxRetriesReached) Error() string {
	return fmt.Sprintf("maximum retries reached (%d) for device %v", e.MaxRetries, e.Device)
}

// ブロードキャストアドレスの設定
var BroadcastIP = network.GetIPv4BroadcastIP()

type Key struct {
	TID echonet_lite.TIDType
}

func MakeKey(msg *echonet_lite.ECHONETLiteMessage) Key {
	return Key{msg.TID}
}

type CallbackCompleteStatus int // プロパティコールバック完了ステータス
const (
	CallbackFinished CallbackCompleteStatus = iota
	CallbackContinue
)

type CallbackFunc func(net.IP, *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error)
type PersistentCallbackFunc func(net.IP, *echonet_lite.ECHONETLiteMessage) error

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

type Session struct {
	mu              sync.RWMutex
	dispatchTable   DispatchTable
	receiveCallback PersistentCallbackFunc
	infCallback     PersistentCallbackFunc
	tid             echonet_lite.TIDType
	eoj             echonet_lite.EOJ
	conn            *network.UDPConnection
	MulticastIP     net.IP
	Debug           bool
	ctx             context.Context                   // コンテキスト
	cancel          context.CancelFunc                // コンテキストのキャンセル関数
	MaxRetries      int                               // 最大再送回数
	RetryInterval   time.Duration                     // 再送間隔
	TimeoutCh       chan SessionTimeoutEvent          // タイムアウト通知用チャンネル
	failedEPCs      map[string][]echonet_lite.EPCType // 失敗したEPCsを保持するマップ
}

// SetTimeoutChannel はタイムアウト通知用チャンネルを設定する
func (s *Session) SetTimeoutChannel(ch chan SessionTimeoutEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TimeoutCh = ch
}

func CreateSession(ctx context.Context, ip net.IP, EOJ echonet_lite.EOJ, debug bool) (*Session, error) {
	// タイムアウトなしのコンテキストを作成（キャンセルのみ可能）
	sessionCtx, cancel := context.WithCancel(ctx)

	multicastIP := echonet_lite.ECHONETLiteMulticastIPv4

	conn, err := network.CreateUDPConnection(sessionCtx, ip, echonet_lite.ECHONETLitePort, multicastIP)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}
	return &Session{
		dispatchTable: make(DispatchTable),
		tid:           echonet_lite.TIDType(1),
		eoj:           EOJ,
		conn:          conn,
		MulticastIP:   multicastIP,
		Debug:         debug,
		ctx:           sessionCtx,
		cancel:        cancel,
		MaxRetries:    3,               // デフォルトの最大再送回数
		RetryInterval: 3 * time.Second, // デフォルトの再送間隔
		failedEPCs:    make(map[string][]echonet_lite.EPCType),
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
				slog.Info("受信終了: 接続が閉じられました")
				break
			}

			// その他のネットワークエラー
			// net.Error.Temporary()はdeprecatedなので、特定のエラータイプで判断する
			if errors.Is(err, net.ErrClosed) {
				// 接続が閉じられた場合
				slog.Info("受信終了: 接続が閉じられました")
				break
			}

			// エラーログを記録
			slog.Error("データ受信中にエラーが発生", "err", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if len(data) == 0 {
			// データが空の場合は次の受信を待つ
			continue
		}

		if s.Debug {
			hexDump := hex.EncodeToString(data)
			slog.Debug("受信データ(hex)", "addr", addr, "hex", hexDump)
		}

		msg, err := echonet_lite.ParseECHONETLiteMessage(data)
		if err != nil {
			slog.Error("パケット解析エラー", "err", err)
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
			entry, ok := s.dispatchTable[key]
			s.mu.RUnlock()

			// Execute callback outside the lock
			if ok {
				for _, esv := range entry.ESVs {
					if esv == msg.ESV {
						var complete CallbackCompleteStatus
						complete, err = entry.Callback(addr.IP, msg)
						if complete == CallbackFinished {
							s.UnregisterCallback(key)
						}
						break
					}
				}
			}
			if err != nil {
				slog.Error("ディスパッチエラー", "err", err)
			}
		case echonet_lite.ESVINF, echonet_lite.ESVINFC:
			// Get the callback while holding the lock
			s.mu.RLock()
			callback := s.infCallback
			s.mu.RUnlock()

			// Execute callback outside the lock
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					slog.Error("Infコールバックエラー", "err", err)
				}
			}
		case echonet_lite.ESVGet, echonet_lite.ESVSetC, echonet_lite.ESVSetI, echonet_lite.ESVINF_REQ:
			s.mu.RLock()
			callback := s.receiveCallback
			s.mu.RUnlock()
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					slog.Error("ReceiveCallbackエラー", "DEOJ", msg.DEOJ, "err", err)
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

func (s *Session) newTID() echonet_lite.TIDType {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tid++
	return s.tid
}

func (s *Session) registerCallback(key Key, ESVs []echonet_lite.ESVType, callback CallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatchTable.Register(key, ESVs, callback)
}

func (s *Session) sendMessage(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	if _, err := s.conn.SendTo(ip, msg.Encode()); err != nil {
		slog.Error("パケット送信エラー", "err", err)
		return err
	}
	if s.Debug {
		fmt.Printf("パケットを送信: %v へ --- %v\n", ip, msg)
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
	return s.sendMessage(ip, msgSend)
}

func (s *Session) Broadcast(SEOJ echonet_lite.EOJ, ESV echonet_lite.ESVType, property echonet_lite.Properties) error {
	msg := &echonet_lite.ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       SEOJ,
		DEOJ:       echonet_lite.NodeProfileObject,
		ESV:        ESV,
		Properties: property,
	}
	ip := s.MulticastIP
	if ip == nil {
		ip = BroadcastIP
	}
	return s.sendMessage(ip, msg)
}

// GetPropertiesCallbackFunc はプロパティ取得のコールバック関数の型。
type GetPropertiesCallbackFunc func(device echonet_lite.IPAndEOJ, success bool, properties echonet_lite.Properties, FailedEPCs []echonet_lite.EPCType) (CallbackCompleteStatus, error)

func (s *Session) CreateGetPropertyMessage(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType) *echonet_lite.ECHONETLiteMessage {
	props := make([]echonet_lite.Property, 0, len(EPCs))
	for _, epc := range EPCs {
		props = append(props, echonet_lite.Property{EPC: epc})
	}
	return &echonet_lite.ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        echonet_lite.ESVGet,
		Properties: props,
	}
}

func (s *Session) prepareStartGetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) (*echonet_lite.ECHONETLiteMessage, Key) {
	msg := s.CreateGetPropertyMessage(device, EPCs)
	key := MakeKey(msg)
	s.registerCallback(key, msg.ESV.ResponseESVs(), func(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		device := echonet_lite.IPAndEOJ{ip, msg.SEOJ}
		if msg.ESV == echonet_lite.ESVGet_Res {
			return callback(device, true, msg.Properties, nil)
		}
		// Getは EDT=nilが失敗
		successProperties := make(echonet_lite.Properties, 0, len(msg.Properties))
		failedEPCs := make([]echonet_lite.EPCType, 0, len(msg.Properties))
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

func (s *Session) StartGetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) (Key, error) {
	msg, key := s.prepareStartGetProperties(device, EPCs, callback)
	if err := s.sendMessage(device.IP, msg); err != nil {
		return Key{}, err
	}
	return key, nil
}

// StartGetPropertiesWithRetry は、プロパティ取得を行い、タイムアウトした場合は go routineで再試行する
func (s *Session) StartGetPropertiesWithRetry(ctx1 context.Context, device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) error {
	desc := fmt.Sprintf("StartGetPropertiesWithRetry(%v, %v)", device, EPCs)

	ctx, cancel := context.WithCancel(ctx1)

	msg, key := s.prepareStartGetProperties(device, EPCs, func(device echonet_lite.IPAndEOJ, success bool, properties echonet_lite.Properties, FailedEPCs []echonet_lite.EPCType) (CallbackCompleteStatus, error) {
		cancel()
		_, err := callback(device, success, properties, FailedEPCs)
		return CallbackFinished, err
	})

	err := s.sendMessage(device.IP, msg)
	if err != nil {
		cancel()
		s.UnregisterCallback(key)
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
				s.UnregisterCallback(key)

				if retryCount > 0 {
					slog.Info("リトライ後に完了", "desc", desc)
				}
				return

			case <-timer.C:
				// タイムアウトした場合
				retryCount++

				if retryCount >= s.MaxRetries {
					// 最大再送回数に達した場合

					slog.Warn("最大再送回数に達しました", "desc", desc, "maxRetries", s.MaxRetries)
					_ = s.notifyDeviceTimeout(device)
					return
				}

				// ログ出力
				slog.Info("リクエストを再送します", "desc", desc, "retry", retryCount, "maxRetries", s.MaxRetries)
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

func (s *Session) notifyDeviceTimeout(device echonet_lite.IPAndEOJ) error {
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

func (s *Session) CreateSetPropertyMessage(device echonet_lite.IPAndEOJ, properties echonet_lite.Properties) *echonet_lite.ECHONETLiteMessage {
	return &echonet_lite.ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        echonet_lite.ESVSetC,
		Properties: properties,
	}
}

// コールバックを登録解除する関数
func (s *Session) UnregisterCallback(key Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dispatchTable, key)
}

// 共通処理を行う内部関数
func (s *Session) sendRequestWithContext(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	msg *echonet_lite.ECHONETLiteMessage,
) (*echonet_lite.ECHONETLiteMessage, error) {
	// 結果を受け取るためのチャネル
	responseCh := make(chan *echonet_lite.ECHONETLiteMessage, 1)

	// キーを取得
	key := MakeKey(msg)

	// コールバックを登録
	s.registerCallback(key, msg.ESV.ResponseESVs(), func(ip net.IP, respMsg *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		// 応答メッセージをチャネルに送信
		select {
		case <-ctx.Done():
			// コンテキストがキャンセルされた場合は何もしない
		default:
			responseCh <- respMsg
		}

		// 必ず登録解除する（ブロードキャストを想定しない）
		s.UnregisterCallback(key)

		return CallbackFinished, nil
	})

	// 関数終了時にコールバックを登録解除するための遅延処理
	callbackUnregistered := false
	defer func() {
		if !callbackUnregistered {
			s.UnregisterCallback(key)
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
			if retryCount > 0 {
				slog.Info("リトライ後に完了", "device", device)
			}
			// 応答を受信した場合
			callbackUnregistered = true // コールバックは既に登録解除されている
			return respMsg, nil

		case <-timer.C:
			// タイムアウトした場合
			retryCount++

			if retryCount >= s.MaxRetries {
				// 最大再送回数に達した場合
				slog.Warn("最大再送回数に達しました", "device", device, "maxRetries", s.MaxRetries)
				return nil, s.notifyDeviceTimeout(device)
			}

			// ログ出力
			slog.Info("リクエストを再送します", "device", device, "retry", retryCount+1, "maxRetries", s.MaxRetries)

			// 再送
			if err := s.sendMessage(device.IP, msg); err != nil {
				return nil, err
			}

			// タイマーをリセット
			timer.Reset(s.RetryInterval)
		}
	}
}

func (s *Session) updateFailedEPCs(device echonet_lite.IPAndEOJ, success echonet_lite.Properties, failed []echonet_lite.EPCType) []echonet_lite.EPCType {
	key := device.Key()

	s.mu.Lock()
	defer s.mu.Unlock()

	// 既存の失敗リストを取得 (存在しない場合は空のスライス)
	existingFailedEPCs := make([]echonet_lite.EPCType, 0)
	if f, ok := s.failedEPCs[key]; ok {
		existingFailedEPCs = append(existingFailedEPCs, f...) // コピーを作成
	}

	// 1. 今回成功したEPCを既存の失敗リストから削除する
	if len(success) > 0 {
		remainingFailedEPCs := make([]echonet_lite.EPCType, 0, len(existingFailedEPCs))
		successEPCs := make(map[echonet_lite.EPCType]struct{}, len(success))
		for _, p := range success {
			successEPCs[p.EPC] = struct{}{}
		}
		for _, epc := range existingFailedEPCs {
			if _, found := successEPCs[epc]; !found {
				remainingFailedEPCs = append(remainingFailedEPCs, epc)
			}
		}
		existingFailedEPCs = remainingFailedEPCs // 更新
	}

	// 2. 今回失敗したEPCのうち、まだ記録されていないものを追加し、戻り値リストを作成する
	newlyFailedForReturn := make([]echonet_lite.EPCType, 0, len(failed))
	if len(failed) > 0 {
		currentFailedSet := make(map[echonet_lite.EPCType]struct{}, len(existingFailedEPCs))
		for _, epc := range existingFailedEPCs {
			currentFailedSet[epc] = struct{}{}
		}

		for _, epc := range failed {
			if _, found := currentFailedSet[epc]; !found {
				// まだ記録されていない失敗
				existingFailedEPCs = append(existingFailedEPCs, epc)     // 内部状態に追加
				newlyFailedForReturn = append(newlyFailedForReturn, epc) // 戻り値リストに追加
				currentFailedSet[epc] = struct{}{}                       // Setにも追加して重複を防ぐ
			}
		}
	}

	// 3. 更新された失敗リストをマップに保存（空なら削除）
	if len(existingFailedEPCs) == 0 {
		delete(s.failedEPCs, key)
	} else {
		s.failedEPCs[key] = existingFailedEPCs
	}

	return newlyFailedForReturn // 今回新たに失敗として記録されたEPCのみを返す
}

// GetProperties - プロパティ取得
func (s *Session) GetProperties(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	EPCs []echonet_lite.EPCType,
) (bool, echonet_lite.Properties, []echonet_lite.EPCType, error) {
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
	success := respMsg.ESV == echonet_lite.ESVGet_Res

	// 成功/失敗のプロパティを分類
	successProperties := make(echonet_lite.Properties, 0, len(respMsg.Properties))
	failedEPCs := make([]echonet_lite.EPCType, 0, len(respMsg.Properties))

	for _, p := range respMsg.Properties {
		if p.EDT != nil {
			successProperties = append(successProperties, p)
		} else {
			failedEPCs = append(failedEPCs, p.EPC)
		}
	}

	failedEPCs = s.updateFailedEPCs(device, successProperties, failedEPCs)

	return success, successProperties, failedEPCs, nil
}

// SetProperties - プロパティ設定
func (s *Session) SetProperties(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	properties echonet_lite.Properties,
) (bool, echonet_lite.Properties, []echonet_lite.EPCType, error) {
	// メッセージを作成
	msg := s.CreateSetPropertyMessage(device, properties)

	// 共通処理を呼び出し
	respMsg, err := s.sendRequestWithContext(ctx, device, msg)

	// エラーチェック
	if err != nil {
		// タイムアウトやコンテキストキャンセルの場合
		failedEPCs := make([]echonet_lite.EPCType, 0, len(properties))
		for _, p := range properties {
			failedEPCs = append(failedEPCs, p.EPC)
		}
		return false, nil, failedEPCs, err
	}

	// 応答を処理
	success := respMsg.ESV == echonet_lite.ESVSet_Res

	// 成功/失敗のプロパティを分類
	successProperties := make(echonet_lite.Properties, 0, len(properties))
	failedEPCs := make([]echonet_lite.EPCType, 0, len(properties))

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
